package middleware

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/cloudreve/Cloudreve/v4/pkg/wopi"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// WopiWriteAccess validates if write access is obtained.
func WopiWriteAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := c.MustGet(wopi.WopiSessionCtx).(*wopi.SessionCache)
		if session.Action != wopi.ActionEdit {
			c.Status(http.StatusNotFound)
			c.Header(wopi.ServerErrorHeader, "read-only access")
			c.Abort()
			return
		}

		c.Next()
	}
}

func ViewerSessionValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		dep := dependency.FromContext(c)
		store := dep.KV()
		settings := dep.SettingProvider()

		accessToken := strings.Split(c.Query(wopi.AccessTokenQuery), ".")
		if len(accessToken) != 2 {
			c.Status(http.StatusForbidden)
			c.Header(wopi.ServerErrorHeader, "malformed access token")
			c.Abort()
			return
		}

		sessionRaw, exist := store.Get(manager.ViewerSessionCachePrefix + accessToken[0])
		if !exist {
			c.Status(http.StatusForbidden)
			c.Header(wopi.ServerErrorHeader, "invalid access token")
			c.Abort()
			return
		}

		session := sessionRaw.(manager.ViewerSessionCache)
		if err := SetUserCtx(c, session.UserID); err != nil {
			c.Status(http.StatusInternalServerError)
			c.Header(wopi.ServerErrorHeader, "user not found")
			c.Abort()
			return
		}

		fileId := hashid.FromContext(c)
		if fileId != session.FileID {
			c.Status(http.StatusForbidden)
			c.Header(wopi.ServerErrorHeader, "invalid file")
			c.Abort()
			return
		}

		// Check if the viewer is still available
		viewers := settings.FileViewers(c)
		var v *types.Viewer
		for _, group := range viewers {
			for _, viewer := range group.Viewers {
				if viewer.ID == session.ViewerID && !viewer.Disabled {
					v = &viewer
					break
				}
			}

			if v != nil {
				break
			}
		}

		if v == nil {
			c.Status(http.StatusInternalServerError)
			c.Header(wopi.ServerErrorHeader, "viewer not found")
			c.Abort()
			return
		}

		util.WithValue(c, manager.ViewerCtx{}, v)
		util.WithValue(c, manager.ViewerSessionCacheCtx{}, &session)
		c.Next()
	}
}
