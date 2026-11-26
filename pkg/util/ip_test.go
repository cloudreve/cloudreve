package util

import (
	"net"
	"testing"
)

func TestNewIPMatcher_CIDR(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{"IPv4 CIDR /24", "192.168.1.0/24", false},
		{"IPv4 CIDR /16", "10.0.0.0/16", false},
		{"IPv4 CIDR /32", "192.168.1.1/32", false},
		{"IPv4 CIDR /0", "0.0.0.0/0", false},
		{"IPv6 CIDR /64", "2001:db8::/64", false},
		{"IPv6 CIDR /128", "::1/128", false},
		{"Invalid CIDR - bad mask", "192.168.1.0/33", true},
		{"Invalid CIDR - bad IP", "999.999.999.999/24", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewIPMatcher(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewIPMatcher(%q) expected error, got nil", tt.filter)
				}
				return
			}
			if err != nil {
				t.Errorf("NewIPMatcher(%q) unexpected error: %v", tt.filter, err)
				return
			}
			if matcher.cidr == nil {
				t.Errorf("NewIPMatcher(%q) expected CIDR matcher, got nil", tt.filter)
			}
		})
	}
}

func TestNewIPMatcher_Range(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{"IPv4 range", "192.168.1.1-192.168.1.255", false},
		{"IPv4 range with spaces", "192.168.1.1 - 192.168.1.255", false},
		{"IPv6 range", "::1-::10", false},
		{"Cross IP version", "192.168.1.1-::1", true},
		{"Invalid start", "abc-192.168.1.255", true},
		{"Invalid end", "192.168.1.1-abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewIPMatcher(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewIPMatcher(%q) expected error, got nil", tt.filter)
				}
				return
			}
			if err != nil {
				t.Errorf("NewIPMatcher(%q) unexpected error: %v", tt.filter, err)
				return
			}
			if matcher.ipStart == nil || matcher.ipEnd == nil {
				t.Errorf("NewIPMatcher(%q) expected range matcher, got nil", tt.filter)
			}
		})
	}
}

func TestNewIPMatcher_Wildcard(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{"One wildcard", "192.168.1.*", false},
		{"Two wildcards", "192.168.*.*", false},
		{"Three wildcards", "192.*.*.*", false},
		{"All wildcards", "*.*.*.*", false},
		{"IPv6 not supported", "2001:db8::*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewIPMatcher(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewIPMatcher(%q) expected error, got nil", tt.filter)
				}
				return
			}
			if err != nil {
				t.Errorf("NewIPMatcher(%q) unexpected error: %v", tt.filter, err)
				return
			}
			if matcher.cidr == nil {
				t.Errorf("NewIPMatcher(%q) expected CIDR matcher (from wildcard), got nil", tt.filter)
			}
		})
	}
}

func TestNewIPMatcher_SingleIP(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{"IPv4 single", "192.168.1.1", false},
		{"IPv6 single", "::1", false},
		{"IPv6 full", "2001:db8::1", false},
		{"Invalid", "not-an-ip", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewIPMatcher(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewIPMatcher(%q) expected error, got nil", tt.filter)
				}
				return
			}
			if err != nil {
				t.Errorf("NewIPMatcher(%q) unexpected error: %v", tt.filter, err)
				return
			}
			if matcher.exact == nil {
				t.Errorf("NewIPMatcher(%q) expected exact IP matcher, got nil", tt.filter)
			}
		})
	}
}

func TestIPMatcher_Match_CIDR(t *testing.T) {
	matcher, err := NewIPMatcher("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		ip    string
		match bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.255", true},
		{"192.168.1.0", true},
		{"192.168.2.1", false},
		{"10.0.0.1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			matched, err := matcher.Match(tt.ip)
			if tt.ip == "invalid" {
				if err == nil {
					t.Errorf("Match(%q) expected error", tt.ip)
				}
				return
			}
			if err != nil {
				t.Errorf("Match(%q) unexpected error: %v", tt.ip, err)
				return
			}
			if matched != tt.match {
				t.Errorf("Match(%q) = %v, want %v", tt.ip, matched, tt.match)
			}
		})
	}
}

func TestIPMatcher_Match_IPv6CIDR(t *testing.T) {
	matcher, err := NewIPMatcher("2001:db8::/32")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		ip    string
		match bool
	}{
		{"2001:db8::1", true},
		{"2001:db8:1234::1", true},
		{"2001:db9::1", false},
		{"::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			matched, err := matcher.Match(tt.ip)
			if err != nil {
				t.Errorf("Match(%q) unexpected error: %v", tt.ip, err)
				return
			}
			if matched != tt.match {
				t.Errorf("Match(%q) = %v, want %v", tt.ip, matched, tt.match)
			}
		})
	}
}

