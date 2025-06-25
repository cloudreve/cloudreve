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
	Version     string // "1.x" or "2.x"
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
}

type TokenAuth struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type DBFS struct {
	UseCursorPagination        bool
	MaxPageSize                int
	MaxRecursiveSearchedFolder int
	UseSSEForSearch            bool
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
	Promotion bool
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
