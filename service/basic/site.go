package basic

import (
	"slices"
	"sort"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/thumb"
	"github.com/cloudreve/Cloudreve/v4/service/user"
	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
)

// SiteConfig 站点全局设置序列
type SiteConfig struct {
	// Basic Section
	InstanceID     string                  `json:"instance_id,omitempty"`
	SiteName       string                  `json:"title,omitempty"`
	Themes         string                  `json:"themes,omitempty"`
	DefaultTheme   string                  `json:"default_theme,omitempty"`
	User           *user.User              `json:"user,omitempty"`
	Logo           string                  `json:"logo,omitempty"`
	LogoLight      string                  `json:"logo_light,omitempty"`
	CustomNavItems []setting.CustomNavItem `json:"custom_nav_items,omitempty"`
	CustomHTML     *setting.CustomHTML     `json:"custom_html,omitempty"`

	// Share section
	ShareDefaultPrivate        bool   `json:"share_default_private,omitempty"`
	DefaultShareLinksInProfile string `json:"default_share_links_in_profile,omitempty"`

	// Login Section
	LoginCaptcha     bool                `json:"login_captcha,omitempty"`
	RegCaptcha       bool                `json:"reg_captcha,omitempty"`
	ForgetCaptcha    bool                `json:"forget_captcha,omitempty"`
	Authn            bool                `json:"authn,omitempty"`
	ReCaptchaKey     string              `json:"captcha_ReCaptchaKey,omitempty"`
	TCaptchaAppID    string              `json:"tcaptcha_app_id,omitempty"`
	CaptchaType      setting.CaptchaType `json:"captcha_type,omitempty"`
	TurnstileSiteID  string              `json:"turnstile_site_id,omitempty"`
	CapInstanceURL   string              `json:"captcha_cap_instance_url,omitempty"`
	CapSiteKey       string              `json:"captcha_cap_site_key,omitempty"`
	CapAssetServer   string              `json:"captcha_cap_asset_server,omitempty"`
	RegisterEnabled  bool                `json:"register_enabled,omitempty"`
	InvitationCode   bool                `json:"invitation_code,omitempty"`
	TosUrl           string              `json:"tos_url,omitempty"`
	PrivacyPolicyUrl string              `json:"privacy_policy_url,omitempty"`
	SSOEnabled       bool                `json:"sso_enabled,omitempty"`
	SSODisplayName   string              `json:"sso_display_name,omitempty"`
	SSOAutoRedirect  bool                `json:"sso_auto_redirect,omitempty"`
	QQConnectEnabled bool                `json:"qq_connect_enabled,omitempty"`
	WeChatEnabled    bool                `json:"wechat_connect_enabled,omitempty"`
	// SmsEnabled tells the login UI to offer phone + code sign-in.
	SmsEnabled bool `json:"sms_enabled,omitempty"`

	// DownloadCDNRoutes exposes configured CDN mirror endpoints so clients
	// can offer a download-route picker (#2987).
	DownloadCDNRoutes []setting.CDNRoute `json:"download_cdn_routes,omitempty"`

	// DownloadCDNShuffle tells clients generated download URLs are already
	// spread across the site URL and all CDN routes server-side (#173), so
	// the manual route picker can be skipped.
	DownloadCDNShuffle bool `json:"download_cdn_shuffle,omitempty"`

	// AbuseCaptcha controls whether the report-abuse dialog shows captcha.
	AbuseCaptcha bool `json:"abuse_captcha,omitempty"`

	// UploadDedup signals clients to send content hashes with upload
	// sessions, enabling server-side duplicate detection / instant upload.
	UploadDedup bool `json:"upload_dedup,omitempty"`

	// TaskNodes lists nodes the current user may target when creating tasks
	// (remote download, archive ops); populated when the group allows node
	// selection, filtered to the group's allowed pool.
	TaskNodes       []TaskNode `json:"task_nodes,omitempty"`
	AllowSelectNode bool       `json:"allow_select_node,omitempty"`

	// Explorer section
	Icons                string                     `json:"icons,omitempty"`
	EmojiPreset          string                     `json:"emoji_preset,omitempty"`
	MapProvider          setting.MapProvider        `json:"map_provider,omitempty"`
	GoogleMapTileType    setting.MapGoogleTileType  `json:"google_map_tile_type,omitempty"`
	MapboxAK             string                     `json:"mapbox_ak,omitempty"`
	FileViewers          []types.ViewerGroup        `json:"file_viewers,omitempty"`
	DefaultViewerMapping types.DefaultViewerMapping `json:"default_viewer_mapping,omitempty"`
	MaxBatchSize         int                        `json:"max_batch_size,omitempty"`
	ThumbnailWidth       int                        `json:"thumbnail_width,omitempty"`
	ThumbnailHeight      int                        `json:"thumbnail_height,omitempty"`
	CustomProps          []types.CustomProps        `json:"custom_props,omitempty"`
	ShowEncryptionStatus bool                       `json:"show_encryption_status,omitempty"`
	FullTextSearch       bool                       `json:"full_text_search,omitempty"`
	// RemoteDownloadProviders lists distinct downloader providers offered by
	// active remote-download-capable nodes, so clients can let users pick
	// between e.g. Aria2 and qBittorrent per download.
	RemoteDownloadProviders []string `json:"remote_download_providers,omitempty"`

	// Thumbnail section
	ThumbExts []string `json:"thumb_exts,omitempty"`

	// App settings
	AppPromotion        bool `json:"app_promotion,omitempty"`
	DesktopAppPromotion bool `json:"desktop_app_promotion,omitempty"`

	//EmailActive          bool      `json:"emailActive"`
	//QQLogin              bool      `json:"QQLogin"`
	//ScoreEnabled         bool      `json:"score_enabled"`
	//ShareScoreRate       string    `json:"share_score_rate"`
	//HomepageViewMethod   string    `json:"home_view_method"`
	//ShareViewMethod      string    `json:"share_view_method"`
	//WopiExts             []string            `json:"wopi_exts"`
	//AppFeedbackLink      string              `json:"app_feedback"`
	//AppForumLink         string              `json:"app_forum"`
}

