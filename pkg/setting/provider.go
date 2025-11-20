package setting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth/requestinfo"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
)

type (
	// Provider provides strong type setting access.
	Provider interface {
		// Site basic information
		SiteBasic(ctx context.Context) *SiteBasic
		// PWA related settings
		PWA(ctx context.Context) *PWASetting
		// RegisterEnabled returns true if public sign-up is enabled.
		RegisterEnabled(ctx context.Context) bool
		// AuthnEnabled returns true if Webauthn is enabled.
		AuthnEnabled(ctx context.Context) bool
		// RegCaptchaEnabled returns true if registration captcha is enabled.
		RegCaptchaEnabled(ctx context.Context) bool
		// LoginCaptchaEnabled returns true if login captcha is enabled.
		LoginCaptchaEnabled(ctx context.Context) bool
		// ForgotPasswordCaptchaEnabled returns true if forgot password captcha is enabled.
		ForgotPasswordCaptchaEnabled(ctx context.Context) bool
		// CaptchaType returns the type of captcha used.
		CaptchaType(ctx context.Context) CaptchaType
		// ReCaptcha returns the Google reCaptcha settings.
		ReCaptcha(ctx context.Context) *ReCaptcha
		// TcCaptcha returns the Tencent Cloud Captcha settings.
		TcCaptcha(ctx context.Context) *TcCaptcha
		// TurnstileCaptcha returns the Cloudflare Turnstile settings.
		TurnstileCaptcha(ctx context.Context) *Turnstile
		// CapCaptcha returns the Cap settings.
		CapCaptcha(ctx context.Context) *Cap
		// EmailActivationEnabled returns true if email activation is required.
		EmailActivationEnabled(ctx context.Context) bool
		// DefaultGroup returns the default group ID for new users.
		DefaultGroup(ctx context.Context) int
		// SMTP returns the SMTP settings.
		SMTP(ctx context.Context) *SMTP
		// SiteURL returns the basic URL.
		SiteURL(ctx context.Context) *url.URL
		// SecretKey returns the secret key for general signature.
		SecretKey(ctx context.Context) string
		// ActivationEmailTemplate returns the email template for activation.
		ActivationEmailTemplate(ctx context.Context) []EmailTemplate
		// ResetEmailTemplate returns the email template for reset password.
		ResetEmailTemplate(ctx context.Context) []EmailTemplate
		// TokenAuth returns token based auth related settings.
		TokenAuth(ctx context.Context) *TokenAuth
		// HashIDSalt returns the salt used for hash ID generation.
		HashIDSalt(ctx context.Context) string
		// DBFS returns the DBFS related settings.
		DBFS(ctx context.Context) *DBFS
		// MaxBatchedFile returns the maximum number of files in a batch operation.
		MaxBatchedFile(ctx context.Context) int
		// UploadSessionTTL returns the TTL of upload session.
		UploadSessionTTL(ctx context.Context) time.Duration
		// MaxOnlineEditSize returns the maximum size of online editing.
		MaxOnlineEditSize(ctx context.Context) int64
		// SlaveRequestSignTTL returns the TTL of slave request signature.
		SlaveRequestSignTTL(ctx context.Context) int
		// ChunkRetryLimit returns the maximum number of chunk retries.
		ChunkRetryLimit(ctx context.Context) int
		// UseChunkBuffer returns true if chunk buffer is enabled.
		UseChunkBuffer(ctx context.Context) bool
		// Queue returns the queue settings.
		Queue(ctx context.Context, queueType QueueType) *QueueSetting
		// EntityUrlCacheMargin returns the safe margin of entity URL cache. URL cache will
		// expire in (EntityUrlValidDuration - EntityUrlCacheMargin).
		EntityUrlCacheMargin(ctx context.Context) int
		// EntityUrlValidDuration returns the valid duration of entity URL.
		EntityUrlValidDuration(ctx context.Context) time.Duration
		// PublicResourceMaxAge returns the max age of public resources.
		PublicResourceMaxAge(ctx context.Context) int
		// MediaMetaEnabled returns true if media meta is enabled.
		MediaMetaEnabled(ctx context.Context) bool
		// MediaMetaExifEnabled returns true if media meta exif is enabled.
		MediaMetaExifEnabled(ctx context.Context) bool
		// MediaMetaExifSizeLimit returns the size limit of media meta exif. first return value is for local sources;
		// second return value is for remote sources.
		MediaMetaExifSizeLimit(ctx context.Context) (int64, int64)
		// MediaMetaExifBruteForce returns true if media meta exif brute force search is enabled.
		MediaMetaExifBruteForce(ctx context.Context) bool
		// MediaMetaMusicEnabled returns true if media meta audio is enabled.
		MediaMetaMusicEnabled(ctx context.Context) bool
		// MediaMetaMusicSizeLimit returns the size limit of media meta audio. first return value is for local sources;
		MediaMetaMusicSizeLimit(ctx context.Context) (int64, int64)
		// MediaMetaFFProbeEnabled returns true if media meta ffprobe is enabled.
		MediaMetaFFProbeEnabled(ctx context.Context) bool
		// MediaMetaFFProbeSizeLimit returns the size limit of media meta ffprobe. first return value is for local sources;
		MediaMetaFFProbeSizeLimit(ctx context.Context) (int64, int64)
		// MediaMetaFFProbePath returns the path of ffprobe executable.
		MediaMetaFFProbePath(ctx context.Context) string
		// MediaMetaGeocodingEnabled returns true if media meta geocoding is enabled.
		MediaMetaGeocodingEnabled(ctx context.Context) bool
		// MediaMetaGeocodingMapboxAK returns the Mapbox access token.
		MediaMetaGeocodingMapboxAK(ctx context.Context) string
		// ThumbSize returns the size limit of thumbnails.
		ThumbSize(ctx context.Context) (int, int)
		// ThumbEncode returns the thumbnail encoding settings.
		ThumbEncode(ctx context.Context) *ThumbEncode
		// BuiltinThumbGeneratorEnabled returns true if builtin thumb generator is enabled.
		BuiltinThumbGeneratorEnabled(ctx context.Context) bool
		// BuiltinThumbMaxSize returns the maximum size of builtin thumb generator.
		BuiltinThumbMaxSize(ctx context.Context) int64
		// TempPath returns the path of temporary directory.
		TempPath(ctx context.Context) string
		// ThumbEntitySuffix returns the suffix of entity thumbnails.
		ThumbEntitySuffix(ctx context.Context) string
		// ThumbSlaveSidecarSuffix returns the suffix of slave sidecar thumbnails.
		ThumbSlaveSidecarSuffix(ctx context.Context) string
		// ThumbGCAfterGen returns true if force GC is invoked after thumb generation.
		ThumbGCAfterGen(ctx context.Context) bool
		// FFMpegPath returns the path of ffmpeg executable.
		FFMpegPath(ctx context.Context) string
		// FFMpegThumbGeneratorEnabled returns true if ffmpeg thumb generator is enabled.
		FFMpegThumbGeneratorEnabled(ctx context.Context) bool
		// FFMpegThumbExts returns the supported extensions of ffmpeg thumb generator.
		FFMpegThumbExts(ctx context.Context) []string
		// FFMpegThumbSeek returns the seek time of ffmpeg thumb generator.
		FFMpegThumbSeek(ctx context.Context) string
		// FFMpegThumbMaxSize returns the maximum size of ffmpeg thumb generator.
		FFMpegThumbMaxSize(ctx context.Context) int64
		// VipsThumbGeneratorEnabled returns true if vips thumb generator is enabled.
		VipsThumbGeneratorEnabled(ctx context.Context) bool
		// VipsThumbExts returns the supported extensions of vips thumb generator.
		VipsThumbExts(ctx context.Context) []string
		// VipsThumbMaxSize returns the maximum size of vips thumb generator.
		VipsThumbMaxSize(ctx context.Context) int64
		// VipsPath returns the path of vips executable.
		VipsPath(ctx context.Context) string
		// LibreOfficeThumbGeneratorEnabled returns true if libreoffice thumb generator is enabled.
		LibreOfficeThumbGeneratorEnabled(ctx context.Context) bool
		// LibreOfficeThumbExts returns the supported extensions of libreoffice thumb generator.
		LibreOfficeThumbExts(ctx context.Context) []string
		// LibreOfficeThumbMaxSize returns the maximum size of libreoffice thumb generator.
		LibreOfficeThumbMaxSize(ctx context.Context) int64
		// LibreOfficePath returns the path of libreoffice executable.
		LibreOfficePath(ctx context.Context) string
		// MusicCoverThumbGeneratorEnabled returns true if music cover thumb generator is enabled.
		MusicCoverThumbGeneratorEnabled(ctx context.Context) bool
		// MusicCoverThumbMaxSize returns the maximum size of music cover thumb generator.
		MusicCoverThumbMaxSize(ctx context.Context) int64
		// MusicCoverThumbExts returns the supported extensions of music cover thumb generator.
		MusicCoverThumbExts(ctx context.Context) []string
		// Cron returns the crontab settings.
		Cron(ctx context.Context, t CronType) string
		// Theme returns the theme settings.
		Theme(ctx context.Context) *Theme
		// Logo returns the logo settings.
		Logo(ctx context.Context) *Logo
		// LegalDocuments returns the legal documents settings.
		LegalDocuments(ctx context.Context) *LegalDocuments
		// Captcha returns the captcha settings.
		Captcha(ctx context.Context) *Captcha
		// ExplorerFrontendSettings returns the explorer frontend settings.
		ExplorerFrontendSettings(ctx context.Context) *ExplorerFrontendSettings
		// SearchCategoryQuery returns the search category query.
		SearchCategoryQuery(ctx context.Context, category SearchCategory) string
		// EmojiPresets returns the emoji presets used in file icon customization.
		EmojiPresets(ctx context.Context) string
		// MapSetting returns the EXIF GPS map related settings.
		MapSetting(ctx context.Context) *MapSetting
		// FolderPropsCacheTTL returns the cache TTL of folder summary.
		FolderPropsCacheTTL(ctx context.Context) int
		// FileViewers returns the file viewers settings.
		FileViewers(ctx context.Context) []types.ViewerGroup
		// ViewerSessionTTL returns the TTL of viewer session.
		ViewerSessionTTL(ctx context.Context) int
		// MimeMapping returns the extension to MIME mapping settings.
		MimeMapping(ctx context.Context) string
		// MaxParallelTransfer returns the maximum parallel transfer in workflows.
		MaxParallelTransfer(ctx context.Context) int
		// ArchiveDownloadSessionTTL returns the TTL of archive download session.
		ArchiveDownloadSessionTTL(ctx context.Context) int
		// AppSetting returns the app related settings.
		AppSetting(ctx context.Context) *AppSetting
		// Avatar returns the avatar settings.
		Avatar(ctx context.Context) *Avatar
		// AvatarProcess returns the avatar process settings.
		AvatarProcess(ctx context.Context) *AvatarProcess
		// UseFirstSiteUrl returns the first site URL.
		AllSiteURLs(ctx context.Context) []*url.URL
		// LibRawThumbGeneratorEnabled returns true if libraw thumb generator is enabled.
		LibRawThumbGeneratorEnabled(ctx context.Context) bool
		// LibRawThumbMaxSize returns the maximum size of libraw thumb generator.
		LibRawThumbMaxSize(ctx context.Context) int64
		// LibRawThumbExts returns the supported extensions of libraw thumb generator.
		LibRawThumbExts(ctx context.Context) []string
		// LibRawThumbPath returns the path of libraw executable.
		LibRawThumbPath(ctx context.Context) string
		// CustomProps returns the custom props settings.
		CustomProps(ctx context.Context) []types.CustomProps
		// CustomNavItems returns the custom nav items settings.
		CustomNavItems(ctx context.Context) []CustomNavItem
		// CustomHTML returns the custom HTML settings.
		CustomHTML(ctx context.Context) *CustomHTML
		// FFMpegExtraArgs returns the extra arguments of ffmpeg thumb generator.
		FFMpegExtraArgs(ctx context.Context) string
		// MasterEncryptKey returns the master encrypt key.
		MasterEncryptKey(ctx context.Context) []byte
		// MasterEncryptKeyVault returns the master encrypt key vault type.
		MasterEncryptKeyVault(ctx context.Context) MasterEncryptKeyVaultType
		// MasterEncryptKeyFile returns the master encrypt key file.
		MasterEncryptKeyFile(ctx context.Context) string
		// ShowEncryptionStatus returns true if encryption status is shown.
		ShowEncryptionStatus(ctx context.Context) bool
	}
	UseFirstSiteUrlCtxKey = struct{}
)

