package thumb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

func NewLibreOfficeGenerator(l logging.Logger, settings setting.Provider) *LibreOfficeGenerator {
	return &LibreOfficeGenerator{l: l, settings: settings}
}

type LibreOfficeGenerator struct {
	settings setting.Provider
	l        logging.Logger
}

func (l *LibreOfficeGenerator) Generate(ctx context.Context, es entitysource.EntitySource, ext string, previous *Result) (*Result, error) {
	if !util.IsInExtensionListExt(l.settings.LibreOfficeThumbExts(ctx), ext) {
		return nil, fmt.Errorf("unsupported video format: %w", ErrPassThrough)
	}

	if es.Entity().Size() > l.settings.LibreOfficeThumbMaxSize(ctx) {
		return nil, fmt.Errorf("file is too big: %w", ErrPassThrough)
	}

	tempOutputPath := filepath.Join(
		util.DataPath(l.settings.TempPath(ctx)),
		thumbTempFolder,
		fmt.Sprintf("soffice_%s", uuid.Must(uuid.NewV4()).String()),
	)

	tempInputPath := ""
	if es.IsLocal() && !es.Entity().Encrypted() {
		tempInputPath = es.LocalPath(ctx)
	} else {
		// If not local policy files, download to temp folder
		tempInputPath = filepath.Join(
			util.DataPath(l.settings.TempPath(ctx)),
			"thumb",
			fmt.Sprintf("soffice_%s.%s", uuid.Must(uuid.NewV4()).String(), ext),
		)

		// Due to limitations of ffmpeg, we need to write the input file to disk first
		tempInputFile, err := util.CreatNestedFile(tempInputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}

		defer os.Remove(tempInputPath)
		defer tempInputFile.Close()

		if _, err = io.Copy(tempInputFile, es); err != nil {
			return &Result{Path: tempOutputPath}, fmt.Errorf("failed to write input file: %w", err)
		}

		tempInputFile.Close()
	}

	// Convert the document to an image
	cmd := exec.CommandContext(ctx, l.settings.LibreOfficePath(ctx), "--headless",
		"--nologo", "--nofirststartwizard", "--invisible", "--norestore", "--convert-to",
		"png", "--outdir", tempOutputPath, tempInputPath)

	// Redirect IO
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	if err := cmd.Run(); err != nil {
		l.l.Warning("Failed to invoke LibreOffice: %s", stdErr.String())
		return &Result{Path: tempOutputPath}, fmt.Errorf("failed to invoke LibreOffice: %w, raw output: %s", err, stdErr.String())
	}

	return &Result{
		Path: filepath.Join(
			tempOutputPath,
			strings.TrimSuffix(filepath.Base(tempInputPath), filepath.Ext(tempInputPath))+".png",
		),
		Continue: true,
		Cleanup:  []func(){func() { _ = os.RemoveAll(tempOutputPath) }},
	}, nil
}

func (l *LibreOfficeGenerator) Priority() int {
	return 50
}

func (l *LibreOfficeGenerator) Enabled(ctx context.Context) bool {
	return l.settings.LibreOfficeThumbGeneratorEnabled(ctx)
}
