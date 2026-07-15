package request

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrUnsafeURL is returned when a user-supplied URL targets an address that
// should not be reachable by server-side fetchers (loopback, private,
// link-local, cloud metadata, multicast, etc.) or uses a non-allowed scheme.
var ErrUnsafeURL = errors.New("URL is not allowed")

var downloadAllowedSchemes = map[string]struct{}{
	"http":   {},
	"https":  {},
	"ftp":    {},
	"ftps":   {},
	"sftp":   {},
	"magnet": {},
}

// trackerAllowedSchemes are schemes valid inside magnet tr=/ws=/xs=/as= params
// that we recognize and validate. Anything else is treated as non-network
// (e.g. dht://) and skipped.
var trackerAllowedSchemes = map[string]struct{}{
	"http":  {},
	"https": {},
	"udp":   {},
	"ws":    {},
	"wss":   {},
}

var bannedHostnames = map[string]struct{}{
	"localhost":             {},
	"localhost.localdomain": {},
	"ip6-localhost":         {},
	"ip6-loopback":          {},
}

// 100.64.0.0/10 (RFC 6598 CGNAT) is not flagged by net.IP.IsPrivate.
var cgnatNet = mustParseCIDR("100.64.0.0/10")

// 169.254.169.254 (cloud instance metadata) is link-local and already covered
// by IsLinkLocalUnicast, but we keep an explicit check for clarity.
var cloudMetadataIP = net.ParseIP("169.254.169.254")

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// SSRFOptions configures ValidateExternalURL.
type SSRFOptions struct {
	// Disabled short-circuits the entire check. Use when the operator has
	// explicitly opted out (e.g. node-level URLValidation.Disabled).
	Disabled bool
	// AllowedHosts contains hostnames whose URLs bypass all subsequent checks
	// (used to whitelist the operator-configured site URL host(s) and any
	// operator-trusted internal hostnames). Compared case-insensitively to
	// the URL's Hostname(). Port is ignored.
	AllowedHosts []string
	// AllowedCIDRs contains CIDR blocks whose IPs bypass the IP-class checks.
	// Resolved IPs falling within any of these are treated as safe even if
	// they would otherwise be rejected (private, link-local, ...). Malformed
	// entries are ignored. Used to opt a LAN range back in.
	AllowedCIDRs []string
	// Resolver is used for DNS lookups; nil uses net.DefaultResolver.
	Resolver *net.Resolver
}

// ValidateExternalURL returns an error wrapping ErrUnsafeURL if the URL would
// reach an internal-only address when fetched by a server-side downloader.
// Magnet links have their tr=/ws=/xs=/as= parameters validated as URLs.
func ValidateExternalURL(ctx context.Context, raw string, opt SSRFOptions) error {
	if opt.Disabled {
		return nil
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty URL: %w", ErrUnsafeURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse URL: %w", ErrUnsafeURL)
	}

	scheme := strings.ToLower(u.Scheme)
	if _, ok := downloadAllowedSchemes[scheme]; !ok {
		return fmt.Errorf("scheme %q not allowed: %w", u.Scheme, ErrUnsafeURL)
	}

	if scheme == "magnet" {
		return validateMagnet(ctx, u, opt)
	}

	return validateHost(ctx, u.Hostname(), opt)
}

func validateMagnet(ctx context.Context, u *url.URL, opt SSRFOptions) error {
	q := u.Query()
	for _, key := range []string{"tr", "ws", "xs", "as"} {
		for _, v := range q[key] {
			sub, err := url.Parse(strings.TrimSpace(v))
			if err != nil {
				return fmt.Errorf("magnet %s=%q: %w", key, v, ErrUnsafeURL)
			}
			if sub.Hostname() == "" {
				continue
			}
			if _, ok := trackerAllowedSchemes[strings.ToLower(sub.Scheme)]; !ok {
				continue
			}
			if err := validateHost(ctx, sub.Hostname(), opt); err != nil {
				return fmt.Errorf("magnet %s=%q: %w", key, v, err)
			}
		}
	}
	return nil
}

func validateHost(ctx context.Context, host string, opt SSRFOptions) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("empty host: %w", ErrUnsafeURL)
	}

	for _, allowed := range opt.AllowedHosts {
		if strings.EqualFold(strings.TrimSpace(allowed), host) {
			return nil
		}
	}

	lowered := strings.ToLower(host)
	if _, banned := bannedHostnames[lowered]; banned {
		return fmt.Errorf("hostname %q is local: %w", host, ErrUnsafeURL)
	}
	if strings.HasSuffix(lowered, ".localhost") {
		return fmt.Errorf("hostname %q is local: %w", host, ErrUnsafeURL)
	}

	allowed := parseCIDRs(opt.AllowedCIDRs)

	if ip := net.ParseIP(host); ip != nil {
		return checkIPWithAllowlist(ip, allowed)
	}

	resolver := opt.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", host, ErrUnsafeURL)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("no addresses for %q: %w", host, ErrUnsafeURL)
	}
	for _, a := range addrs {
		if err := checkIPWithAllowlist(a.IP, allowed); err != nil {
			return err
		}
	}
	return nil
}