// NewProvider creates a new setting provider.
func NewProvider(root SettingStoreAdapter) Provider {
	return &settingProvider{
		adapterChain: root,
	}
}

const (
	stringListDefault          = "DEFAULT"
	stringListDefaultSeparator = ","
)

var defaultBoolSet = &boolset.BooleanSet{}

type (
	SiteHostAllowListGetter interface {
		AllowedHost() []string
	}
	settingProvider struct {
		adapterChain SettingStoreAdapter
	}
)

func (s *settingProvider) ShowEncryptionStatus(ctx context.Context) bool {
	return s.getBoolean(ctx, "show_encryption_status", true)
}

func (s *settingProvider) MasterEncryptKeyFile(ctx context.Context) string {
	return s.getString(ctx, "encrypt_master_key_file", "")
}

func (s *settingProvider) MasterEncryptKeyVault(ctx context.Context) MasterEncryptKeyVaultType {
	return MasterEncryptKeyVaultType(s.getString(ctx, "encrypt_master_key_vault", "setting"))
}

func (s *settingProvider) MasterEncryptKey(ctx context.Context) []byte {
	encoded := s.getString(ctx, "encrypt_master_key", "")
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}
	return key
}

func (s *settingProvider) CustomHTML(ctx context.Context) *CustomHTML {
	return &CustomHTML{
		HeadlessFooter: s.getString(ctx, "headless_footer_html", ""),
		HeadlessBody:   s.getString(ctx, "headless_bottom_html", ""),
		SidebarBottom:  s.getString(ctx, "sidebar_bottom_html", ""),
	}
}
func (s *settingProvider) CustomNavItems(ctx context.Context) []CustomNavItem {
	raw := s.getString(ctx, "custom_nav_items", "[]")
	var items []CustomNavItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CustomNavItem{}
	}
	return items
}
func (s *settingProvider) CustomProps(ctx context.Context) []types.CustomProps {
	raw := s.getString(ctx, "custom_props", "[]")
	var props []types.CustomProps
	if err := json.Unmarshal([]byte(raw), &props); err != nil {
		return []types.CustomProps{}
	}
	return props
}

