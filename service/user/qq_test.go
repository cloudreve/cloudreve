package user

import (
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

func TestValidateQQConfig(t *testing.T) {
	require.Error(t, validateQQConfig(&setting.QQConnect{}))
	require.Error(t, validateQQConfig(&setting.QQConnect{Enabled: true}))
	require.Error(t, validateQQConfig(&setting.QQConnect{Enabled: true, AppID: "id"}))
	require.NoError(t, validateQQConfig(&setting.QQConnect{Enabled: true, AppID: "id", AppSecret: "secret"}))
}

func TestParseQQTokenResponse(t *testing.T) {
	token, err := parseQQTokenResponse("access_token=ABC123&expires_in=7776000&refresh_token=DEF")
	require.NoError(t, err)
	require.Equal(t, "ABC123", token)

	// JSONP error reply carries no access_token.
	_, err = parseQQTokenResponse(`callback({"error":100016,"error_description":"app secret error"});`)
	require.Error(t, err)

	_, err = parseQQTokenResponse("")
	require.Error(t, err)
	_, err = parseQQTokenResponse("expires_in=7776000")
	require.Error(t, err)
}

func TestParseQQOpenIDResponse(t *testing.T) {
	openID, err := parseQQOpenIDResponse(`callback( {"client_id":"APPID","openid":"OPENIDXYZ"} );`, "APPID")
	require.NoError(t, err)
	require.Equal(t, "OPENIDXYZ", openID)

	// A token minted for a different app must not be accepted.
	_, err = parseQQOpenIDResponse(`callback( {"client_id":"OTHER","openid":"OPENIDXYZ"} );`, "APPID")
	require.Error(t, err)

	_, err = parseQQOpenIDResponse(`callback( {"client_id":"APPID"} );`, "APPID")
	require.Error(t, err)
	_, err = parseQQOpenIDResponse(`callback( not-json );`, "APPID")
	require.Error(t, err)
	_, err = parseQQOpenIDResponse(`{"client_id":"APPID","openid":"X"}`, "APPID")
	require.Error(t, err)
	_, err = parseQQOpenIDResponse("", "APPID")
	require.Error(t, err)
}
