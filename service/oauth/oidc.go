package oauth

import (
	"net/url"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
)

type DiscoveryService struct{}

type JWKService struct{}

func (s *DiscoveryService) Get(c *gin.Context) *DiscoveryResponse {
	issuer := oidcIssuer(c)
	return &DiscoveryResponse{
		Issuer:                issuer.String(),
		AuthorizationEndpoint: oidcEndpoint(issuer, "/session/authorize"),
		TokenEndpoint:         oidcEndpoint(issuer, constants.APIPrefix+"/session/oauth/token"),
		UserInfoEndpoint:      oidcEndpoint(issuer, constants.APIPrefix+"/session/oauth/userinfo"),
		JWKSURI:               oidcEndpoint(issuer, constants.APIPrefix+"/session/oauth/jwks"),
		ResponseTypesSupported: []string{
			"code",
		},
		GrantTypesSupported: []string{
			"authorization_code",
		},
		SubjectTypesSupported: []string{
			"public",
		},
		IDTokenSigningAlgValuesSupported: []string{
			"RS256",
		},
		TokenEndpointAuthMethods: []string{
			"client_secret_post",
		},
		CodeChallengeMethodsSupported: []string{
			"S256",
		},
		ScopesSupported: []string{
			types.ScopeOpenID,
			types.ScopeProfile,
			types.ScopeEmail,
		},
		ClaimsSupported: []string{
			"sub",
			"name",
			"preferred_username",
			"picture",
			"updated_at",
			"email",
			"email_verified",
		},
	}
}

func (s *JWKService) Get(c *gin.Context) (*auth.JWKSet, error) {
	dep := dependency.FromContext(c)
	return auth.OIDCJWKSet(c, dep.SettingClient())
}

func oidcIssuer(c *gin.Context) *url.URL {
	dep := dependency.FromContext(c)
	issuer := *dep.SettingProvider().SiteURL(setting.UseFirstSiteUrl(c))
	issuer.RawQuery = ""
	issuer.Fragment = ""
	return &issuer
}

func oidcEndpoint(issuer *url.URL, endpoint string) string {
	route, _ := url.Parse(endpoint)
	return issuer.ResolveReference(route).String()
}
