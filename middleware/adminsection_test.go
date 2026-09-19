package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

func perms(bits ...types.GroupPermission) *boolset.BooleanSet {
	b := &boolset.BooleanSet{}
	for _, p := range bits {
		boolset.Set(int(p), true, b)
	}
	return b
}

func newAdminRequest(t *testing.T, permissions *boolset.BooleanSet) *gin.Context {
	t.Helper()
	w := httptest.NewRecorder()
	c := gin.CreateTestContextOnly(w, testEngine)
	c.Request = httptest.NewRequest("GET", "/api/v4/admin/summary", nil)
	dep := dependency.NewDependency(
		dependency.WithLogger(logging.NewConsoleLogger(logging.LevelDebug)),
	)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), dependency.DepCtx{}, dep))
	u := &ent.User{
		ID: 2,
		Edges: ent.UserEdges{
			Group: &ent.Group{Permissions: permissions},
		},
	}
	util.WithValue(c, inventory.UserCtx{}, u)
	return c
}

func TestIsAdminOrDelegated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := IsAdminOrDelegated()

	t.Run("full admin passes", func(t *testing.T) {
		c := newAdminRequest(t, perms(types.GroupPermissionIsAdmin))
		handler(c)
		if c.IsAborted() {
			t.Fatal("full admin was rejected")
		}
	})

	t.Run("delegated admin passes", func(t *testing.T) {
		c := newAdminRequest(t, perms(types.GroupPermissionAdminUsers))
		handler(c)
		if c.IsAborted() {
			t.Fatal("delegated admin was rejected")
		}
	})

	t.Run("regular user rejected", func(t *testing.T) {
		c := newAdminRequest(t, perms(types.GroupPermissionShare))
		handler(c)
		if !c.IsAborted() {
			t.Fatal("regular user was not rejected")
		}
	})
}

func TestAdminSection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := AdminSection(types.GroupPermissionAdminUsers, types.GroupPermissionAdminGroups)

	t.Run("full admin passes any section", func(t *testing.T) {
		c := newAdminRequest(t, perms(types.GroupPermissionIsAdmin))
		handler(c)
		if c.IsAborted() {
			t.Fatal("full admin was rejected")
		}
	})

	t.Run("matching delegated section passes", func(t *testing.T) {
		c := newAdminRequest(t, perms(types.GroupPermissionAdminUsers))
		handler(c)
		if c.IsAborted() {
			t.Fatal("delegated admin with matching section was rejected")
		}
	})

	t.Run("non-matching delegated section rejected", func(t *testing.T) {
		c := newAdminRequest(t, perms(types.GroupPermissionAdminFiles))
		handler(c)
		if !c.IsAborted() {
			t.Fatal("delegated admin without matching section was not rejected")
		}
	})

	t.Run("regular user rejected", func(t *testing.T) {
		c := newAdminRequest(t, perms())
		handler(c)
		if !c.IsAborted() {
			t.Fatal("regular user was not rejected")
		}
	})
}
