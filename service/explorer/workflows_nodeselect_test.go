package explorer

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	entnode "github.com/cloudreve/Cloudreve/v4/ent/node"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// nodeSelDepStub exposes only the dependencies resolveNodeSelection touches.
type nodeSelDepStub struct {
	dependency.Dep
	nodeClient inventory.NodeClient
	hasher     hashid.Encoder
}

func (d *nodeSelDepStub) NodeClient() inventory.NodeClient { return d.nodeClient }
func (d *nodeSelDepStub) HashIDEncoder() hashid.Encoder    { return d.hasher }

func enableCaps(flags ...int) *boolset.BooleanSet {
	b := boolset.BooleanSet(make([]byte, 4))
	for _, f := range flags {
		b[f/8] |= 1 << uint(f%8)
	}
	return &b
}

func TestResolveNodeSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	mk := func(name string, status entnode.Status, caps *boolset.BooleanSet) *ent.Node {
		return client.Node.Create().SetName(name).SetStatus(status).SetType(entnode.TypeSlave).
			SetCapabilities(caps).SaveX(ctx)
	}
	downloadCapable := mk("dl", entnode.StatusActive, enableCaps(int(types.NodeCapabilityRemoteDownload)))
	archiveOnly := mk("zip", entnode.StatusActive, enableCaps(int(types.NodeCapabilityCreateArchive)))
	suspended := mk("off", entnode.StatusSuspended, enableCaps(int(types.NodeCapabilityRemoteDownload)))

	seq := 0
	newUser := func(settings *types.GroupSetting) *ent.User {
		seq++
		g := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).
			SetSettings(settings).SaveX(ctx)
		u := client.User.Create().SetEmail(fmt.Sprintf("u%d@example.com", seq)).SetNick("u").SetStatus("active").SetGroup(g).SaveX(ctx)
		return client.User.Query().WithGroup().Where(entuser.ID(u.ID)).OnlyX(ctx)
	}

	hasher, err := hashid.New("node-sel-test-salt")
	require.NoError(t, err)
	dep := &nodeSelDepStub{nodeClient: inventory.NewNodeClient(client), hasher: hasher}

	newCtx := func(u *ent.User) *gin.Context {
		engine := gin.New()
		engine.ContextWithFallback = true
		c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
		c.Request = httptest.NewRequest("POST", "/", nil)
		util.WithValue(c, dependency.DepCtx{}, dep)
		util.WithValue(c, inventory.UserCtx{}, u)
		return c
	}

	t.Run("empty target returns allowed pool for auto dispatch", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowedNodes: []int{downloadCapable.ID}})
		sel, err := resolveNodeSelection(newCtx(u), dep, "", types.NodeCapabilityRemoteDownload)
		require.NoError(t, err)
		require.Equal(t, 0, sel.TargetNodeID)
		require.Equal(t, []int{downloadCapable.ID}, sel.AllowedNodes)
	})

	t.Run("explicit pick within allowed pool wins", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowedNodes: []int{downloadCapable.ID}, AllowSelectNode: true})
		sel, err := resolveNodeSelection(newCtx(u), dep, hashid.EncodeNodeID(hasher, downloadCapable.ID),
			types.NodeCapabilityRemoteDownload)
		require.NoError(t, err)
		require.Equal(t, downloadCapable.ID, sel.TargetNodeID)
	})

	t.Run("pick rejected when group disallows selection", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowSelectNode: false})
		_, err := resolveNodeSelection(newCtx(u), dep, hashid.EncodeNodeID(hasher, downloadCapable.ID),
			types.NodeCapabilityRemoteDownload)
		require.Error(t, err)
	})

	t.Run("pick outside allowed pool is rejected", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowedNodes: []int{archiveOnly.ID}, AllowSelectNode: true})
		_, err := resolveNodeSelection(newCtx(u), dep, hashid.EncodeNodeID(hasher, downloadCapable.ID),
			types.NodeCapabilityRemoteDownload)
		require.Error(t, err)
	})

	t.Run("node without the task capability is rejected", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowSelectNode: true})
		_, err := resolveNodeSelection(newCtx(u), dep, hashid.EncodeNodeID(hasher, archiveOnly.ID),
			types.NodeCapabilityRemoteDownload)
		require.Error(t, err)
	})

	t.Run("suspended node is rejected", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowSelectNode: true})
		_, err := resolveNodeSelection(newCtx(u), dep, hashid.EncodeNodeID(hasher, suspended.ID),
			types.NodeCapabilityRemoteDownload)
		require.Error(t, err)
	})

	t.Run("invalid node hashid is rejected", func(t *testing.T) {
		u := newUser(&types.GroupSetting{AllowSelectNode: true})
		_, err := resolveNodeSelection(newCtx(u), dep, "!!!", types.NodeCapabilityRemoteDownload)
		require.Error(t, err)
	})
}
