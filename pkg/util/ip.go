package util

import (
	"fmt"
	"net"
	"strings"
)

// IPMatcher checks whether an IP address matches a given filter.
// The filter can be:
//   - A single IP address (e.g., "192.168.1.1" or "::1")
//   - A CIDR notation (e.g., "192.168.1.0/24" or "2001:db8::/32")
//   - An IP range (e.g., "192.168.1.1-192.168.1.255")
//   - A wildcard pattern (e.g., "192.168.1.*" or "192.168.*.*")
type IPMatcher struct {
	cidr    *net.IPNet
	ipStart net.IP
	ipEnd   net.IP
	exact   net.IP
}

// NewIPMatcher creates an IPMatcher from a filter string.
// Supported formats:
//   - CIDR: "192.168.1.0/24", "2001:db8::/32"
//   - Range: "192.168.1.1-192.168.1.255", "::1-::10"
//   - Wildcard: "192.168.1.*", "192.168.*.*", "10.*.*.*"
//   - Single IP: "192.168.1.1", "::1"
func NewIPMatcher(filter string) (*IPMatcher, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return nil, fmt.Errorf("empty IP filter")
	}

	// Try CIDR notation first
	if strings.Contains(filter, "/") {
		_, cidrNet, err := net.ParseCIDR(filter)
		if err == nil {
			return &IPMatcher{cidr: cidrNet}, nil
		}
		// If it has "/" but isn't valid CIDR, continue to try other formats
	}

	// Try IP range (e.g., "192.168.1.1-192.168.1.255")
	if strings.Count(filter, "-") == 1 {
		parts := strings.SplitN(filter, "-", 2)
		start := net.ParseIP(strings.TrimSpace(parts[0]))
		end := net.ParseIP(strings.TrimSpace(parts[1]))
		if start != nil && end != nil {
			// Ensure same IP version
			if (start.To4() != nil) == (end.To4() != nil) {
				return &IPMatcher{ipStart: start, ipEnd: end}, nil
			}
			return nil, fmt.Errorf("IP range must contain same IP version: %s", filter)
		}
	}

	// Try wildcard pattern (e.g., "192.168.1.*" or "192.168.*.*")
	if strings.Contains(filter, "*") {
		cidr, err := wildcardToCIDR(filter)
		if err == nil {
			return &IPMatcher{cidr: cidr}, nil
		}
		return nil, err
	}

	// Try single exact IP
	ip := net.ParseIP(filter)
	if ip != nil {
		return &IPMatcher{exact: ip}, nil
	}

	return nil, fmt.Errorf("invalid IP filter format: %s", filter)
}

// Match checks if the given IP address matches the filter.
// The ip parameter should be a string representation of an IP address.
// Returns an error if the IP string is invalid.
func (m *IPMatcher) Match(ip string) (bool, error) {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false, fmt.Errorf("invalid IP address: %s", ip)
	}

	return m.MatchIP(parsedIP), nil
}

// MatchIP checks if the given parsed IP address matches the filter.
func (m *IPMatcher) MatchIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	switch {
	case m.cidr != nil:
		return m.cidr.Contains(ip)
	case m.ipStart != nil && m.ipEnd != nil:
		return ipInRange(ip, m.ipStart, m.ipEnd)
	case m.exact != nil:
		return m.exact.Equal(ip)
	}

	return false
}

// ipInRange checks if ip is within the inclusive range [start, end].
func ipInRange(ip, start, end net.IP) bool {
	// Normalize to 16-byte representation for comparison
	ip16 := ip.To16()
	start16 := start.To16()
	end16 := end.To16()

	return bytesCompare(start16, ip16) <= 0 && bytesCompare(ip16, end16) <= 0
}

// bytesCompare compares two byte slices lexicographically.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func bytesCompare(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// wildcardToCIDR converts a wildcard IP pattern to a CIDR net.
// Only supports IPv4 wildcards (e.g., "192.168.1.*" -> "192.168.1.0/24").
func wildcardToCIDR(pattern string) (*net.IPNet, error) {
	pattern = strings.TrimSpace(pattern)
	parts := strings.Split(pattern, ".")

	if len(parts) != 4 {
		return nil, fmt.Errorf("wildcard pattern currently only supports IPv4: %s", pattern)
	}

	// Count how many fixed octets
	fixedOctets := 0
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "*" {
			// Fill remaining octets with 0 for the network address
			for j := i; j < 4; j++ {
				parts[j] = "0"
			}
			break
		}
		fixedOctets++
	}

	// All wildcards would be 0.0.0.0/0
	if fixedOctets == 0 {
		return &net.IPNet{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.CIDRMask(0, 32),
		}, nil
	}

	// Construct CIDR network
	cidrStr := fmt.Sprintf("%s/%d", strings.Join(parts, "."), fixedOctets*8)
	_, cidrNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid wildcard pattern: %s (%w)", pattern, err)
	}

	return cidrNet, nil
}

// IsCIDRNotation checks if a filter string is a valid CIDR notation.
func IsCIDRNotation(filter string) bool {
	_, _, err := net.ParseCIDR(strings.TrimSpace(filter))
	return err == nil
}

// IPFilterToCIDRs converts an IP filter string to a list of CIDR networks.
// This is useful when you need to convert user-friendly IP filters
// to CIDR notation for display or storage.
// Supports CIDR, wildcard, single IP, and IP range (expanded).
func IPFilterToCIDRs(filter string) ([]*net.IPNet, error) {
	matcher, err := NewIPMatcher(filter)
	if err != nil {
		return nil, err
	}

	switch {
	case matcher.cidr != nil:
		return []*net.IPNet{matcher.cidr}, nil
	case matcher.exact != nil:
		// Single IP -> /32 or /128
		if matcher.exact.To4() != nil {
			return []*net.IPNet{{
				IP:   matcher.exact,
				Mask: net.CIDRMask(32, 32),
			}}, nil
		}
		return []*net.IPNet{{
			IP:   matcher.exact,
			Mask: net.CIDRMask(128, 128),
		}}, nil
	case matcher.ipStart != nil && matcher.ipEnd != nil:
		// IP range - return as two /32 or /128 CIDRs
		// Note: This is a simplified representation; the caller should
		// use Match/MatchIP for exact range matching
		cidrs := make([]*net.IPNet, 0)
		if matcher.ipStart.To4() != nil {
			cidrs = append(cidrs, &net.IPNet{
				IP:   matcher.ipStart,
				Mask: net.CIDRMask(32, 32),
			})
			cidrs = append(cidrs, &net.IPNet{
				IP:   matcher.ipEnd,
				Mask: net.CIDRMask(32, 32),
			})
		} else {
			cidrs = append(cidrs, &net.IPNet{
				IP:   matcher.ipStart,
				Mask: net.CIDRMask(128, 128),
			})
			cidrs = append(cidrs, &net.IPNet{
				IP:   matcher.ipEnd,
				Mask: net.CIDRMask(128, 128),
			})
		}
		return cidrs, nil
	}

	return nil, fmt.Errorf("cannot convert filter to CIDR: %s", filter)
}