func (s *settingProvider) License(ctx context.Context) string {
	return s.getString(ctx, "license", "")
}

func (s *settingProvider) AvatarProcess(ctx context.Context) *AvatarProcess {
	return &AvatarProcess{
		Path:        s.getString(ctx, "avatar_path", "avatar"),
		MaxFileSize: s.getInt64(ctx, "avatar_size", 4194304),
		MaxWidth:    s.getInt(ctx, "avatar_size_l", 200),
	}
}

func (s *settingProvider) Avatar(ctx context.Context) *Avatar {
	return &Avatar{
		Gravatar: s.getString(ctx, "gravatar_server", ""),
		Path:     s.getString(ctx, "avatar_path", "avatar"),
	}
}

func (s *settingProvider) FileViewers(ctx context.Context) []types.ViewerGroup {
	raw := s.getString(ctx, "file_viewers", "[]")
	var viewers []types.ViewerGroup
	if err := json.Unmarshal([]byte(raw), &viewers); err != nil {
		return []types.ViewerGroup{}
	}

	return viewers
}

func (s *settingProvider) AppSetting(ctx context.Context) *AppSetting {
	return &AppSetting{
		Promotion: s.getBoolean(ctx, "show_app_promotion", false),
	}
}

func (s *settingProvider) MaxParallelTransfer(ctx context.Context) int {
	return s.getInt(ctx, "max_parallel_transfer", 4)
}

