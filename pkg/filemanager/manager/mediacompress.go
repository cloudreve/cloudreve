package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/mediaprocesstask"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/crontab"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

// skipMediaProcessEnqueueKey marks a context as originating from the media
// compression write-back, so CompleteUpload does not re-enqueue a compression
// task for the freshly written (already compressed) version — which would loop.
type skipMediaProcessEnqueueKey struct{}

func withSkipMediaProcessEnqueue(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipMediaProcessEnqueueKey{}, true)
}

func shouldSkipMediaProcessEnqueue(ctx context.Context) bool {
	v, _ := ctx.Value(skipMediaProcessEnqueueKey{}).(bool)
	return v
}

const mediaCompressTempFolder = "media_compress"

type (
	// MediaCompressTask is the queue task that compresses a single blob recorded
	// in the media_process_task table (APP-101). One task == one image.
	MediaCompressTask struct {
		*queue.DBTask

		l     logging.Logger
		state *MediaCompressTaskState
	}

	MediaCompressTaskState struct {
		// RowID is the media_process_task row id this task processes.
		RowID int `json:"row_id"`
	}
)

func init() {
	queue.RegisterResumableTaskFactory(queue.MediaCompressTaskType, NewMediaCompressTaskFromModel)
	queue.RegisterResumableTaskFactory(queue.MediaCompressVideoTaskType, NewMediaCompressTaskFromModel)

	// Cron: enqueue bounded batches of pending media onto their dedicated queues.
	// Image and video are independent lanes so either can run with the other off,
	// and a long video transcode never blocks image compression (APP-103). Mirrors
	// pkg/filemanager/manager/recycle.go.
	crontab.Register(setting.CronTypeMediaProcess, func(ctx context.Context) {
		dep := dependency.FromContext(ctx)
		sp := dep.SettingProvider()

		if mp := sp.MediaProcess(ctx); mp.ImageEnabled {
			enqueueMediaProcessLane(ctx, dep, mediaprocesstask.MediaTypeImage, mp.BatchSize,
				queue.MediaCompressTaskType, dep.MediaProcessQueue(ctx))
		}
		if v := sp.MediaProcessVideo(ctx); v.Enabled {
			enqueueMediaProcessLane(ctx, dep, mediaprocesstask.MediaTypeVideo, v.BatchSize,
				queue.MediaCompressVideoTaskType, dep.MediaVideoQueue(ctx))
		}
	})
}

// enqueueMediaProcessLane lists a bounded batch of pending rows of the given media
// type and enqueues one MediaCompressTask per row onto the provided queue, tagged
// with taskType so it lands on the right (image vs video) lane.
func enqueueMediaProcessLane(ctx context.Context, dep dependency.Dep, mediaType mediaprocesstask.MediaType, batchSize int, taskType string, q queue.Queue) {
	l := dep.Logger()
	rows, err := dep.MediaProcessClient().ListPending(ctx, mediaType, batchSize)
	if err != nil {
		l.Error("Failed to list pending media process tasks (%s): %s", mediaType, err)
		return
	}
	if len(rows) == 0 {
		return
	}

	uc := dep.UserClient()
	enqueued := 0
	for _, row := range rows {
		// The task must carry an owner (DBTask.Owner() is dereferenced when the
		// queue persists the task); attribute it to the blob's owner.
		owner, err := uc.GetByID(ctx, row.OwnerID)
		if err != nil {
			l.Error("Failed to load owner %d for media compress row %d: %s", row.OwnerID, row.ID, err)
			continue
		}
		t, err := newMediaCompressTask(ctx, row.ID, owner, taskType)
		if err != nil {
			l.Error("Failed to create media compress task for row %d: %s", row.ID, err)
			continue
		}
		if err := q.QueueTask(ctx, t); err != nil {
			l.Error("Failed to queue media compress task for row %d: %s", row.ID, err)
			continue
		}
		enqueued++
	}
	l.Info("Enqueued %d media compress task(s) (%s) from cron.", enqueued, mediaType)
}

