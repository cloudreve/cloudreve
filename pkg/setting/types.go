package setting

import (
	"time"
)

type PWASetting struct {
	SmallIcon       string
	MediumIcon      string
	LargeIcon       string
	Display         string
	ThemeColor      string
	BackgroundColor string
}

type SiteBasic struct {
	Name        string
	Title       string
	ID          string
	Description string
	Script      string
}

type CaptchaType string

const (
	CaptchaNormal    = CaptchaType("normal")
	CaptchaReCaptcha = CaptchaType("recaptcha")
	CaptchaTcaptcha  = CaptchaType("tcaptcha")
	CaptchaTurnstile = CaptchaType("turnstile")
	CaptchaCap       = CaptchaType("cap")
)

type ReCaptcha struct {
	Key    string
	Secret string
}

type TcCaptcha struct {
	AppID        string
	AppSecretKey string
	SecretID     string
	SecretKey    string
}

type Turnstile struct {
	Key    string
	Secret string
}

type Cap struct {
	InstanceURL string
	SiteKey     string
	SecretKey   string
	AssetServer string
}

type SMTP struct {
	FromName        string
	From            string
	Host            string
	ReplyTo         string
	User            string
	Password        string
	ForceEncryption bool
	Port            int
	Keepalive       int
	AuthType        string
}

type TokenAuth struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// SSO holds inbound single sign-on (OIDC consumer) settings.
type SSO struct {
	Enabled         bool
	DisplayName     string
	Issuer          string
	ClientID        string
	ClientSecret    string
	Scopes          string
	RegisterEnabled bool
	AutoRedirect    bool
}

// QQConnect holds the QQ互联 (connect.qq.com) OAuth2 application config.
// Unlike the generic OIDC consumer, QQ Connect has no discovery document or
// id_token; identity resolves through /oauth2.0/me openid.
type QQConnect struct {
	Enabled         bool
	AppID           string
	AppSecret       string
	RegisterEnabled bool
}

// WeChatConnect holds the WeChat Open Platform (open.weixin.qq.com) scan
// login config. Identity resolves through the token response's unionid,
// falling back to openid.
type WeChatConnect struct {
	Enabled         bool
	AppID           string
	AppSecret       string
	RegisterEnabled bool
}

// SmsGateway holds the generic HTTP SMS gateway config used for phone
// verification codes. Endpoint/BodyTemplate accept `{phone}` and `{code}`
// placeholders; Headers is a newline-separated `Key: Value` list.
type SmsGateway struct {
	Enabled         bool
	Endpoint        string
	Method          string
	Headers         string
	BodyTemplate    string
	RegisterEnabled bool
}

type EmailFilterMode int

const (
	EmailFilterDisabled EmailFilterMode = iota
	EmailFilterWhitelist
	EmailFilterBlacklist
)

// EmailFilter holds sign-up email restriction settings.
type EmailFilter struct {
	Mode              EmailFilterMode
	List              []string
	DisableSubAddress bool
	// SubAddressChars are the characters treated as sub-address separators in
	// the local part when DisableSubAddress is on. Empty falls back to "+".
	SubAddressChars string
}

type DBFS struct {
	UseCursorPagination        bool
	MaxPageSize                int
	MaxRecursiveSearchedFolder int
	UseSSEForSearch            bool
	// DedupScope controls hash-based duplicate detection on upload:
	// "off" disables it, "owner" dedups against the uploader's own
	// entities, "global" dedups across all users.
	DedupScope string
}

type (
	QueueType    string
	QueueSetting struct {
		WorkerNum          int
		MaxExecution       time.Duration
		BackoffFactor      float64
		BackoffMaxDuration time.Duration
		MaxRetry           int
		RetryDelay         time.Duration
	}
)

type ThumbEncode struct {
	Quality int
	Format  string
}

var (
	QueueTypeMediaMeta      = QueueType("media_meta")
	QueueTypeIOIntense      = QueueType("io_intense")
	QueueTypeThumb          = QueueType("thumb")
	QueueTypeEntityRecycle  = QueueType("recycle")
	QueueTypeSlave          = QueueType("slave")
	QueueTypeRemoteDownload = QueueType("remote_download")
)

