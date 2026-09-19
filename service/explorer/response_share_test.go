package explorer

import (
	"context"
	"net/url"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

// stubSettingProvider satisfies the SettingProvider dependency; only
// ExposeUserEmail and ShareDefaults are exercised by BuildUserRedacted.
type stubSettingProvider struct {
	setting.Provider
}

func (stubSettingProvider) ExposeUserEmail(context.Context) bool { return false }

func (stubSettingProvider) ShareDefaults(context.Context) *setting.ShareDefaults {
	return &setting.ShareDefaults{}
}

// TestBuildShareNoteVisibility ensures the owner-defined share note is only
// exposed to the share owner, never to visitors (upstream #3570).
func TestBuildShareNoteVisibility(t *testing.T) {
	dep := dependency.NewDependency(dependency.WithSettingProvider(stubSettingProvider{}))
	ctx := context.WithValue(context.Background(), dependency.DepCtx{}, dep)

	hasher, err := hashid.New("test-salt")
	require.NoError(t, err)
	base := &url.URL{Scheme: "https", Host: "example.com"}

	owner := &ent.User{ID: 1, Email: "owner@example.com", Settings: &types.UserSetting{}}
	visitor := &ent.User{ID: 2, Email: "visitor@example.com", Settings: &types.UserSetting{}}
	anonymous := &ent.User{ID: 0, Settings: &types.UserSetting{}}

	newShare := func() *ent.Share {
		return &ent.Share{
			ID:    1,
			Props: &types.ShareProps{Note: "tax receipts Q3"},
		}
	}

	t.Run("owner sees note", func(t *testing.T) {
		res := BuildShare(ctx, newShare(), base, hasher, owner, owner,
			"receipts.zip", types.FileTypeFile, true, false)
		require.Equal(t, "tax receipts Q3", res.Note)
	})

	t.Run("unlocked visitor does not see note", func(t *testing.T) {
		res := BuildShare(ctx, newShare(), base, hasher, visitor, owner,
			"receipts.zip", types.FileTypeFile, true, false)
		require.Empty(t, res.Note)
	})

	t.Run("anonymous visitor does not see note", func(t *testing.T) {
		res := BuildShare(ctx, newShare(), base, hasher, anonymous, owner,
			"receipts.zip", types.FileTypeFile, true, false)
		require.Empty(t, res.Note)
	})

	t.Run("locked share does not leak note to visitor", func(t *testing.T) {
		res := BuildShare(ctx, newShare(), base, hasher, visitor, owner,
			"receipts.zip", types.FileTypeFile, false, false)
		require.Empty(t, res.Note)
	})
}
