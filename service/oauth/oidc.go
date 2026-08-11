package oauth

import (
	"net/url"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/gin-gonic/gin"
)

type DiscoveryService struct{}

type JWKService struct{}

func (s *DiscoveryService) Get(c *gin.Context) *DiscoveryResponse {
	issuer := oidcIssuer(c)
	return &DiscoveryResponse{
		Issuer:                issuer.String(),
		AuthorizationEndpoint: routes.MasterOIDCEndpointUrl(issuer, "/session/authorize"),
		TokenEndpoint:         routes.MasterOIDCEndpointUrl(issuer, constants.APIPrefix+"/session/oauth/token"),
		UserInfoEndpoint:      routes.MasterOIDCEndpointUrl(issuer, constants.APIPrefix+"/session/oauth/userinfo"),
		JWKSURI:               routes.MasterOIDCEndpointUrl(issuer, constants.APIPrefix+"/session/oauth/jwks"),
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
	return auth.OIDCJWKSet(dep.SettingProvider().OIDCSigningPrivateKey(c))
}

func oidcIssuer(c *gin.Context) *url.URL {
	dep := dependency.FromContext(c)
	issuer := *dep.SettingProvider().SiteURL(c)
	issuer.RawQuery = ""
	issuer.Fragment = ""
	return &issuer
}
