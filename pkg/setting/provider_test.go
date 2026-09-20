package setting

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubAdapter map[string]any

func (s stubAdapter) Get(_ context.Context, name string, defaultVal any) any {
	if v, ok := s[name]; ok {
		return v
	}
	return defaultVal
}

func TestDownloadCDNRoutes(t *testing.T) {
	ctx := context.Background()

	p := NewProvider(stubAdapter{})
	require.Empty(t, p.DownloadCDNRoutes(ctx))

	p = NewProvider(stubAdapter{
		"download_cdn_routes": "Line 1=https://cdn1.example.com/\n\n" +
			"not-a-route\n" +
			"cdn2 = https://cdn2.example.com/base/\n" +
			"bad=ftp://example.com\n" +
			"also-bad=notaurl",
	})
	routes := p.DownloadCDNRoutes(ctx)
	require.Equal(t, []CDNRoute{
		{Name: "Line 1", URL: "https://cdn1.example.com"},
		{Name: "cdn2", URL: "https://cdn2.example.com/base"},
	}, routes)
}

func TestDownloadURLBase(t *testing.T) {
	ctx := context.Background()

	// Shuffle disabled: always the resolved site URL.
	p := NewProvider(stubAdapter{
		"siteURL":             "https://a.example.com,https://b.example.com",
		"download_cdn_routes": "cdn1=https://cdn1.example.com",
	})
	require.Equal(t, "https://a.example.com", p.DownloadURLBase(ctx).String())

	// Shuffle enabled without routes: still the site URL.
	p = NewProvider(stubAdapter{
		"siteURL":              "https://a.example.com",
		"download_cdn_shuffle": "1",
	})
	require.Equal(t, "https://a.example.com", p.DownloadURLBase(ctx).String())

	// Shuffle enabled: every draw lands on site URL or a configured route,
	// and all endpoints are reached over enough draws.
	p = NewProvider(stubAdapter{
		"siteURL":              "https://a.example.com",
		"download_cdn_shuffle": "1",
		"download_cdn_routes":  "cdn1=https://cdn1.example.com\ncdn2=https://cdn2.example.com",
	})
	seen := map[string]bool{}
	for i := 0; i < 300; i++ {
		seen[p.DownloadURLBase(ctx).String()] = true
	}
	require.Equal(t, map[string]bool{
		"https://a.example.com":    true,
		"https://cdn1.example.com": true,
		"https://cdn2.example.com": true,
	}, seen)

	// UseFirstSiteUrl pins the primary site URL even with shuffle on.
	pinned := context.WithValue(ctx, UseFirstSiteUrlCtxKey{}, true)
	require.Equal(t, "https://a.example.com", p.DownloadURLBase(pinned).String())

	// Invalid route entries never leak into the pool.
	p = NewProvider(stubAdapter{
		"siteURL":              "https://a.example.com",
		"download_cdn_shuffle": "1",
		"download_cdn_routes":  "bad=ftp://x.example.com",
	})
	for i := 0; i < 50; i++ {
		require.Equal(t, "https://a.example.com", p.DownloadURLBase(ctx).String())
	}
}