func (s *settingProvider) ArchiveDownloadSessionTTL(ctx context.Context) int {
	return s.getInt(ctx, "archive_timeout", 20)
}

func (s *settingProvider) ViewerSessionTTL(ctx context.Context) int {
	return s.getInt(ctx, "viewer_session_timeout", 36000)
}

func (s *settingProvider) MapSetting(ctx context.Context) *MapSetting {
	return &MapSetting{
		Provider:       MapProvider(s.getString(ctx, "map_provider", "openstreetmap")),
		GoogleTileType: MapGoogleTileType(s.getString(ctx, "map_google_tile_type", "roadmap")),
		MapboxAK:       s.getString(ctx, "map_mapbox_ak", ""),
	}
}

func (s *settingProvider) MimeMapping(ctx context.Context) string {
	return s.getString(ctx, "mime_mapping", "{}")
}

func (s *settingProvider) Logo(ctx context.Context) *Logo {
	return &Logo{
		Normal: s.getString(ctx, "site_logo", "/static/img/logo.svg"),
		Light:  s.getString(ctx, "site_logo_light", "/static/img/logo_light.svg"),
	}
}

func (s *settingProvider) ExplorerFrontendSettings(ctx context.Context) *ExplorerFrontendSettings {
	return &ExplorerFrontendSettings{
		Icons: s.getString(ctx, "explorer_icons", "[]"),
	}
}

func (s *settingProvider) SearchCategoryQuery(ctx context.Context, category SearchCategory) string {
	return s.getString(ctx, fmt.Sprintf("explorer_category_%s_query", category), "")
}

func (s *settingProvider) Captcha(ctx context.Context) *Captcha {
	return &Captcha{
		Height:             s.getInt(ctx, "captcha_height", 60),
		Width:              s.getInt(ctx, "captcha_width", 240),
		Mode:               CaptchaMode(s.getInt(ctx, "captcha_mode", int(CaptchaModeNumberAlphabet))),
		ComplexOfNoiseText: s.getInt(ctx, "captcha_ComplexOfNoiseText", 0),
		ComplexOfNoiseDot:  s.getInt(ctx, "captcha_ComplexOfNoiseDot", 0),
		IsShowHollowLine:   s.getBoolean(ctx, "captcha_IsShowHollowLine", false),
		IsShowNoiseDot:     s.getBoolean(ctx, "captcha_IsShowNoiseDot", false),
		IsShowNoiseText:    s.getBoolean(ctx, "captcha_IsShowNoiseText", false),
		IsShowSlimeLine:    s.getBoolean(ctx, "captcha_IsShowSlimeLine", false),
		IsShowSineLine:     s.getBoolean(ctx, "captcha_IsShowSineLine", false),
		Length:             s.getInt(ctx, "captcha_CaptchaLen", 6),
	}
}

func (s *settingProvider) LegalDocuments(ctx context.Context) *LegalDocuments {
	return &LegalDocuments{
		PrivacyPolicy:  s.getString(ctx, "privacy_policy_url", ""),
		TermsOfService: s.getString(ctx, "tos_url", ""),
	}
}

