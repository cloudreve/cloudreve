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

	// MediaProcessSetting carries the media post-processing (image compression)
	// parameters (APP-101), all editable from the admin panel.
	MediaProcessSetting struct {
		ImageEnabled bool   // master switch for image compression
		Engine       string // "vips" | "ffmpeg"
		WorkerNum    int    // compression concurrency (dedicated key; default 1)
		BatchSize    int    // max rows the cron enqueues per run
		Quality      int    // 1..100
		Format       string // "keep" | "webp" | "jpeg" | "png"
		ExtraArgs    string // extra engine flags
		ResultMode   string // "version" | "replace" | "auto"
		MinSize      int64  // skip blobs smaller than this (bytes)
	}

	// MediaProcessVideoSetting carries the deferred video transcoding parameters
	// (APP-103), all editable from the admin panel. The engine is always ffmpeg
	// (vips does not process video). Video runs on its own dedicated queue so a
	// long transcode never blocks image compression.
	MediaProcessVideoSetting struct {
		Enabled       bool   // master switch for video transcoding
		Codec         string // video codec, e.g. "libx264" | "libx265"
		CRF           int    // constant rate factor (lower = better quality/bigger)
		Preset        string // ffmpeg preset (speed/size trade-off), e.g. "medium"
		Container     string // "keep" | "mp4" | "webm" — output container/extension
		MaxResolution string // downscale cap, e.g. "1920x1080" (empty = no cap)
		AudioCodec    string // audio codec, e.g. "aac"
		AudioBitrate  string // audio bitrate, e.g. "128k"
		ExtraArgs     string // extra ffmpeg flags
		WorkerNum     int    // transcoding concurrency (dedicated key; default 1)
		BatchSize     int    // max rows the cron enqueues per run
		Threads       int    // ffmpeg -threads per encode (approximate CPU cap)
		Nice          bool   // run ffmpeg at low priority (best effort, non-Windows)
		MinSize       int64  // skip blobs smaller than this (bytes)
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
	QueueTypeMediaProcess   = QueueType("media_process")
	// QueueTypeMediaProcessVideo is the dedicated queue for video transcoding
	// (APP-103), kept separate from media_process so a long transcode does not
	// block image compression.
	QueueTypeMediaProcessVideo = QueueType("media_process_video")
)

type CronType string

var (
	CronTypeEntityCollect    = CronType("entity_collect")
	CronTypeTrashBinCollect  = CronType("trash_bin_collect")
	CronTypeOauthCredRefresh = CronType("oauth_cred_refresh")
	CronTypeMediaProcess     = CronType("media_process")
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
