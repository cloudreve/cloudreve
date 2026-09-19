package thumb

import (
	"context"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/dhowden/tag"
	"github.com/gofrs/uuid"
	"os"
	"path/filepath"
)

func NewMusicCoverGenerator(l logging.Logger, settings setting.Provider) *MusicCoverGenerator {
	return &MusicCoverGenerator{l: l, settings: settings}
}

type MusicCoverGenerator struct {
	l        logging.Logger
	settings setting.Provider
}

func (v *MusicCoverGenerator) Generate(ctx context.Context, es entitysource.EntitySource, ext string, previous *Result) (*Result, error) {
	if !util.IsExtInList(v.settings.MusicCoverThumbExts(ctx), ext) {
		return nil, fmt.Errorf("unsupported music format: %w", ErrPassThrough)
	}

	if es.Entity().Size() > v.settings.MusicCoverThumbMaxSize(ctx) {
		return nil, fmt.Errorf("file is too big: %w", ErrPassThrough)
	}

	m, err := tag.ReadFrom(es)
	if err != nil {
		return nil, fmt.Errorf("faield to read audio tags from file: %w", err)
	}

	p := m.Picture()
	if p == nil || len(p.Data) == 0 {
		return nil, fmt.Errorf("no cover found in given file")
	}

	thumbExt := ".jpg"
	if p.Ext != "" {
		thumbExt = p.Ext
	}

	tempPath := filepath.Join(
		util.DataPath(v.settings.TempPath(ctx)),
		thumbTempFolder,
		fmt.Sprintf("thumb_%s.%s", uuid.Must(uuid.NewV4()).String(), thumbExt),
	)

	thumbFile, err := util.CreatNestedFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	defer thumbFile.Close()

	if _, err := thumbFile.Write(p.Data); err != nil {
		return &Result{Path: tempPath}, fmt.Errorf("failed to write cover to file: %w", err)
	}

	return &Result{
		Path:     tempPath,
		Continue: true,
		Cleanup:  []func(){func() { _ = os.Remove(tempPath) }},
	}, nil
}

func (v *MusicCoverGenerator) Priority() int {
	return 50
}

func (v *MusicCoverGenerator) Enabled(ctx context.Context) bool {
	return v.settings.MusicCoverThumbGeneratorEnabled(ctx)
}