func NewMediaCompressTask(ctx context.Context, rowID int, owner *ent.User) (queue.Task, error) {
	return newMediaCompressTask(ctx, rowID, owner, queue.MediaCompressTaskType)
}

// newMediaCompressTask builds a MediaCompressTask tagged with the given queue task
// type, which selects the lane it runs on: MediaCompressTaskType (image) →
// MediaProcessQueue, MediaCompressVideoTaskType (video) → MediaVideoQueue.
func newMediaCompressTask(ctx context.Context, rowID int, owner *ent.User, taskType string) (queue.Task, error) {
	state := &MediaCompressTaskState{RowID: rowID}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return &MediaCompressTask{
		DBTask: &queue.DBTask{
			// DirectOwner must be non-nil: the queue dereferences Owner().ID when
			// persisting the task (pkg/queue/task.go).
			DirectOwner: owner,
			Task: &ent.Task{
				Type:          taskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
		},
	}, nil
}

func NewMediaCompressTaskFromModel(t *ent.Task) queue.Task {
	return &MediaCompressTask{
		DBTask: &queue.DBTask{
			Task: t,
		},
	}
}

func (m *MediaCompressTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	state := &MediaCompressTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	mpClient := dep.MediaProcessClient()
	row, err := mpClient.GetByID(ctx, state.RowID)
	if err != nil {
		// Row gone (e.g. cleaned up): nothing to do.
		m.l.Warning("Media process task row %d not found, skipping: %s", state.RowID, err)
		return task.StatusCompleted, nil
	}

	// Idempotency: only act on a pending row.
	if row.Status != mediaprocesstask.StatusPending {
		return task.StatusCompleted, nil
	}

	// Master-switch gate per media type: image and video have independent switches
	// so one lane can be off while the other runs (APP-103).
	sp := dep.SettingProvider()
	enabled := sp.MediaProcess(ctx).ImageEnabled
	if row.MediaType == mediaprocesstask.MediaTypeVideo {
		enabled = sp.MediaProcessVideo(ctx).Enabled
	}
	if !enabled {
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped})
		return task.StatusCompleted, nil
	}

	// Only the write-back "version" mode is implemented in the MVP; replace/auto
	// are follow-ups (see ticket §7.1). Any other value falls back to version.
	if _, err := mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{
		Status:       mediaprocesstask.StatusProcessing,
		BumpAttempts: true,
	}); err != nil {
		return task.StatusError, fmt.Errorf("failed to mark processing: %w", err)
	}

	if err := m.compress(ctx, dep, row); err != nil {
		// Persist the error; the queue backoff will retry up to MaxRetry, after
		// which the row stays "processing" — a follow-up sweep can requeue it.
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{
			Status: mediaprocesstask.StatusFailed,
			Error:  err.Error(),
		})
		return task.StatusError, err
	}

	return task.StatusCompleted, nil
}