func (s *settingProvider) FolderPropsCacheTTL(ctx context.Context) int {
	return s.getInt(ctx, "folder_props_timeout", 300)
}

func (s *settingProvider) EmojiPresets(ctx context.Context) string {
	return s.getString(ctx, "emojis", "{}")
}

func (s *settingProvider) Theme(ctx context.Context) *Theme {
	return &Theme{
		Themes:       s.getString(ctx, "theme_options", "{}"),
		DefaultTheme: s.getString(ctx, "defaultTheme", ""),
	}
}

func (s *settingProvider) Cron(ctx context.Context, t CronType) string {
	return s.getString(ctx, "cron_"+string(t), "@hourly")
}

func (s *settingProvider) BuiltinThumbGeneratorEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_builtin_enabled", true)
}

func (s *settingProvider) BuiltinThumbMaxSize(ctx context.Context) int64 {
	return s.getInt64(ctx, "thumb_builtin_max_size", 78643200)
}

func (s *settingProvider) MusicCoverThumbGeneratorEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_music_cover_enabled", true)
}

func (s *settingProvider) MusicCoverThumbMaxSize(ctx context.Context) int64 {
	return s.getInt64(ctx, "thumb_music_cover_max_size", 1073741824)
}

func (s *settingProvider) MusicCoverThumbExts(ctx context.Context) []string {
	return s.getStringList(ctx, "thumb_music_cover_exts", []string{})
}

func (s *settingProvider) FFMpegPath(ctx context.Context) string {
	return s.getString(ctx, "thumb_ffmpeg_path", "ffmpeg")
}

func (s *settingProvider) FFMpegThumbGeneratorEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_ffmpeg_enabled", false)
}

func (s *settingProvider) FFMpegThumbExts(ctx context.Context) []string {
	return s.getStringList(ctx, "thumb_ffmpeg_exts", []string{})
}

func (s *settingProvider) FFMpegThumbSeek(ctx context.Context) string {
	return s.getString(ctx, "thumb_ffmpeg_seek", "00:00:01.00")
}

func (s *settingProvider) FFMpegExtraArgs(ctx context.Context) string {
	return s.getString(ctx, "thumb_ffmpeg_extra_args", "")
}

func (s *settingProvider) FFMpegThumbMaxSize(ctx context.Context) int64 {
	return s.getInt64(ctx, "thumb_ffmpeg_max_size", 10737418240)
}

func (s *settingProvider) VipsThumbGeneratorEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_vips_enabled", false)
}

func (s *settingProvider) VipsThumbMaxSize(ctx context.Context) int64 {
	return s.getInt64(ctx, "thumb_vips_max_size", 78643200)
}

func (s *settingProvider) VipsThumbExts(ctx context.Context) []string {
	return s.getStringList(ctx, "thumb_vips_exts", []string{})
}

func (s *settingProvider) VipsPath(ctx context.Context) string {
	return s.getString(ctx, "thumb_vips_path", "vips")
}

func (s *settingProvider) LibRawThumbGeneratorEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_libraw_enabled", false)
}

func (s *settingProvider) LibRawThumbMaxSize(ctx context.Context) int64 {
	return s.getInt64(ctx, "thumb_libraw_max_size", 78643200)
}

func (s *settingProvider) LibRawThumbExts(ctx context.Context) []string {
	return s.getStringList(ctx, "thumb_libraw_exts", []string{})
}

func (s *settingProvider) LibRawThumbPath(ctx context.Context) string {
	return s.getString(ctx, "thumb_libraw_path", "simple_dcraw")
}

func (s *settingProvider) LibreOfficeThumbGeneratorEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_libreoffice_enabled", false)
}

func (s *settingProvider) LibreOfficeThumbMaxSize(ctx context.Context) int64 {
	return s.getInt64(ctx, "thumb_libreoffice_max_size", 78643200)
}

func (s *settingProvider) LibreOfficePath(ctx context.Context) string {
	return s.getString(ctx, "thumb_libreoffice_path", "soffice")
}

func (s *settingProvider) LibreOfficeThumbExts(ctx context.Context) []string {
	return s.getStringList(ctx, "thumb_libreoffice_exts", []string{})
}

func (s *settingProvider) ThumbSize(ctx context.Context) (int, int) {
	return s.getInt(ctx, "thumb_width", 400), s.getInt(ctx, "thumb_height", 300)
}

func (s *settingProvider) ThumbEncode(ctx context.Context) *ThumbEncode {
	return &ThumbEncode{
		Format:  s.getString(ctx, "thumb_encode_method", "jpg"),
		Quality: s.getInt(ctx, "thumb_encode_quality", 85),
	}
}

func (s *settingProvider) ThumbEntitySuffix(ctx context.Context) string {
	return s.getString(ctx, "thumb_entity_suffix", "{blob_path}/{blob_name}._thumb")
}

