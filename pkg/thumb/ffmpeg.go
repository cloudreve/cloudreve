package thumb

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

const (
	urlTimeout = time.Duration(1) * time.Hour
)

func NewFfmpegGenerator(l logging.Logger, settings setting.Provider) *FfmpegGenerator {
	return &FfmpegGenerator{l: l, settings: settings}
}

type FfmpegGenerator struct {
	l        logging.Logger
	settings setting.Provider
}

func (f *FfmpegGenerator) Generate(ctx context.Context, es entitysource.EntitySource, ext string, previous *Result) (*Result, error) {
	if !util.IsInExtensionListExt(f.settings.FFMpegThumbExts(ctx), ext) {
		return nil, fmt.Errorf("unsupported video format: %w", ErrPassThrough)
	}

	if es.Entity().Size() > f.settings.FFMpegThumbMaxSize(ctx) {
		return nil, fmt.Errorf("file is too big: %w", ErrPassThrough)
	}

	tempOutputPath := filepath.Join(
		util.DataPath(f.settings.TempPath(ctx)),
		thumbTempFolder,
		fmt.Sprintf("thumb_%s.png", uuid.Must(uuid.NewV4()).String()),
	)

	if err := util.CreatNestedFolder(filepath.Dir(tempOutputPath)); err != nil {
		return nil, fmt.Errorf("failed to create temp folder: %w", err)
	}

	input := ""
	expire := time.Now().Add(urlTimeout)
	if es.IsLocal() && !es.Entity().Encrypted() {
		input = es.LocalPath(ctx)
	} else {
		opts := []entitysource.EntitySourceOption{
			entitysource.WithContext(ctx),
			entitysource.WithExpire(&expire),
		}
		if !es.Entity().Encrypted() {
			opts = append(opts, entitysource.WithNoInternalProxy())
		}
		src, err := es.Url(ctx, opts...)
		if err != nil {
			return &Result{Path: tempOutputPath}, fmt.Errorf("failed to get entity url: %w", err)
		}

		input = src.Url
	}

	// Invoke ffmpeg
	w, h := f.settings.ThumbSize(ctx)
	scaleOpt := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", w, h)
	args := []string{
		"-ss", f.settings.FFMpegThumbSeek(ctx),
	}

	extraArgs := f.settings.FFMpegExtraArgs(ctx)
	if extraArgs != "" {
		args = append(args, strings.Split(extraArgs, " ")...)
	}

	args = append(args, []string{
		"-i", input,
		"-vf", scaleOpt,
		"-vframes", "1",
		tempOutputPath,
	}...)
	cmd := exec.CommandContext(ctx, f.settings.FFMpegPath(ctx), args...)

	// Redirect IO
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		f.l.Warning("Failed to invoke ffmpeg: %s", stdErr.String())
		return &Result{Path: tempOutputPath}, fmt.Errorf("failed to invoke ffmpeg: %w, raw output: %s", err, stdErr.String())
	}

	return &Result{Path: tempOutputPath}, nil
}

func (f *FfmpegGenerator) Priority() int {
	return 200
}

func (f *FfmpegGenerator) Enabled(ctx context.Context) bool {
	return f.settings.FFMpegThumbGeneratorEnabled(ctx)
}
