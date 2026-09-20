package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/samber/lo"
)

type (
	RemoteDownloadTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *RemoteDownloadTaskState
		node     cluster.Node
		d        downloader.Downloader
		progress queue.Progresses
	}
	RemoteDownloadTaskPhase string
	RemoteDownloadTaskState struct {
		SrcFileUri string `json:"src_file_uri,omitempty"`
		SrcUri     string `json:"src_uri,omitempty"`
		Dst        string `json:"dst,omitempty"`
		FileName   string `json:"file_name,omitempty"`
		// Provider pins the task to a node offering this downloader provider
		// (e.g. "aria2"/"qbittorrent"). Empty = pool picks any capable node.
		Provider           string                 `json:"provider,omitempty"`
		HTTPUsername       string                 `json:"http_username,omitempty"`
		HTTPPassword       string                 `json:"http_password,omitempty"`
		HTTPHeaders        []string               `json:"http_headers,omitempty"`
		Handle             *downloader.TaskHandle `json:"handle,omitempty"`
		Status             *downloader.TaskStatus `json:"status,omitempty"`
		NodeState          `json:",inline"`
		Phase              RemoteDownloadTaskPhase `json:"phase,omitempty"`
		SlaveUploadTaskID  int                     `json:"slave__upload_task_id,omitempty"`
		SlaveUploadState   *SlaveUploadTaskState   `json:"slave_upload_state,omitempty"`
		GetTaskStatusTried int                     `json:"get_task_status_tried,omitempty"`
		Transferred        map[int]interface{}     `json:"transferred,omitempty"`
		Failed             int                     `json:"failed,omitempty"`
		// ResumeMonitorAfterTransfer marks an early transfer batch taken while
		// the download is still running: after it completes, return to Monitor
		// instead of awaiting seeding.
		ResumeMonitorAfterTransfer bool `json:"resume_monitor_after_transfer,omitempty"`
	}
)

const (
	RemoteDownloadTaskPhaseNotStarted   RemoteDownloadTaskPhase = ""
	RemoteDownloadTaskPhaseMonitor                              = "monitor"
	RemoteDownloadTaskPhaseTransfer                             = "transfer"
	RemoteDownloadTaskPhaseAwaitSeeding                         = "seeding"

	GetTaskStatusMaxTries = 5

	// maxDownloadFiles bounds the number of files a single download task
	// may import, guarding against crafted torrents that declare absurd
	// file counts.
	maxDownloadFiles = 10_000

	SummaryKeyDownloadStatus = "download"
	SummaryKeySrcStr         = "src_str"

	ProgressTypeRelocateTransferCount = "relocate"
	ProgressTypeUploadSinglePrefix    = "upload_single_"

	SummaryKeySrcMultiple    = "src_multiple"
	SummaryKeySrcDstPolicyID = "dst_policy_id"
	SummaryKeyDstPolicyName  = "dst_policy_name"
	SummaryKeyFailed         = "failed"
	SummaryKeyTotal          = "total"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.RemoteDownloadTaskType, NewRemoteDownloadTaskFromModel)
}

// RemoteDownloadTaskOption carries optional per-task parameters supplied by
// the user: a custom output file name and HTTP basic-auth credentials for
// plain HTTP(S) sources.
type RemoteDownloadTaskOption struct {
	FileName     string
	Provider     string
	HTTPUsername string
	HTTPPassword string
	HTTPHeaders  []string
	// NodeSel carries the user/group node constraints for dispatch.
	NodeSel NodeSelection
}

