package user

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

// Parameter contexts for SSO services.
type (
	SSOLoginParameterCtx    struct{}
	SSOCallbackParameterCtx struct{}
	SSOExchangeParameterCtx struct{}
)

const (
	ssoStateTTL     = 600 // seconds the authorization state stays valid
	ssoTicketTTL    = 60  // seconds the one-time login ticket stays valid
	ssoDiscoveryTTL = 3600
	ssoJWKSTTL      = 900
)

// ssoState is the one-time server-side state stored for an in-flight SSO flow.
// Storing it in KV (single-use) prevents replay and binds the callback to the
// exact login attempt that generated it.
type ssoState struct {
	Nonce    string
	Redirect string
	// LinkUserID is non-zero for account-link flows (QQ Connect): the
	// resolved external identity binds to this user instead of signing in.
	LinkUserID int
}

// SSOLoginService starts an inbound OIDC flow by redirecting the browser to
// the configured identity provider.
type SSOLoginService struct {
	Redirect string `form:"redirect" json:"redirect"`
}

// SSOLogin builds the authorization URL and redirects to the IdP.
func (service *SSOLoginService) SSOLogin(c *gin.Context) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	sso := settings.SSO(c)

	redirect := sanitizeSSORedirect(service.Redirect)
	if err := validateSSOConfig(sso); err != nil {
		redirectToSigninWithError(c, settings, "sso_not_configured")
		return
	}

	discovery, err := ssoDiscovery(c, dep, sso)
	if err != nil {
		dep.Logger().Warning("SSO discovery failed: %s", err)
		redirectToSigninWithError(c, settings, "sso_discovery_failed")
		return
	}

	state := ssoState{
		Nonce:    util.RandStringRunesCrypto(32),
		Redirect: redirect,
	}
	stateKey := ssoStateKey()
	if err := dep.KV().Set(stateKey, state, ssoStateTTL); err != nil {
		dep.Logger().Warning("Failed to persist SSO state: %s", err)
		redirectToSigninWithError(c, settings, "sso_state_failed")
		return
	}

	authorize, err := url.Parse(discovery.AuthorizationEndpoint)
	if err != nil {
		redirectToSigninWithError(c, settings, "sso_discovery_failed")
		return
	}
	q := authorize.Query()
	q.Set("response_type", "code")
	q.Set("client_id", sso.ClientID)
	q.Set("redirect_uri", ssoCallbackURL(settings, c))
	q.Set("scope", sso.Scopes)
	q.Set("state", stateKey)
	q.Set("nonce", state.Nonce)
	authorize.RawQuery = q.Encode()

	c.Redirect(http.StatusFound, authorize.String())
}

// SSOCallbackService completes the inbound OIDC flow.
type SSOCallbackService struct {
	Code             string `form:"code" json:"code"`
	State            string `form:"state" json:"state"`
	Error            string `form:"error" json:"error"`
	ErrorDescription string `form:"error_description" json:"error_description"`
}

