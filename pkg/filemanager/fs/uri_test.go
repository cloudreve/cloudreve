package fs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShareUriPasswordRoundTrip(t *testing.T) {
	for _, password := range []string{"plain", "with space", "a:b@c/d?e", "😀😴😭", "$%^&*"} {
		u, err := NewUriFromString(NewShareUri("abc123", password))
		require.NoError(t, err, "password %q", password)
		require.Equal(t, "abc123", u.ID(""))
		require.Equal(t, password, u.Password(), "password %q", password)
	}

	u, err := NewUriFromString(NewShareUri("abc123", ""))
	require.NoError(t, err)
	require.Equal(t, "abc123", u.ID(""))
	require.Equal(t, "", u.Password())
}
