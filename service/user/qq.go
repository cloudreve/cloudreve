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
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
)

type (
	// QQLoginParameterCtx marks the QQ Connect authorization start route.
	QQLoginParameterCtx struct{}
	// QQCallbackParameterCtx marks the QQ Connect redirect target.
	QQCallbackParameterCtx struct{}
	// SsoUnbindParameterCtx marks the external-binding removal route.
	SsoUnbindParameterCtx struct{}

	// QQLoginService starts the QQ Connect OAuth2 flow. `link=1` binds the
	// QQ identity to the currently signed-in user instead of signing in.
	QQLoginService struct {
		Redirect string `form:"redirect"`
		Link     bool   `form:"link"`
	}

	// QQCallbackService completes the QQ Connect flow.
	QQCallbackService struct {
		Code  string `form:"code"`
		State string `form:"state"`
	}

	// SsoUnbindService removes the caller's binding at a provider.
	SsoUnbindService struct {
		Provider string `uri:"provider" binding:"required"`
	}
)

const (
	qqAuthorizeEndpoint = "https://graph.qq.com/oauth2.0/authorize"
	qqTokenEndpoint     = "https://graph.qq.com/oauth2.0/token"
	qqOpenIDEndpoint    = "https://graph.qq.com/oauth2.0/me"
	qqUserInfoEndpoint  = "https://graph.qq.com/user/get_user_info"
	// qqConnectMailDomain is the synthetic domain used for provisioned
	// accounts; QQ exposes no email claim so a real address cannot exist.
	qqConnectMailDomain = "connect.qq.local"
)

// qqOpenIDResponse is the JSONP payload of /oauth2.0/me.
type qqOpenIDResponse struct {
	ClientID string `json:"client_id"`
	OpenID   string `json:"openid"`
}

// qqUserInfoResponse is the profile document from get_user_info.
type qqUserInfoResponse struct {
	Ret      int    `json:"ret"`
	Nickname string `json:"nickname"`
}

func validateQQConfig(qq *setting.QQConnect) error {
	if !qq.Enabled || qq.AppID == "" || qq.AppSecret == "" {
		return errors.New("qq connect not enabled or not configured")
	}
	return nil
}

func qqCallbackURL(settings setting.Provider, c *gin.Context) string {
	return settings.SiteURL(c).ResolveReference(&url.URL{Path: "api/v4/session/qq/callback"}).String()
}

// QQLogin redirects the browser to the QQ authorization page.
func (service *QQLoginService) Login(c *gin.Context) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	qq := settings.QQConnect(c)

	if err := validateQQConfig(qq); err != nil {
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
		dep.Logger().Warning("Failed to persist QQ state: %s", err)
		redirectToSigninWithError(c, settings, "sso_state_failed")
		return
	}

	authorize, _ := url.Parse(qqAuthorizeEndpoint)
	q := authorize.Query()
	q.Set("response_type", "code")
	q.Set("client_id", qq.AppID)
	q.Set("redirect_uri", qqCallbackURL(settings, c))
	q.Set("state", stateKey)
	q.Set("scope", "get_user_info")
	authorize.RawQuery = q.Encode()

	c.Redirect(http.StatusFound, authorize.String())
}

