package dbfs

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/aclentry"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

// groupFolderFixture builds an owner with a populated folder and a member
// user in a separate group. Returns (member, owner, sharedDir, nestedFile).
func groupFolderFixture(t *testing.T, client *ent.Client) (*ent.User, *ent.User, *ent.File, *ent.File) {
	t.Helper()
	ctx := context.Background()

	ownerGroup := client.Group.Create().SetName("owners").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	memberGroup := client.Group.Create().SetName("members").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)

	owner := client.User.Create().SetEmail("owner@example.com").SetNick("o").SetGroup(ownerGroup).SaveX(ctx)
	ownerRoot := client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(owner).SaveX(ctx)
	sharedDir := client.File.Create().SetName("team_dir").SetType(int(types.FileTypeFolder)).SetOwner(owner).SetParent(ownerRoot).SaveX(ctx)
	nested := client.File.Create().SetName("nested.txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SetParent(sharedDir).SaveX(ctx)

	member := client.User.Create().SetEmail("member@example.com").SetNick("m").SetGroup(memberGroup).SaveX(ctx)
	client.File.Create().SetName(inventory.RootFolderName).SetType(int(types.FileTypeFolder)).SetOwner(member).SaveX(ctx)

	// Attach the primary-group edge the navigator relies on.
	member.Edges.Group = memberGroup
	return member, owner, sharedDir, nested
}

func groupFolderNavigator(t *testing.T, client *ent.Client, user *ent.User) (*sharedWithMeNavigator, hashid.Encoder) {
	t.Helper()
	hasher, err := hashid.New("group-folder-test-salt")
	require.NoError(t, err)
	n := NewSharedWithMeNavigator(user, inventory.NewFileClient(client, conf.SQLiteDB, hasher),
		inventory.NewAclClient(client, conf.SQLiteDB),
		logging.NewConsoleLogger(logging.LevelError), &setting.DBFS{MaxPageSize: 50}, hasher)
	return n.(*sharedWithMeNavigator), hasher
}

func aclGrant(t *testing.T, client *ent.Client, fileID int, st aclentry.SubjectType, sid int, perms ...types.AclPermission) {
	t.Helper()
	bs := &boolset.BooleanSet{}
	for _, p := range perms {
		boolset.Set(int(p), true, bs)
	}
	client.AclEntry.Create().
		SetFileID(fileID).SetSubjectType(st).SetSubjectID(sid).SetPermissions(bs).
		SaveX(context.Background())
}

func sharedWithMeURI(t *testing.T, parts ...string) *fs.URI {
	t.Helper()
	uri, err := fs.NewUriFromString("cloudreve://" + string(constants.FileSystemSharedWithMe))
	require.NoError(t, err)
	return uri.Join(parts...)
}

func TestSharedWithMeListsAclGrantedFolder(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	member, _, sharedDir, _ := groupFolderFixture(t, client)
	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeGroup, member.Edges.Group.ID, types.AclPermRead)

	n, _ := groupFolderNavigator(t, client, member)
	root, err := n.To(ctx, sharedWithMeURI(t))
	require.NoError(t, err)

	res, err := n.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 50}})
	require.NoError(t, err)
	require.Len(t, res.Files, 1)
	require.Equal(t, "team_dir", res.Files[0].Name())

	// The entry carries the ACL-derived capability set and real owner.
	require.NotNil(t, res.Files[0].Capabilities())
	require.True(t, res.Files[0].Capabilities().Enabled(int(NavigatorCapabilityListChildren)))
	require.False(t, res.Files[0].Capabilities().Enabled(int(NavigatorCapabilityUploadFile)))
	require.Equal(t, sharedDir.OwnerID, res.Files[0].Owner().ID)
	require.Equal(t, sharedWithMeURI(t, hashid.EncodeFileID(n.hasher, sharedDir.ID)).String(), res.Files[0].Uri(false).String())
}

