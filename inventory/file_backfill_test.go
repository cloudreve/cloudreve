package inventory

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeBackfillFile(t *testing.T, client *ent.Client, fileType types.FileType, name string, ownerID int, size int64, primaryEntity, policyID int) *ent.File {
	t.Helper()
	stm := client.File.Create().
		SetType(int(fileType)).
		SetName(name).
		SetOwnerID(ownerID).
		SetSize(size)
	if primaryEntity > 0 {
		stm.SetPrimaryEntity(primaryEntity)
	}
	if policyID > 0 {
		stm.SetStoragePolicyFiles(policyID)
	}
	f, err := stm.Save(context.Background())
	require.NoError(t, err)
	return f
}

func backfillFileIDs(files []*ent.File) []int {
	out := make([]int, len(files))
	for i, f := range files {
		out[i] = f.ID
	}
	return out
}

// TestBackfillCandidateFiles covers APP-102: the candidate query returns only
// file-type rows with a primary entity and size >= minSize, honors the owner and
// storage-policy scopes, and paginates by ascending id via the afterID cursor.
func TestBackfillCandidateFiles(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	c := NewFileClient(client, "sqlite", nil)

	// Minimal graph so File FKs (owner_id, storage_policy_files) resolve.
	grp, err := client.Group.Create().
		SetName("g").
		SetPermissions(&boolset.BooleanSet{}).
		SetSettings(&types.GroupSetting{}).
		Save(ctx)
	require.NoError(t, err)
	u1, err := client.User.Create().SetEmail("u1@example.com").SetNick("u1").SetGroupUsers(grp.ID).Save(ctx)
	require.NoError(t, err)
	u2, err := client.User.Create().SetEmail("u2@example.com").SetNick("u2").SetGroupUsers(grp.ID).Save(ctx)
	require.NoError(t, err)
	p1 := makePolicy(t, client, "p1")
	p2 := makePolicy(t, client, "p2")

	const minSize = int64(200000)

	// Eligible: files owned by u1, big enough, with a primary entity.
	f1 := makeBackfillFile(t, client, types.FileTypeFile, "a.jpg", u1.ID, 300000, 101, p1.ID)
	f2 := makeBackfillFile(t, client, types.FileTypeFile, "b.png", u1.ID, 500000, 102, p1.ID)
	// Excluded: below min size.
	makeBackfillFile(t, client, types.FileTypeFile, "small.jpg", u1.ID, 1000, 103, p1.ID)
	// Excluded: no primary entity.
	makeBackfillFile(t, client, types.FileTypeFile, "noentity.jpg", u1.ID, 400000, 0, p1.ID)
	// Excluded: folder, not a file.
	makeBackfillFile(t, client, types.FileTypeFolder, "folder", u1.ID, 400000, 104, p1.ID)
	// Belongs to u2 (excluded when scoping to u1).
	makeBackfillFile(t, client, types.FileTypeFile, "other.jpg", u2.ID, 400000, 105, p1.ID)
	// Owned by u1 but on a different storage policy.
	f7 := makeBackfillFile(t, client, types.FileTypeFile, "policy2.jpg", u1.ID, 400000, 106, p2.ID)

	// Count scoped to u1, all policies: f1, f2, f7.
	total, err := c.CountBackfillCandidateFiles(ctx, u1.ID, nil, minSize)
	require.NoError(t, err)
	assert.Equal(t, 3, total)

	// List scoped to u1, ordered by ascending id.
	files, err := c.ListBackfillCandidateFiles(ctx, 0, 100, u1.ID, nil, minSize)
	require.NoError(t, err)
	require.Len(t, files, 3)
	assert.Equal(t, []int{f1.ID, f2.ID, f7.ID}, backfillFileIDs(files))

	// Cursor pagination: a batch of 2, then the remainder after the last id.
	page1, err := c.ListBackfillCandidateFiles(ctx, 0, 2, u1.ID, nil, minSize)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	page2, err := c.ListBackfillCandidateFiles(ctx, page1[len(page1)-1].ID, 2, u1.ID, nil, minSize)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, f7.ID, page2[0].ID)

	// Storage-policy scope: policy p1 only → f1, f2 (f7 is on p2).
	pol1, err := c.CountBackfillCandidateFiles(ctx, u1.ID, []int{p1.ID}, minSize)
	require.NoError(t, err)
	assert.Equal(t, 2, pol1)

	// No owner scope (userID=0): includes u2's file too → f1, f2, u2's, f7.
	all, err := c.CountBackfillCandidateFiles(ctx, 0, nil, minSize)
	require.NoError(t, err)
	assert.Equal(t, 4, all)
}