func TestIPMatcher_Match_Range(t *testing.T) {
	matcher, err := NewIPMatcher("192.168.1.10-192.168.1.20")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		ip    string
		match bool
	}{
		{"192.168.1.10", true},
		{"192.168.1.15", true},
		{"192.168.1.20", true},
		{"192.168.1.9", false},
		{"192.168.1.21", false},
		{"192.168.2.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			matched, err := matcher.Match(tt.ip)
			if err != nil {
				t.Errorf("Match(%q) unexpected error: %v", tt.ip, err)
				return
			}
			if matched != tt.match {
				t.Errorf("Match(%q) = %v, want %v", tt.ip, matched, tt.match)
			}
		})
	}
}

func TestIPMatcher_Match_Wildcard(t *testing.T) {
	matcher, err := NewIPMatcher("192.168.*.*")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		ip    string
		match bool
	}{
		{"192.168.1.1", true},
		{"192.168.255.255", true},
		{"192.168.0.0", true},
		{"192.169.1.1", false},
		{"10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			matched, err := matcher.Match(tt.ip)
			if err != nil {
				t.Errorf("Match(%q) unexpected error: %v", tt.ip, err)
				return
			}
			if matched != tt.match {
				t.Errorf("Match(%q) = %v, want %v", tt.ip, matched, tt.match)
			}
		})
	}
}

func TestIPMatcher_Match_SingleIP(t *testing.T) {
	matcher, err := NewIPMatcher("192.168.1.1")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		ip    string
		match bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.2", false},
		{"192.168.1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			matched, _ := matcher.Match(tt.ip)
			if matched != tt.match {
				t.Errorf("Match(%q) = %v, want %v", tt.ip, matched, tt.match)
			}
		})
	}
}

func TestIPMatcher_MatchIP_WithNetIP(t *testing.T) {
	matcher, err := NewIPMatcher("10.0.0.0/8")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// Test MatchIP with pre-parsed net.IP
	ip := net.ParseIP("10.255.255.255")
	if !matcher.MatchIP(ip) {
		t.Errorf("MatchIP(%v) = false, want true", ip)
	}

	ip = net.ParseIP("11.0.0.1")
	if matcher.MatchIP(ip) {
		t.Errorf("MatchIP(%v) = true, want false", ip)
	}

	// Test nil IP
	if matcher.MatchIP(nil) {
		t.Errorf("MatchIP(nil) = true, want false")
	}
}

func TestWildcardToCIDR(t *testing.T) {
	tests := []struct {
		pattern  string
		cidr     string
		wantErr  bool
	}{
		{"192.168.1.*", "192.168.1.0/24", false},
		{"192.168.*.*", "192.168.0.0/16", false},
		{"192.*.*.*", "192.0.0.0/8", false},
		{"*.*.*.*", "0.0.0.0/0", false},
		{"10.0.*.*", "10.0.0.0/16", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			cidrNet, err := wildcardToCIDR(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Errorf("wildcardToCIDR(%q) expected error, got nil", tt.pattern)
				}
				return
			}
			if err != nil {
				t.Errorf("wildcardToCIDR(%q) unexpected error: %v", tt.pattern, err)
				return
			}
			if cidrNet.String() != tt.cidr {
				t.Errorf("wildcardToCIDR(%q) = %q, want %q", tt.pattern, cidrNet.String(), tt.cidr)
			}
		})
	}
}

func TestIsCIDRNotation(t *testing.T) {
	tests := []struct {
		filter string
		isCIDR bool
	}{
		{"192.168.1.0/24", true},
		{"2001:db8::/32", true},
		{"0.0.0.0/0", true},
		{"192.168.1.1", false},
		{"not-a-cidr", false},
		{"192.168.1.*", false},
		{"192.168.1.1-192.168.1.255", false},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			if got := IsCIDRNotation(tt.filter); got != tt.isCIDR {
				t.Errorf("IsCIDRNotation(%q) = %v, want %v", tt.filter, got, tt.isCIDR)
			}
		})
	}
}

func TestIPMatcher_BackwardCompatible(t *testing.T) {
	// Ensures existing exact IP filtering still works
	matcher, err := NewIPMatcher("123.45.67.89")
	if err != nil {
		t.Fatalf("Failed to create matcher for exact IP: %v", err)
	}

	matched, err := matcher.Match("123.45.67.89")
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}
	if !matched {
		t.Error("Exact IP should match")
	}

	matched, err = matcher.Match("123.45.67.90")
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}
	if matched {
		t.Error("Different exact IP should not match")
	}
}

func TestIPMatcher_IPv6Range(t *testing.T) {
	matcher, err := NewIPMatcher("::1-::5")
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	tests := []struct {
		ip    string
		match bool
	}{
		{"::1", true},
		{"::3", true},
		{"::5", true},
		{"::6", false},
		{"::0", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			matched, err := matcher.Match(tt.ip)
			if err != nil {
				t.Errorf("Match(%q) unexpected error: %v", tt.ip, err)
				return
			}
			if matched != tt.match {
				t.Errorf("Match(%q) = %v, want %v", tt.ip, matched, tt.match)
			}
		})
	}
}
