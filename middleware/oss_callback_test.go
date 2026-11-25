package middleware

import (
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/oss"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ossCallbackTestUsers struct {
	inventory.UserClient
}

func (ossCallbackTestUsers) GetLoginUserByID(_ context.Context, id int) (*ent.User, error) {
	return &ent.User{ID: id}, nil
}

type ossCallbackTestDep struct {
	dependency.Dep
}

func (ossCallbackTestDep) RequestClient(...request.Option) request.Client { return nil }

type ossCallbackTestBody struct {
	io.Reader
	read int
}

func (b *ossCallbackTestBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (*ossCallbackTestBody) Close() error { return nil }

func TestOSSCallbackAuthBoundsBodyAndPreservesSession(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey})
	const payload = `{"name":"file.txt","source_name":"uploads/file.txt","size":1,"pic_info":""}`
	oversized := payload + strings.Repeat(" ", oss.MaxCallbackBodySize+4096-len(payload))

	for _, test := range []struct {
		name           string
		body           string
		unknownLength  bool
		missingSession bool
		wrongKey       bool
		wrongPolicy    bool
		wantStatus     int
		wantRead       int
	}{
		{name: "oversized", body: oversized, wantStatus: http.StatusRequestEntityTooLarge, wantRead: oss.MaxCallbackBodySize + 1},
		{name: "unknown length", body: oversized, unknownLength: true, wantStatus: http.StatusRequestEntityTooLarge, wantRead: oss.MaxCallbackBodySize + 1},
		{name: "invalid signature", body: payload, wantStatus: http.StatusUnauthorized, wantRead: len(payload)},
		{name: "missing session", body: oversized, missingSession: true, wantStatus: http.StatusUnauthorized},
		{name: "wrong callback secret", body: oversized, wrongKey: true, wantStatus: http.StatusUnauthorized},
		{name: "wrong policy", body: oversized, wrongPolicy: true, wantStatus: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := logging.NewConsoleLogger(logging.LevelError)
			kv := cache.NewMemoStore("", logger)
			require.NoError(t, kv.Set("oss_public_key", pemKey, 600))
			session := fs.UploadSession{
				UID: 1, CallbackSecret: "callback-test-secret",
				Policy: &ent.StoragePolicy{Type: types.PolicyTypeOss},
			}
			if test.wrongPolicy {
				session.Policy.Type = types.PolicyTypeLocal
			}
			sessionKey := manager.UploadSessionCachePrefix + "session"
			if !test.missingSession {
				require.NoError(t, kv.Set(sessionKey, session, 600))
			}
			before, existed := kv.Store.Load(sessionKey)
			dep := ossCallbackTestDep{Dep: dependency.NewDependency(
				dependency.WithLogger(logger),
				dependency.WithKV(kv),
				dependency.WithUserClient(ossCallbackTestUsers{}),
			)}
			router := gin.New()
			router.ContextWithFallback = true
			router.Use(func(c *gin.Context) {
				util.WithValue(c, dependency.DepCtx{}, dep)
				util.WithValue(c, logging.LoggerCtx{}, logger)
				c.Next()
			})
			nextCalled := false
			router.POST("/api/v4/callback/oss/:sessionID/:key",
				UseUploadSession(types.PolicyTypeOss),
				OSSCallbackAuth(),
				func(c *gin.Context) {
					nextCalled = true
					require.Equal(t, session.UID, inventory.UserIDFromContext(c))
					body, err := io.ReadAll(c.Request.Body)
					require.NoError(t, err)
					require.Equal(t, payload, string(body))
					c.Status(http.StatusNoContent)
				},
			)

			path := "/api/v4/callback/oss/session/" + session.CallbackSecret
			if test.wrongKey {
				path += "-wrong"
			}
			body := &ossCallbackTestBody{Reader: strings.NewReader(test.body)}
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Body = body
			req.ContentLength = int64(len(test.body))
			req.Header.Set("Authorization", base64.StdEncoding.EncodeToString([]byte("invalid signature")))
			if test.unknownLength {
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			require.Equal(t, test.wantStatus, recorder.Code)
			require.Equal(t, test.wantRead, body.read)
			require.False(t, nextCalled)
			after, exists := kv.Store.Load(sessionKey)
			require.Equal(t, existed, exists)
			require.Equal(t, before, after, "callback rejection must preserve the session and its expiry")

			if test.wantStatus == http.StatusRequestEntityTooLarge {
				var response serializer.GeneralUploadCallbackFailed
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, "Callback request body too large.", response.Error)

				digest := md5.Sum([]byte(path + "\n" + payload))
				signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.MD5, digest[:])
				require.NoError(t, err)
				validRequest := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
				validRequest.Header.Set("Authorization", base64.StdEncoding.EncodeToString(signature))
				validRecorder := httptest.NewRecorder()
				router.ServeHTTP(validRecorder, validRequest)

				require.Equal(t, http.StatusNoContent, validRecorder.Code)
				require.True(t, nextCalled, "the original session must still accept a valid callback")
			}
		})
	}
}