// compress runs the full pipeline for one row: read → compress/transcode →
// write-back as a new version → mark done. The read, size/idempotency guards,
// write-back and quota accounting are shared between image and video; only the
// engine step branches by media_type. Returns nil on success or a documented skip.
func (m *MediaCompressTask) compress(ctx context.Context, dep dependency.Dep, row *ent.MediaProcessTask) error {
	mpClient := dep.MediaProcessClient()
	sp := dep.SettingProvider()
	isVideo := row.MediaType == mediaprocesstask.MediaTypeVideo

	owner, err := dep.UserClient().GetByID(ctx, row.OwnerID)
	if err != nil {
		return fmt.Errorf("failed to load owner %d: %w", row.OwnerID, err)
	}

	fm := NewFileManager(dep, owner)
	defer fm.Recycle()

	es, err := fm.GetEntitySource(ctx, row.EntityID)
	if err != nil {
		// Blob gone (recycled/deleted): self-heal by skipping.
		m.l.Warning("Entity %d unavailable, skipping media compress: %s", row.EntityID, err)
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped, Error: err.Error()})
		return nil
	}
	defer es.Close()

	minSize := sp.MediaProcess(ctx).MinSize
	if isVideo {
		minSize = sp.MediaProcessVideo(ctx).MinSize
	}
	originalSize := es.Entity().Size()
	if originalSize < minSize {
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped})
		return nil
	}

	// Resolve the logical file for the write-back URI + name/ext.
	file, err := fm.TraverseFile(ctx, row.FileID)
	if err != nil {
		m.l.Warning("File %d unavailable, skipping media compress: %s", row.FileID, err)
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped, Error: err.Error()})
		return nil
	}
	sourceExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))

	// Idempotency signature + the props recorded on completion. Both branch by
	// media type: image encodes engine+quality+format, video encodes the ffmpeg
	// transcode parameters. A matching signature means the same parameters already
	// ran against the same original bytes → skip.
	var (
		signature string
		props     *types.MediaProcessTaskProps
	)
	if isVideo {
		v := sp.MediaProcessVideo(ctx)
		signature = fmt.Sprintf("ffmpeg|%s|crf%d|%s|%s|%s|%s|%d",
			v.Codec, v.CRF, v.Preset, v.Container, v.MaxResolution, v.AudioCodec, originalSize)
		props = &types.MediaProcessTaskProps{
			Engine: "ffmpeg", Codec: v.Codec, Quality: v.CRF, Format: v.Container,
			OriginalSize: originalSize, Signature: signature,
		}
	} else {
		mp := sp.MediaProcess(ctx)
		signature = fmt.Sprintf("%s|q%d|%s|%d", mp.Engine, mp.Quality, mp.Format, originalSize)
		props = &types.MediaProcessTaskProps{
			Engine: mp.Engine, Quality: mp.Quality, Format: mp.Format,
			OriginalSize: originalSize, Signature: signature,
		}
	}
	if row.Props != nil && row.Props.Signature == signature {
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped})
		return nil
	}

	// Materialize a local input path (download if the blob is remote/encrypted).
	tempDir := filepath.Join(util.DataPath(sp.TempPath(ctx)), mediaCompressTempFolder)
	if err := util.CreatNestedFolder(tempDir); err != nil {
		return fmt.Errorf("failed to create temp folder: %w", err)
	}
	inputPath, cleanupInput, err := materializeLocalInput(ctx, es, tempDir, sourceExt)
	if err != nil {
		return fmt.Errorf("failed to materialize input: %w", err)
	}
	defer cleanupInput()

	// Compress/transcode.
	var (
		outputPath string
		targetExt  string
	)
	if isVideo {
		outputPath, targetExt, err = compressVideo(ctx, sp.MediaProcessVideo(ctx), sp, inputPath, sourceExt, tempDir)
	} else {
		outputPath, targetExt, err = compressImage(ctx, sp.MediaProcess(ctx), sp, inputPath, sourceExt, tempDir)
	}
	if err != nil {
		if err == errUnsupportedFormat {
			m.l.Info("Unsupported %s format %q for row %d, skipping.", row.MediaType, sourceExt, row.ID)
			_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{Status: mediaprocesstask.StatusSkipped})
			return nil
		}
		return err
	}
	defer os.Remove(outputPath)

	outInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to stat compressed output: %w", err)
	}
	resultSize := outInfo.Size()

	// No gain (or larger): keep the original, mark done with the signature so it
	// is not retried with the same parameters. This guard is more often hit for
	// video (an already well-encoded clip may not shrink on re-encode).
	if resultSize == 0 || resultSize >= originalSize {
		m.l.Info("Compression yielded no gain for row %d (%d -> %d), keeping original.", row.ID, originalSize, resultSize)
		_, _ = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{
			Status:     mediaprocesstask.StatusDone,
			ResultSize: originalSize,
			Props:      props,
		})
		return nil
	}

	// Write-back as a new version via manager.Update. This reuses the whole upload
	// path (any driver, incl. remote) and applies the quota StorageDiff
	// automatically. The context guard prevents CompleteUpload from re-enqueuing.
	outFile, err := os.Open(outputPath)
	if err != nil {
		return fmt.Errorf("failed to open compressed output: %w", err)
	}
	defer outFile.Close()

	req := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:  file.Uri(false),
			Size: resultSize,
		},
		File:   outFile,
		Seeker: outFile,
		Mode:   fs.ModeOverwrite,
	}

	writeCtx := withSkipMediaProcessEnqueue(dbfs.WithBypassOwnerCheck(ctx))
	if _, err := fm.Update(writeCtx, req, fs.WithEntityType(types.EntityTypeVersion)); err != nil {
		return fmt.Errorf("failed to write back compressed version: %w", err)
	}

	_, err = mpClient.SetStatus(ctx, row.ID, &inventory.MediaProcessStatusArgs{
		Status:     mediaprocesstask.StatusDone,
		ResultSize: resultSize,
		Props:      props,
	})
	if err != nil {
		return fmt.Errorf("failed to mark done: %w", err)
	}

	m.l.Info("Compressed row %d (%s): %s %d -> %d bytes (target .%s).", row.ID, row.MediaType, sourceExt, originalSize, resultSize, targetExt)
	return nil
}