func (s *settingProvider) ThumbSlaveSidecarSuffix(ctx context.Context) string {
	return s.getString(ctx, "thumb_slave_sidecar_suffix", "._thumb_sidecar")
}

func (s *settingProvider) ThumbGCAfterGen(ctx context.Context) bool {
	return s.getBoolean(ctx, "thumb_gc_after_gen", false)
}

func (s *settingProvider) TempPath(ctx context.Context) string {
	return s.getString(ctx, "temp_path", "temp")
}

func (s *settingProvider) MediaMetaFFProbePath(ctx context.Context) string {
	return s.getString(ctx, "media_meta_ffprobe_path", "ffprobe")
}

func (s *settingProvider) MediaMetaFFProbeSizeLimit(ctx context.Context) (int64, int64) {
	return s.getInt64(ctx, "media_meta_ffprobe_size_local", 0), s.getInt64(ctx, "media_meta_ffprobe_size_remote", 0)
}

func (s *settingProvider) MediaMetaFFProbeEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "media_meta_ffprobe", true)
}

func (s *settingProvider) MediaMetaMusicSizeLimit(ctx context.Context) (int64, int64) {
	return s.getInt64(ctx, "media_meta_music_size_local", 0), s.getInt64(ctx, "media_meta_music_size_remote", 0)
}

func (s *settingProvider) MediaMetaMusicEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "media_meta_music", true)
}

func (s *settingProvider) MediaMetaExifBruteForce(ctx context.Context) bool {
	return s.getBoolean(ctx, "media_meta_exif_brute_force", false)
}

func (s *settingProvider) MediaMetaExifSizeLimit(ctx context.Context) (int64, int64) {
	return s.getInt64(ctx, "media_meta_exif_size_local", 0), s.getInt64(ctx, "media_meta_exif_size_remote", 0)
}

func (s *settingProvider) MediaMetaExifEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "media_meta_exif", true)
}

func (s *settingProvider) MediaMetaEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "media_meta", true)
}

func (s *settingProvider) MediaMetaGeocodingEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "media_meta_geocoding", false)
}

func (s *settingProvider) MediaMetaGeocodingMapboxAK(ctx context.Context) string {
	return s.getString(ctx, "media_meta_geocoding_mapbox_ak", "")
}

func (s *settingProvider) PublicResourceMaxAge(ctx context.Context) int {
	return s.getInt(ctx, "public_resource_maxage", 0)
}

func (s *settingProvider) EntityUrlCacheMargin(ctx context.Context) int {
	return s.getInt(ctx, "entity_url_cache_margin", 600)
}

func (s *settingProvider) EntityUrlValidDuration(ctx context.Context) time.Duration {
	return time.Duration(s.getInt(ctx, "entity_url_default_ttl", 3600)) * time.Second
}

func (s *settingProvider) Queue(ctx context.Context, queueType QueueType) *QueueSetting {
	queueTypeStr := string(queueType)
	return &QueueSetting{
		WorkerNum:          s.getInt(ctx, "queue_"+queueTypeStr+"_worker_num,", 15),
		MaxExecution:       time.Duration(s.getInt(ctx, "queue_"+queueTypeStr+"_max_execution", 86400)) * time.Second,
		BackoffFactor:      s.getFloat64(ctx, "queue_"+queueTypeStr+"_backoff_factor", 4),
		BackoffMaxDuration: time.Duration(s.getInt(ctx, "queue_"+queueTypeStr+"_backoff_max_duration", 3600)) * time.Second,
		MaxRetry:           s.getInt(ctx, "queue_"+queueTypeStr+"_max_retry", 5),
		RetryDelay:         time.Duration(s.getInt(ctx, "queue_"+queueTypeStr+"_retry_delay", 5)) * time.Second,
	}
}

func (s *settingProvider) UseChunkBuffer(ctx context.Context) bool {
	return s.getBoolean(ctx, "use_temp_chunk_buffer", true)
}

func (s *settingProvider) ChunkRetryLimit(ctx context.Context) int {
	return s.getInt(ctx, "chunk_retries", 3)
}

func (s *settingProvider) SlaveRequestSignTTL(ctx context.Context) int {
	return s.getInt(ctx, "slave_api_timeout", 60)
}

func (s *settingProvider) MaxOnlineEditSize(ctx context.Context) int64 {
	return int64(s.getInt(ctx, "maxEditSize", 52428800))
}

func (s *settingProvider) UploadSessionTTL(ctx context.Context) time.Duration {
	return time.Duration(s.getInt(ctx, "upload_session_timeout", 86400)) * time.Second
}

func (s *settingProvider) MaxBatchedFile(ctx context.Context) int {
	return s.getInt(ctx, "max_batched_file", 3000)
}

func (s *settingProvider) DBFS(ctx context.Context) *DBFS {
	return &DBFS{
		UseCursorPagination:        s.getBoolean(ctx, "use_cursor_pagination", true),
		MaxPageSize:                s.getInt(ctx, "max_page_size", 2000),
		MaxRecursiveSearchedFolder: s.getInt(ctx, "max_recursive_searched_folder", 65535),
		UseSSEForSearch:            s.getBoolean(ctx, "use_sse_for_search", false),
	}
}

