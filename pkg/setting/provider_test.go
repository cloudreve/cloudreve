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