// materializeLocalInput returns a local filesystem path for the entity bytes. If
// the source is a local, unencrypted blob it uses its path directly (no cleanup);
// otherwise it streams the source into a temp file (cleaned up by the returned fn).
func materializeLocalInput(ctx context.Context, es entitySourceReader, tempDir, ext string) (string, func(), error) {
	if es.IsLocal() && !es.Entity().Encrypted() {
		return es.LocalPath(ctx), func() {}, nil
	}

	name := fmt.Sprintf("in_%s.%s", uuid.Must(uuid.NewV4()).String(), ext)
	tempInput := filepath.Join(tempDir, name)
	f, err := util.CreatNestedFile(tempInput)
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to create temp input: %w", err)
	}
	if _, err := io.Copy(f, es); err != nil {
		f.Close()
		os.Remove(tempInput)
		return "", func() {}, fmt.Errorf("failed to download entity: %w", err)
	}
	f.Close()

	return tempInput, func() { os.Remove(tempInput) }, nil
}

var errUnsupportedFormat = fmt.Errorf("unsupported image format")

// compressImage runs the configured engine on a local input file and returns the
// compressed output path + its extension. Engine default is vips (better for
// stills); ffmpeg is the alternative.
func compressImage(ctx context.Context, mp *setting.MediaProcessSetting, settings setting.Provider, inputPath, sourceExt, tempDir string) (string, string, error) {
	targetExt := normalizeFormat(mp.Format, sourceExt)
	if targetExt == "" {
		return "", "", errUnsupportedFormat
	}

	outputPath := filepath.Join(tempDir, fmt.Sprintf("out_%s.%s", uuid.Must(uuid.NewV4()).String(), targetExt))

	var (
		bin  string
		args []string
	)
	switch strings.ToLower(mp.Engine) {
	case "ffmpeg":
		bin = settings.FFMpegPath(ctx)
		a, err := ffmpegCompressArgs(inputPath, outputPath, targetExt, mp.Quality)
		if err != nil {
			return "", "", err
		}
		args = a
	default: // vips
		bin = settings.VipsPath(ctx)
		args = []string{"copy", inputPath, vipsOutputSpec(outputPath, targetExt, mp.Quality)}
	}

	if extra := strings.TrimSpace(mp.ExtraArgs); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return "", "", fmt.Errorf("failed to invoke %s: %w, output: %s", bin, err, stdErr.String())
	}

	return outputPath, targetExt, nil
}