func (s *settingProvider) HashIDSalt(ctx context.Context) string {
	return s.getString(ctx, "hash_id_salt", "")
}

func (s *settingProvider) TokenAuth(ctx context.Context) *TokenAuth {
	return &TokenAuth{
		AccessTokenTTL:  time.Duration(s.getInt(ctx, "access_token_ttl", 3600)) * time.Second,
		RefreshTokenTTL: time.Duration(s.getInt(ctx, "refresh_token_ttl", 15552000)) * time.Second,
	}
}

func (s *settingProvider) ResetEmailTemplate(ctx context.Context) []EmailTemplate {
	src := s.getString(ctx, "mail_reset_template", "[]")
	var templates []EmailTemplate
	if err := json.Unmarshal([]byte(src), &templates); err != nil {
		return []EmailTemplate{}
	}

	return templates
}

func (s *settingProvider) ActivationEmailTemplate(ctx context.Context) []EmailTemplate {
	src := s.getString(ctx, "mail_activation_template", "[]")
	var templates []EmailTemplate
	if err := json.Unmarshal([]byte(src), &templates); err != nil {
		return []EmailTemplate{}
	}

	return templates
}

func (s *settingProvider) SecretKey(ctx context.Context) string {
	return s.getString(ctx, "secret_key", "")
}

func (s *settingProvider) AllSiteURLs(ctx context.Context) []*url.URL {
	rawUrls := s.getStringList(ctx, "siteURL", []string{"http://localhost"})
	if len(rawUrls) == 0 {
		rawUrls = []string{"http://localhost"}
	}

	urls := make([]*url.URL, 0, len(rawUrls))
	for _, u := range rawUrls {
		parsedURL, err := url.Parse(u)
		if err != nil {
			continue
		}
		urls = append(urls, parsedURL)
	}
	return urls
}

func (s *settingProvider) SiteURL(ctx context.Context) *url.URL {
	rawUrls := s.getStringList(ctx, "siteURL", []string{"http://localhost"})
	if len(rawUrls) == 0 {
		rawUrls = []string{"http://localhost"}
	}

	urls := make([]*url.URL, 0, len(rawUrls))
	for _, u := range rawUrls {
		parsedURL, err := url.Parse(u)
		if err != nil {
			continue
		}
		urls = append(urls, parsedURL)
	}

	reqInfo := requestinfo.RequestInfoFromContext(ctx)
	_, useFirst := ctx.Value(UseFirstSiteUrlCtxKey{}).(bool)
	if !useFirst && reqInfo != nil && reqInfo.Host != "" {
		for _, u := range urls {
			if (u.Host) == reqInfo.Host {
				return u
			}
		}
	}

	return urls[0]
}

func (s *settingProvider) SMTP(ctx context.Context) *SMTP {
	return &SMTP{
		FromName:        s.getString(ctx, "fromName", ""),
		From:            s.getString(ctx, "fromAdress", ""),
		Host:            s.getString(ctx, "smtpHost", ""),
		ReplyTo:         s.getString(ctx, "replyTo", ""),
		User:            s.getString(ctx, "smtpUser", ""),
		Password:        s.getString(ctx, "smtpPass", ""),
		ForceEncryption: s.getBoolean(ctx, "smtpEncryption", false),
		Port:            s.getInt(ctx, "smtpPort", 25),
		Keepalive:       s.getInt(ctx, "mail_keepalive", 30),
	}
}

func (s *settingProvider) DefaultGroup(ctx context.Context) int {
	return s.getInt(ctx, "default_group", 2)
}

func (s *settingProvider) EmailActivationEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "email_active", false)
}

func (s *settingProvider) TcCaptcha(ctx context.Context) *TcCaptcha {
	return &TcCaptcha{
		AppID:        s.getString(ctx, "captcha_TCaptcha_CaptchaAppId", ""),
		AppSecretKey: s.getString(ctx, "captcha_TCaptcha_AppSecretKey", ""),
		SecretID:     s.getString(ctx, "captcha_TCaptcha_SecretId", ""),
		SecretKey:    s.getString(ctx, "captcha_TCaptcha_SecretKey", ""),
	}
}

func (s *settingProvider) TurnstileCaptcha(ctx context.Context) *Turnstile {
	return &Turnstile{
		Secret: s.getString(ctx, "captcha_turnstile_site_secret", ""),
		Key:    s.getString(ctx, "captcha_turnstile_site_key", ""),
	}
}

