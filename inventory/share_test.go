package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/stretchr/testify/require"
)

func TestIsValidShareChecksOwnerAccess(t *testing.T) {
	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	allowedGroup := &ent.Group{Permissions: permissions}

	tests := []struct {
		name    string
		status  entuser.Status
		group   *ent.Group
		wantErr error
	}{
		{name: "active owner with share permission", status: entuser.StatusActive, group: allowedGroup},
		{name: "active owner without share permission", status: entuser.StatusActive, group: &ent.Group{Permissions: &boolset.BooleanSet{}}, wantErr: ErrSourceFileInvalid},
		{name: "missing group", status: entuser.StatusActive, wantErr: ErrSourceFileInvalid},
		{name: "missing permissions", status: entuser.StatusActive, group: &ent.Group{}, wantErr: ErrSourceFileInvalid},
		{name: "manually banned owner", status: entuser.StatusManualBanned, group: allowedGroup, wantErr: ErrOwnerInactive},
		{name: "system banned owner", status: entuser.StatusSysBanned, group: allowedGroup, wantErr: ErrOwnerInactive},
		{name: "inactive owner", status: entuser.StatusInactive, group: allowedGroup, wantErr: ErrOwnerInactive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := &ent.User{ID: 1, Status: tt.status}
			owner.SetGroup(tt.group)
			share := &ent.Share{}
			share.SetUser(owner)
			share.SetFile(&ent.File{OwnerID: owner.ID, FileChildren: 1})

			require.ErrorIs(t, IsValidShare(share), tt.wantErr)
		})
	}
}

func TestShareClientRevalidatesOwnerGroup(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	group := client.Group.Create().SetName("sharing enabled").SetPermissions(permissions).SaveX(ctx)
	restrictedGroup := client.Group.Create().SetName("sharing disabled").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	owner := client.User.Create().SetEmail("owner@example.com").SetNick("owner").SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	file := client.File.Create().SetName("shared.txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(root).SaveX(ctx)
	share := client.Share.Create().SetUser(owner).SetFile(file).SaveX(ctx)
	shareClient := NewShareClient(client, conf.SQLiteDB, nil)

	// Share-info and listing callers only request the owner and file edges.
	ctx = context.WithValue(ctx, LoadShareUser{}, true)
	ctx = context.WithValue(ctx, LoadShareFile{}, true)
	checkShare := func(wantErr error) {
		t.Helper()
		current, err := shareClient.GetByID(ctx, share.ID)
		require.NoError(t, err)
		require.ErrorIs(t, IsValidShare(current), wantErr)
	}

	checkShare(nil)
	client.Group.UpdateOne(group).SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	checkShare(ErrSourceFileInvalid)
	client.Group.UpdateOne(group).SetPermissions(permissions).SaveX(ctx)
	checkShare(nil)
	client.User.UpdateOne(owner).SetGroup(restrictedGroup).SaveX(ctx)
	checkShare(ErrSourceFileInvalid)
	client.User.UpdateOne(owner).SetGroup(group).SaveX(ctx)
	checkShare(nil)
}
