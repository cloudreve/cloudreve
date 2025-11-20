package thumb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

func NewVipsGenerator(l logging.Logger, settings setting.Provider) *VipsGenerator {
	return &VipsGenerator{l: l, settings: settings}
}

type VipsGenerator struct {
	l        logging.Logger
	settings setting.Provider
}

func (v *VipsGenerator) Generate(ctx context.Context, es entitysource.EntitySource, ext string, previous *Result) (*Result, error) {
	if !util.IsInExtensionListExt(v.settings.VipsThumbExts(ctx), ext) {
		return nil, fmt.Errorf("unsupported video format: %w", ErrPassThrough)
	}

	if es.Entity().Size() > v.settings.VipsThumbMaxSize(ctx) {
		return nil, fmt.Errorf("file is too big: %w", ErrPassThrough)
	}

	outputOpt := ".png"
	encode := v.settings.ThumbEncode(ctx)
	if encode.Format == "jpg" || encode.Format == "webp" {
		outputOpt = fmt.Sprintf(".%s[Q=%d]", encode.Format, encode.Quality)
	}

	input := "[descriptor=0]"
	usePipe := true
	if runtime.GOOS == "windows" {
		// Pipe IO is not working on Windows for VIPS
		if es.IsLocal() && !es.Entity().Encrypted() {
			// escape [ and ] in file name
			input = fmt.Sprintf("[filename=\"%s\"]", es.LocalPath(ctx))
			usePipe = false
		} else {
			usePipe = false
			// If not local policy files, download to temp folder
			tempPath := filepath.Join(
				util.DataPath(v.settings.TempPath(ctx)),
				"thumb",
				fmt.Sprintf("vips_%s.%s", uuid.Must(uuid.NewV4()).String(), ext),
			)
			input = fmt.Sprintf("[filename=\"%s\"]", tempPath)

			// Due to limitations of ffmpeg, we need to write the input file to disk first
			tempInputFile, err := util.CreatNestedFile(tempPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create temp file: %w", err)
			}

			defer os.Remove(tempPath)
			defer tempInputFile.Close()

			if _, err = io.Copy(tempInputFile, es); err != nil {
				return &Result{Path: tempPath}, fmt.Errorf("failed to write input file: %w", err)
			}

			tempInputFile.Close()
		}
	}

	w, h := v.settings.ThumbSize(ctx)
	cmd := exec.CommandContext(ctx,
		v.settings.VipsPath(ctx), "thumbnail_source", input, outputOpt, strconv.Itoa(w),
		"--height", strconv.Itoa(h))

	tempPath := filepath.Join(
		util.DataPath(v.settings.TempPath(ctx)),
		thumbTempFolder,
		fmt.Sprintf("thumb_%s", uuid.Must(uuid.NewV4()).String()),
	)

	thumbFile, err := util.CreatNestedFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	defer thumbFile.Close()

	// Redirect IO
	var vipsErr bytes.Buffer
	if usePipe {
		cmd.Stdin = es
	}
	cmd.Stdout = thumbFile
	cmd.Stderr = &vipsErr

	if err := cmd.Run(); err != nil {
		v.l.Warning("Failed to invoke vips: %s", vipsErr.String())
		return &Result{Path: tempPath}, fmt.Errorf("failed to invoke vips: %w, raw output: %s", err, vipsErr.String())
	}

	return &Result{Path: tempPath}, nil
}

func (v *VipsGenerator) Priority() int {
	return 100
}

func (v *VipsGenerator) Enabled(ctx context.Context) bool {
	return v.settings.VipsThumbGeneratorEnabled(ctx)
}
