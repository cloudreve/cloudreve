package mime

import (
	"context"
	"encoding/json"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"mime"
	"path"
)

type MimeDetector interface {
	// TypeByName returns the mime type by file name.
	TypeByName(ext string) string
}

type mimeDetector struct {
	mapping map[string]string
}

func NewMimeDetector(ctx context.Context, settings setting.Provider, l logging.Logger) MimeDetector {
	mappingStr := settings.MimeMapping(ctx)
	mapping := make(map[string]string)
	if err := json.Unmarshal([]byte(mappingStr), &mapping); err != nil {
		l.Error("Failed to unmarshal mime mapping: %s, fallback to empty mapping", err)
	}

	return &mimeDetector{
		mapping: mapping,
	}
}

func (d *mimeDetector) TypeByName(p string) string {
	ext := path.Ext(p)
	if m, ok := d.mapping[ext]; ok {
		return m
	}

	m := mime.TypeByExtension(ext)
	if m != "" {
		return m
	}

	// Fallback
	return "application/octet-stream"
}
