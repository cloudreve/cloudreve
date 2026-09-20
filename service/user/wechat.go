package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
)

type (
	// WeChatLoginParameterCtx marks the WeChat scan-login start route.
	WeChatLoginParameterCtx struct{}
	// WeChatCallbackParameterCtx marks the WeChat redirect target.
	WeChatCallbackParameterCtx struct{}

	// WeChatLoginService starts the WeChat Open Platform scan flow.
	// `link=1` binds the WeChat identity to the currently signed-in user
	// instead of signing in.
	WeChatLoginService struct {
		Redirect string `form:"redirect"`
		Link     bool   `form:"link"`
	}

	// WeChatCallbackService completes the WeChat flow.
	WeChatCallbackService struct {
		Code  string `form:"code"`
		State string `form:"state"`
	}
)

const (
	wechatAuthorizeEndpoint = "https://open.weixin.qq.com/connect/qrconnect"
	wechatTokenEndpoint     = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wechatUserInfoEndpoint  = "https://api.weixin.qq.com/sns/userinfo"
	// wechatConnectMailDomain is the synthetic domain used for provisioned
	// accounts; WeChat exposes no email claim so a real address cannot exist.
	wechatConnectMailDomain = "connect.wechat.local"
)

// wechatTokenResponse is the JSON payload of /sns/oauth2/access_token.
// unionid is the stable identity across the operator's WeChat apps and is
// preferred over openid when present.
type wechatTokenResponse struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// wechatUserInfoResponse is the profile document from /sns/userinfo.
type wechatUserInfoResponse struct {
	Nickname string `json:"nickname"`
	ErrCode  int    `json:"errcode"`
}

func validateWeChatConfig(wx *setting.WeChatConnect) error {
	if !wx.Enabled || wx.AppID == "" || wx.AppSecret == "" {
		return errors.New("wechat connect not enabled or not configured")
	}
	return nil
}

func wechatCallbackURL(settings setting.Provider, c *gin.Context) string {
	return settings.SiteURL(c).ResolveReference(&url.URL{Path: "api/v4/session/wechat/callback"}).String()
}

// Login redirects the browser to the WeChat scan QR page.
func (service *WeChatLoginService) Login(c *gin.Context) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	wx := settings.WeChatConnect(c)

	if err := validateWeChatConfig(wx); err != nil {
		redirectToSigninWithError(c, settings, "sso_not_configured")
		return
	}

	state := ssoState{
		Redirect: sanitizeSSORedirect(service.Redirect),
	}
	if service.Link {
		u := inventory.UserFromContext(c)
		if inventory.IsAnonymousUser(u) {
			redirectToSigninWithError(c, settings, "sso_state_failed")
			return
		}
		state.LinkUserID = u.ID
	}

	stateKey := ssoStateKey()
	if err := dep.KV().Set(stateKey, state, ssoStateTTL); err != nil {
		dep.Logger().Warning("Failed to persist WeChat state: %s", err)
		redirectToSigninWithError(c, settings, "sso_state_failed")
		return
	}

	authorize, _ := url.Parse(wechatAuthorizeEndpoint)
	q := authorize.Query()
	q.Set("appid", wx.AppID)
	q.Set("redirect_uri", wechatCallbackURL(settings, c))
	q.Set("response_type", "code")
	q.Set("scope", "snsapi_login")
	q.Set("state", stateKey)
	authorize.RawQuery = q.Encode()
	// The WeChat qrconnect page requires the trailing #wechat_redirect
	// fragment; without it the QR code is not rendered.
	authorize.Fragment = "wechat_redirect"

	c.Redirect(http.StatusFound, authorize.String())
}

