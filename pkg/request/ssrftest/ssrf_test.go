// Package ssrftest contains tests for pkg/request SSRF validation. It lives
// in a separate directory so it can compile even when the parent package's
// pre-existing _test.go files do not.
package ssrftest

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/stretchr/testify/assert"
)

func TestValidateExternalURL_Scheme(t *testing.T) {
	ctx := context.Background()
	cases := []string{
		"file:///etc/passwd",
		"gopher://example.com/_GET",
		"dict://127.0.0.1:11211/stat",
		"ldap://x",
		"javascript:alert(1)",
		"",
	}
	for _, raw := range cases {
		err := request.ValidateExternalURL(ctx, raw, request.SSRFOptions{})
		assert.ErrorIs(t, err, request.ErrUnsafeURL, "raw=%q", raw)
	}
}

func TestValidateExternalURL_LocalLiterals(t *testing.T) {
	ctx := context.Background()
	cases := []string{
		"http://localhost/x",
		"http://LOCALHOST:8080/x",
		"http://ip6-localhost/x",
		"http://service.localhost/x",
		"https://localhost.localdomain/x",
	}
	for _, raw := range cases {
		err := request.ValidateExternalURL(ctx, raw, request.SSRFOptions{})
		assert.ErrorIs(t, err, request.ErrUnsafeURL, "raw=%q", raw)
	}
}

func TestValidateExternalURL_IPLiterals(t *testing.T) {
	ctx := context.Background()
	cases := []string{
		"http://127.0.0.1:7777/secret",
		"http://[::1]:7780/v6secret",
		"http://10.0.0.5/internal",
		"http://172.16.5.5/internal",
		"http://192.168.1.1/router",
		"http://169.254.169.254/latest/meta",
		"http://169.254.42.42/link-local",
		"http://[fe80::1]/link-local-v6",
		"http://[fc00::1]/ula",
		"http://0.0.0.0/x",
		"http://[::]/x",
		"http://100.64.0.1/cgnat",
		"http://[::ffff:127.0.0.1]/v4mapped",
		"http://[::ffff:10.0.0.1]/v4mappedpriv",
		// IPv4-in-IPv6 transition forms wrapping internal targets.
		"http://[64:ff9b::127.0.0.1]/nat64-loopback",
		"http://[64:ff9b::a9fe:a9fe]/nat64-metadata",
		"http://[64:ff9b::169.254.169.254]/nat64-metadata-dec",
		"http://[64:ff9b::a00:1]/nat64-private",
		"http://[2002:7f00:0001::]/6to4-loopback",
		"http://[2002:a9fe:a9fe::]/6to4-metadata",
		"http://[2002:0a00:0001::]/6to4-private",
		"http://[2001:0000:4136:e378:8000:63bf:f5ff:fffe]/teredo-10.0.0.1",
		"http://[::127.0.0.1]/v4compatible",
		"http://224.0.0.1/multicast",
	}
	for _, raw := range cases {
		err := request.ValidateExternalURL(ctx, raw, request.SSRFOptions{})
		assert.ErrorIs(t, err, request.ErrUnsafeURL, "raw=%q", raw)
	}
}

func TestValidateExternalURL_PublicIP(t *testing.T) {
	ctx := context.Background()
	cases := []string{
		"http://1.1.1.1/",
		"https://8.8.8.8/",
		"http://[2606:4700:4700::1111]/",
		// Transition forms wrapping *public* IPv4 remain allowed.
		"http://[64:ff9b::808:808]/nat64-public",
		"http://[2002:0808:0808::]/6to4-public",
	}
	for _, raw := range cases {
		err := request.ValidateExternalURL(ctx, raw, request.SSRFOptions{})
		assert.NoError(t, err, "raw=%q", raw)
	}
}

func TestValidateExternalURL_AllowedHostBypass(t *testing.T) {
	ctx := context.Background()

	opt := request.SSRFOptions{AllowedHosts: []string{"localhost"}}
	for _, raw := range []string{
		"http://localhost/file.zip",
		"http://LOCALHOST:8080/file.zip",
	} {
		assert.NoError(t, request.ValidateExternalURL(ctx, raw, opt), "raw=%q", raw)
	}

	opt = request.SSRFOptions{AllowedHosts: []string{"cloudreve.example.com"}}
	assert.NoError(t, request.ValidateExternalURL(ctx, "https://cloudreve.example.com/x", opt))

	opt = request.SSRFOptions{AllowedHosts: []string{"127.0.0.1"}}
	assert.NoError(t, request.ValidateExternalURL(ctx, "http://127.0.0.1:7777/secret", opt))
}

func TestValidateExternalURL_Disabled(t *testing.T) {
	ctx := context.Background()
	opt := request.SSRFOptions{Disabled: true}

	// Everything passes when the operator has explicitly disabled validation.
	for _, raw := range []string{
		"http://127.0.0.1:7777/secret",
		"http://localhost/x",
		"file:///etc/passwd",
		"http://169.254.169.254/latest",
	} {
		assert.NoError(t, request.ValidateExternalURL(ctx, raw, opt), "raw=%q", raw)
	}
}

func TestValidateExternalURL_AllowedCIDR(t *testing.T) {
	ctx := context.Background()

	// LAN range allowlist lets a NAS at 192.168.10.50 through.
	opt := request.SSRFOptions{AllowedCIDRs: []string{"192.168.10.0/24"}}
	assert.NoError(t, request.ValidateExternalURL(ctx,
		"http://192.168.10.50/share/file.zip", opt))

	// IPs outside the allowlisted CIDR remain rejected.
	err := request.ValidateExternalURL(ctx,
		"http://192.168.11.50/share/file.zip", opt)
	assert.ErrorIs(t, err, request.ErrUnsafeURL)

	// IPv6 ULA allowlist.
	opt = request.SSRFOptions{AllowedCIDRs: []string{"fd00::/8"}}
	assert.NoError(t, request.ValidateExternalURL(ctx,
		"http://[fd12:3456::1]/x", opt))

	// Malformed CIDRs are silently dropped without affecting other entries.
	opt = request.SSRFOptions{AllowedCIDRs: []string{"not-a-cidr", "10.0.0.0/8"}}
	assert.NoError(t, request.ValidateExternalURL(ctx, "http://10.5.5.5/x", opt))
}

func TestValidateExternalURL_Magnet(t *testing.T) {
	ctx := context.Background()

	// DHT-only magnet (no tracker/web seed) is allowed.
	assert.NoError(t, request.ValidateExternalURL(ctx,
		"magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a",
		request.SSRFOptions{}))

	// Internal tracker URL is rejected.
	err := request.ValidateExternalURL(ctx,
		"magnet:?xt=urn:btih:abc&tr=http://127.0.0.1:7777/announce",
		request.SSRFOptions{})
	assert.ErrorIs(t, err, request.ErrUnsafeURL)

	// Internal web seed is rejected.
	err = request.ValidateExternalURL(ctx,
		"magnet:?xt=urn:btih:abc&ws=http://10.0.0.1/seed",
		request.SSRFOptions{})
	assert.ErrorIs(t, err, request.ErrUnsafeURL)

	// Public tracker is allowed.
	assert.NoError(t, request.ValidateExternalURL(ctx,
		"magnet:?xt=urn:btih:abc&tr=udp://tracker.opentrackr.org:1337/announce",
		request.SSRFOptions{}))
}