// NewRemoteDownloadTask creates a new RemoteDownloadTask
func NewRemoteDownloadTask(ctx context.Context, src string, srcFile, dst string, opts *RemoteDownloadTaskOption) (queue.Task, error) {
	state := &RemoteDownloadTaskState{
		SrcUri:     src,
		SrcFileUri: srcFile,
		Dst:        dst,
		NodeState:  NodeState{},
	}
	if opts != nil {
		state.FileName = sanitizeFileName(opts.FileName)
		state.Provider = opts.Provider
		state.HTTPUsername = opts.HTTPUsername
		state.HTTPPassword = opts.HTTPPassword
		state.HTTPHeaders = opts.HTTPHeaders
		state.NodeState = opts.NodeSel.state()
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &RemoteDownloadTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.RemoteDownloadTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func NewRemoteDownloadTaskFromModel(task *ent.Task) queue.Task {
	return &RemoteDownloadTask{
		DBTask: &queue.DBTask{
			Task: task,
		},
	}
}

func (m *RemoteDownloadTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	// unmarshal state
	state := &RemoteDownloadTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	// Resolve a user-picked downloader provider to a preferred node. Runs only
	// until a node is locked in; falls back to any capable node if no match.
	if m.state.NodeID == 0 && m.state.Provider != "" {
		m.state.NodeID = providerNodeID(ctx, dep, m.state.Provider)
	}

	// select node
	node, err := allocateNode(ctx, dep, &m.state.NodeState, types.NodeCapabilityRemoteDownload)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to allocate node: %w", err)
	}
	m.node = node

	// create downloader instance
	if m.d == nil {
		d, err := node.CreateDownloader(ctx, dep.RequestClient(), dep.SettingProvider())
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to create downloader: %w", err)
		}

		m.d = d
	}

	next := task.StatusCompleted
	switch m.state.Phase {
	case RemoteDownloadTaskPhaseNotStarted:
		next, err = m.createDownloadTask(ctx, dep)
	case RemoteDownloadTaskPhaseMonitor, RemoteDownloadTaskPhaseAwaitSeeding:
		next, err = m.monitor(ctx, dep)
	case RemoteDownloadTaskPhaseTransfer:
		if m.node.IsMaster() {
			next, err = m.masterTransfer(ctx, dep)
		} else {
			next, err = m.slaveTransfer(ctx, dep)
		}
	}

	newStateStr, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	m.Lock()
	m.Task.PrivateState = string(newStateStr)
	m.Unlock()
	return next, err
}

func (m *RemoteDownloadTask) createDownloadTask(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	if m.state.Handle != nil {
		m.state.Phase = RemoteDownloadTaskPhaseMonitor
		return task.StatusSuspending, nil
	}

	user := inventory.UserFromContext(ctx)
	torrentUrl := m.state.SrcUri

	// SSRF policy is owned by the node that actually performs the fetch.
	// Validate the user-supplied URL (m.state.SrcUri) against the assigned
	// node's URLValidation policy, with the operator-configured site URL
	// host(s) always added to the allowlist. We don't validate SrcFileUri
	// resolution because that's a Cloudreve-internal entity URL pointing at
	// the user's own torrent file.
	if m.state.SrcUri != "" {
		opt := buildSSRFOptions(ctx, dep, m.node.Settings(ctx))
		if err := request.ValidateExternalURL(ctx, m.state.SrcUri, opt); err != nil {
			return task.StatusError, fmt.Errorf("url rejected: %s (%w)", err, queue.CriticalErr)
		}
	}

	if m.state.SrcFileUri != "" {
		// Target is a torrent file
		uri, err := fs.NewUriFromString(m.state.SrcFileUri)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to parse src file uri: %s (%w)", err, queue.CriticalErr)
		}

		fm := manager.NewFileManager(dep, user)
		expire := time.Now().Add(dep.SettingProvider().EntityUrlValidDuration(ctx))
		torrentUrls, _, err := fm.GetEntityUrls(ctx, []manager.GetEntityUrlArgs{
			{URI: uri},
		}, fs.WithUrlExpire(&expire))
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to get torrent entity urls: %w", err)
		}

		if len(torrentUrls) == 0 {
			return task.StatusError, fmt.Errorf("no torrent urls found")
		}

		torrentUrl = torrentUrls[0].Url
	}

	options, taskUrl := m.buildDownloadOptions(ctx, user.Edges.Group.Settings.RemoteDownloadOptions, torrentUrl)

	// Create download task
	handle, err := m.d.CreateTask(ctx, taskUrl, options)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to create download task: %w", err)
	}

	m.state.Handle = handle
	m.state.Phase = RemoteDownloadTaskPhaseMonitor
	return task.StatusSuspending, nil
}

