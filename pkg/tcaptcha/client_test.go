package tcaptcha

import (
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

func TestSignProducesStableAuthorization(t *testing.T) {
	auth := sign("AKIDTEST", "secret", `{"CaptchaType":9}`, 1700000000)

	require.True(t, strings.HasPrefix(auth, "TC3-HMAC-SHA256 Credential=AKIDTEST/2023-11-14/captcha/tc3_request"))
	require.Contains(t, auth, "SignedHeaders=content-type;host;x-tc-action")
	require.Contains(t, auth, "Signature=")

	// Signature is the last 64 hex chars and deterministic.
	sig := auth[strings.LastIndex(auth, "Signature=")+len("Signature="):]
	require.Len(t, sig, 64)
	require.Equal(t, auth, sign("AKIDTEST", "secret", `{"CaptchaType":9}`, 1700000000))
	require.NotEqual(t, auth, sign("AKIDTEST", "secret2", `{"CaptchaType":9}`, 1700000000))
}

func TestVerifyRejectsBadAppID(t *testing.T) {
	ok, err := Verify(t.Context(), nil, &setting.TcCaptcha{AppID: "not-a-number"}, "t", "r", "1.2.3.4")
	require.Error(t, err)
	require.False(t, ok)
}
