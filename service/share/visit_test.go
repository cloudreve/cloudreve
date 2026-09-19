package share

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// stubSettingProvider satisfies the SettingProvider dependency; only
// ShareDefaults, SiteURL and ExposeUserEmail are exercised by
// ListInUserProfile and the share response builder.
type stubSettingProvider struct {
	setting.Provider
	defaults *setting.ShareDefaults
}

func (s stubSettingProvider) ShareDefaults(context.Context) *setting.ShareDefaults {
	return s.defaults
}

func (stubSettingProvider) SiteURL(context.Context) *url.URL {
	return &url.URL{Scheme: "https", Host: "example.com"}
}

func (stubSettingProvider) ExposeUserEmail(context.Context) bool { return false }

// TestListInUserProfileVisibilityResolution ensures a user's empty
// share_links_in_profile inherits the site-wide default while explicit
// values always win (upstream #3390).
func TestListInUserProfileVisibilityResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	hasher, err := hashid.New("test-salt")
	require.NoError(t, err)

	group := client.Group.Create().SetName("g").SetPermissions(&boolset.BooleanSet{}).SaveX(ctx)

	newOwner := func(email string, level types.ShareLinksInProfileLevel) int {
		owner := client.User.Create().
			SetEmail(email).
			SetNick("owner").
			SetStatus("active").
			SetGroup(group).
			SetSettings(&types.UserSetting{ShareLinksInProfile: level}).
			SaveX(ctx)
		file := client.File.Create().SetName("shared.txt").SetType(int(types.FileTypeFile)).SetOwner(owner).SaveX(ctx)
		client.Share.Create().SetUser(owner).SetFile(file).SaveX(ctx)                   // public share
		client.Share.Create().SetUser(owner).SetFile(file).SetPassword("pw").SaveX(ctx) // private share
		return owner.ID
	}

	newCtx := func(dep dependency.Dep, requesterID int) *gin.Context {
		w := httptest.NewRecorder()
		engine := gin.New()
		engine.ContextWithFallback = true
		c := gin.CreateTestContextOnly(w, engine)
		c.Request = httptest.NewRequest("GET", "/", nil)
		util.WithValue(c, dependency.DepCtx{}, dep)
		util.WithValue(c, inventory.UserCtx{}, client.User.GetX(context.Background(), requesterID))
		return c
	}

	newDep := func(defaults *setting.ShareDefaults) dependency.Dep {
		return dependency.NewDependency(
			dependency.WithKV(cache.NewMemoStore("", nil)),
			dependency.WithLogger(logging.NewConsoleLogger(logging.LevelDebug)),
			dependency.WithUserClient(inventory.NewUserClient(client)),
			dependency.WithShareClient(inventory.NewShareClient(client, conf.SQLiteDB, hasher)),
			dependency.WithHashIDEncoder(hasher),
			dependency.WithSettingProvider(stubSettingProvider{defaults: defaults}),
		)
	}

	svc := &ListShareService{PageSize: 20}

	tests := []struct {
		name        string
		userLevel   types.ShareLinksInProfileLevel
		siteDefault types.ShareLinksInProfileLevel
		wantCount   int
		wantErr     bool
	}{
		{name: "unset inherits site default all_share", userLevel: types.ProfilePublicShareOnly, siteDefault: types.ProfileAllShare, wantCount: 2},
		{name: "unset inherits site default public only", userLevel: types.ProfilePublicShareOnly, siteDefault: types.ProfilePublicShareOnly, wantCount: 1},
		{name: "unset inherits site default hide", userLevel: types.ProfilePublicShareOnly, siteDefault: types.ProfileHideShare, wantErr: true},
		{name: "explicit all_share overrides site public", userLevel: types.ProfileAllShare, siteDefault: types.ProfilePublicShareOnly, wantCount: 2},
		{name: "explicit public_share overrides site all", userLevel: types.ProfileSharePublic, siteDefault: types.ProfileAllShare, wantCount: 1},
		{name: "explicit hide overrides site all", userLevel: types.ProfileHideShare, siteDefault: types.ProfileAllShare, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid := newOwner(tt.name+"@example.com", tt.userLevel)
			dep := newDep(&setting.ShareDefaults{LinksInProfile: tt.siteDefault})
			c := newCtx(dep, uid)

			res, err := svc.ListInUserProfile(c, uid)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, res.Shares, tt.wantCount)
		})
	}
}