// Callback exchanges the code, resolves the WeChat unionid/openid, then
// either binds it to the linking user or signs the bound account in.
func (service *WeChatCallbackService) Callback(c *gin.Context) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	fail := func(code string) {
		redirectToSigninWithError(c, settings, code)
	}

	wx := settings.WeChatConnect(c)
	if err := validateWeChatConfig(wx); err != nil {
		fail("sso_not_configured")
		return
	}
	if service.State == "" || service.Code == "" {
		fail("sso_invalid_response")
		return
	}

	rawState, ok := dep.KV().Get(service.State)
	if !ok {
		fail("sso_state_expired")
		return
	}
	_ = dep.KV().Delete("", service.State)
	state, ok := rawState.(ssoState)
	if !ok {
		fail("sso_state_expired")
		return
	}

	// The endpoints are constants, but DNS resolution is not: reject a
	// weixin host that resolves to a private address before issuing
	// credentialed requests.
	for _, endpoint := range []string{wechatTokenEndpoint, wechatUserInfoEndpoint} {
		if err := request.ValidateExternalURL(c, endpoint, request.SSRFOptions{}); err != nil {
			dep.Logger().Warning("WeChat endpoint rejected by SSRF check: %s", err)
			fail("sso_exchange_failed")
			return
		}
	}

	httpClient := dep.RequestClient()
	token, err := wechatExchangeCode(c, httpClient, service.Code, wx)
	if err != nil {
		dep.Logger().Warning("WeChat token exchange failed: %s", err)
		fail("sso_exchange_failed")
		return
	}

	// Prefer unionid: it identifies the same person across every app in the
	// operator's WeChat open-platform account.
	identity := token.UnionID
	if identity == "" {
		identity = token.OpenID
	}
	if identity == "" {
		fail("sso_token_invalid")
		return
	}

	bindings := dep.SsoBindingClient()

	// Link mode: attach the identity to the signed-in user and return to the
	// security settings tab.
	if state.LinkUserID != 0 {
		if _, err := bindings.Bind(c, state.LinkUserID, inventory.SsoProviderWeChat, identity); err != nil {
			dep.Logger().Info("WeChat link rejected: %s", err)
			if errors.Is(err, inventory.ErrSsoBindingConflict) {
				fail("sso_account_unavailable")
				return
			}
			fail("sso_state_failed")
			return
		}
		recordUserEvent(c, dep, state.LinkUserID, types.EventLinkAccount, map[string]any{"provider": inventory.SsoProviderWeChat})
		dest := settings.SiteURL(c).ResolveReference(&url.URL{Path: "settings", RawQuery: "tab=security"})
		c.Redirect(http.StatusFound, dest.String())
		return
	}

	var targetUser *ent.User
	binding, err := bindings.Get(c, inventory.SsoProviderWeChat, identity)
	switch {
	case err == nil:
		ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
		targetUser, err = dep.UserClient().GetByID(ctx, binding.UserID)
		if err != nil {
			dep.Logger().Warning("WeChat binding resolved to missing user %d: %s", binding.UserID, err)
			fail("sso_account_unavailable")
			return
		}
	case ent.IsNotFound(err):
		if !wx.RegisterEnabled {
			dep.Logger().Info("WeChat login rejected: provisioning disabled")
			fail("sso_account_unavailable")
			return
		}
		targetUser, err = wechatProvisionUser(c, dep, httpClient, token, wx)
		if err != nil {
			dep.Logger().Warning("WeChat provisioning failed: %s", err)
			fail("sso_account_unavailable")
			return
		}
		if _, err := bindings.Bind(c, targetUser.ID, inventory.SsoProviderWeChat, identity); err != nil {
			dep.Logger().Warning("WeChat binding failed: %s", err)
			fail("sso_state_failed")
			return
		}
	default:
		dep.Logger().Warning("WeChat binding lookup failed: %s", err)
		fail("sso_state_failed")
		return
	}

	if targetUser, err = dep.UserClient().LiftExpiredBan(c, targetUser); err != nil {
		fail("sso_account_unavailable")
		return
	}
	if err := checkUserStatus(c, targetUser); err != nil {
		dep.Logger().Info("WeChat login rejected: %s", err)
		fail("sso_account_unavailable")
		return
	}

	ticket := ssoTicketKey()
	if err := dep.KV().Set(ticket, targetUser.ID, ssoTicketTTL); err != nil {
		fail("sso_state_failed")
		return
	}

	callback := settings.SiteURL(c).ResolveReference(&url.URL{Path: "callback/sso"})
	q := callback.Query()
	q.Set("ticket", ticket)
	if state.Redirect != "" {
		q.Set("redirect", state.Redirect)
	}
	callback.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, callback.String())
}

// wechatExchangeCode trades the authorization code for an access token.
// Unlike QQ, WeChat answers with JSON; the token response already carries
// openid and unionid.
func wechatExchangeCode(c *gin.Context, client request.Client, code string, wx *setting.WeChatConnect) (*wechatTokenResponse, error) {
	endpoint, _ := url.Parse(wechatTokenEndpoint)
	q := endpoint.Query()
	q.Set("appid", wx.AppID)
	q.Set("secret", wx.AppSecret)
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")
	endpoint.RawQuery = q.Encode()

	resp, err := client.
		Request(http.MethodGet, endpoint.String(), nil,
			request.WithContext(c),
			request.WithTimeout(15*time.Second),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}

	return parseWeChatToken(resp)
}

// parseWeChatToken unwraps the JSON token reply. WeChat reports failures as
// HTTP 200 with a nonzero errcode, so both paths must be checked.
func parseWeChatToken(body string) (*wechatTokenResponse, error) {
	var token wechatTokenResponse
	if err := json.Unmarshal([]byte(body), &token); err != nil {
		return nil, fmt.Errorf("malformed token response: %w", err)
	}
	if token.ErrCode != 0 || token.AccessToken == "" {
		return nil, fmt.Errorf("token error %d: %s", token.ErrCode, token.ErrMsg)
	}
	return &token, nil
}

// wechatProvisionUser creates the local account for a new WeChat identity.
// WeChat has no email claim, so a synthetic address under
// connect.wechat.local is used; the nickname comes from userinfo and falls
// back to an openid suffix.
func wechatProvisionUser(c *gin.Context, dep dependency.Dep, client request.Client, token *wechatTokenResponse, wx *setting.WeChatConnect) (*ent.User, error) {
	nick := ""
	endpoint, _ := url.Parse(wechatUserInfoEndpoint)
	q := endpoint.Query()
	q.Set("access_token", token.AccessToken)
	q.Set("openid", token.OpenID)
	endpoint.RawQuery = q.Encode()

	if resp, err := client.
		Request(http.MethodGet, endpoint.String(), nil,
			request.WithContext(c),
			request.WithTimeout(10*time.Second),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse(); err == nil {
		var info wechatUserInfoResponse
		if err := json.Unmarshal([]byte(resp), &info); err == nil && info.ErrCode == 0 {
			nick = strings.TrimSpace(info.Nickname)
		}
	} else {
		dep.Logger().Warning("WeChat userinfo request failed: %s", err)
	}

	if nick == "" {
		nick = "WeChat user " + token.OpenID[len(token.OpenID)-min(6, len(token.OpenID)):]
	}
	if len(nick) > 100 {
		nick = nick[:100]
	}

	return dep.UserClient().Create(c, &inventory.NewUserArgs{
		Email:   fmt.Sprintf("wx_%s@%s", token.OpenID, wechatConnectMailDomain),
		Nick:    nick,
		Status:  user.StatusActive,
		GroupID: dep.SettingProvider().DefaultGroup(c),
	})
}