// SSOCallback validates the response, resolves the user and redirects to the
// SPA with a one-time ticket. Tokens never appear in the URL.
func (service *SSOCallbackService) SSOCallback(c *gin.Context) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	fail := func(code string) {
		redirectToSigninWithError(c, settings, code)
	}

	if service.Error != "" {
		dep.Logger().Info("SSO callback error from IdP: %s (%s)", service.Error, service.ErrorDescription)
		fail("sso_denied")
		return
	}

	sso := settings.SSO(c)
	if err := validateSSOConfig(sso); err != nil {
		fail("sso_not_configured")
		return
	}
	if service.State == "" || service.Code == "" {
		fail("sso_invalid_response")
		return
	}

	// Single-use state: Get then Delete so a captured state cannot be replayed.
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

	discovery, err := ssoDiscovery(c, dep, sso)
	if err != nil {
		dep.Logger().Warning("SSO discovery failed: %s", err)
		fail("sso_discovery_failed")
		return
	}

	validate := ssoURLValidator(c, sso)
	httpClient := dep.RequestClient()

	tokens, err := exchangeOIDCCode(c, httpClient, discovery.TokenEndpoint, service.Code, ssoCallbackURL(settings, c), sso.ClientID, sso.ClientSecret, validate)
	if err != nil {
		dep.Logger().Warning("SSO token exchange failed: %s", err)
		fail("sso_exchange_failed")
		return
	}

	claims, err := ssoVerifyIDToken(c, dep, sso, discovery, tokens.IDToken, state.Nonce)
	if err != nil {
		dep.Logger().Warning("SSO id_token validation failed: %s", err)
		fail("sso_token_invalid")
		return
	}

	// Profile claims resolve across standard and AD FS-style alternates;
	// AD FS userinfo returns only sub, so the ID token is the primary
	// source for name/email/upn (#3572).
	email := firstEmailClaim(claims.Email, claims.UPN, claims.UniqueName)
	name := claims.Name
	if name == "" {
		name = strings.TrimSpace(claims.GivenName + " " + claims.FamilyName)
	}
	preferred := claims.PreferredUsername
	if preferred == "" {
		preferred = claims.UniqueName
	}

	// Some providers omit email from the ID token; fall back to userinfo.
	if email == "" && tokens.AccessToken != "" && discovery.UserinfoEndpoint != "" {
		info, err := fetchOIDCUserInfo(c, httpClient, discovery.UserinfoEndpoint, tokens.AccessToken, validate)
		if err != nil {
			dep.Logger().Warning("SSO userinfo request failed: %s", err)
		} else {
			email = firstEmailClaim(info.Email, info.UPN, info.UniqueName)
			if name == "" {
				name = info.Name
				if name == "" {
					name = strings.TrimSpace(info.GivenName + " " + info.FamilyName)
				}
			}
			if preferred == "" {
				preferred = info.PreferredUsername
				if preferred == "" {
					preferred = info.UniqueName
				}
			}
		}
	}
	if email == "" {
		fail("sso_no_email")
		return
	}

	targetUser, err := ssoResolveUser(c, dep, sso, email, name, preferred)
	if err != nil {
		dep.Logger().Info("SSO login rejected for %q: %s", email, err)
		fail("sso_account_unavailable")
		return
	}

	// One-time ticket -> frontend exchanges it for a token pair via JSON API.
	ticket := ssoTicketKey()
	if err := dep.KV().Set(ticket, targetUser.ID, ssoTicketTTL); err != nil {
		dep.Logger().Warning("Failed to persist SSO ticket: %s", err)
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

// SSOExchangeService trades a one-time callback ticket for a session token.
type SSOExchangeService struct {
	Ticket string `form:"ticket" json:"ticket" binding:"required"`
}

// SSOExchange validates the ticket and issues a builtin token pair.
func (service *SSOExchangeService) SSOExchange(c *gin.Context) (any, error) {
	dep := dependency.FromContext(c)

	raw, ok := dep.KV().Get(service.Ticket)
	if !ok {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid or expired SSO ticket", nil)
	}
	_ = dep.KV().Delete("", service.Ticket)

	uid, ok := raw.(int)
	if !ok {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid SSO ticket", nil)
	}

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	u, err := dep.UserClient().GetByID(ctx, uid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
	}
	if u, err = dep.UserClient().LiftExpiredBan(c, u); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to lift expired ban", err)
	}
	if err := checkUserStatus(c, u); err != nil {
		return nil, err
	}

	util.WithValue(c, inventory.UserCtx{}, u)
	return IssueToken(c)
}

