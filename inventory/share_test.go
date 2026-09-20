package inventory

import (
	"context"
	"testing"
	"time"

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

func TestIsValidShareMultiFile(t *testing.T) {
	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	group := &ent.Group{Permissions: permissions}
	owner := &ent.User{ID: 1, Status: entuser.StatusActive}
	owner.SetGroup(group)

	alive := &ent.File{OwnerID: owner.ID, FileChildren: 1}
	dead := &ent.File{OwnerID: owner.ID, FileChildren: 0}
	foreign := &ent.File{OwnerID: 2, FileChildren: 1}

	newShare := func(files ...*ent.File) *ent.Share {
		s := &ent.Share{}
		s.SetUser(owner)
		s.SetFile(dead)
		s.Edges.Files = files
		return s
	}

	// Multi-file shares stay valid while at least one linked file is alive.
	require.NoError(t, IsValidShare(newShare(dead, alive)))
	require.NoError(t, IsValidShare(newShare(alive)))
	require.ErrorIs(t, IsValidShare(newShare(dead)), ErrSourceFileInvalid)
	require.ErrorIs(t, IsValidShare(newShare(foreign)), ErrSourceFileInvalid)
	require.ErrorIs(t, IsValidShare(newShare(dead, foreign)), ErrSourceFileInvalid)

	// Anchor-dead but linked-alive shares remain valid.
	require.NoError(t, IsValidShare(newShare(dead, dead, alive)))
}

func TestShareUpsertFileIDs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	group := client.Group.Create().SetName("g").SetPermissions(permissions).SaveX(ctx)
	owner := client.User.Create().SetEmail("owner@example.com").SetNick("owner").SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	f1 := client.File.Create().SetName("a.txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(root).SaveX(ctx)
	f2 := client.File.Create().SetName("b.txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(root).SaveX(ctx)

	shareClient := NewShareClient(client, conf.SQLiteDB, nil)

	// Multi-file share: files edge holds the full set, anchor included.
	s, err := shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID: owner.ID,
		FileID:  f1.ID,
		FileIDs: []int{f1.ID, f2.ID},
	})
	require.NoError(t, err)

	loadCtx := context.WithValue(ctx, LoadShareFiles{}, true)
	loaded, err := shareClient.GetByID(loadCtx, s.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Edges.Files, 2)

	// Single-file share: no files edge — legacy behavior unchanged.
	single, err := shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID: owner.ID,
		FileID:  f1.ID,
		FileIDs: []int{f1.ID},
	})
	require.NoError(t, err)
	loaded, err = shareClient.GetByID(loadCtx, single.ID)
	require.NoError(t, err)
	require.Empty(t, loaded.Edges.Files)
}

func TestShareClientListListedOnly(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	permissions := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionShare, true, permissions)
	group := client.Group.Create().SetName("g").SetPermissions(permissions).SaveX(ctx)
	owner := client.User.Create().SetEmail("owner@example.com").SetNick("owner").SetGroup(group).SaveX(ctx)
	root := client.File.Create().SetName(RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	mkfile := func(name string) *ent.File {
		return client.File.Create().SetName(name).SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(root).SaveX(ctx)
	}

	shareClient := NewShareClient(client, conf.SQLiteDB, nil)
	past := time.Now().Add(-time.Hour)

	// Visible: listed, no password, unexpired.
	listed, err := shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID: owner.ID, FileID: mkfile("public-report.pdf").ID, ListedPublicly: true,
	})
	require.NoError(t, err)
	// Hidden: not opted in.
	_, err = shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID: owner.ID, FileID: mkfile("private-notes.txt").ID,
	})
	require.NoError(t, err)
	// Hidden: opted in but password-protected (e.g. edited private later).
	_, err = shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID: owner.ID, FileID: mkfile("secret.zip").ID, Password: "pw", ListedPublicly: true,
	})
	require.NoError(t, err)
	// Hidden: opted in but expired.
	_, err = shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID: owner.ID, FileID: mkfile("old.zip").ID, ListedPublicly: true, Expires: &past,
	})
	require.NoError(t, err)
	// Visible via covered-file name match (multi-file share).
	anchor := mkfile("bundle")
	multi, err := shareClient.Upsert(ctx, &CreateShareParams{
		OwnerID:        owner.ID,
		FileID:         anchor.ID,
		FileIDs:        []int{anchor.ID, mkfile("quarterly-figures.xlsx").ID},
		ListedPublicly: true,
	})
	require.NoError(t, err)

	listArgs := func(query string) *ListShareArgs {
		return &ListShareArgs{
			PaginationArgs: &PaginationArgs{PageSize: 20},
			ListedOnly:     true,
			Query:          query,
		}
	}
	ids := func(res *ListShareResult) []int {
		out := make([]int, 0, len(res.Shares))
		for _, s := range res.Shares {
			out = append(out, s.ID)
		}
		return out
	}

	res, err := shareClient.List(ctx, listArgs(""))
	require.NoError(t, err)
	require.ElementsMatch(t, []int{listed.ID, multi.ID}, ids(res))

	// Case-insensitive substring match on anchor name.
	res, err = shareClient.List(ctx, listArgs("REPORT"))
	require.NoError(t, err)
	require.Equal(t, []int{listed.ID}, ids(res))

	// Covered-file name match surfaces the multi-file share.
	res, err = shareClient.List(ctx, listArgs("quarterly"))
	require.NoError(t, err)
	require.Equal(t, []int{multi.ID}, ids(res))

	res, err = shareClient.List(ctx, listArgs("nonexistent"))
	require.NoError(t, err)
	require.Empty(t, res.Shares)
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
