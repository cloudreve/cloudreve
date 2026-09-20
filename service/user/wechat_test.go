package user

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWeChatToken(t *testing.T) {
	token, err := parseWeChatToken(`{"access_token":"AT","openid":"o","unionid":"u"}`)
	require.NoError(t, err)
	require.Equal(t, "AT", token.AccessToken)
	require.Equal(t, "u", token.UnionID)

	_, err = parseWeChatToken(`{"errcode":40029,"errmsg":"invalid code"}`)
	require.Error(t, err)

	_, err = parseWeChatToken(`{"openid":"o"}`) // no access_token
	require.Error(t, err)

	_, err = parseWeChatToken(`not json`)
	require.Error(t, err)
}