// ssoResolveUser finds or provisions the local account for an SSO identity.
func ssoResolveUser(c *gin.Context, dep dependency.Dep, sso *setting.SSO, email, name, preferred string) (*ent.User, error) {
	userClient := dep.UserClient()

	u, err := userClient.GetByEmail(context.WithValue(c, inventory.LoadUserGroup{}, true), email)
	if err == nil {
		if u, err = userClient.LiftExpiredBan(c, u); err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to lift expired ban", err)
		}
		if err := checkUserStatus(c, u); err != nil {
			return nil, err
		}
		return u, nil
	}

	if !sso.RegisterEnabled {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "SSO account provisioning is disabled", nil)
	}
	if err := CheckEmailAllowed(dep.SettingProvider().EmailFilter(c), email); err != nil {
		return nil, err
	}

	nick := preferred
	if nick == "" {
		nick = name
	}
	if nick == "" {
		nick = strings.Split(email, "@")[0]
	}

	newUser, err := userClient.Create(c, &inventory.NewUserArgs{
		Email:   email,
		Nick:    nick,
		Status:  user.StatusActive,
		GroupID: dep.SettingProvider().DefaultGroup(c),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to provision SSO user: %w", err)
	}

	return newUser, nil
}

// banError renders the ban error. When a ban reason is configured it is
// sent as the message so the frontend can render it next to the
// localized "blocked" text (#2478).
func banError(u *ent.User, fallback string) error {
	msg := fallback
	if u.BanReason != "" {
		msg = u.BanReason
	}
	return serializer.NewError(serializer.CodeUserBaned, msg, nil)
}

func checkUserStatus(c *gin.Context, u *ent.User) error {
	switch u.Status {
	case user.StatusSysBanned, user.StatusManualBanned:
		return banError(u, "User is banned")
	case user.StatusInactive:
		return serializer.NewError(serializer.CodeUserNotActivated, "User is not activated", nil)
	}
	return checkLoginIPWhitelist(c.ClientIP(), inventory.EffectiveGroup(u))
}

// CheckEmailAllowed enforces the sign-up email filter. Shared by the classic
// register endpoint and SSO account provisioning.
func CheckEmailAllowed(filter *setting.EmailFilter, email string) error {
	local, domain, found := strings.Cut(email, "@")
	if !found {
		return serializer.NewError(serializer.CodeParamErr, "Invalid email", nil)
	}

	if filter.DisableSubAddress {
		chars := filter.SubAddressChars
		if chars == "" {
			chars = "+"
		}
		if strings.ContainsAny(local, chars) {
			return serializer.NewError(serializer.CodeParamErr, "Sub-address emails are not allowed", nil)
		}
	}

	domain = strings.ToLower(domain)
	listed := false
	for _, entry := range filter.List {
		if entry == domain || strings.HasSuffix(domain, "."+entry) {
			listed = true
			break
		}
	}

	switch filter.Mode {
	case setting.EmailFilterWhitelist:
		if !listed {
			return serializer.NewError(serializer.CodeParamErr, "Email domain is not allowed", nil)
		}
	case setting.EmailFilterBlacklist:
		if listed {
			return serializer.NewError(serializer.CodeEmailProviderBaned, "Email domain is not allowed", nil)
		}
	}

	return nil
}

func validateSSOConfig(sso *setting.SSO) error {
	if !sso.Enabled || sso.Issuer == "" || sso.ClientID == "" {
		return errors.New("sso not enabled or not configured")
	}
	return nil
}

// ssoURLValidator wraps SSRF validation: the operator-trusted issuer host is
// allowlisted, endpoints on other internal hosts are rejected.
func ssoURLValidator(c *gin.Context, sso *setting.SSO) func(raw string) error {
	allowed := []string{}
	if u, err := url.Parse(sso.Issuer); err == nil && u.Hostname() != "" {
		allowed = append(allowed, u.Hostname())
	}
	return func(raw string) error {
		return request.ValidateExternalURL(c, raw, request.SSRFOptions{AllowedHosts: allowed})
	}
}