// parseCIDRs parses CIDR strings, silently dropping malformed entries — the
// admin gets validation feedback at config-save time, so noisy errors during a
// download are not useful.
func parseCIDRs(raw []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(raw))
	for _, r := range raw {
		_, n, err := net.ParseCIDR(strings.TrimSpace(r))
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

func checkIPWithAllowlist(ip net.IP, allowed []*net.IPNet) error {
	// Normalize IPv4-in-IPv6 transition forms so both the allowlist and the
	// class check operate on the address the packet actually reaches.
	ip = effectiveIP(ip)
	for _, n := range allowed {
		if n.Contains(ip) {
			return nil
		}
	}
	return checkIP(ip)
}

// nat64WellKnownPrefix is 64:ff9b::/96 (RFC 6052 §2.1), the 12-byte prefix
// that wraps an IPv4 address in its last 32 bits for NAT64 gateways.
var nat64WellKnownPrefix = []byte{
	0x00, 0x64, 0xff, 0x9b,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00,
}

// teredoPrefix is 2001:0000::/32 (RFC 4380). In Teredo, the client's IPv4 is
// the last 32 bits of the address XORed with 0xff.
var teredoPrefix = []byte{0x20, 0x01, 0x00, 0x00}

// effectiveIP unwraps IPv4-in-IPv6 transition forms to the IPv4 address the
// packet ultimately reaches, so checkIP classifies the real target rather
// than the (often globally-routable) IPv6 wrapper. Go's net.IP builtins
// (IsLoopback, IsPrivate, IsLinkLocalUnicast, ...) inspect only the outer
// address and would otherwise accept, for example, 64:ff9b::a9fe:a9fe as a
// safe public IPv6 even though it routes to 169.254.169.254 on a NAT64 host.
//
// Covers:
//   - IPv4-mapped     ::ffff:a.b.c.d          (RFC 4291 §2.5.5.2)
//   - NAT64 WKP       64:ff9b::a.b.c.d        (RFC 6052 §2.1)
//   - 6to4            2002:AABB:CCDD::        (RFC 3056)
//   - Teredo          2001:0000:...           (RFC 4380)
//   - IPv4-compatible ::a.b.c.d               (RFC 4291 §2.5.5.1, deprecated)
func effectiveIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	// IPv4 literal or IPv4-mapped ::ffff:a.b.c.d — To4 returns the embedded v4.
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	v6 := ip.To16()
	if v6 == nil {
		return ip
	}

	// NAT64 well-known prefix 64:ff9b::/96.
	if bytes.HasPrefix(v6, nat64WellKnownPrefix) {
		return net.IPv4(v6[12], v6[13], v6[14], v6[15]).To4()
	}

	// 6to4 2002::/16 — bytes 2-5 encode the embedded IPv4.
	if v6[0] == 0x20 && v6[1] == 0x02 {
		return net.IPv4(v6[2], v6[3], v6[4], v6[5]).To4()
	}

	// Teredo 2001:0000::/32 — client IPv4 is last 4 bytes XOR 0xff.
	if bytes.HasPrefix(v6, teredoPrefix) {
		return net.IPv4(v6[12]^0xff, v6[13]^0xff, v6[14]^0xff, v6[15]^0xff).To4()
	}

	// IPv4-compatible ::a.b.c.d — top 12 bytes are zero. Skip :: and ::1 so
	// the outer classifier still catches them as unspecified/loopback IPv6.
	var zero [12]byte
	if bytes.Equal(v6[:12], zero[:]) {
		if last := binary.BigEndian.Uint32(v6[12:16]); last > 1 {
			return net.IPv4(v6[12], v6[13], v6[14], v6[15]).To4()
		}
	}

	return ip
}

func checkIP(ip net.IP) error {
	ip = effectiveIP(ip)
	if ip == nil {
		return fmt.Errorf("invalid IP: %w", ErrUnsafeURL)
	}
	switch {
	case ip.IsLoopback():
		return fmt.Errorf("loopback address %s: %w", ip, ErrUnsafeURL)
	case ip.IsUnspecified():
		return fmt.Errorf("unspecified address %s: %w", ip, ErrUnsafeURL)
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return fmt.Errorf("link-local address %s: %w", ip, ErrUnsafeURL)
	case ip.IsPrivate():
		return fmt.Errorf("private address %s: %w", ip, ErrUnsafeURL)
	case ip.IsMulticast():
		return fmt.Errorf("multicast address %s: %w", ip, ErrUnsafeURL)
	case ip.Equal(cloudMetadataIP):
		return fmt.Errorf("cloud metadata address %s: %w", ip, ErrUnsafeURL)
	case cgnatNet.Contains(ip):
		return fmt.Errorf("CGNAT address %s: %w", ip, ErrUnsafeURL)
	}
	return nil
}