// QQCallback exchanges the code, resolves the QQ openid, then either binds
// it to the linking user or signs the bound account in.
func (service *QQCallbackService) Callback(c *gin.Context) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	fail := func(code string) {
		redirectToSigninWithError(c, settings, code)
	}

	qq := settings.QQConnect(c)
	if err := validateQQConfig(qq); err != nil {
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
	// graph.qq.com that resolves to a private address before issuing
	// credentialed requests.
	for _, endpoint := range []string{qqTokenEndpoint, qqOpenIDEndpoint, qqUserInfoEndpoint} {
		if err := request.ValidateExternalURL(c, endpoint, request.SSRFOptions{}); err != nil {
			dep.Logger().Warning("QQ endpoint rejected by SSRF check: %s", err)
			fail("sso_exchange_failed")
			return
		}
	}

	httpClient := dep.RequestClient()
	token, err := qqExchangeCode(c, httpClient, service.Code, qqCallbackURL(settings, c), qq)
	if err != nil {
		dep.Logger().Warning("QQ token exchange failed: %s", err)
		fail("sso_exchange_failed")
		return
	}

	openID, err := qqFetchOpenID(c, httpClient, token, qq.AppID)
	if err != nil {
		dep.Logger().Warning("QQ openid request failed: %s", err)
		fail("sso_token_invalid")
		return
	}

	bindings := dep.SsoBindingClient()

	// Link mode: attach the openid to the signed-in user and return to the
	// security settings tab.
	if state.LinkUserID != 0 {
		if _, err := bindings.Bind(c, state.LinkUserID, inventory.SsoProviderQQ, openID); err != nil {
			dep.Logger().Info("QQ link rejected: %s", err)
			if errors.Is(err, inventory.ErrSsoBindingConflict) {
				fail("sso_account_unavailable")
				return
			}
			fail("sso_state_failed")
			return
		}
		recordUserEvent(c, dep, state.LinkUserID, types.EventLinkAccount, map[string]any{"provider": inventory.SsoProviderQQ})
		dest := settings.SiteURL(c).ResolveReference(&url.URL{Path: "settings", RawQuery: "tab=security"})
		c.Redirect(http.StatusFound, dest.String())
		return
	}

	var targetUser *ent.User
	binding, err := bindings.Get(c, inventory.SsoProviderQQ, openID)
	switch {
	case err == nil:
		ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
		targetUser, err = dep.UserClient().GetByID(ctx, binding.UserID)
		if err != nil {
			dep.Logger().Warning("QQ binding resolved to missing user %d: %s", binding.UserID, err)
			fail("sso_account_unavailable")
			return
		}
	case ent.IsNotFound(err):
		if !qq.RegisterEnabled {
			dep.Logger().Info("QQ login rejected: provisioning disabled")
			fail("sso_account_unavailable")
			return
		}
		targetUser, err = qqProvisionUser(c, dep, httpClient, token, openID, qq)
		if err != nil {
			dep.Logger().Warning("QQ provisioning failed: %s", err)
			fail("sso_account_unavailable")
			return
		}
		if _, err := bindings.Bind(c, targetUser.ID, inventory.SsoProviderQQ, openID); err != nil {
			dep.Logger().Warning("QQ binding failed: %s", err)
			fail("sso_state_failed")
			return
		}
	default:
		dep.Logger().Warning("QQ binding lookup failed: %s", err)
		fail("sso_state_failed")
		return
	}

	if targetUser, err = dep.UserClient().LiftExpiredBan(c, targetUser); err != nil {
		fail("sso_account_unavailable")
		return
	}
	if err := checkUserStatus(c, targetUser); err != nil {
		dep.Logger().Info("QQ login rejected: %s", err)
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

// Delete removes the caller's binding at the given provider. Removing the
// last viable sign-in method (no password, no passkey, no other binding)
// is refused so an SSO-provisioned account cannot lock itself out.
func (service *SsoUnbindService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if service.Provider != inventory.SsoProviderQQ && service.Provider != inventory.SsoProviderWeChat {
		return serializer.NewError(serializer.CodeParamErr, "Unknown provider", nil)
	}

	bindings := dep.SsoBindingClient()
	if u.Password == "" {
		passkeys, err := dep.UserClient().ListPasskeys(c, u.ID)
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to get user passkey", err)
		}
		others, err := bindings.ListByUser(c, u.ID)
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to get user linked accounts", err)
		}
		otherProviders := 0
		for _, b := range others {
			if b.Provider != service.Provider {
				otherProviders++
			}
		}
		if len(passkeys) == 0 && otherProviders == 0 {
			return serializer.NewError(serializer.CodeParamErr, "Cannot unlink the only sign-in method", nil)
		}
	}

	if err := bindings.Unbind(c, u.ID, service.Provider); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to unlink account", err)
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUnlinkAccount,
		activity.Extra(map[string]any{"provider": service.Provider}))
	return nil
}