func ssoCallbackURL(settings setting.Provider, c *gin.Context) string {
	return settings.SiteURL(c).ResolveReference(&url.URL{Path: "api/v4/session/sso/callback"}).String()
}

func redirectToSigninWithError(c *gin.Context, settings setting.Provider, code string) {
	dest := settings.SiteURL(c).ResolveReference(&url.URL{Path: "session"})
	q := dest.Query()
	q.Set("sso_error", code)
	dest.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, dest.String())
}

func ssoStateKey() string  { return "sso_state_" + util.RandStringRunesCrypto(32) }
func ssoTicketKey() string { return "sso_ticket_" + util.RandStringRunesCrypto(32) }

// sanitizeSSORedirect keeps only same-origin relative paths so a crafted
// login URL cannot bounce the browser to an external site after sign-in.
func sanitizeSSORedirect(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return ""
	}
	return raw
}

// ssoDiscovery returns the provider metadata, KV-cached by issuer.
func ssoDiscovery(c *gin.Context, dep dependency.Dep, sso *setting.SSO) (*auth.OIDCDiscovery, error) {
	key := "sso_discovery_" + ssoIssuerHash(sso.Issuer)
	if cached, ok := dep.KV().Get(key); ok {
		if doc, ok := cached.(auth.OIDCDiscovery); ok {
			return &doc, nil
		}
	}

	validate := ssoURLValidator(c, sso)
	doc, err := fetchOIDCDiscovery(c, dep.RequestClient(), sso.Issuer, validate)
	if err != nil {
		return nil, err
	}

	// Validate each endpoint as well; a compromised discovery doc must not
	// become an SSRF proxy.
	for _, endpoint := range []string{doc.TokenEndpoint, doc.JWKSURI, doc.UserinfoEndpoint} {
		if endpoint == "" {
			continue
		}
		if err := validate(endpoint); err != nil {
			return nil, fmt.Errorf("unsafe OIDC endpoint %q: %w", endpoint, err)
		}
	}

	_ = dep.KV().Set(key, *doc, ssoDiscoveryTTL)
	return doc, nil
}

// ssoVerifyIDToken verifies the ID token, refetching JWKS once when the
// signing key is unknown (key rotation).
func ssoVerifyIDToken(c *gin.Context, dep dependency.Dep, sso *setting.SSO, doc *auth.OIDCDiscovery, idToken, nonce string) (*auth.OIDCIDTokenClaims, error) {
	jwksKey := "sso_jwks_" + ssoIssuerHash(doc.JWKSURI)
	validate := ssoURLValidator(c, sso)

	fetch := func() (*auth.JWKSet, error) {
		jwks, err := fetchOIDCJWKS(c, dep.RequestClient(), doc.JWKSURI, validate)
		if err != nil {
			return nil, err
		}
		_ = dep.KV().Set(jwksKey, *jwks, ssoJWKSTTL)
		return jwks, nil
	}

	var jwks *auth.JWKSet
	if cached, ok := dep.KV().Get(jwksKey); ok {
		if j, ok := cached.(auth.JWKSet); ok {
			jwks = &j
		}
	}
	if jwks == nil {
		var err error
		jwks, err = fetch()
		if err != nil {
			return nil, err
		}
	}

	claims, err := auth.VerifyOIDCIDToken(idToken, doc.Issuer, sso.ClientID, nonce, jwks)
	if errors.Is(err, auth.ErrOIDCKeyNotFound) {
		if jwks, err = fetch(); err != nil {
			return nil, err
		}
		claims, err = auth.VerifyOIDCIDToken(idToken, doc.Issuer, sso.ClientID, nonce, jwks)
	}

	return claims, err
}

func ssoIssuerHash(issuer string) string {
	sum := sha256.Sum256([]byte(issuer))
	return hex.EncodeToString(sum[:8])
}