func (s *settingProvider) CapCaptcha(ctx context.Context) *Cap {
	return &Cap{
		InstanceURL: s.getString(ctx, "captcha_cap_instance_url", ""),
		SiteKey:     s.getString(ctx, "captcha_cap_site_key", ""),
		SecretKey:   s.getString(ctx, "captcha_cap_secret_key", ""),
		AssetServer: s.getString(ctx, "captcha_cap_asset_server", "jsdelivr"),
	}
}

func (s *settingProvider) ReCaptcha(ctx context.Context) *ReCaptcha {
	return &ReCaptcha{
		Secret: s.getString(ctx, "captcha_ReCaptchaSecret", ""),
		Key:    s.getString(ctx, "captcha_ReCaptchaKey", ""),
	}
}

func (s *settingProvider) CaptchaType(ctx context.Context) CaptchaType {
	return CaptchaType(s.getString(ctx, "captcha_type", string(CaptchaNormal)))
}

func (s *settingProvider) RegCaptchaEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "reg_captcha", false)
}

func (s *settingProvider) LoginCaptchaEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "login_captcha", false)
}

func (s *settingProvider) ForgotPasswordCaptchaEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "forget_captcha", false)
}

func (s *settingProvider) AuthnEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "authn_enabled", false)
}

func (s *settingProvider) RegisterEnabled(ctx context.Context) bool {
	return s.getBoolean(ctx, "register_enabled", false)
}

func (s *settingProvider) SiteBasic(ctx context.Context) *SiteBasic {
	return &SiteBasic{
		Name:        s.getString(ctx, "siteName", ""),
		Title:       s.getString(ctx, "siteTitle", ""),
		ID:          s.getString(ctx, "siteID", ""),
		Description: s.getString(ctx, "siteDes", ""),
		Script:      s.getString(ctx, "siteScript", ""),
	}
}

func (s *settingProvider) PWA(ctx context.Context) *PWASetting {
	return &PWASetting{
		SmallIcon:       s.getString(ctx, "pwa_small_icon", ""),
		MediumIcon:      s.getString(ctx, "pwa_medium_icon", ""),
		LargeIcon:       s.getString(ctx, "pwa_large_icon", ""),
		Display:         s.getString(ctx, "pwa_display", ""),
		ThemeColor:      s.getString(ctx, "pwa_theme_color", ""),
		BackgroundColor: s.getString(ctx, "pwa_background_color", ""),
	}
}

func IsTrueValue(val string) bool {
	return val == "1" || val == "true"
}

func (s *settingProvider) getInt(ctx context.Context, name string, defaultVal int) int {
	val := s.adapterChain.Get(ctx, name, defaultVal)
	if intVal, ok := val.(int); ok {
		return intVal
	}

	strVal := val.(string)
	if intVal, err := strconv.Atoi(strVal); err == nil {
		return intVal
	}

	return defaultVal
}

func (s *settingProvider) getInt64(ctx context.Context, name string, defaultVal int64) int64 {
	val := s.adapterChain.Get(ctx, name, defaultVal)
	if intVal, ok := val.(int64); ok {
		return intVal
	}

	strVal := val.(string)
	if intVal, err := strconv.ParseInt(strVal, 10, 64); err == nil {
		return intVal
	}

	return defaultVal
}

func (s *settingProvider) getFloat64(ctx context.Context, name string, defaultVal float64) float64 {
	val := s.adapterChain.Get(ctx, name, defaultVal)
	if intVal, ok := val.(float64); ok {
		return intVal
	}

	strVal := val.(string)
	if intVal, err := strconv.ParseFloat(strVal, 64); err == nil {
		return intVal
	}

	return defaultVal
}

func (s *settingProvider) getBoolean(ctx context.Context, name string, defaultVal bool) bool {
	val := s.adapterChain.Get(ctx, name, defaultVal)
	if intVal, ok := val.(bool); ok {
		return intVal
	}

	strVal := val.(string)
	return IsTrueValue(strVal)
}

func (s *settingProvider) getString(ctx context.Context, name string, defaultVal string) string {
	val := s.adapterChain.Get(ctx, name, defaultVal)
	return val.(string)
}

func (s *settingProvider) getStringList(ctx context.Context, name string, defaultVal []string) []string {
	res, _ := s.getStringListRaw(ctx, name, defaultVal)
	return res
}

func (s *settingProvider) getStringListRaw(ctx context.Context, name string, defaultVal []string) ([]string, string) {
	val := s.getString(ctx, name, stringListDefault)
	if val == stringListDefault {
		return defaultVal, val
	}

	return strings.Split(val, stringListDefaultSeparator), val
}

func (s *settingProvider) getBoolSet(ctx context.Context, name string) *boolset.BooleanSet {
	val := s.getString(ctx, name, "")
	if val == "" {
		return defaultBoolSet
	}

	res, err := boolset.FromString(val)
	if err != nil {
		return defaultBoolSet
	}

	return res
}

func UseFirstSiteUrl(ctx context.Context) context.Context {
	return context.WithValue(ctx, UseFirstSiteUrlCtxKey{}, true)
}
