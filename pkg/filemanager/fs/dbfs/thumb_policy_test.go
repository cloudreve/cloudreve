package dbfs

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

func thumbUploadReq(t *testing.T, f *DBFS, u int, name string, size int64, preferredPolicy int) *fs.UploadRequest {
	t.Helper()
	uri, err := fs.NewUriFromString(fs.NewMyUri(hashid.EncodeUserID(f.hasher, u)) + "/" + name)
	require.NoError(t, err)
	thumbType := types.EntityTypeThumbnail
	return &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:                    uri,
			Size:                   size,
			UploadSessionID:        uuid.Must(uuid.NewV4()).String(),
			ExpireAt:               time.Now().Add(time.Hour),
			EntityType:             &thumbType,
			PreferredStoragePolicy: preferredPolicy,
		},
	}
}

func TestPrepareUploadThumbPolicyOverride(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	u, src, f := dedupUploadFixture(t, client, "off", 1<<40)

	dst := client.StoragePolicy.Create().SetName("thumbs").SetType("local").
		SetStatus(storagepolicy.StatusActive).SetSettings(&types.PolicySetting{}).SaveX(ctx)

	// Create the backing file so a thumbnail entity can attach; completing
	// releases the upload-session lock the thumb request would hit.
	s, err := f.PrepareUpload(ctx, uploadReq(t, f.hasher, u, "img.png", 1024, ""))
	require.NoError(t, err)
	require.Equal(t, src.ID, s.Policy.ID)
	_, err = f.CompleteUpload(ctx, s)
	require.NoError(t, err)

	// Thumbnail upload with a designated policy resolves to it, not the
	// file's own policy.
	s, err = f.PrepareUpload(ctx, thumbUploadReq(t, f, u.ID, "img.png", 64, dst.ID))
	require.NoError(t, err)
	require.Equal(t, dst.ID, s.Policy.ID)
	_, err = f.CompleteUpload(ctx, s)
	require.NoError(t, err)

	// Without a designation the thumb follows the file's policy.
	s, err = f.PrepareUpload(ctx, thumbUploadReq(t, f, u.ID, "img.png", 64, 0))
	require.NoError(t, err)
	require.Equal(t, src.ID, s.Policy.ID)
}