// qqExchangeCode trades the authorization code for an access token. QQ
// answers with a form-encoded body (access_token=...&expires_in=...), not
// JSON.
func qqExchangeCode(c *gin.Context, client request.Client, code, redirectURI string, qq *setting.QQConnect) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {qq.AppID},
		"client_secret": {qq.AppSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	resp, err := client.
		Request(http.MethodPost, qqTokenEndpoint, strings.NewReader(form.Encode()),
			request.WithContext(c),
			request.WithTimeout(15*time.Second),
			request.WithHeader(http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}

	return parseQQTokenResponse(resp)
}

// parseQQTokenResponse extracts the access token from QQ's form-encoded
// token reply. Error responses arrive as JSONP callback({"error":...})
// and fail the lookup naturally.
func parseQQTokenResponse(body string) (string, error) {
	if values, err := url.ParseQuery(strings.TrimSpace(body)); err == nil {
		if token := values.Get("access_token"); token != "" {
			return token, nil
		}
	}
	return "", errors.New("unexpected token response")
}

// qqFetchOpenID unwraps the JSONP response of /oauth2.0/me and verifies the
// echoed client_id to guard against token substitution.
func qqFetchOpenID(c *gin.Context, client request.Client, accessToken, appID string) (string, error) {
	endpoint, _ := url.Parse(qqOpenIDEndpoint)
	q := endpoint.Query()
	q.Set("access_token", accessToken)
	endpoint.RawQuery = q.Encode()

	resp, err := client.
		Request(http.MethodGet, endpoint.String(), nil,
			request.WithContext(c),
			request.WithTimeout(10*time.Second),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return "", fmt.Errorf("openid request failed: %w", err)
	}

	return parseQQOpenIDResponse(resp, appID)
}

// parseQQOpenIDResponse unwraps the JSONP reply of /oauth2.0/me and
// verifies the echoed client_id to guard against token substitution.
func parseQQOpenIDResponse(body, appID string) (string, error) {
	body = strings.TrimSpace(body)
	start := strings.Index(body, "(")
	end := strings.LastIndex(body, ")")
	if start < 0 || end <= start {
		return "", errors.New("malformed openid response")
	}

	var payload qqOpenIDResponse
	if err := json.Unmarshal([]byte(body[start+1:end]), &payload); err != nil {
		return "", fmt.Errorf("malformed openid payload: %w", err)
	}
	if payload.ClientID != appID || payload.OpenID == "" {
		return "", errors.New("openid client mismatch")
	}
	return payload.OpenID, nil
}

// qqProvisionUser creates the local account for a new QQ identity. QQ has
// no email claim, so a synthetic address under connect.qq.local is used;
// the nickname falls back to an openid suffix.
func qqProvisionUser(c *gin.Context, dep dependency.Dep, client request.Client, accessToken, openID string, qq *setting.QQConnect) (*ent.User, error) {
	nick := ""
	endpoint, _ := url.Parse(qqUserInfoEndpoint)
	q := endpoint.Query()
	q.Set("access_token", accessToken)
	q.Set("oauth_consumer_key", qq.AppID)
	q.Set("openid", openID)
	endpoint.RawQuery = q.Encode()

	if resp, err := client.
		Request(http.MethodGet, endpoint.String(), nil,
			request.WithContext(c),
			request.WithTimeout(10*time.Second),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse(); err == nil {
		var info qqUserInfoResponse
		if err := json.Unmarshal([]byte(resp), &info); err == nil && info.Ret == 0 {
			nick = strings.TrimSpace(info.Nickname)
		}
	} else {
		dep.Logger().Warning("QQ userinfo request failed: %s", err)
	}

	if nick == "" {
		nick = "QQ user " + openID[len(openID)-min(6, len(openID)):]
	}
	if len(nick) > 100 {
		nick = nick[:100]
	}

	return dep.UserClient().Create(c, &inventory.NewUserArgs{
		Email:   fmt.Sprintf("qq_%s@%s", openID, qqConnectMailDomain),
		Nick:    nick,
		Status:  user.StatusActive,
		GroupID: dep.SettingProvider().DefaultGroup(c),
	})
}

// recordUserEvent logs an account event attributed to a user ID rather
// than the request actor (link-mode callbacks run for the linking user).
func recordUserEvent(c *gin.Context, dep dependency.Dep, uid int, event int, extra map[string]any) {
	opts := []activity.Opt{activity.Actor(uid)}
	if extra != nil {
		opts = append(opts, activity.Extra(extra))
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), event, opts...)
}
