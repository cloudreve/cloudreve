package wopi

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"net/url"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

var (
	ErrActionNotSupported = errors.New("action not supported by current wopi endpoint")

	queryPlaceholders = map[string]string{
		"BUSINESS_USER":           "",
		"DC_LLCC":                 "lng",
		"DISABLE_ASYNC":           "",
		"DISABLE_CHAT":            "",
		"EMBEDDED":                "true",
		"FULLSCREEN":              "true",
		"HOST_SESSION_ID":         "",
		"SESSION_CONTEXT":         "",
		"RECORDING":               "",
		"THEME_ID":                "darkmode",
		"UI_LLCC":                 "lng",
		"VALIDATOR_TEST_CATEGORY": "",
	}
)

const (
	SessionCachePrefix    = "wopi_session_"
	AccessTokenQuery      = "access_token"
	OverwriteHeader       = WopiHeaderPrefix + "Override"
	ServerErrorHeader     = WopiHeaderPrefix + "ServerError"
	RenameRequestHeader   = WopiHeaderPrefix + "RequestedName"
	LockTokenHeader       = WopiHeaderPrefix + "Lock"
	ItemVersionHeader     = WopiHeaderPrefix + "ItemVersion"
	SuggestedTargetHeader = WopiHeaderPrefix + "SuggestedTarget"

	MethodLock           = "LOCK"
	MethodUnlock         = "UNLOCK"
	MethodRefreshLock    = "REFRESH_LOCK"
	MethodPutRelative    = "PUT_RELATIVE"
	wopiSrcPlaceholder   = "WOPI_SOURCE"
	wopiSrcParamDefault  = "WOPISrc"
	languageParamDefault = "lang"
	WopiHeaderPrefix     = "X-WOPI-"

	LockDuration = time.Duration(30) * time.Minute
)

func GenerateWopiSrc(ctx context.Context, action types.ViewerAction, viewer *types.Viewer, viewerSession *manager.ViewerSession) (*url.URL, error) {
	dep := dependency.FromContext(ctx)
	base := dep.SettingProvider().SiteURL(setting.UseFirstSiteUrl(ctx))
	hasher := dep.HashIDEncoder()

	availableActions, ok := viewer.WopiActions[viewerSession.File.Ext()]
	if !ok {
		return nil, ErrActionNotSupported
	}

	var (
		src string
	)
	fallbackOrder := []types.ViewerAction{action, types.ViewerActionView, types.ViewerActionEdit}
	for _, a := range fallbackOrder {
		if src, ok = availableActions[a]; ok {
			break
		}
	}

	if src == "" {
		return nil, ErrActionNotSupported
	}

	actionUrl, err := generateActionUrl(src,
		routes.MasterWopiSrc(base, hashid.EncodeFileID(hasher, viewerSession.File.ID())).String())
	if err != nil {
		return nil, err
	}

	return actionUrl, nil
}

// Replace query parameters in action URL template. Some placeholders need to be replaced
// at the frontend, e.g. `THEME_ID`.
func generateActionUrl(src string, fileSrc string) (*url.URL, error) {
	src = strings.ReplaceAll(src, "<", "")
	src = strings.ReplaceAll(src, ">", "")
	actionUrl, err := url.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action url: %s", err)
	}

	queries := actionUrl.Query()
	srcReplaced := false
	queryReplaced := url.Values{}
	for k := range queries {
		if placeholder, ok := queryPlaceholders[queries.Get(k)]; ok {
			if placeholder != "" {
				queryReplaced.Set(k, placeholder)
			}

			continue
		}

		if queries.Get(k) == wopiSrcPlaceholder {
			queryReplaced.Set(k, fileSrc)
			srcReplaced = true
			continue
		}

		queryReplaced.Set(k, queries.Get(k))
	}

	if !srcReplaced {
		queryReplaced.Set(wopiSrcParamDefault, fileSrc)
	}

	// LibreOffice require this flag to show correct language
	queryReplaced.Set(languageParamDefault, "lng")

	actionUrl.RawQuery = queryReplaced.Encode()
	return actionUrl, nil
}
