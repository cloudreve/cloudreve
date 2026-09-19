package workflows

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiteralDirPrefix(t *testing.T) {
	cases := []struct {
		rule string
		want string
	}{
		{"uploads/{uid}/{path}", "uploads"},
		{"uploads/{uid}", "uploads"},
		{"{uid}/{path}", ""},
		{"blob/store/{uid}", "blob/store"},
		{"", ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, literalDirPrefix(c.rule), "rule %q", c.rule)
	}
}