// TaskNode is the minimal public node descriptor for task targeting.
type TaskNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type (
	GetSettingService struct {
		Section string `uri:"section" binding:"required"`
	}
	GetSettingParamCtx struct{}
)

func (s *GetSettingService) GetSiteConfig(c *gin.Context) (*SiteConfig, error) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	switch s.Section {
	case "login":
		legalDocs := settings.LegalDocuments(c)
		sso := settings.SSO(c)
		qq := settings.QQConnect(c)
		wx := settings.WeChatConnect(c)
		return &SiteConfig{
			LoginCaptcha:     settings.LoginCaptchaEnabled(c),
			RegCaptcha:       settings.RegCaptchaEnabled(c),
			ForgetCaptcha:    settings.ForgotPasswordCaptchaEnabled(c),
			Authn:            settings.AuthnEnabled(c),
			RegisterEnabled:  settings.RegisterEnabled(c),
			InvitationCode:   settings.InvitationCodeRequired(c),
			PrivacyPolicyUrl: legalDocs.PrivacyPolicy,
			TosUrl:           legalDocs.TermsOfService,
			SSOEnabled:       sso.Enabled && sso.Issuer != "" && sso.ClientID != "",
			SSODisplayName:   sso.DisplayName,
			SSOAutoRedirect:  sso.AutoRedirect,
			QQConnectEnabled: qq.Enabled && qq.AppID != "",
			WeChatEnabled:    wx.Enabled && wx.AppID != "",
			SmsEnabled:       settings.SmsGateway(c).Enabled && settings.SmsGateway(c).Endpoint != "",
		}, nil
	case "explorer":
		explorerSettings := settings.ExplorerFrontendSettings(c)
		mapSettings := settings.MapSetting(c)
		fileViewers := settings.FileViewers(c)
		customProps := settings.CustomProps(c)
		maxBatchSize := settings.MaxBatchedFile(c)
		showEncryptionStatus := settings.ShowEncryptionStatus(c)
		w, h := settings.ThumbSize(c)
		for i := range fileViewers {
			for j := range fileViewers[i].Viewers {
				fileViewers[i].Viewers[j].WopiActions = nil
			}
		}
		return &SiteConfig{
			MaxBatchSize:            maxBatchSize,
			FileViewers:             fileViewers,
			DefaultViewerMapping:    settings.DefaultViewerMapping(c),
			Icons:                   explorerSettings.Icons,
			MapProvider:             mapSettings.Provider,
			GoogleMapTileType:       mapSettings.GoogleTileType,
			MapboxAK:                mapSettings.MapboxAK,
			ThumbnailWidth:          w,
			ThumbnailHeight:         h,
			CustomProps:             customProps,
			ShowEncryptionStatus:    showEncryptionStatus,
			FullTextSearch:          settings.FTSEnabled(c),
			RemoteDownloadProviders: remoteDownloadProviders(c, dep),
		}, nil
	case "emojis":
		emojis := settings.EmojiPresets(c)
		return &SiteConfig{
			EmojiPreset: emojis,
		}, nil
	case "app":
		appSetting := settings.AppSetting(c)
		return &SiteConfig{
			AppPromotion:        appSetting.Promotion,
			DesktopAppPromotion: appSetting.DesktopPromotion,
		}, nil
	case "thumb":
		// Return supported thumbnail extensions from enabled generators.
		exts := map[string]bool{}
		if settings.BuiltinThumbGeneratorEnabled(c) {
			for _, e := range thumb.BuiltinSupportedExts {
				exts[e] = true
			}
		}
		if settings.FFMpegThumbGeneratorEnabled(c) {
			for _, e := range settings.FFMpegThumbExts(c) {
				exts[strings.ToLower(e)] = true
			}
		}
		if settings.VipsThumbGeneratorEnabled(c) {
			for _, e := range settings.VipsThumbExts(c) {
				exts[strings.ToLower(e)] = true
			}
		}
		if settings.LibreOfficeThumbGeneratorEnabled(c) {
			for _, e := range settings.LibreOfficeThumbExts(c) {
				exts[strings.ToLower(e)] = true
			}
		}
		if settings.MusicCoverThumbGeneratorEnabled(c) {
			for _, e := range settings.MusicCoverThumbExts(c) {
				exts[strings.ToLower(e)] = true
			}
		}
		if settings.LibRawThumbGeneratorEnabled(c) {
			for _, e := range settings.LibRawThumbExts(c) {
				exts[strings.ToLower(e)] = true
			}
		}

		// map -> sorted slice
		result := make([]string, 0, len(exts))
		for e := range exts {
			result = append(result, e)
		}
		sort.Strings(result)
		return &SiteConfig{ThumbExts: result}, nil
	default:
		break
	}

	u := inventory.UserFromContext(c)
	lang := ""
	if u != nil {
		lang = u.Settings.Language
	}
	if lang == "" {
		lang = setting.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
	}
	siteBasic := settings.SiteBasicLocalized(c, lang)
	themes := settings.Theme(c)
	userRes := user.BuildUser(u, dep.HashIDEncoder())
	logo := settings.Logo(c)
	reCaptcha := settings.ReCaptcha(c)
	capCaptcha := settings.CapCaptcha(c)
	appSetting := settings.AppSetting(c)
	customNavItems := settings.CustomNavItems(c)
	customHTML := settings.CustomHTML(c)
	shareDefaults := settings.ShareDefaults(c)
	taskNodes, allowSelect := taskNodesForUser(c, dep, u)
	return &SiteConfig{
		InstanceID:                 siteBasic.ID,
		SiteName:                   siteBasic.Name,
		Themes:                     themes.Themes,
		DefaultTheme:               themes.DefaultTheme,
		User:                       &userRes,
		Logo:                       logo.Normal,
		LogoLight:                  logo.Light,
		CaptchaType:                settings.CaptchaType(c),
		TCaptchaAppID:              settings.TcCaptcha(c).AppID,
		TurnstileSiteID:            settings.TurnstileCaptcha(c).Key,
		ReCaptchaKey:               reCaptcha.Key,
		CapInstanceURL:             capCaptcha.InstanceURL,
		CapSiteKey:                 capCaptcha.SiteKey,
		CapAssetServer:             capCaptcha.AssetServer,
		AppPromotion:               appSetting.Promotion,
		CustomNavItems:             customNavItems,
		CustomHTML:                 customHTML,
		ShareDefaultPrivate:        shareDefaults.PrivateByDefault,
		DefaultShareLinksInProfile: string(shareDefaults.LinksInProfile),
		DownloadCDNRoutes:          settings.DownloadCDNRoutes(c),
		DownloadCDNShuffle:         settings.DownloadCDNShuffle(c),
		AbuseCaptcha:               settings.AbuseCaptchaEnabled(c),
		UploadDedup:                settings.DBFS(c).DedupScope != "off",
		TaskNodes:                  taskNodes,
		AllowSelectNode:            allowSelect,
	}, nil
}

