package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

type stubSettingProvider struct {
	setting.Provider
}

func (stubSettingProvider) AuditLogEnabled(context.Context, int) bool { return true }

type stubTokenAuth struct {
	uid    int
	scopes []string
}

func (s stubTokenAuth) Issue(ctx context.Context, args *auth.IssueTokenArgs) (*auth.Token, error) {
	return nil, nil
}

func (s stubTokenAuth) VerifyAndRetrieveUser(c *gin.Context) (bool, error) {
	if c.GetHeader(auth.AuthorizationHeader) == "Bearer good" {
		util.WithValue(c, inventory.UserIDCtx{}, s.uid)
		if s.scopes != nil {
			util.WithValue(c, auth.ScopeContextKey{}, s.scopes)
		}
	}
	return false, nil
}

func (s stubTokenAuth) Refresh(ctx context.Context, refreshToken string) (*auth.Token, error) {
	return nil, nil
}

func (s stubTokenAuth) Claims(ctx context.Context, tokenStr string) (*auth.Claims, error) {
	return nil, nil
}

func newDavAuthDep(t *testing.T, groupPerms *boolset.BooleanSet, scopes []string) dependency.Dep {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })

	group := client.Group.Create().
		SetName("g").
		SetPermissions(groupPerms).
		SaveX(context.Background())

	u := client.User.Create().
		SetEmail("dav@test.com").
		SetNick("dav").
		SetStatus("active").
		SetPassword("x").
		SetGroup(group).
		SaveX(context.Background())

	return dependency.NewDependency(
		dependency.WithKV(cache.NewMemoStore("", nil)),
		dependency.WithLogger(logging.NewConsoleLogger(logging.LevelDebug)),
		dependency.WithUserClient(inventory.NewUserClient(client)),
		dependency.WithTokenAuth(stubTokenAuth{uid: u.ID, scopes: scopes}),
		dependency.WithSettingProvider(stubSettingProvider{}),
		dependency.WithActivityClient(inventory.NewActivityClient(client, conf.SQLiteDB)),
	)
}

func newDavRequest(t *testing.T, dep dependency.Dep, method, authHeader string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c := gin.CreateTestContextOnly(w, testEngine)
	req := httptest.NewRequest(method, "/dav/file.txt", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	c.Request = req.WithContext(context.WithValue(req.Context(), dependency.DepCtx{}, dep))
	return c, w
}

func TestWebDAVBearerAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := WebDAVAuth()

	davPerms := &boolset.BooleanSet{}
	boolset.Set(int(types.GroupPermissionWebDAV), true, davPerms)
	readOnlyPerms := &boolset.BooleanSet{}
	boolset.Set(int(types.GroupPermissionWebDAV), true, readOnlyPerms)
	boolset.Set(int(types.GroupPermissionWebDAVReadOnly), true, readOnlyPerms)

	t.Run("valid bearer token authenticates", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, nil)
		c, _ := newDavRequest(t, dep, http.MethodGet, "Bearer good")
		handler(c)
		if c.IsAborted() {
			t.Fatal("valid bearer token was rejected")
		}
		u := inventory.UserFromContext(c)
		if u == nil || u.Email != "dav@test.com" {
			t.Fatalf("expected bearer user in ctx, got %+v", u)
		}
	})

	t.Run("invalid bearer token rejected", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, nil)
		c, _ := newDavRequest(t, dep, http.MethodGet, "Bearer bad")
		handler(c)
		if !c.IsAborted() || c.Writer.Status() != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", c.Writer.Status())
		}
	})

	t.Run("no credentials still issues basic+bearer challenge", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, nil)
		c, w := newDavRequest(t, dep, http.MethodGet, "")
		handler(c)
		if c.Writer.Status() != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", c.Writer.Status())
		}
		if challenges := w.Header().Values("WWW-Authenticate"); len(challenges) != 2 {
			t.Fatalf("expected Basic+Bearer challenges, got %v", challenges)
		}
	})

	t.Run("options without credentials passes", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, nil)
		c, _ := newDavRequest(t, dep, http.MethodOptions, "")
		handler(c)
		if c.IsAborted() {
			t.Fatal("unauthenticated OPTIONS was rejected")
		}
	})

	t.Run("bearer user without group dav permission forbidden", func(t *testing.T) {
		dep := newDavAuthDep(t, &boolset.BooleanSet{}, nil)
		c, _ := newDavRequest(t, dep, http.MethodGet, "Bearer good")
		handler(c)
		if c.Writer.Status() != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", c.Writer.Status())
		}
	})

	t.Run("group read-only blocks bearer writes", func(t *testing.T) {
		dep := newDavAuthDep(t, readOnlyPerms, nil)
		c, _ := newDavRequest(t, dep, http.MethodPut, "Bearer good")
		handler(c)
		if c.Writer.Status() != http.StatusForbidden {
			t.Fatalf("expected 403 for PUT, got %d", c.Writer.Status())
		}
		c, _ = newDavRequest(t, dep, http.MethodGet, "Bearer good")
		handler(c)
		if c.IsAborted() {
			t.Fatal("read-only bearer GET was rejected")
		}
	})

	t.Run("scoped token without Files.Write is read-only", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, []string{types.ScopeFilesRead})
		c, _ := newDavRequest(t, dep, http.MethodDelete, "Bearer good")
		handler(c)
		if c.Writer.Status() != http.StatusForbidden {
			t.Fatalf("expected 403 for DELETE, got %d", c.Writer.Status())
		}
		c, _ = newDavRequest(t, dep, http.MethodGet, "Bearer good")
		handler(c)
		if c.IsAborted() {
			t.Fatal("scoped read token GET was rejected")
		}
	})

	t.Run("scoped token without Files.Read forbidden", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, []string{"User.Read"})
		c, _ := newDavRequest(t, dep, http.MethodGet, "Bearer good")
		handler(c)
		if c.Writer.Status() != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", c.Writer.Status())
		}
	})

	t.Run("basic auth path unchanged", func(t *testing.T) {
		dep := newDavAuthDep(t, davPerms, nil)
		c, _ := newDavRequest(t, dep, http.MethodGet, "Basic dGVzdDp0ZXN0")
		handler(c)
		// No dav account exists -> 401 via the basic credential path.
		if c.Writer.Status() != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", c.Writer.Status())
		}
	})
}