func TestSharedWithMeResolvesGroupFolderSubtree(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	member, owner, sharedDir, nested := groupFolderFixture(t, client)
	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeGroup, member.Edges.Group.ID,
		types.AclPermRead, types.AclPermCreate)

	n, hasher := groupFolderNavigator(t, client, member)

	root, err := n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID)))
	require.NoError(t, err)
	require.Equal(t, sharedDir.ID, root.Model.ID)
	require.Equal(t, owner.ID, root.Owner().ID)
	require.True(t, root.Capabilities().Enabled(int(NavigatorCapabilityUploadFile)))
	require.True(t, root.Capabilities().Enabled(int(NavigatorCapabilityListChildren)))
	require.False(t, root.Capabilities().Enabled(int(NavigatorCapabilityDeleteFile)))

	// Deeper elements walk the owner's subtree.
	child, err := n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID), "nested.txt"))
	require.NoError(t, err)
	require.Equal(t, nested.ID, child.Model.ID)
	require.True(t, child.Capabilities().Enabled(int(NavigatorCapabilityDownloadFile)))

	// Children of the subtree list like a regular folder.
	res, err := n.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 50}})
	require.NoError(t, err)
	require.Len(t, res.Files, 1)
	require.Equal(t, "nested.txt", res.Files[0].Name())
	require.True(t, res.Files[0].Uri(false).String() == sharedWithMeURI(t,
		hashid.EncodeFileID(hasher, sharedDir.ID), "nested.txt").String())
}

func TestSharedWithMeDeniesUngranted(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	member, _, sharedDir, _ := groupFolderFixture(t, client)
	n, hasher := groupFolderNavigator(t, client, member)

	// No ACL at all.
	_, err := n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID)))
	require.Error(t, err)

	// Grant without the read bit is not a browse grant.
	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeGroup, member.Edges.Group.ID, types.AclPermCreate)
	_, err = n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID)))
	require.Error(t, err)

	// Grant to a different group does not leak.
	other := client.Group.Create().SetName("other").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeGroup, other.ID, types.AclPermRead)
	_, err = n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID)))
	require.Error(t, err)

	// Everyone-tier grants allow browsing but do not surface in the listing.
	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeEveryone, 0, types.AclPermRead)
	_, err = n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID)))
	require.NoError(t, err)

	root, err := n.To(ctx, sharedWithMeURI(t))
	require.NoError(t, err)
	res, err := n.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 50}})
	require.NoError(t, err)
	require.Empty(t, res.Files)
}

func TestSharedWithMeMembershipGrant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	member, _, sharedDir, _ := groupFolderFixture(t, client)

	// Grant lands on a group the user holds only through a membership.
	teamGroup := client.Group.Create().SetName("team").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	m := client.GroupMembership.Create().SetUserID(member.ID).SetGroupID(teamGroup.ID).SaveX(ctx)
	m.Edges.Group = teamGroup
	member.Edges.Memberships = []*ent.GroupMembership{m}

	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeGroup, teamGroup.ID, types.AclPermRead)

	n, hasher := groupFolderNavigator(t, client, member)
	root, err := n.To(ctx, sharedWithMeURI(t))
	require.NoError(t, err)
	res, err := n.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 50}})
	require.NoError(t, err)
	require.Len(t, res.Files, 1)

	_, err = n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID)))
	require.NoError(t, err)
}

func TestSharedWithMeSymbolicShortcutResolves(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	member, _, _, _ := groupFolderFixture(t, client)

	// A saved share shortcut of the member's own.
	shortcut := client.File.Create().SetName("saved_share").SetType(int(types.FileTypeFolder)).
		SetOwner(member).SetIsSymbolic(true).SaveX(ctx)

	n, hasher := groupFolderNavigator(t, client, member)

	// No ACL on the owner's dir — resolution rides on ownership of the shortcut.
	res, err := n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, shortcut.ID)))
	require.NoError(t, err)
	require.Equal(t, shortcut.ID, res.Model.ID)
	require.Equal(t, member.ID, res.Owner().ID)
}

func TestSharedWithMeUserSubjectGrant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	member, _, sharedDir, nested := groupFolderFixture(t, client)
	aclGrant(t, client, sharedDir.ID, aclentry.SubjectTypeUser, member.ID, types.AclPermRead)

	n, hasher := groupFolderNavigator(t, client, member)
	root, err := n.To(ctx, sharedWithMeURI(t))
	require.NoError(t, err)
	res, err := n.Children(ctx, root, &ListArgs{Page: &inventory.PaginationArgs{PageSize: 50}})
	require.NoError(t, err)
	require.Len(t, res.Files, 1)

	file, err := n.To(ctx, sharedWithMeURI(t, hashid.EncodeFileID(hasher, sharedDir.ID), nested.Name))
	require.NoError(t, err)
	require.Equal(t, nested.ID, file.Model.ID)
}