// taskNodesForUser returns the nodes a user may target for tasks: active
// nodes with any task capability, intersected with the group's allowed pool.
// Returns nil when the group disallows selection.
func taskNodesForUser(c *gin.Context, dep dependency.Dep, u *ent.User) ([]TaskNode, bool) {
	if u == nil || u.Edges.Group == nil || !u.Edges.Group.Settings.AllowSelectNode {
		return nil, false
	}

	nodes, err := dep.NodeClient().ListActiveNodes(c, nil)
	if err != nil {
		return nil, true
	}

	allowed := u.Edges.Group.Settings.AllowedNodes
	taskCaps := []types.NodeCapability{
		types.NodeCapabilityCreateArchive,
		types.NodeCapabilityExtractArchive,
		types.NodeCapabilityRemoteDownload,
	}
	res := make([]TaskNode, 0, len(nodes))
	for _, n := range nodes {
		if len(allowed) > 0 && !slices.Contains(allowed, n.ID) {
			continue
		}
		capable := false
		for _, cap := range taskCaps {
			if n.Capabilities != nil && n.Capabilities.Enabled(int(cap)) {
				capable = true
				break
			}
		}
		if capable {
			res = append(res, TaskNode{ID: hashid.EncodeNodeID(dep.HashIDEncoder(), n.ID), Name: n.Name})
		}
	}
	return res, true
}

