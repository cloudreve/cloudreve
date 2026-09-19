package user

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestActivateEmailChange verifies the signed-link confirmation path: pending
// address comes from KV, the change applies once, and replays/concurrent
// claims are rejected.
func TestActivateEmailChange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)
	u := client.User.Create().
		SetEmail("old@example.com").
		SetNick("change").
		SetStatus("active").
		SetGroup(group).
		SetSettings(&types.UserSetting{}).
		SaveX(ctx)
	client.User.Create().
		SetEmail("taken@example.com").
		SetNick("taken").
		SetStatus("active").
		SetGroup(group).
		SetSettings(&types.UserSetting{}).
		SaveX(ctx)

	kv := cache.NewMemoStore("", nil)
	dep := dependency.NewDependency(
		dependency.WithUserClient(inventory.NewUserClient(client)),
		dependency.WithKV(kv),
	)

	newCtx := func(uid int) *gin.Context {
		engine := gin.New()
		engine.ContextWithFallback = true
		c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
		c.Request = httptest.NewRequest("GET", "/", nil)
		util.WithValue(c, dependency.DepCtx{}, dep)
		util.WithValue(c, hashid.ObjectIDCtx{}, uid)
		return c
	}

	// No pending change → rejected.
	require.Error(t, ActivateEmailChange(newCtx(u.ID)))

	// Pending change applies.
	require.NoError(t, kv.Set(fmt.Sprintf("%s%d", emailChangeSessionKey, u.ID), "new@example.com", 60))
	require.NoError(t, ActivateEmailChange(newCtx(u.ID)))
	require.Equal(t, "new@example.com", client.User.GetX(ctx, u.ID).Email)

	// Replay after success is rejected (KV entry consumed).
	require.Error(t, ActivateEmailChange(newCtx(u.ID)))

	// Address claimed between request and confirm is rejected.
	require.NoError(t, kv.Set(fmt.Sprintf("%s%d", emailChangeSessionKey, u.ID), "taken@example.com", 60))
	require.Error(t, ActivateEmailChange(newCtx(u.ID)))
	require.Equal(t, "new@example.com", client.User.GetX(ctx, u.ID).Email)
}
