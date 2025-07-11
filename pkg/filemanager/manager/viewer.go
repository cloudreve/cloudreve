package manager

import (
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

type (
	ViewerSession struct {
		ID          string  `json:"id"`
		AccessToken string  `json:"access_token"`
		Expires     int64   `json:"expires"`
		File        fs.File `json:"-"`
	}
	ViewerSessionCache struct {
		ID       string
		Uri      string
		UserID   int
		FileID   int
		ViewerID string
		Version  string
		Token    string
	}
	ViewerSessionCacheCtx struct{}
	ViewerCtx             struct{}
)

const (
	ViewerSessionCachePrefix = "viewer_session_"

	sessionExpiresPadding = 10
)

func init() {
	gob.Register(ViewerSessionCache{})
}

func (m *manager) CreateViewerSession(ctx context.Context, uri *fs.URI, version string, viewer *types.Viewer) (*ViewerSession, error) {
	file, err := m.fs.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithNotRoot())
	if err != nil {
		return nil, err
	}

	versionType := types.EntityTypeVersion
	found, desired := fs.FindDesiredEntity(file, version, m.hasher, &versionType)
	if !found {
		return nil, fs.ErrEntityNotExist
	}

	if desired.Size() > m.settings.MaxOnlineEditSize(ctx) {
		return nil, fs.ErrFileSizeTooBig
	}

	sessionID := uuid.Must(uuid.NewV4()).String()
	token := util.RandStringRunes(128)
	sessionCache := &ViewerSessionCache{
		ID:       sessionID,
		Uri:      file.Uri(false).String(),
		UserID:   m.user.ID,
		ViewerID: viewer.ID,
		FileID:   file.ID(),
		Version:  version,
		Token:    fmt.Sprintf("%s.%s", sessionID, token),
	}
	ttl := m.settings.ViewerSessionTTL(ctx)
	if err := m.kv.Set(ViewerSessionCachePrefix+sessionID, *sessionCache, ttl); err != nil {
		return nil, err
	}

	return &ViewerSession{
		File:        file,
		ID:          sessionID,
		AccessToken: sessionCache.Token,
		Expires:     time.Now().Add(time.Duration(ttl-sessionExpiresPadding) * time.Second).UnixMilli(),
	}, nil
}

func ViewerSessionFromContext(ctx context.Context) *ViewerSessionCache {
	return ctx.Value(ViewerSessionCacheCtx{}).(*ViewerSessionCache)
}

func ViewerFromContext(ctx context.Context) *types.Viewer {
	return ctx.Value(ViewerCtx{}).(*types.Viewer)
}
