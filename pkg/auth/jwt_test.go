package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestFirstPartyRequiresVerifiedAccessToken(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	encoder, err := hashid.New("first-party-auth-test")
	require.NoError(t, err)
	secret := []byte("first-party-auth-test-secret")
	tokenAuth := NewTokenAuth(encoder, nil, secret, nil, logging.NewConsoleLogger(logging.LevelError), nil, nil)

	for _, test := range []struct {
		name        string
		clientID    string
		tokenType   TokenType
		signingKey  []byte
		subject     string
		expires     time.Time
		allowed     bool
		verifyError bool
	}{
		{
			name: "first-party access", tokenType: TokenTypeAccess, signingKey: secret,
			subject: hashid.EncodeUserID(encoder, 1), expires: time.Now().Add(time.Hour), allowed: true,
		},
		{
			name: "OAuth access without scopes", tokenType: TokenTypeAccess, signingKey: secret,
			subject: hashid.EncodeUserID(encoder, 1), expires: time.Now().Add(time.Hour), clientID: "oauth-client",
		},
		{
			name: "invalid signature", tokenType: TokenTypeAccess, signingKey: []byte("wrong-secret"),
			subject: hashid.EncodeUserID(encoder, 1), expires: time.Now().Add(time.Hour),
		},
		{
			name: "expired access", tokenType: TokenTypeAccess, signingKey: secret,
			subject: hashid.EncodeUserID(encoder, 1), expires: time.Now().Add(-time.Hour),
		},
		{
			name: "refresh token", tokenType: TokenTypeRefresh, signingKey: secret,
			subject: hashid.EncodeUserID(encoder, 1), expires: time.Now().Add(time.Hour), verifyError: true,
		},
		{
			name: "invalid user ID", tokenType: TokenTypeAccess, signingKey: secret,
			subject: hashid.EncodeFileID(encoder, 1), expires: time.Now().Add(time.Hour), verifyError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, engine := gin.CreateTestContext(httptest.NewRecorder())
			engine.ContextWithFallback = true
			c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
				TokenType: test.tokenType,
				ClientID:  test.clientID,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: test.subject, ExpiresAt: jwt.NewNumericDate(test.expires),
				},
			}).SignedString(test.signingKey)
			require.NoError(t, err)
			c.Request.Header.Set(AuthorizationHeader, TokenHeaderPrefix+token)

			_, err = tokenAuth.VerifyAndRetrieveUser(c)
			if test.verifyError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			err = CheckFirstParty(c)
			if test.allowed {
				require.NoError(t, err)
				require.Equal(t, 1, inventory.UserIDFromContext(c))
			} else {
				var appErr serializer.AppError
				require.ErrorAs(t, err, &appErr)
				require.Equal(t, serializer.CodeNoPermissionErr, appErr.Code)
			}
		})
	}
}
