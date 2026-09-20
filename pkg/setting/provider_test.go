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

func TestLocalizedSetting(t *testing.T) {
	ctx := context.Background()

	p := NewProvider(stubAdapter{
		"siteName":      "Cloudreve",
		"siteName_i18n": `{"zh-CN":"云盘","zh":"简中","*":"Intl"}`,
	})

	// Exact tag wins over the bare-subtag and wildcard entries.
	require.Equal(t, "云盘", p.Localized(ctx, "siteName", "zh-CN"))
	// Regional request falls back to the bare primary-subtag entry.
	require.Equal(t, "简中", p.Localized(ctx, "siteName", "zh-TW"))
	// Bare request may match a regional entry.
	require.Equal(t, "简中", p.Localized(ctx, "siteName", "zh"))
	// Wildcard entry covers unmatched languages.
	require.Equal(t, "Intl", p.Localized(ctx, "siteName", "fr-FR"))
	// Empty lang or missing map falls back to the base value.
	require.Equal(t, "Cloudreve", p.Localized(ctx, "siteName", ""))
	require.Equal(t, "", p.Localized(ctx, "other", "zh-CN"))

	// Without a wildcard, unmatched languages get the base value.
	p = NewProvider(stubAdapter{"siteName": "Cloudreve", "siteName_i18n": `{"zh-CN":"云盘"}`})
	require.Equal(t, "Cloudreve", p.Localized(ctx, "siteName", "fr-FR"))

	// Malformed i18n JSON is ignored.
	p = NewProvider(stubAdapter{"siteName": "Cloudreve", "siteName_i18n": "{bad"})
	require.Equal(t, "Cloudreve", p.Localized(ctx, "siteName", "zh-CN"))
}

func TestSiteBasicLocalized(t *testing.T) {
	p := NewProvider(stubAdapter{
		"siteName":      "Cloudreve",
		"siteName_i18n": `{"zh-CN":"云盘"}`,
		"siteDes":       "desc",
		"siteDes_i18n":  `{"zh-CN":"描述"}`,
	})
	b := p.SiteBasicLocalized(context.Background(), "zh-CN")
	require.Equal(t, "云盘", b.Name)
	require.Equal(t, "描述", b.Description)
	b = p.SiteBasicLocalized(context.Background(), "")
	require.Equal(t, "Cloudreve", b.Name)
	require.Equal(t, "desc", b.Description)
}

func TestParseAcceptLanguage(t *testing.T) {
	require.Equal(t, "zh-CN", ParseAcceptLanguage("zh-CN,zh;q=0.9,en;q=0.8"))
	require.Equal(t, "en-US", ParseAcceptLanguage("en-US;q=0.7"))
	require.Equal(t, "", ParseAcceptLanguage(""))
}
