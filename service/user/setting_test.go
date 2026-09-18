package user

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestPatchUserSettingShareDefaults verifies the tri-state private-share
// override and profile-visibility validation (upstream #3390).
func TestPatchUserSettingShareDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().
		SetEmail("patch@example.com").
		SetNick("patch").
		SetStatus("active").
		SetGroup(group).
		SetSettings(&types.UserSetting{}).
		SaveX(ctx)

	newCtx := func() *gin.Context {
		engine := gin.New()
		engine.ContextWithFallback = true
		c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
		c.Request = httptest.NewRequest("PATCH", "/", nil)
		util.WithValue(c, dependency.DepCtx{}, dependency.NewDependency(
			dependency.WithUserClient(inventory.NewUserClient(client)),
		))
		util.WithValue(c, inventory.UserCtx{}, u)
		return c
	}

	strPtr := func(s string) *string { return &s }

	patch := func(s *PatchUserSetting) error {
		return s.Patch(newCtx())
	}

	// Set override on.
	require.NoError(t, patch(&PatchUserSetting{ShareDefaultPrivate: strPtr("true")}))
	require.NotNil(t, u.Settings.ShareDefaultPrivate)
	require.True(t, *u.Settings.ShareDefaultPrivate)

	// Set override off.
	require.NoError(t, patch(&PatchUserSetting{ShareDefaultPrivate: strPtr("false")}))
	require.NotNil(t, u.Settings.ShareDefaultPrivate)
	require.False(t, *u.Settings.ShareDefaultPrivate)

	// Persisted to DB.
	persisted := client.User.GetX(ctx, u.ID)
	require.NotNil(t, persisted.Settings.ShareDefaultPrivate)
	require.False(t, *persisted.Settings.ShareDefaultPrivate)

	// Empty clears the override back to inheritance.
	require.NoError(t, patch(&PatchUserSetting{ShareDefaultPrivate: strPtr("")}))
	require.Nil(t, u.Settings.ShareDefaultPrivate)

	// Malformed value rejected.
	require.Error(t, patch(&PatchUserSetting{ShareDefaultPrivate: strPtr("yes")}))

	// Valid profile visibility values accepted.
	for _, v := range []string{"", "public_share", "all_share", "hide_share"} {
		require.NoError(t, patch(&PatchUserSetting{ShareLinksInProfile: strPtr(v)}))
		require.Equal(t, types.ShareLinksInProfileLevel(v), u.Settings.ShareLinksInProfile)
	}

	// Invalid profile visibility rejected.
	require.Error(t, patch(&PatchUserSetting{ShareLinksInProfile: strPtr("friends_only")}))
}
