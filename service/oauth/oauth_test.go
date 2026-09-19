package oauth

import (
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/stretchr/testify/require"
)

func TestValidateClientAuth(t *testing.T) {
	confidential := &ent.OAuthClient{Secret: "s3cret"}
	public := &ent.OAuthClient{Secret: ""}

	// Confidential clients: secret required and must match.
	require.Error(t, validateClientAuth(confidential, "", "challenge"))
	require.Error(t, validateClientAuth(confidential, "wrong", "challenge"))
	require.NoError(t, validateClientAuth(confidential, "s3cret", ""))

	// Public clients: secret ignored, PKCE challenge mandatory.
	require.Error(t, validateClientAuth(public, "", ""))
	require.Error(t, validateClientAuth(public, "anything", ""))
	require.NoError(t, validateClientAuth(public, "", "challenge"))
	require.NoError(t, validateClientAuth(public, "stale-known-secret", "challenge"))
}