// buildDownloadOptions overlays per-task options (custom file name, HTTP
// credentials) onto the group's remote-download options. Custom name and
// credentials only apply to plain HTTP(S) source URLs on aria2; qBittorrent
// accepts a torrent rename and carries HTTP auth in the URL userinfo.
func (m *RemoteDownloadTask) buildDownloadOptions(ctx context.Context, base map[string]interface{}, srcUrl string) (map[string]interface{}, string) {
	if m.state.FileName == "" && m.state.HTTPUsername == "" && len(m.state.HTTPHeaders) == 0 {
		return base, srcUrl
	}

	options := maps.Clone(base)
	if options == nil {
		options = map[string]interface{}{}
	}

	isHttpSrc := m.state.SrcFileUri == "" && (strings.HasPrefix(m.state.SrcUri, "http://") || strings.HasPrefix(m.state.SrcUri, "https://"))
	switch m.node.Settings(ctx).Provider {
	case types.DownloaderProviderQBittorrent:
		if m.state.FileName != "" {
			options["rename"] = m.state.FileName
		}
		if isHttpSrc {
			if m.state.HTTPUsername != "" {
				if u, err := url.Parse(srcUrl); err == nil {
					u.User = url.UserPassword(m.state.HTTPUsername, m.state.HTTPPassword)
					srcUrl = u.String()
				}
			}
			// qBittorrent's add API only accepts a cookie field, not arbitrary
			// headers — pass through any Cookie: line the user supplied.
			for _, h := range m.state.HTTPHeaders {
				if k, v, ok := strings.Cut(h, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "cookie") {
					options["cookie"] = strings.TrimSpace(v)
				}
			}
		}
	case types.DownloaderProviderYtDlp:
		if isHttpSrc {
			// "output" becomes a yt-dlp -o template: reject path separators
			// and "%(" template fields so a user-chosen name cannot escape
			// the task temp dir or expand yt-dlp metadata.
			if m.state.FileName != "" && !strings.ContainsAny(m.state.FileName, `/\`) &&
				!strings.Contains(m.state.FileName, "%(") && m.state.FileName != ".." {
				options["output"] = m.state.FileName
			}
			if m.state.HTTPUsername != "" {
				options["username"] = m.state.HTTPUsername
				options["password"] = m.state.HTTPPassword
			}
			if len(m.state.HTTPHeaders) > 0 {
				options["add_headers"] = m.state.HTTPHeaders
			}
		}
	default:
		if isHttpSrc {
			if m.state.FileName != "" {
				options["out"] = m.state.FileName
			}
			if m.state.HTTPUsername != "" {
				options["http-user"] = m.state.HTTPUsername
				options["http-passwd"] = m.state.HTTPPassword
			}
			if len(m.state.HTTPHeaders) > 0 {
				options["header"] = m.state.HTTPHeaders
			}
		}
	}
	return options, srcUrl
}

// buildSSRFOptions composes the SSRF policy for a download: the assigned
// node's URLValidation settings, plus the operator-configured site URL hosts
// (always allowlisted so users can fetch files served by Cloudreve itself).
func buildSSRFOptions(ctx context.Context, dep dependency.Dep, node *types.NodeSetting) request.SSRFOptions {
	opt := request.SSRFOptions{}
	if node != nil && node.URLValidation != nil {
		opt.Disabled = node.URLValidation.Disabled
		opt.AllowedHosts = append(opt.AllowedHosts, node.URLValidation.AllowedHosts...)
		opt.AllowedCIDRs = append(opt.AllowedCIDRs, node.URLValidation.AllowedCIDRs...)
	}
	for _, u := range dep.SettingProvider().AllSiteURLs(ctx) {
		if h := u.Hostname(); h != "" {
			opt.AllowedHosts = append(opt.AllowedHosts, h)
		}
	}
	return opt
}

func (m *RemoteDownloadTask) monitor(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	resumeAfter := time.Duration(m.node.Settings(ctx).Interval) * time.Second

	// Update task status
	status, err := m.d.Info(ctx, m.state.Handle)
	if err != nil {
		if errors.Is(err, downloader.ErrTaskNotFount) && m.state.Status != nil {
			// If task is not found, but it previously existed, consider it as canceled
			m.l.Warning("task not found, consider it as canceled")
			return task.StatusCanceled, nil
		}

		m.state.GetTaskStatusTried++
		if m.state.GetTaskStatusTried >= GetTaskStatusMaxTries {
			return task.StatusError, fmt.Errorf("failed to get task status after %d retry: %w", m.state.GetTaskStatusTried, err)
		}

		m.l.Warning("failed to get task info: %s, will retry.", err)
		m.ResumeAfter(resumeAfter)
		return task.StatusSuspending, nil
	}

	// Follow to new handle if needed
	if status.FollowedBy != nil {
		m.l.Info("Task handle updated to %v", status.FollowedBy)
		m.state.Handle = status.FollowedBy
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}

	if m.state.Status == nil || m.state.Status.Total != status.Total {
		m.l.Info("download size changed, re-validate files.")
		// Group per-task volume cap: abort once the resolved size exceeds it.
		var maxSize int64
		if u := inventory.UserFromContext(ctx); u != nil && u.Edges.Group != nil {
			maxSize = u.Edges.Group.Settings.Aria2MaxFileSize
		}
		if maxSize > 0 && status.Total > maxSize {
			m.state.Status = status
			return task.StatusError, fmt.Errorf("download size %d exceeds group limit %d (%w)", status.Total, maxSize, queue.CriticalErr)
		}
		// First time to get status / total size changed, check user capacity
		if err := m.validateFiles(ctx, dep, status); err != nil {
			m.state.Status = status
			return task.StatusError, fmt.Errorf("failed to validate files: %s (%w)", err, queue.CriticalErr)
		}
	}

	m.state.Status = status
	m.state.GetTaskStatusTried = 0

	m.l.Debug("Monitor %q task state: %s", status.Name, status.State)
	switch status.State {
	case downloader.StatusSeeding:
		m.l.Info("Download task seeding")
		if m.state.Phase == RemoteDownloadTaskPhaseMonitor {
			// Not transferred
			m.state.Phase = RemoteDownloadTaskPhaseTransfer
			return task.StatusSuspending, nil
		} else if !m.node.Settings(ctx).WaitForSeeding {
			// Skip seeding
			m.l.Info("Download task seeding skipped.")
			return task.StatusCompleted, nil
		} else {
			// Still seeding
			m.ResumeAfter(resumeAfter)
			return task.StatusSuspending, nil
		}
	case downloader.StatusCompleted:
		m.l.Info("Download task completed")
		if m.state.Phase == RemoteDownloadTaskPhaseMonitor {
			// Not transferred
			m.state.Phase = RemoteDownloadTaskPhaseTransfer
			return task.StatusSuspending, nil
		}
		// Seeding complete
		m.l.Info("Download task seeding completed")
		return task.StatusCompleted, nil
	case downloader.StatusDownloading:
		if m.hasEarlyTransferCandidates(status) {
			m.l.Info("Some files already completed, starting early transfer.")
			m.state.Phase = RemoteDownloadTaskPhaseTransfer
			m.state.ResumeMonitorAfterTransfer = true
			m.ResumeAfter(0)
			return task.StatusSuspending, nil
		}
		m.ResumeAfter(resumeAfter)
		return task.StatusSuspending, nil
	case downloader.StatusUnknown, downloader.StatusError:
		return task.StatusError, fmt.Errorf("download task failed with state %q (%w), errorMsg: %s", status.State, queue.CriticalErr, status.ErrorMessage)
	}

	m.ResumeAfter(resumeAfter)
	return task.StatusSuspending, nil
}

// hasEarlyTransferCandidates reports whether any selected file finished
// downloading but has not been transferred to the user's storage yet.
func (m *RemoteDownloadTask) hasEarlyTransferCandidates(status *downloader.TaskStatus) bool {
	for _, f := range status.Files {
		if f.Selected && f.Progress >= 1 {
			if _, ok := m.state.Transferred[f.Index]; !ok {
				return true
			}
		}
	}
	return false
}

func (m *RemoteDownloadTask) slaveTransfer(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	u := inventory.UserFromContext(ctx)
	if m.state.Transferred == nil {
		m.state.Transferred = make(map[int]interface{})
	}

	if m.state.SlaveUploadTaskID == 0 {
		dstUri, err := fs.NewUriFromString(m.state.Dst)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to parse dst uri %q: %s (%w)", m.state.Dst, err, queue.CriticalErr)
		}

		// Create slave upload task
		payload := &SlaveUploadTaskState{
			Files:       []SlaveUploadEntity{},
			MaxParallel: dep.SettingProvider().MaxParallelTransfer(ctx),
			UserID:      u.ID,
		}

		// Construct files to be transferred
		for _, f := range m.state.Status.Files {
			if !f.Selected {
				continue
			}

			// Skip already transferred
			if _, ok := m.state.Transferred[f.Index]; ok {
				continue
			}

			// During early transfer batches only completed files are picked.
			if m.state.ResumeMonitorAfterTransfer && f.Progress < 1 {
				continue
			}

			dst := dstUri.JoinRaw(sanitizeFileName(f.Name))
			src := path.Join(m.state.Status.SavePath, f.Name)
			payload.Files = append(payload.Files, SlaveUploadEntity{
				Src:   src,
				Uri:   dst,
				Size:  f.Size,
				Index: f.Index,
			})
		}

		payloadStr, err := json.Marshal(payload)
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to marshal payload: %w", err)
		}

		taskId, err := m.node.CreateTask(ctx, queue.SlaveUploadTaskType, string(payloadStr))
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to create slave task: %w", err)
		}

		m.state.NodeState.progress = nil
		m.state.SlaveUploadTaskID = taskId
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}

	m.l.Info("Checking slave upload task %d...", m.state.SlaveUploadTaskID)
	t, err := m.node.GetTask(ctx, m.state.SlaveUploadTaskID, true)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get slave task: %w", err)
	}

	m.Lock()
	m.state.NodeState.progress = t.Progress
	m.Unlock()

	m.state.SlaveUploadState = &SlaveUploadTaskState{}
	if err := json.Unmarshal([]byte(t.PrivateState), m.state.SlaveUploadState); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal slave compress state: %s (%w)", err, queue.CriticalErr)
	}

	if t.Status == task.StatusError || t.Status == task.StatusCompleted {
		if len(m.state.SlaveUploadState.Transferred) < len(m.state.SlaveUploadState.Files) {
			// Not all files transferred, retry
			slaveTaskId := m.state.SlaveUploadTaskID
			m.state.SlaveUploadTaskID = 0
			for i, _ := range m.state.SlaveUploadState.Transferred {
				m.state.Transferred[m.state.SlaveUploadState.Files[i].Index] = struct{}{}
			}

			m.l.Warning("Slave task %d failed to transfer %d files, retrying...", slaveTaskId, len(m.state.SlaveUploadState.Files)-len(m.state.SlaveUploadState.Transferred))
			if m.state.ResumeMonitorAfterTransfer {
				// Early transfer batch failed while the download continues -
				// return to monitoring; the final transfer pass retries them.
				m.state.ResumeMonitorAfterTransfer = false
				m.state.Phase = RemoteDownloadTaskPhaseMonitor
				m.ResumeAfter(0)
				return task.StatusSuspending, nil
			}
			return task.StatusError, fmt.Errorf(
				"slave task failed to transfer %d files, first 5 errors: %s",
				len(m.state.SlaveUploadState.Files)-len(m.state.SlaveUploadState.Transferred),
				m.state.SlaveUploadState.First5TransferErrors,
			)
		} else {
			if m.state.ResumeMonitorAfterTransfer {
				for i := range m.state.SlaveUploadState.Transferred {
					m.state.Transferred[m.state.SlaveUploadState.Files[i].Index] = struct{}{}
				}
				m.state.SlaveUploadTaskID = 0
				m.state.ResumeMonitorAfterTransfer = false
				m.state.Phase = RemoteDownloadTaskPhaseMonitor
				m.ResumeAfter(0)
				return task.StatusSuspending, nil
			}
			m.state.Phase = RemoteDownloadTaskPhaseAwaitSeeding
			m.ResumeAfter(0)
			return task.StatusSuspending, nil
		}
	}

	if t.Status == task.StatusCanceled {
		return task.StatusError, fmt.Errorf("slave task canceled (%w)", queue.CriticalErr)
	}

	m.l.Info("Slave task %d is still uploading, resume after 30s.", m.state.SlaveUploadTaskID)
	m.ResumeAfter(time.Second * 30)
	return task.StatusSuspending, nil
}

func (m *RemoteDownloadTask) masterTransfer(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	if m.state.Transferred == nil {
		m.state.Transferred = make(map[int]interface{})
	}

	maxParallel := dep.SettingProvider().MaxParallelTransfer(ctx)
	wg := sync.WaitGroup{}
	worker := make(chan int, maxParallel)
	for i := 0; i < maxParallel; i++ {
		worker <- i
	}

	// Sum up total count and select files
	totalCount := 0
	totalSize := int64(0)
	allFiles := make([]downloader.TaskFile, 0, len(m.state.Status.Files))
	for _, f := range m.state.Status.Files {
		if f.Selected {
			// Early transfer batches only pick completed files; files still
			// downloading are left for the final transfer pass.
			if m.state.ResumeMonitorAfterTransfer && f.Progress < 1 {
				continue
			}
			allFiles = append(allFiles, f)
			totalSize += f.Size
			totalCount++
		}
	}

	m.Lock()
	m.progress = make(queue.Progresses)
	m.progress[ProgressTypeUploadCount] = &queue.Progress{Total: int64(totalCount)}
	m.progress[ProgressTypeUpload] = &queue.Progress{Total: totalSize}
	m.Unlock()

	dstUri, err := fs.NewUriFromString(m.state.Dst)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse dst uri: %s (%w)", err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)
	failed := int64(0)
	ae := serializer.NewAggregateError()

	transferFunc := func(workerId int, file downloader.TaskFile) {
		sanitizedName := sanitizeFileName(file.Name)
		dst := dstUri.JoinRaw(sanitizedName)
		src := filepath.FromSlash(path.Join(m.state.Status.SavePath, file.Name))
		m.l.Info("Uploading file %s to %s...", src, sanitizedName, dst)

		progressKey := fmt.Sprintf("%s%d", ProgressTypeUploadSinglePrefix, workerId)
		m.Lock()
		m.progress[progressKey] = &queue.Progress{Identifier: dst.String(), Total: file.Size}
		fileProgress := m.progress[progressKey]
		uploadProgress := m.progress[ProgressTypeUpload]
		uploadCountProgress := m.progress[ProgressTypeUploadCount]
		m.Unlock()

		defer func() {
			atomic.AddInt64(&uploadCountProgress.Current, 1)
			worker <- workerId
			wg.Done()
		}()

		fileStream, err := os.Open(src)
		if err != nil {
			m.l.Warning("Failed to open file %s: %s", src, err.Error())
			atomic.AddInt64(&uploadProgress.Current, file.Size)
			atomic.AddInt64(&failed, 1)
			ae.Add(file.Name, fmt.Errorf("failed to open file: %w", err))
			return
		}

		defer fileStream.Close()

		fileData := &fs.UploadRequest{
			Props: &fs.UploadProps{
				Uri:  dst,
				Size: file.Size,
			},
			ProgressFunc: func(current, diff int64, total int64) {
				atomic.AddInt64(&fileProgress.Current, diff)
				atomic.AddInt64(&uploadProgress.Current, diff)
			},
			File: fileStream,
		}

		_, err = fm.Update(ctx, fileData, fs.WithNoEntityType())
		if err != nil {
			m.l.Warning("Failed to upload file %s: %s", src, err.Error())
			atomic.AddInt64(&failed, 1)
			atomic.AddInt64(&uploadProgress.Current, file.Size)
			ae.Add(file.Name, fmt.Errorf("failed to upload file: %w", err))
			return
		}

		m.Lock()
		m.state.Transferred[file.Index] = nil
		m.Unlock()
	}

	// Start upload files
	for _, file := range allFiles {
		// Check if file is already transferred
		if _, ok := m.state.Transferred[file.Index]; ok {
			m.l.Info("File %s already transferred, skipping...", file.Name)
			m.Lock()
			atomic.AddInt64(&m.progress[ProgressTypeUpload].Current, file.Size)
			atomic.AddInt64(&m.progress[ProgressTypeUploadCount].Current, 1)
			m.Unlock()
			continue
		}

		select {
		case <-ctx.Done():
			return task.StatusError, ctx.Err()
		case workerId := <-worker:
			wg.Add(1)

			go transferFunc(workerId, file)
		}
	}

	wg.Wait()
	if failed > 0 {
		if m.state.ResumeMonitorAfterTransfer {
			// Early batch failed while the download continues - return to
			// monitoring; the final transfer pass retries the failed files.
			m.l.Warning("Early transfer batch failed for %d file(s), will retry after download completes.", failed)
			m.state.ResumeMonitorAfterTransfer = false
			m.state.Phase = RemoteDownloadTaskPhaseMonitor
			m.ResumeAfter(0)
			return task.StatusSuspending, nil
		}
		m.state.Failed = int(failed)
		m.l.Error("Failed to transfer %d file(s).", failed)
		return task.StatusError, fmt.Errorf("failed to transfer %d file(s), first 5 errors: %s", failed, ae.FormatFirstN(5))
	}

	m.l.Info("All files transferred.")
	if m.state.ResumeMonitorAfterTransfer {
		m.state.ResumeMonitorAfterTransfer = false
		m.state.Phase = RemoteDownloadTaskPhaseMonitor
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}
	m.state.Phase = RemoteDownloadTaskPhaseAwaitSeeding
	return task.StatusSuspending, nil
}

func (m *RemoteDownloadTask) awaitSeeding(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	return task.StatusSuspending, nil
}

func (m *RemoteDownloadTask) validateFiles(ctx context.Context, dep dependency.Dep, status *downloader.TaskStatus) error {
	// Validate files
	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)

	dstUri, err := fs.NewUriFromString(m.state.Dst)
	if err != nil {
		return fmt.Errorf("failed to parse dst uri: %w", err)
	}

	selectedFiles := lo.Filter(status.Files, func(f downloader.TaskFile, _ int) bool {
		return f.Selected
	})
	if len(selectedFiles) == 0 {
		return fmt.Errorf("no selected file found in download task")
	}

	// Bound the entity count a single task can create — a crafted torrent
	// declaring hundreds of thousands of files is a database bomb.
	if len(selectedFiles) > maxDownloadFiles {
		return fmt.Errorf("download task selects %d files, exceeds the %d limit", len(selectedFiles), maxDownloadFiles)
	}

	validateArgs := lo.Map(selectedFiles, func(f downloader.TaskFile, _ int) fs.PreValidateFile {
		return fs.PreValidateFile{
			Name:     sanitizeFileName(f.Name),
			Size:     f.Size,
			OmitName: f.Name == "",
		}
	})

	if err := fm.PreValidateUpload(ctx, dstUri, validateArgs...); err != nil {
		return fmt.Errorf("failed to pre-validate files: %w", err)
	}

	return nil
}

func (m *RemoteDownloadTask) Cleanup(ctx context.Context) error {
	if m.state.Handle != nil {
		if err := m.d.Cancel(ctx, m.state.Handle); err != nil {
			m.l.Warning("failed to cancel download task: %s", err)
		}
	}

	if m.state.Status != nil && m.node.IsMaster() && m.state.Status.SavePath != "" {
		if err := os.RemoveAll(m.state.Status.SavePath); err != nil {
			m.l.Warning("failed to remove download temp folder: %s", err)
		}
	}

	return nil
}

// SetDownloadTarget sets the files to download for the task
func (m *RemoteDownloadTask) SetDownloadTarget(ctx context.Context, args ...*downloader.SetFileToDownloadArgs) error {
	if m.state.Handle == nil {
		return fmt.Errorf("download task not created")
	}

	return m.d.SetFilesToDownload(ctx, m.state.Handle, args...)
}

// CancelDownload cancels the download task
func (m *RemoteDownloadTask) CancelDownload(ctx context.Context) error {
	if m.state.Handle == nil {
		return nil
	}

	return m.d.Cancel(ctx, m.state.Handle)
}

func (m *RemoteDownloadTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	// unmarshal state
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	var status *downloader.TaskStatus
	if m.state.Status != nil {
		status = &*m.state.Status

		// Redact save path
		status.SavePath = ""
	}

	failed := m.state.Failed
	if m.state.SlaveUploadState != nil && m.state.Phase != RemoteDownloadTaskPhaseTransfer {
		failed = len(m.state.SlaveUploadState.Files) - len(m.state.SlaveUploadState.Transferred)
	}

	return &queue.Summary{
		Phase:  string(m.state.Phase),
		NodeID: m.state.NodeID,
		Props: map[string]any{
			SummaryKeySrcStr:         m.state.SrcUri,
			SummaryKeySrc:            m.state.SrcFileUri,
			SummaryKeyDst:            m.state.Dst,
			SummaryKeyFailed:         failed,
			SummaryKeyDownloadStatus: status,
		},
	}
}

func (m *RemoteDownloadTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()

	merged := make(queue.Progresses)
	for k, v := range m.progress {
		merged[k] = v
	}

	if m.state.NodeState.progress != nil {
		for k, v := range m.state.NodeState.progress {
			merged[k] = v
		}
	}

	return merged
}

func sanitizeFileName(name string) string {
	r := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return r.Replace(name)
}

// providerNodeID resolves a downloader provider name to the lowest-ID active
// node offering it for remote download. Returns 0 when no node matches, which
// makes the pool fall back to any capable node.
func providerNodeID(ctx context.Context, dep dependency.Dep, provider string) int {
	nodes, err := dep.NodeClient().ListActiveNodes(ctx, nil)
	if err != nil {
		return 0
	}
	return PickProviderNode(nodes, provider)
}

func PickProviderNode(nodes []*ent.Node, provider string) int {
	best := 0
	for _, n := range nodes {
		if n.Capabilities == nil || !n.Capabilities.Enabled(int(types.NodeCapabilityRemoteDownload)) ||
			n.Settings == nil || string(n.Settings.Provider) != provider {
			continue
		}
		if best == 0 || n.ID < best {
			best = n.ID
		}
	}
	return best
}