// normalizeFormat resolves the target extension from the format setting. "keep"
// preserves the source extension (only for compressible raster types). An empty
// return means the source is not compressible with this configuration.
func normalizeFormat(format, sourceExt string) string {
	f := strings.ToLower(strings.TrimSpace(format))
	if f == "" || f == "keep" {
		f = sourceExt
	}
	switch f {
	case "jpg", "jpeg":
		return "jpg"
	case "webp":
		return "webp"
	case "png":
		return "png"
	default:
		return ""
	}
}

// vipsOutputSpec builds the vips output filename with save options that trigger
// re-encoding at the requested quality.
func vipsOutputSpec(outputPath, targetExt string, quality int) string {
	switch targetExt {
	case "jpg", "webp":
		return fmt.Sprintf("%s[Q=%d]", outputPath, quality)
	case "png":
		return fmt.Sprintf("%s[compression=9]", outputPath)
	default:
		return outputPath
	}
}

// ffmpegCompressArgs maps the quality (1..100, higher = better) onto ffmpeg's
// per-codec quality flag.
func ffmpegCompressArgs(inputPath, outputPath, targetExt string, quality int) ([]string, error) {
	base := []string{"-y", "-i", inputPath}
	switch targetExt {
	case "jpg":
		// mjpeg qscale: 2 (best) .. 31 (worst).
		q := 31 - int(float64(quality)/100.0*29.0)
		if q < 2 {
			q = 2
		}
		if q > 31 {
			q = 31
		}
		return append(base, "-q:v", strconv.Itoa(q), outputPath), nil
	case "webp":
		return append(base, "-quality", strconv.Itoa(quality), outputPath), nil
	case "png":
		return append(base, "-compression_level", "100", outputPath), nil
	default:
		return nil, errUnsupportedFormat
	}
}

// compressVideo transcodes a local video with ffmpeg (the only video engine; vips
// does not process video) and returns the output path + its extension. The target
// container defaults to "keep" (same extension as the source, so the file is not
// renamed). CPU is bounded by -threads and, when enabled, a low process priority
// via nice (best effort, non-Windows only).
func compressVideo(ctx context.Context, v *setting.MediaProcessVideoSetting, settings setting.Provider, inputPath, sourceExt, tempDir string) (string, string, error) {
	targetExt := normalizeVideoContainer(v.Container, sourceExt)
	if targetExt == "" {
		return "", "", errUnsupportedFormat
	}

	outputPath := filepath.Join(tempDir, fmt.Sprintf("out_%s.%s", uuid.Must(uuid.NewV4()).String(), targetExt))

	codec := strings.TrimSpace(v.Codec)
	if codec == "" {
		codec = "libx264"
	}
	args := []string{"-y", "-i", inputPath, "-c:v", codec, "-crf", strconv.Itoa(v.CRF)}
	if preset := strings.TrimSpace(v.Preset); preset != "" {
		args = append(args, "-preset", preset)
	}
	if scale := videoScaleFilter(v.MaxResolution); scale != "" {
		args = append(args, "-vf", scale)
	}
	if ac := strings.TrimSpace(v.AudioCodec); ac != "" {
		args = append(args, "-c:a", ac)
		if ab := strings.TrimSpace(v.AudioBitrate); ab != "" {
			args = append(args, "-b:a", ab)
		}
	}
	if v.Threads > 0 {
		args = append(args, "-threads", strconv.Itoa(v.Threads))
	}
	if extra := strings.TrimSpace(v.ExtraArgs); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	args = append(args, outputPath)

	bin := settings.FFMpegPath(ctx)
	// Yield CPU to real traffic by running the encode at low priority. `nice` is a
	// POSIX tool; skip it on Windows where it is unavailable.
	if v.Nice && runtime.GOOS != "windows" {
		args = append([]string{"-n", "10", bin}, args...)
		bin = "nice"
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return "", "", fmt.Errorf("failed to invoke %s: %w, output: %s", bin, err, stdErr.String())
	}

	return outputPath, targetExt, nil
}

