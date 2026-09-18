package user

import (
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

func TestCheckEmailAllowed(t *testing.T) {
	tests := []struct {
		name    string
		filter  *setting.EmailFilter
		email   string
		wantErr bool
		code    int
	}{
		{
			name:   "disabled filter allows any domain",
			filter: &setting.EmailFilter{Mode: setting.EmailFilterDisabled},
			email:  "a@anything.com",
		},
		{
			name: "whitelist allows listed domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterWhitelist,
				List: []string{"example.com"},
			},
			email: "a@example.com",
		},
		{
			name: "whitelist allows subdomain of listed domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterWhitelist,
				List: []string{"example.com"},
			},
			email: "a@mail.example.com",
		},
		{
			name: "whitelist rejects unlisted domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterWhitelist,
				List: []string{"example.com"},
			},
			email:   "a@other.com",
			wantErr: true,
			code:    serializer.CodeParamErr,
		},
		{
			name: "whitelist does not suffix-match partial domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterWhitelist,
				List: []string{"ample.com"},
			},
			email:   "a@example.com",
			wantErr: true,
			code:    serializer.CodeParamErr,
		},
		{
			name: "blacklist rejects listed domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterBlacklist,
				List: []string{"spam.com"},
			},
			email:   "a@spam.com",
			wantErr: true,
			code:    serializer.CodeEmailProviderBaned,
		},
		{
			name: "blacklist rejects subdomain of listed domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterBlacklist,
				List: []string{"spam.com"},
			},
			email:   "a@mx.spam.com",
			wantErr: true,
			code:    serializer.CodeEmailProviderBaned,
		},
		{
			name: "blacklist allows unlisted domain",
			filter: &setting.EmailFilter{
				Mode: setting.EmailFilterBlacklist,
				List: []string{"spam.com"},
			},
			email: "a@ok.com",
		},
		{
			name: "sub-address rejected when disabled",
			filter: &setting.EmailFilter{
				Mode:              setting.EmailFilterDisabled,
				DisableSubAddress: true,
			},
			email:   "a+tag@example.com",
			wantErr: true,
			code:    serializer.CodeParamErr,
		},
		{
			name: "sub-address allowed when enabled",
			filter: &setting.EmailFilter{
				Mode:              setting.EmailFilterDisabled,
				DisableSubAddress: false,
			},
			email: "a+tag@example.com",
		},
		{
			name: "custom sub-address chars rejected",
			filter: &setting.EmailFilter{
				Mode:              setting.EmailFilterDisabled,
				DisableSubAddress: true,
				SubAddressChars:   "+-.",
			},
			email:   "a-tag@example.com",
			wantErr: true,
			code:    serializer.CodeParamErr,
		},
		{
			name: "custom sub-address chars allow plain local part",
			filter: &setting.EmailFilter{
				Mode:              setting.EmailFilterDisabled,
				DisableSubAddress: true,
				SubAddressChars:   "+-.",
			},
			email: "user@example.com",
		},
		{
			name: "dot in domain not treated as sub-address",
			filter: &setting.EmailFilter{
				Mode:              setting.EmailFilterDisabled,
				DisableSubAddress: true,
				SubAddressChars:   "+-.",
			},
			email: "user@mail.example.com",
		},
		{
			name:    "invalid email rejected",
			filter:  &setting.EmailFilter{Mode: setting.EmailFilterDisabled},
			email:   "not-an-email",
			wantErr: true,
			code:    serializer.CodeParamErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckEmailAllowed(tt.filter, tt.email)
			if tt.wantErr {
				require.Error(t, err)
				if appErr, ok := err.(serializer.AppError); ok {
					require.Equal(t, tt.code, appErr.Code)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSanitizeSSORedirect(t *testing.T) {
	require.Equal(t, "", sanitizeSSORedirect(""))
	require.Equal(t, "", sanitizeSSORedirect("https://evil.com"))
	require.Equal(t, "", sanitizeSSORedirect("//evil.com"))
	require.Equal(t, "", sanitizeSSORedirect("evil.com/path"))
	require.Equal(t, "/home", sanitizeSSORedirect("/home"))
	require.Equal(t, "/share/s/abc", sanitizeSSORedirect("/share/s/abc"))
}

func TestFirstEmailClaim(t *testing.T) {
	require.Equal(t, "user@corp.com", firstEmailClaim("User@Corp.com"))
	require.Equal(t, "user@corp.com", firstEmailClaim("  User@Corp.COM  "))
	require.Equal(t, "user@corp.com", firstEmailClaim("", `CORP\user`, "user@corp.com"))
	require.Equal(t, "", firstEmailClaim("", `CORP\user`, "not-an-email"))
	require.Equal(t, "", firstEmailClaim())
}

func TestCheckLoginIPWhitelist(t *testing.T) {
	group := func(list ...string) *ent.Group {
		return &ent.Group{Settings: &types.GroupSetting{LoginIPWhitelist: list}}
	}

	// Empty whitelist allows everything.
	require.NoError(t, checkLoginIPWhitelist("1.2.3.4", nil))
	require.NoError(t, checkLoginIPWhitelist("1.2.3.4", group()))

	// Exact IP match.
	require.NoError(t, checkLoginIPWhitelist("10.0.0.5", group("10.0.0.5", "192.168.0.0/16")))
	require.Error(t, checkLoginIPWhitelist("10.0.0.6", group("10.0.0.5")))

	// CIDR ranges, v4 and v6.
	require.NoError(t, checkLoginIPWhitelist("192.168.1.9", group("192.168.0.0/16")))
	require.NoError(t, checkLoginIPWhitelist("fd00::42", group("fd00::/8")))
	require.Error(t, checkLoginIPWhitelist("8.8.8.8", group("192.168.0.0/16", "fd00::/8")))

	// Malformed entries are skipped rather than locking everyone out.
	require.NoError(t, checkLoginIPWhitelist("1.2.3.4", group("not-an-ip", "1.2.3.4")))
}
