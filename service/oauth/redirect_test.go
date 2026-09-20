package oauth

import "testing"

func TestRedirectURIMatches(t *testing.T) {
	for _, test := range []struct {
		registered string
		requested  string
		matches    bool
	}{
		{"http://127.0.0.1/callback", "http://127.0.0.1:49152/callback", true},
		{"http://[::1]/callback", "http://[::1]:49152/callback", true},
		{"http://127.0.0.1:8000/callback?a=1", "http://127.0.0.1:65535/callback?a=1", true},
		{"/callback/desktop", "/callback/desktop", true},
		{"cloudreve://mount", "cloudreve://mount", true},
		{"https://app.example/callback", "https://app.example/callback", true},
		{"http://localhost/callback", "http://localhost:49152/callback", false},
		{"https://127.0.0.1/callback", "https://127.0.0.1:49152/callback", false},
		{"http://192.168.0.1/callback", "http://192.168.0.1:49152/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.2:49152/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1.evil.example:49152/callback", false},
		{"http://127.0.0.1/callback", "http://user@127.0.0.1:49152/callback", false},
		{"http://user@127.0.0.1/callback", "http://user@127.0.0.1/callback", false},
		{"http://127.0.0.1/callback", "https://127.0.0.1:49152/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:0/callback", false},
		{"http://127.0.0.1:0/callback", "http://127.0.0.1:0/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:65536/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:abc/callback", false},
		{"http://127.0.0.1:abc/callback", "http://127.0.0.1:abc/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:/callback", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:49152/callback#", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:49152/callback#fragment", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:49152/other", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:49152/%63allback", false},
		{"http://127.0.0.1/callback?a=1", "http://127.0.0.1:49152/callback?a=2", false},
		{"http://127.0.0.1/callback", "http://127.0.0.1:49152/callback?", false},
	} {
		t.Run(test.registered+" -> "+test.requested, func(t *testing.T) {
			if got := redirectURIMatches(test.registered, test.requested); got != test.matches {
				t.Fatalf("matches = %v, want %v", got, test.matches)
			}
		})
	}
}