type CronType string

var (
	CronTypeEntityCollect    = CronType("entity_collect")
	CronTypeTrashBinCollect  = CronType("trash_bin_collect")
	CronTypeOauthCredRefresh = CronType("oauth_cred_refresh")
	CronTypeGrantExpire      = CronType("grant_expire")
	CronTypeAuditCleanup     = CronType("audit_cleanup")
)

type Theme struct {
	Themes       string
	DefaultTheme string
}

type Logo struct {
	Normal string
	Light  string
}

type LegalDocuments struct {
	PrivacyPolicy  string
	TermsOfService string
}

type CaptchaMode int

const (
	CaptchaModeNumber = CaptchaMode(iota)
	CaptchaModeAlphabet
	CaptchaModeArithmetic
	CaptchaModeNumberAlphabet
)

type Captcha struct {
	Height             int
	Width              int
	Mode               CaptchaMode
	ComplexOfNoiseText int
	ComplexOfNoiseDot  int
	IsShowHollowLine   bool
	IsShowNoiseDot     bool
	IsShowNoiseText    bool
	IsShowSlimeLine    bool
	IsShowSineLine     bool
	Length             int
}

type ExplorerFrontendSettings struct {
	Icons string
}

type MapProvider string

const (
	MapProviderOpenStreetMap = MapProvider("openstreetmap")
	MapProviderGoogle        = MapProvider("google")
	MapProviderMapbox        = MapProvider("mapbox")
)

type MapGoogleTileType string

const (
	MapGoogleTileTypeRegular   = MapGoogleTileType("regular")
	MapGoogleTileTypeSatellite = MapGoogleTileType("satellite")
	MapGoogleTileTypeTerrain   = MapGoogleTileType("terrain")
)

type MapSetting struct {
	Provider       MapProvider
	GoogleTileType MapGoogleTileType
	MapboxAK       string
}

// Viewer related

type (
	SearchCategory string
)

const (
	CategoryUnknown  = SearchCategory("unknown")
	CategoryImage    = SearchCategory("image")
	CategoryVideo    = SearchCategory("video")
	CategoryAudio    = SearchCategory("audio")
	CategoryDocument = SearchCategory("document")
)

type AppSetting struct {
	Promotion        bool
	DesktopPromotion bool
}

type EmailTemplate struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Language string `json:"language"`
}

type Avatar struct {
	Gravatar string `json:"gravatar"`
	Path     string `json:"path"`
}

type AvatarProcess struct {
	Path        string `json:"path"`
	MaxFileSize int64  `json:"max_file_size"`
	MaxWidth    int    `json:"max_width"`
}

type CustomNavItem struct {
	Icon string `json:"icon"`
	Name string `json:"name"`
	URL  string `json:"url"`
	// Scope limits display: "" or "public" shows to everyone, "user" only
	// to logged-in accounts (#3578).
	Scope string `json:"scope,omitempty"`
}

type CustomHTML struct {
	HeadlessFooter string `json:"headless_footer,omitempty"`
	HeadlessBody   string `json:"headless_bottom,omitempty"`
	SidebarBottom  string `json:"sidebar_bottom,omitempty"`
}

type FTSIndexType string

const (
	FTSIndexTypeNone        = FTSIndexType("")
	FTSIndexTypeMeilisearch = FTSIndexType("meilisearch")
)

type FTSExtractorType string

const (
	FTSExtractorTypeNone = FTSExtractorType("")
	FTSExtractorTypeTika = FTSExtractorType("tika")
)

type FTSIndexMeilisearchSetting struct {
	Endpoint         string
	APIKey           string
	PageSize         int
	EmbeddingEnbaled bool
	EmbeddingSetting string
}

type FTSTikaExtractorSetting struct {
	Endpoint    string
	Exts        []string
	MaxFileSize int64
}

type MasterEncryptKeyVaultType string

const (
	MasterEncryptKeyVaultTypeSetting = MasterEncryptKeyVaultType("setting")
	MasterEncryptKeyVaultTypeEnv     = MasterEncryptKeyVaultType("env")
	MasterEncryptKeyVaultTypeFile    = MasterEncryptKeyVaultType("file")
)