const (
	CaptchaSessionPrefix = "captcha_session_"
	CaptchaTTL           = 1800 // 30 minutes
)

type (
	CaptchaResponse struct {
		Image  string `json:"image"`
		Ticket string `json:"ticket"`
	}
)

// GetCaptchaImage generates captcha session
func GetCaptchaImage(c *gin.Context) *CaptchaResponse {
	dep := dependency.FromContext(c)
	captchaSettings := dep.SettingProvider().Captcha(c)
	var configD = base64Captcha.ConfigCharacter{
		Height:             captchaSettings.Height,
		Width:              captchaSettings.Width,
		Mode:               int(captchaSettings.Mode),
		ComplexOfNoiseText: captchaSettings.ComplexOfNoiseText,
		ComplexOfNoiseDot:  captchaSettings.ComplexOfNoiseDot,
		IsShowHollowLine:   captchaSettings.IsShowHollowLine,
		IsShowNoiseDot:     captchaSettings.IsShowNoiseDot,
		IsShowNoiseText:    captchaSettings.IsShowNoiseText,
		IsShowSlimeLine:    captchaSettings.IsShowSlimeLine,
		IsShowSineLine:     captchaSettings.IsShowSineLine,
		CaptchaLen:         captchaSettings.Length,
	}

	// 生成验证码
	idKeyD, capD := base64Captcha.GenerateCaptcha("", configD)

	base64stringD := base64Captcha.CaptchaWriteToBase64Encoding(capD)

	return &CaptchaResponse{
		Image:  base64stringD,
		Ticket: idKeyD,
	}
}

// remoteDownloadProviders returns the distinct downloader providers offered by
// active nodes with remote-download capability, in stable sorted order.
func remoteDownloadProviders(c *gin.Context, dep dependency.Dep) []string {
	nodes, err := dep.NodeClient().ListActiveNodes(c, nil)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var providers []string
	for _, n := range nodes {
		if n.Capabilities == nil || !n.Capabilities.Enabled(int(types.NodeCapabilityRemoteDownload)) ||
			n.Settings == nil || n.Settings.Provider == "" || seen[string(n.Settings.Provider)] {
			continue
		}
		seen[string(n.Settings.Provider)] = true
		providers = append(providers, string(n.Settings.Provider))
	}
	sort.Strings(providers)
	return providers
}
