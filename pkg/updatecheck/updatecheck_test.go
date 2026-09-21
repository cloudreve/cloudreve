package updatecheck

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"4.19.1", "4.19.2", true},
		{"v4.19.1", "v4.19.2", true},
		{"4.19.2", "4.19.2", false},
		{"4.19.2", "4.19.1", false},
		{"4.19.2", "4.20.0", true},
		{"4.19.9", "4.20.0", true},
		{"4.19.9", "5.0.0", true},
		{"4.19.2-rc1", "4.19.2", true},
		{"4.19.2", "4.19.2-rc1", false},
		{"5.0.0", "4.99.9", false},
		{"4.19", "4.19.1", true},
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}