// normalizeVideoContainer resolves the output container/extension. "keep" (default)
// preserves the source extension — but only for recognised video containers, so a
// non-video file yields "" and is skipped. An explicit container overrides it (the
// file is not renamed, matching the image caveat: the bytes change container but
// the name/extension stays — hence "keep" is the recommended default).
func normalizeVideoContainer(container, sourceExt string) string {
	c := strings.ToLower(strings.TrimSpace(container))
	if c == "" || c == "keep" {
		c = strings.ToLower(strings.TrimSpace(sourceExt))
	}
	switch c {
	case "mp4", "m4v", "mov", "mkv", "webm", "avi", "flv", "wmv", "mpeg", "mpg", "ts", "m2ts", "3gp", "ogv":
		return c
	default:
		return ""
	}
}

// videoScaleFilter builds an ffmpeg -vf scale filter that caps the output at the
// given WxH without upscaling and keeping the aspect ratio, forcing even
// dimensions (required by yuv420p). An empty or unparseable spec disables scaling.
func videoScaleFilter(maxResolution string) string {
	spec := strings.TrimSpace(maxResolution)
	if spec == "" {
		return ""
	}
	parts := strings.SplitN(strings.ToLower(spec), "x", 2)
	if len(parts) != 2 {
		return ""
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return ""
	}
	return fmt.Sprintf("scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2", w, h)
}

// entitySourceReader is the subset of entitysource.EntitySource the compression
// pipeline consumes (kept local to avoid widening the import surface).
type entitySourceReader interface {
	io.Reader
	IsLocal() bool
	LocalPath(ctx context.Context) string
	Entity() fs.Entity
}

// enqueueMediaProcessIfEligible registers the just-uploaded primary entity as a
// pending image-compression row, when the global switch + the owner's opt-in +
// mime/size gates all pass (APP-101). Called from CompleteUpload on the master.
// It is a no-op on the compression write-back path (context guard) so it never
// loops on its own output.
func (m *manager) enqueueMediaProcessIfEligible(ctx context.Context, session *fs.UploadSession, file fs.File) {
	if m.stateless || file == nil || m.user == nil || m.user.Settings == nil {
		return
	}
	if shouldSkipMediaProcessEnqueue(ctx) {
		return
	}

	mimeType := ""
	if session != nil && session.Props != nil {
		mimeType = session.Props.MimeType
	}
	if mimeType == "" {
		mimeType = m.dep.MimeDetector(ctx).TypeByName(file.Name())
	}
	mimeType = strings.ToLower(mimeType)

	// Route by mime to the matching media type; each has its own master switch,
	// per-user opt-in and min-size gate (APP-101 image, APP-103 video).
	var (
		mediaType mediaprocesstask.MediaType
		minSize   int64
	)
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		mp := m.settings.MediaProcess(ctx)
		if !mp.ImageEnabled || !m.user.Settings.AutoCompressImages {
			return
		}
		mediaType, minSize = mediaprocesstask.MediaTypeImage, mp.MinSize
	case strings.HasPrefix(mimeType, "video/"):
		v := m.settings.MediaProcessVideo(ctx)
		if !v.Enabled || !m.user.Settings.AutoCompressVideos {
			return
		}
		mediaType, minSize = mediaprocesstask.MediaTypeVideo, v.MinSize
	default:
		return
	}

	entity := file.PrimaryEntity()
	if entity == nil || entity.Size() < minSize {
		return
	}

	mpClient := m.dep.MediaProcessClient()
	if active, err := mpClient.HasActive(ctx, entity.ID()); err != nil {
		m.l.Warning("media process: active-row check failed for entity %d: %s", entity.ID(), err)
		return
	} else if active {
		return
	}

	if _, err := mpClient.Enqueue(ctx, &inventory.MediaProcessEnqueueArgs{
		EntityID:  entity.ID(),
		FileID:    file.ID(),
		OwnerID:   m.user.ID,
		MediaType: mediaType,
	}); err != nil {
		m.l.Warning("media process: failed to enqueue pending row for entity %d: %s", entity.ID(), err)
	}
}
