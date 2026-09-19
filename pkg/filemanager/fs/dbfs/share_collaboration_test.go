package dbfs

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

func shareWithProps(props *types.ShareProps) *ent.Share {
	return &ent.Share{Props: props}
}

func TestShareCapabilities(t *testing.T) {
	capsOf := func(props *types.ShareProps) *boolset.BooleanSet {
		n := &shareNavigator{share: shareWithProps(props)}
		return n.shareCapabilities()
	}
	enabled := func(bs *boolset.BooleanSet, c NavigatorCapability) bool {
		return bs.Enabled(int(c))
	}

	t.Run("no props grants read only", func(t *testing.T) {
		caps := capsOf(nil)
		require.True(t, enabled(caps, NavigatorCapabilityListChildren))
		require.True(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.True(t, enabled(caps, NavigatorCapabilityEnterFolder))
		require.False(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.False(t, enabled(caps, NavigatorCapabilityCreateFile))
		require.False(t, enabled(caps, NavigatorCapabilityRenameFile))
		require.False(t, enabled(caps, NavigatorCapabilityDeleteFile))
		require.False(t, enabled(caps, NavigatorCapabilitySoftDelete))
	})

	t.Run("allow upload grants upload create and lock", func(t *testing.T) {
		caps := capsOf(&types.ShareProps{AllowUpload: true})
		require.True(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.True(t, enabled(caps, NavigatorCapabilityCreateFile))
		require.True(t, enabled(caps, NavigatorCapabilityLockFile))
		require.True(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.True(t, enabled(caps, NavigatorCapabilityListChildren))
		require.False(t, enabled(caps, NavigatorCapabilityRenameFile))
		require.False(t, enabled(caps, NavigatorCapabilityDeleteFile))
	})

	t.Run("allow edit grants all write caps", func(t *testing.T) {
		caps := capsOf(&types.ShareProps{AllowEdit: true})
		require.True(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.True(t, enabled(caps, NavigatorCapabilityCreateFile))
		require.True(t, enabled(caps, NavigatorCapabilityLockFile))
		require.True(t, enabled(caps, NavigatorCapabilityRenameFile))
		require.True(t, enabled(caps, NavigatorCapabilityDeleteFile))
		require.True(t, enabled(caps, NavigatorCapabilitySoftDelete))
	})

	t.Run("preview only keeps read caps", func(t *testing.T) {
		caps := capsOf(&types.ShareProps{PreviewOnly: true})
		require.True(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.True(t, enabled(caps, NavigatorCapabilityListChildren))
		require.False(t, enabled(caps, NavigatorCapabilityUploadFile))
	})

	t.Run("upload only strips read and edit", func(t *testing.T) {
		caps := capsOf(&types.ShareProps{UploadOnly: true})
		require.True(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.True(t, enabled(caps, NavigatorCapabilityCreateFile))
		require.True(t, enabled(caps, NavigatorCapabilityLockFile))
		require.False(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.False(t, enabled(caps, NavigatorCapabilityListChildren))
	})

	t.Run("upload only wins over edit", func(t *testing.T) {
		caps := capsOf(&types.ShareProps{UploadOnly: true, AllowEdit: true})
		require.True(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.False(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.False(t, enabled(caps, NavigatorCapabilityListChildren))
		require.False(t, enabled(caps, NavigatorCapabilityRenameFile))
		require.False(t, enabled(caps, NavigatorCapabilityDeleteFile))
		require.False(t, enabled(caps, NavigatorCapabilitySoftDelete))
	})
}

func TestCanMoveOrCopyTo(t *testing.T) {
	shareA := func() *fs.URI {
		u, err := fs.NewUriFromString("cloudreve://aaa@share/folder")
		require.NoError(t, err)
		return u
	}
	shareB := func() *fs.URI {
		u, err := fs.NewUriFromString("cloudreve://bbb@share/folder")
		require.NoError(t, err)
		return u
	}
	my := func() *fs.URI {
		u, err := fs.NewUriFromString("cloudreve://my/folder")
		require.NoError(t, err)
		return u
	}
	trash := func() *fs.URI {
		u, err := fs.NewUriFromString("cloudreve://trash/folder")
		require.NoError(t, err)
		return u
	}

	require.True(t, canMoveOrCopyTo(shareA(), shareA(), true), "same-share copy")
	require.True(t, canMoveOrCopyTo(shareA(), shareA(), false), "same-share move")
	require.False(t, canMoveOrCopyTo(shareA(), shareB(), true), "cross-share copy")
	require.False(t, canMoveOrCopyTo(shareA(), shareB(), false), "cross-share move")
	require.False(t, canMoveOrCopyTo(shareA(), my(), true), "share to my copy")
	require.False(t, canMoveOrCopyTo(shareA(), my(), false), "share to my move")
	require.False(t, canMoveOrCopyTo(my(), shareA(), true), "my to share copy")
	require.False(t, canMoveOrCopyTo(my(), shareA(), false), "my to share move")
	require.True(t, canMoveOrCopyTo(my(), my(), true), "my copy")
	require.True(t, canMoveOrCopyTo(my(), my(), false), "my move")
	require.True(t, canMoveOrCopyTo(my(), trash(), false), "my to trash move")
	require.False(t, canMoveOrCopyTo(my(), trash(), true), "my to trash copy")
	require.True(t, canMoveOrCopyTo(trash(), my(), false), "trash restore")
}

func TestWritePermitted(t *testing.T) {
	owner := &ent.User{ID: 1}
	visitor := &ent.User{ID: 2}

	newOwnedFile := func(caps *boolset.BooleanSet) *File {
		f := newFile(nil, &ent.File{ID: 10, Name: "f", OwnerID: owner.ID})
		f.OwnerModel = owner
		f.CapabilitiesBs = caps
		return f
	}

	writeCaps := &boolset.BooleanSet{}
	boolset.Set(int(NavigatorCapabilityRenameFile), true, writeCaps)
	readCaps := &boolset.BooleanSet{}
	boolset.Set(int(NavigatorCapabilityDownloadFile), true, readCaps)

	ownerFS := &DBFS{user: owner}
	visitorFS := &DBFS{user: visitor}

	require.True(t, ownerFS.writePermitted(newOwnedFile(nil), NavigatorCapabilityRenameFile), "owner always permitted")
	require.True(t, visitorFS.writePermitted(newOwnedFile(writeCaps), NavigatorCapabilityRenameFile), "visitor with cap")
	require.False(t, visitorFS.writePermitted(newOwnedFile(readCaps), NavigatorCapabilityRenameFile), "visitor without cap")
	require.False(t, visitorFS.writePermitted(newOwnedFile(nil), NavigatorCapabilityRenameFile), "visitor nil caps")
}

type downloadCountShareClient struct {
	inventory.ShareClient
	calls int
	err   error
}

func (c *downloadCountShareClient) Downloaded(_ context.Context, _ *ent.Share) error {
	c.calls++
	return c.err
}

func TestShareNavigatorExecuteHookPreviewOnly(t *testing.T) {
	newNav := func(props *types.ShareProps) (*shareNavigator, *downloadCountShareClient) {
		sc := &downloadCountShareClient{}
		return &shareNavigator{
			l:           logging.NewConsoleLogger(logging.LevelError),
			shareClient: sc,
			share:       shareWithProps(props),
		}, sc
	}

	t.Run("explicit download denied", func(t *testing.T) {
		n, sc := newNav(&types.ShareProps{PreviewOnly: true})
		ctx := context.WithValue(context.Background(), IsDownloadCtxKey{}, true)
		err := n.ExecuteHook(ctx, fs.HookTypeBeforeDownload, nil)
		require.ErrorIs(t, err, ErrShareDownloadDisabled)
		require.Zero(t, sc.calls, "denied downloads must not bump the counter")
	})

	t.Run("preview fetch allowed", func(t *testing.T) {
		n, sc := newNav(&types.ShareProps{PreviewOnly: true})
		err := n.ExecuteHook(context.Background(), fs.HookTypeBeforeDownload, nil)
		require.NoError(t, err)
		require.Equal(t, 1, sc.calls)
	})

	t.Run("normal share counts download", func(t *testing.T) {
		n, sc := newNav(&types.ShareProps{})
		ctx := context.WithValue(context.Background(), IsDownloadCtxKey{}, true)
		err := n.ExecuteHook(ctx, fs.HookTypeBeforeDownload, nil)
		require.NoError(t, err)
		require.Equal(t, 1, sc.calls)
	})
}

func TestShareCapabilitiesAcl(t *testing.T) {
	owner := &ent.User{ID: 1}
	visitor := &ent.User{ID: 2}

	enabled := func(bs *boolset.BooleanSet, c NavigatorCapability) bool {
		return bs.Enabled(int(c))
	}
	aclBs := func(perms ...types.AclPermission) *boolset.BooleanSet {
		bs := &boolset.BooleanSet{}
		for _, p := range perms {
			boolset.Set(int(p), true, bs)
		}
		return bs
	}

	t.Run("matched acl overrides share props", func(t *testing.T) {
		n := &shareNavigator{
			user:    visitor,
			owner:   owner,
			share:   shareWithProps(&types.ShareProps{AllowEdit: true}),
			aclCaps: aclBs(types.AclPermRead),
		}
		caps := n.shareCapabilities()
		require.True(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.False(t, enabled(caps, NavigatorCapabilityRenameFile))
		require.False(t, enabled(caps, NavigatorCapabilityDeleteFile))
	})

	t.Run("explicit empty acl revokes fallback", func(t *testing.T) {
		n := &shareNavigator{
			user:    visitor,
			owner:   owner,
			share:   shareWithProps(nil),
			aclCaps: aclBs(),
		}
		caps := n.shareCapabilities()
		require.False(t, enabled(caps, NavigatorCapabilityDownloadFile))
		require.False(t, enabled(caps, NavigatorCapabilityListChildren))
	})

	t.Run("no match falls back to share props", func(t *testing.T) {
		n := &shareNavigator{
			user:  visitor,
			owner: owner,
			share: shareWithProps(&types.ShareProps{AllowUpload: true}),
		}
		caps := n.shareCapabilities()
		require.True(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.True(t, enabled(caps, NavigatorCapabilityDownloadFile))
	})

	t.Run("owner bypasses acl", func(t *testing.T) {
		n := &shareNavigator{
			user:    owner,
			owner:   owner,
			share:   shareWithProps(nil),
			aclCaps: aclBs(),
		}
		caps := n.shareCapabilities()
		require.True(t, enabled(caps, NavigatorCapabilityDownloadFile))
	})

	t.Run("create update delete bits map", func(t *testing.T) {
		caps := aclPermsToCapabilities(aclBs(types.AclPermCreate, types.AclPermUpdate, types.AclPermDelete))
		require.True(t, enabled(caps, NavigatorCapabilityUploadFile))
		require.True(t, enabled(caps, NavigatorCapabilityCreateFile))
		require.True(t, enabled(caps, NavigatorCapabilityRenameFile))
		require.True(t, enabled(caps, NavigatorCapabilityUpdateMetadata))
		require.True(t, enabled(caps, NavigatorCapabilityDeleteFile))
		require.True(t, enabled(caps, NavigatorCapabilitySoftDelete))
		require.False(t, enabled(caps, NavigatorCapabilityDownloadFile))
	})
}
