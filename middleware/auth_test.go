package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type webDAVAuthTestUsers struct {
	inventory.UserClient
	getActiveByDavAccount func(context.Context, string, string) (*ent.User, error)
}

func (u webDAVAuthTestUsers) GetActiveByDavAccount(ctx context.Context, email, password string) (*ent.User, error) {
	return u.getActiveByDavAccount(ctx, email, password)
}

func TestWebDAVAuth(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(oldMode)
	})

	tests := []struct {
		name            string
		method          string
		body            string
		readOnlyAllowed bool
	}{
		{name: "GET", method: http.MethodGet, readOnlyAllowed: true},
		{name: "HEAD", method: http.MethodHead, readOnlyAllowed: true},
		{name: "OPTIONS", method: http.MethodOptions, readOnlyAllowed: true},
		{name: "POST", method: http.MethodPost, readOnlyAllowed: true},
		{name: "PROPFIND", method: "PROPFIND", readOnlyAllowed: true},
		{name: "DELETE", method: http.MethodDelete},
		{name: "PUT", method: http.MethodPut},
		{name: "MKCOL", method: "MKCOL"},
		{name: "COPY", method: "COPY"},
		{name: "MOVE", method: "MOVE"},
		{name: "LOCK", method: "LOCK"},
		{name: "UNLOCK", method: "UNLOCK"},
		{
			name:   "PROPPATCH/set",
			method: "PROPPATCH",
			body:   `<D:propertyupdate xmlns:D="DAV:" xmlns:X="urn:test"><D:set><D:prop><X:note>updated</X:note></D:prop></D:set></D:propertyupdate>`,
		},
		{
			name:   "PROPPATCH/remove",
			method: "PROPPATCH",
			body:   `<D:propertyupdate xmlns:D="DAV:" xmlns:X="urn:test"><D:remove><D:prop><X:note/></D:prop></D:remove></D:propertyupdate>`,
		},
		{name: "PATCH", method: http.MethodPatch},
		{name: "unknown method", method: "CUSTOM"},
	}

	for _, mode := range []struct {
		name     string
		readOnly bool
	}{
		{name: "read only", readOnly: true},
		{name: "read write"},
	} {
		t.Run(mode.name, func(t *testing.T) {
			account := &ent.DavAccount{Options: &boolset.BooleanSet{}}
			boolset.Set(types.DavAccountReadOnly, mode.readOnly, account.Options)
			group := &ent.Group{Permissions: &boolset.BooleanSet{}}
			boolset.Set(types.GroupPermissionWebDAV, true, group.Permissions)
			user := &ent.User{ID: 1, Email: "owner@example.com"}
			user.SetGroup(group)
			user.SetDavAccounts([]*ent.DavAccount{account})

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					dep := dependency.NewDependency(
						dependency.WithLogger(logging.NewConsoleLogger(logging.LevelError)),
						dependency.WithUserClient(webDAVAuthTestUsers{
							getActiveByDavAccount: func(_ context.Context, email, password string) (*ent.User, error) {
								require.Equal(t, user.Email, email)
								require.Equal(t, "dav-test-password", password)
								return user, nil
							},
						}),
					)
					router := gin.New()
					router.ContextWithFallback = true
					router.Use(func(c *gin.Context) {
						util.WithValue(c, dependency.DepCtx{}, dep)
						c.Next()
					}, WebDAVAuth())
					called := false
					router.Handle(test.method, "/dav/file.txt", func(c *gin.Context) {
						called = true
						require.Same(t, user, inventory.UserFromContext(c))
						c.Status(http.StatusNoContent)
					})

					req := httptest.NewRequest(test.method, "/dav/file.txt", strings.NewReader(test.body))
					req.SetBasicAuth(user.Email, "dav-test-password")
					if test.body != "" {
						req.Header.Set("Content-Type", "application/xml")
					}
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, req)

					allowed := !mode.readOnly || test.readOnlyAllowed
					wantStatus := http.StatusForbidden
					if allowed {
						wantStatus = http.StatusNoContent
					}
					require.Equal(t, wantStatus, recorder.Code)
					require.Equal(t, allowed, called, "denied requests must not reach the DAV handler")
				})
			}
		})
	}

	t.Run("anonymous OPTIONS", func(t *testing.T) {
		router := gin.New()
		router.OPTIONS("/dav/file.txt", WebDAVAuth(), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/dav/file.txt", nil))

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})
}