// fetchOIDCDiscovery retrieves the provider metadata document.
func fetchOIDCDiscovery(c *gin.Context, client request.Client, issuer string, validate func(raw string) error) (*auth.OIDCDiscovery, error) {
	wellKnown := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	if err := validate(wellKnown); err != nil {
		return nil, err
	}

	resp, err := client.
		Request(http.MethodGet, wellKnown, nil,
			request.WithContext(c),
			request.WithTimeout(10*time.Second),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", auth.ErrOIDCDiscoveryFailed, err)
	}

	var doc auth.OIDCDiscovery
	if err := json.Unmarshal([]byte(resp), &doc); err != nil {
		return nil, fmt.Errorf("malformed discovery document: %w", auth.ErrOIDCDiscoveryFailed)
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	return &doc, nil
}

// fetchOIDCJWKS retrieves the provider's signing keys.
func fetchOIDCJWKS(c *gin.Context, client request.Client, jwksURI string, validate func(raw string) error) (*auth.JWKSet, error) {
	if err := validate(jwksURI); err != nil {
		return nil, err
	}

	resp, err := client.
		Request(http.MethodGet, jwksURI, nil,
			request.WithContext(c),
			request.WithTimeout(10*time.Second),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	var jwks auth.JWKSet
	if err := json.Unmarshal([]byte(resp), &jwks); err != nil {
		return nil, fmt.Errorf("malformed JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, auth.ErrOIDCKeyNotFound
	}

	return &jwks, nil
}

// exchangeOIDCCode trades the authorization code for tokens at the token
// endpoint. Client secret is sent via HTTP basic auth (client_secret_basic).
func exchangeOIDCCode(c *gin.Context, client request.Client, tokenEndpoint, code, redirectURI, clientID, clientSecret string, validate func(raw string) error) (*auth.OIDCTokenResponse, error) {
	if err := validate(tokenEndpoint); err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}
	header := http.Header{
		"Content-Type":  {"application/x-www-form-urlencoded"},
		"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(url.QueryEscape(clientID)+":"+url.QueryEscape(clientSecret)))},
		"Accept":        {"application/json"},
	}

	resp, err := client.
		Request(http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()),
			request.WithContext(c),
			request.WithTimeout(15*time.Second),
			request.WithHeader(header),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	var tokens auth.OIDCTokenResponse
	if err := json.Unmarshal([]byte(resp), &tokens); err != nil {
		return nil, fmt.Errorf("malformed token response: %w", err)
	}
	if tokens.IDToken == "" {
		return nil, fmt.Errorf("missing id_token: %w", auth.ErrOIDCInvalidToken)
	}

	return &tokens, nil
}

// fetchOIDCUserInfo retrieves claims from the userinfo endpoint, used when the
// ID token omits the email claim.
func fetchOIDCUserInfo(c *gin.Context, client request.Client, endpoint, accessToken string, validate func(raw string) error) (*auth.OIDCUserInfo, error) {
	if err := validate(endpoint); err != nil {
		return nil, err
	}

	header := http.Header{
		"Authorization": {"Bearer " + accessToken},
		"Accept":        {"application/json"},
	}
	resp, err := client.
		Request(http.MethodGet, endpoint, nil,
			request.WithContext(c),
			request.WithTimeout(10*time.Second),
			request.WithHeader(header),
		).
		CheckHTTPResponse(http.StatusOK).
		GetResponse()
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}

	var info auth.OIDCUserInfo
	if err := json.Unmarshal([]byte(resp), &info); err != nil {
		return nil, fmt.Errorf("malformed userinfo response: %w", err)
	}

	return &info, nil
}

// firstEmailClaim returns the first candidate usable as an email address.
// AD FS upn/unique_name are accepted only when email-shaped, since
// DOMAIN\user and non-routable identifiers cannot serve as addresses.
func firstEmailClaim(candidates ...string) string {
	for _, c := range candidates {
		c = strings.ToLower(strings.TrimSpace(c))
		if strings.Contains(c, "@") {
			return c
		}
	}
	return ""
}
