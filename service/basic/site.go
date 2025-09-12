package basic

import (
    "sort"
    "strings"

    "github.com/cloudreve/Cloudreve/v4/application/dependency"
    "github.com/cloudreve/Cloudreve/v4/inventory"
    "github.com/cloudreve/Cloudreve/v4/inventory/types"
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

	// Login Section
	LoginCaptcha     bool                `json:"login_captcha,omitempty"`
	RegCaptcha       bool                `json:"reg_captcha,omitempty"`
	ForgetCaptcha    bool                `json:"forget_captcha,omitempty"`
	Authn            bool                `json:"authn,omitempty"`
	ReCaptchaKey     string              `json:"captcha_ReCaptchaKey,omitempty"`
	CaptchaType      setting.CaptchaType `json:"captcha_type,omitempty"`
	TurnstileSiteID  string              `json:"turnstile_site_id,omitempty"`
	CapInstanceURL   string              `json:"captcha_cap_instance_url,omitempty"`
	CapSiteKey       string              `json:"captcha_cap_site_key,omitempty"`
	CapAssetServer   string              `json:"captcha_cap_asset_server,omitempty"`
	RegisterEnabled  bool                `json:"register_enabled,omitempty"`
	TosUrl           string              `json:"tos_url,omitempty"`
	PrivacyPolicyUrl string              `json:"privacy_policy_url,omitempty"`

	// Explorer section
	Icons             string                    `json:"icons,omitempty"`
	EmojiPreset       string                    `json:"emoji_preset,omitempty"`
	MapProvider       setting.MapProvider       `json:"map_provider,omitempty"`
	GoogleMapTileType setting.MapGoogleTileType `json:"google_map_tile_type,omitempty"`
	FileViewers       []types.ViewerGroup       `json:"file_viewers,omitempty"`
	MaxBatchSize      int                       `json:"max_batch_size,omitempty"`
	ThumbnailWidth    int                       `json:"thumbnail_width,omitempty"`
	ThumbnailHeight   int                       `json:"thumbnail_height,omitempty"`
    CustomProps       []types.CustomProps       `json:"custom_props,omitempty"`

    // Thumbnail section
    ThumbExts []string `json:"thumb_exts,omitempty"`

	// App settings
	AppPromotion bool `json:"app_promotion,omitempty"`

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
		return &SiteConfig{
			LoginCaptcha:     settings.LoginCaptchaEnabled(c),
			RegCaptcha:       settings.RegCaptchaEnabled(c),
			ForgetCaptcha:    settings.ForgotPasswordCaptchaEnabled(c),
			Authn:            settings.AuthnEnabled(c),
			RegisterEnabled:  settings.RegisterEnabled(c),
			PrivacyPolicyUrl: legalDocs.PrivacyPolicy,
			TosUrl:           legalDocs.TermsOfService,
		}, nil
	case "explorer":
		explorerSettings := settings.ExplorerFrontendSettings(c)
		mapSettings := settings.MapSetting(c)
		fileViewers := settings.FileViewers(c)
		customProps := settings.CustomProps(c)
		maxBatchSize := settings.MaxBatchedFile(c)
		w, h := settings.ThumbSize(c)
		for i := range fileViewers {
			for j := range fileViewers[i].Viewers {
				fileViewers[i].Viewers[j].WopiActions = nil
			}
		}
		return &SiteConfig{
			MaxBatchSize:      maxBatchSize,
			FileViewers:       fileViewers,
			Icons:             explorerSettings.Icons,
			MapProvider:       mapSettings.Provider,
			GoogleMapTileType: mapSettings.GoogleTileType,
			ThumbnailWidth:    w,
			ThumbnailHeight:   h,
			CustomProps:       customProps,
		}, nil
	case "emojis":
		emojis := settings.EmojiPresets(c)
		return &SiteConfig{
			EmojiPreset: emojis,
		}, nil
	case "app":
		appSetting := settings.AppSetting(c)
        return &SiteConfig{
            AppPromotion: appSetting.Promotion,
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
	siteBasic := settings.SiteBasic(c)
	themes := settings.Theme(c)
	userRes := user.BuildUser(u, dep.HashIDEncoder())
	logo := settings.Logo(c)
	reCaptcha := settings.ReCaptcha(c)
	capCaptcha := settings.CapCaptcha(c)
	appSetting := settings.AppSetting(c)
	customNavItems := settings.CustomNavItems(c)
	customHTML := settings.CustomHTML(c)
	return &SiteConfig{
		InstanceID:      siteBasic.ID,
		SiteName:        siteBasic.Name,
		Themes:          themes.Themes,
		DefaultTheme:    themes.DefaultTheme,
		User:            &userRes,
		Logo:            logo.Normal,
		LogoLight:       logo.Light,
		CaptchaType:     settings.CaptchaType(c),
		TurnstileSiteID: settings.TurnstileCaptcha(c).Key,
		ReCaptchaKey:    reCaptcha.Key,
		CapInstanceURL:  capCaptcha.InstanceURL,
		CapSiteKey:      capCaptcha.SiteKey,
		CapAssetServer:  capCaptcha.AssetServer,
		AppPromotion:    appSetting.Promotion,
		CustomNavItems:  customNavItems,
		CustomHTML:      customHTML,
	}, nil
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
