package explorer

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// policyDepStub exposes only the dependencies decodeAllowedPolicy touches.
type policyDepStub struct {
	dependency.Dep
	policyClient inventory.StoragePolicyClient
	hasher       hashid.Encoder
}

func (d *policyDepStub) StoragePolicyClient() inventory.StoragePolicyClient { return d.policyClient }
func (d *policyDepStub) HashIDEncoder() hashid.Encoder                      { return d.hasher }

func TestDecodeAllowedPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	mk := func(name string) *ent.StoragePolicy {
		return client.StoragePolicy.Create().SetName(name).SetType("local").
			SetStatus(storagepolicy.StatusActive).SaveX(ctx)
	}
	def := mk("def")
	a := mk("a")
	outside := mk("outside")

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).
		SetStoragePolicies(def).AddAllowedPolicies(a).SaveX(ctx)

	hasher, err := hashid.New("policy-test-salt")
	require.NoError(t, err)
	dep := &policyDepStub{policyClient: inventory.NewStoragePolicyClient(client, nil), hasher: hasher}

	engine := gin.New()
	engine.ContextWithFallback = true
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), engine)
	c.Request = httptest.NewRequest("GET", "/", nil)
	util.WithValue(c, dependency.DepCtx{}, dep)

	t.Run("allowed policy decodes", func(t *testing.T) {
		got, err := decodeAllowedPolicy(c, dep, []*ent.Group{group}, hashid.EncodePolicyID(hasher, a.ID))
		require.NoError(t, err)
		require.Equal(t, a.ID, got.ID)
	})

	t.Run("legacy default policy decodes", func(t *testing.T) {
		got, err := decodeAllowedPolicy(c, dep, []*ent.Group{group}, hashid.EncodePolicyID(hasher, def.ID))
		require.NoError(t, err)
		require.Equal(t, def.ID, got.ID)
	})

	t.Run("policy outside the group set is rejected", func(t *testing.T) {
		_, err := decodeAllowedPolicy(c, dep, []*ent.Group{group}, hashid.EncodePolicyID(hasher, outside.ID))
		require.Error(t, err)
	})

	t.Run("invalid hashid is rejected", func(t *testing.T) {
		_, err := decodeAllowedPolicy(c, dep, []*ent.Group{group}, "!!!")
		require.Error(t, err)
	})
}
