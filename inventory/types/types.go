package types

import (
	"time"
)

// UserSetting 用户其他配置
type (
	UserSetting struct {
		ProfileOff          bool                     `json:"profile_off,omitempty"`
		PreferredTheme      string                   `json:"preferred_theme,omitempty"`
		VersionRetention    bool                     `json:"version_retention,omitempty"`
		VersionRetentionExt []string                 `json:"version_retention_ext,omitempty"`
		VersionRetentionMax int                      `json:"version_retention_max,omitempty"`
		Pined               []PinedFile              `json:"pined,omitempty"`
		Language            string                   `json:"email_language,omitempty"`
		DisableViewSync     bool                     `json:"disable_view_sync,omitempty"`
		FsViewMap           map[string]ExplorerView  `json:"fs_view_map,omitempty"`
		ShareLinksInProfile ShareLinksInProfileLevel `json:"share_links_in_profile,omitempty"`
		// ShareDefaultPrivate overrides the site-wide private-share default
		// for this user. nil means inherit the site default.
		ShareDefaultPrivate *bool `json:"share_default_private,omitempty"`
		// PreferredViewers maps file extensions (without dot, lowercase) to
		// viewer IDs chosen via "always open with". Synced server-side so the
		// preference follows the user across devices.
		PreferredViewers map[string]string `json:"preferred_viewers,omitempty"`
		// TrashRetention overrides the group's trash retention in seconds.
		// 0 means inherit the group setting. Applied when a file is moved to
		// trash — already-trashed files keep their original expiry.
		TrashRetention int `json:"trash_retention,omitempty"`
		// PreferredPolicy is the user's default storage policy, chosen from
		// the policies allowed for their group. 0 means the group default.
		PreferredPolicy int `json:"preferred_policy,omitempty"`
		// DismissedAnnouncement stores the announcement content the user last
		// dismissed. When the admin edits the announcement it differs from
		// this value and the modal is shown again.
		DismissedAnnouncement string `json:"dismissed_announcement,omitempty"`
	}

	// LBPolicyRef binds a child storage policy to a load_balance policy with
	// a selection weight. Weight 0 counts as 1.
	LBPolicyRef struct {
		PolicyID int `json:"policy"`
		Weight   int `json:"weight,omitempty"`
	}

	ShareLinksInProfileLevel string

	PinedFile struct {
		Uri  string `json:"uri"`
		Name string `json:"name,omitempty"`
	}

	// GroupSetting 用户组其他配置
	GroupSetting struct {
		CompressSize          int64                  `json:"compress_size,omitempty"` // 可压缩大小
		DecompressSize        int64                  `json:"decompress_size,omitempty"`
		RemoteDownloadOptions map[string]interface{} `json:"remote_download_options,omitempty"` // 离线下载用户组配置
		SourceBatchSize       int                    `json:"source_batch,omitempty"`
		Aria2BatchSize        int                    `json:"aria2_batch,omitempty"`
		// Aria2TaskLimit caps concurrent active remote-download tasks per user.
		Aria2TaskLimit int `json:"aria2_task_limit,omitempty"`
		// Aria2MaxFileSize caps the total byte size of a single download task.
		Aria2MaxFileSize int64 `json:"aria2_max_file_size,omitempty"`
		MaxWalkedFiles   int   `json:"max_walked_files,omitempty"`
		TrashRetention   int   `json:"trash_retention,omitempty"`
		RedirectedSource bool  `json:"redirected_source,omitempty"`
		// LoginIPWhitelist restricts sign-in to the given IPs/CIDR ranges.
		// Empty means no restriction.
		LoginIPWhitelist []string `json:"login_ip_whitelist,omitempty"`
		// DefaultPinned is a list of share entity IDs seeded as share
		// shortcuts for members of this group on file-system init.
		DefaultPinned []int `json:"default_pinned,omitempty"`
		// AllowedNodes restricts which nodes this group's tasks may run on.
		// Empty means all nodes are eligible.
		AllowedNodes []int `json:"allowed_nodes,omitempty"`
		// AllowSelectNode lets members pick a preferred node when creating
		// tasks (remote download, archive create/extract).
		AllowSelectNode bool `json:"allow_select_node,omitempty"`
	}

	// PolicySetting 非公有的存储策略属性
	PolicySetting struct {
		// Upyun访问Token
		Token string `json:"token"`
		// 允许的文件扩展名
		FileType []string `json:"file_type"`
		// IsFileTypeDenyList Whether above list is a deny list.
		IsFileTypeDenyList bool `json:"is_file_type_deny_list,omitempty"`
		// FileRegexp 文件扩展名正则表达式
		NameRegexp string `json:"file_regexp,omitempty"`
		// IsNameRegexp Whether above regexp is a deny list.
		IsNameRegexpDenyList bool `json:"is_name_regexp_deny_list,omitempty"`
		// AllowNativeName permits characters only illegal on Windows
		// filesystems (:*?"<>|) in file names. Path separators and dot-names
		// stay illegal (#3065).
		AllowNativeName bool `json:"allow_native_name,omitempty"`
		// OauthRedirect Oauth 重定向地址
		OauthRedirect string `json:"od_redirect,omitempty"`
		// CustomProxy whether to use custom-proxy to get file content
		CustomProxy bool `json:"custom_proxy,omitempty"`
		// ProxyServer 反代地址
		ProxyServer string `json:"proxy_server,omitempty"`
		// InternalProxy whether to use Cloudreve internal proxy to get file content
		InternalProxy bool `json:"internal_proxy,omitempty"`
		// OdDriver OneDrive 驱动器定位符
		OdDriver string `json:"od_driver,omitempty"`
		// Region 区域代码
		Region string `json:"region,omitempty"`
		// ServerSideEndpoint 服务端请求使用的 Endpoint，为空时使用 Policy.Server 字段
		ServerSideEndpoint string `json:"server_side_endpoint,omitempty"`
		// 分片上传的分片大小
		ChunkSize int64 `json:"chunk_size,omitempty"`
		// 每秒对存储端的 API 请求上限
		TPSLimit float64 `json:"tps_limit,omitempty"`
		// 每秒 API 请求爆发上限
		TPSLimitBurst int `json:"tps_limit_burst,omitempty"`
		// Set this to `true` to force the request to use path-style addressing,
		// i.e., `http://s3.amazonaws.com/BUCKET/KEY `
		S3ForcePathStyle bool `json:"s3_path_style"`
		// File extensions that support thumbnail generation using native policy API.
		ThumbExts []string `json:"thumb_exts,omitempty"`
		// Whether to support all file extensions for thumbnail generation.
		ThumbSupportAllExts bool `json:"thumb_support_all_exts,omitempty"`
		// ThumbMaxSize indicates the maximum allowed size of a thumbnail. 0 indicates that no limit is set.
		ThumbMaxSize int64 `json:"thumb_max_size,omitempty"`
		// Whether to upload file through server's relay.
		Relay bool `json:"relay,omitempty"`
		// Whether to pre allocate space for file before upload in physical disk.
		PreAllocate bool `json:"pre_allocate,omitempty"`
		// MediaMetaExts file extensions that support media meta generation using native policy API.
		MediaMetaExts []string `json:"media_meta_exts,omitempty"`
		// MediaMetaGeneratorProxy whether to use local proxy to generate media meta.
		MediaMetaGeneratorProxy bool `json:"media_meta_generator_proxy,omitempty"`
		// ThumbGeneratorProxy whether to use local proxy to generate thumbnail.
		ThumbGeneratorProxy bool `json:"thumb_generator_proxy,omitempty"`
		// NativeMediaProcessing whether to use native media processing API from storage provider.
		NativeMediaProcessing bool `json:"native_media_processing"`
		// S3DeleteBatchSize the number of objects to delete in each batch.
		S3DeleteBatchSize int `json:"s3_delete_batch_size,omitempty"`
		// StreamSaver whether to use stream saver to download file in Web.
		StreamSaver bool `json:"stream_saver,omitempty"`
		// UseCname whether to use CNAME for endpoint (OSS).
		UseCname bool `json:"use_cname,omitempty"`
		// LBPolicies binds weighted child policies to a load_balance policy.
		// Children must be concrete (non-load_balance) policies.
		LBPolicies []LBPolicyRef `json:"lb_policies,omitempty"`
		// CDN domain does not need to be signed.
		SourceAuth bool `json:"source_auth,omitempty"`
		// QiniuUploadCdn whether to use CDN for Qiniu upload.
		QiniuUploadCdn bool `json:"qiniu_upload_cdn,omitempty"`
		// ChunkConcurrency the number of chunks to upload concurrently.
		ChunkConcurrency int `json:"chunk_concurrency,omitempty"`
		// Whether to enable file encryption.
		Encryption bool `json:"encryption,omitempty"`
	}

	FileType         int
	EntityType       int
	GroupPermission  int
	FilePermission   int
	DavAccountOption int
	NodeCapability   int
	AclPermission    int

	NodeSetting struct {
		Provider            DownloaderProvider `json:"provider,omitempty"`
		*QBittorrentSetting `json:"qbittorrent,omitempty"`
		*Aria2Setting       `json:"aria2,omitempty"`
		*YtdlpSetting       `json:"ytdlp,omitempty"`
		// 下载监控间隔
		Interval       int  `json:"interval,omitempty"`
		WaitForSeeding bool `json:"wait_for_seeding,omitempty"`
		// URLValidation controls SSRF policy applied to user-supplied URLs
		// fetched by this node's downloader. nil means the secure default
		// (validation on, no extra allowlist) — existing nodes upgraded in
		// place stay protected without admin action.
		URLValidation *URLValidationSetting `json:"url_validation,omitempty"`
	}

	URLValidationSetting struct {
		// Disabled turns the SSRF check off entirely on this node. Only set
		// this when the downloader runs in a network segment that cannot
		// reach any internal asset (e.g. dedicated egress namespace).
		Disabled bool `json:"disabled,omitempty"`
		// AllowedHosts is a list of hostnames or IP literals that bypass all
		// checks. Exact, case-insensitive match against url.Hostname().
		AllowedHosts []string `json:"allowed_hosts,omitempty"`
		// AllowedCIDRs is a list of CIDR blocks (IPv4 or IPv6) whose IPs are
		// treated as safe even if they would otherwise be rejected (private,
		// link-local, etc.). Use this to whitelist a LAN range like
		// "192.168.10.0/24" so a local NAS can be fetched.
		AllowedCIDRs []string `json:"allowed_cidrs,omitempty"`
	}

	DownloaderProvider string

	QBittorrentSetting struct {
		Server   string         `json:"server,omitempty"`
		User     string         `json:"user,omitempty"`
		Password string         `json:"password,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
		TempPath string         `json:"temp_path,omitempty"`
	}

	Aria2Setting struct {
		Server   string         `json:"server,omitempty"`
		Token    string         `json:"token,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
		TempPath string         `json:"temp_path,omitempty"`
	}

	// YtdlpSetting configures a yt-dlp CLI downloader. Options keys become
	// "--<key> <value>" CLI flags verbatim (bool true -> flag only).
	YtdlpSetting struct {
		Binary   string         `json:"binary,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
		TempPath string         `json:"temp_path,omitempty"`
	}

	TaskPublicState struct {
		Error            string          `json:"error,omitempty"`
		ErrorHistory     []string        `json:"error_history,omitempty"`
		ExecutedDuration time.Duration   `json:"executed_duration,omitempty"`
		RetryCount       int             `json:"retry_count,omitempty"`
		ResumeTime       int64           `json:"resume_time,omitempty"`
		SlaveTaskProps   *SlaveTaskProps `json:"slave_task_props,omitempty"`
	}

	SlaveTaskProps struct {
		NodeID            int    `json:"node_id,omitempty"`
		MasterSiteURl     string `json:"master_site_u_rl,omitempty"`
		MasterSiteID      string `json:"master_site_id,omitempty"`
		MasterSiteVersion string `json:"master_site_version,omitempty"`
	}

	EntityProps struct {
		UnlinkOnly      bool             `json:"unlink_only,omitempty"`
		EncryptMetadata *EncryptMetadata `json:"encrypt_metadata,omitempty"`
		// RecycleFailCount tracks consecutive driver-delete failures during
		// entity recycling. Entities reaching the threshold are force-removed
		// from the DB so one un-deletable blob cannot stall the sweep forever.
		RecycleFailCount int `json:"recycle_fail_count,omitempty"`
	}

	Cipher string

	EncryptMetadata struct {
		Algorithm    Cipher `json:"algorithm"`
		Key          []byte `json:"key"`
		KeyPlainText []byte `json:"key_plain_text,omitempty"`
		IV           []byte `json:"iv"`
	}

	DavAccountProps struct {
	}

	PolicyType string

	FileProps struct {
		View *ExplorerView `json:"view,omitempty"`
	}

	ExplorerView struct {
		PageSize       int              `json:"page_size" binding:"min=50"`
		Order          string           `json:"order,omitempty" binding:"max=255"`
		OrderDirection string           `json:"order_direction,omitempty" binding:"eq=asc|eq=desc"`
		View           string           `json:"view,omitempty" binding:"eq=list|eq=grid|eq=gallery"`
		Thumbnail      bool             `json:"thumbnail,omitempty"`
		GalleryWidth   int              `json:"gallery_width,omitempty" binding:"min=50,max=500"`
		Columns        []ListViewColumn `json:"columns,omitempty" binding:"max=1000"`
	}

	ListViewColumn struct {
		Type  int             `json:"type" binding:"min=0"`
		Width *int            `json:"width,omitempty"`
		Props *ColumTypeProps `json:"props,omitempty"`
	}

	ColumTypeProps struct {
		MetadataKey   string `json:"metadata_key,omitempty" binding:"max=255"`
		CustomPropsID string `json:"custom_props_id,omitempty" binding:"max=255"`
	}

	ShareProps struct {
		// Whether to share view setting from owner
		ShareView bool `json:"share_view,omitempty"`
		// Whether to automatically show readme file in share view
		ShowReadMe bool `json:"show_read_me,omitempty"`
		// Whether to hide the readme file itself from the share listing
		// (only meaningful together with ShowReadMe)
		HideReadMe bool `json:"hide_readme,omitempty"`
		// Whether share visitors can upload new files into the shared folder
		AllowUpload bool `json:"allow_upload,omitempty"`
		// Whether share visitors can rename, move and delete files (implies upload)
		AllowEdit bool `json:"allow_edit,omitempty"`
		// Whether share visitors can browse and preview but not download
		PreviewOnly bool `json:"preview_only,omitempty"`
		// Whether share visitors can upload but cannot list or download (drop box)
		UploadOnly bool `json:"upload_only,omitempty"`
		// Owner-defined note/alias for identifying the share in My Shares;
		// never exposed to share visitors (#3570).
		Note string `json:"note,omitempty"`
	}

	OAuthClientProps struct {
		Description     string `json:"description,omitempty"`
		Icon            string `json:"icon,omitempty"`
		RefreshTokenTTL int64  `json:"refresh_token_ttl,omitempty"` // in seconds, 0 means default
	}

	FileTypeIconSetting struct {
		Exts      []string `json:"exts"`
		Icon      string   `json:"icon,omitempty"`
		Color     string   `json:"color,omitempty"`
		ColorDark string   `json:"color_dark,omitempty"`
		Img       string   `json:"img,omitempty"`
	}
)

const (
	GroupPermissionIsAdmin = GroupPermission(iota)
	GroupPermissionIsAnonymous
	GroupPermissionShare
	GroupPermissionWebDAV
	GroupPermissionArchiveDownload
	GroupPermissionArchiveTask
	GroupPermissionWebDAVProxy
	GroupPermissionShareDownload
	// GroupPermissionShareFree lets members access paid shares without
	// purchasing (e.g. staff or VIP groups).
	GroupPermissionShareFree
	GroupPermissionRemoteDownload
	GroupPermission_CommunityPlaceholder2
	GroupPermissionRedirectedSource // not used
	GroupPermissionAdvanceDelete
	GroupPermission_CommunityPlaceholder3
	GroupPermission_CommunityPlaceholder4
	// GroupPermissionSetExplicitUser allows members to manage per-file ACL
	// entries (Permissions dialog) on files they own.
	GroupPermissionSetExplicitUser
	GroupPermissionIgnoreFileOwnership // not used
	GroupPermissionUniqueRedirectDirectLink
	// GroupPermissionWebDAVReadOnly restricts the group's WebDAV access to
	// read operations — write methods (PUT, MKCOL, DELETE, COPY, MOVE, LOCK,
	// PROPPATCH) are rejected even if the group's WebDAV access is enabled.
	GroupPermissionWebDAVReadOnly
	// Delegated admin section permissions. A group with any of these bits —
	// but without GroupPermissionIsAdmin — is a delegated administrator that
	// can only access the corresponding admin sections. GroupPermissionIsAdmin
	// implies all sections.
	GroupPermissionAdminUsers
	GroupPermissionAdminGroups
	GroupPermissionAdminFiles
	GroupPermissionAdminShares
	GroupPermissionAdminStorage
	GroupPermissionAdminQueue
	GroupPermissionAdminSettings
	GroupPermissionAdminPayment
	GroupPermissionAdminEvents
	GroupPermissionAdminReports
	// GroupPermissionShareSell allows members to set a points price on
	// shares they create (paid shares).
	GroupPermissionShareSell
	// GroupPermissionSharePublicList allows members to list their shares in
	// the public share directory exposed to anonymous visitors.
	GroupPermissionSharePublicList
)

// AclPermission is a bit position in an ACL entry's permission bitmask.
// Read grants listing/download, Create grants uploads and new entries,
// Update grants rename/metadata/content changes, Delete grants removal.
const (
	AclPermRead AclPermission = iota
	AclPermCreate
	AclPermUpdate
	AclPermDelete
)

// AclPermissionList maps each ACL bit to its API-facing key.
var AclPermissionList = map[AclPermission]string{
	AclPermRead:   "read",
	AclPermCreate: "create",
	AclPermUpdate: "update",
	AclPermDelete: "delete",
}

// DelegatedAdminPermissions lists every per-section admin permission bit.
// GroupPermissionIsAdmin implies all of them.
func DelegatedAdminPermissions() []GroupPermission {
	return []GroupPermission{
		GroupPermissionAdminUsers,
		GroupPermissionAdminGroups,
		GroupPermissionAdminFiles,
		GroupPermissionAdminShares,
		GroupPermissionAdminStorage,
		GroupPermissionAdminQueue,
		GroupPermissionAdminSettings,
		GroupPermissionAdminPayment,
		GroupPermissionAdminEvents,
		GroupPermissionAdminReports,
	}
}

// AdminPermissionBits returns all admin-capable bits, including IsAdmin.
func AdminPermissionBits() []GroupPermission {
	return append([]GroupPermission{GroupPermissionIsAdmin}, DelegatedAdminPermissions()...)
}

const (
	NodeCapabilityNone NodeCapability = iota
	NodeCapabilityCreateArchive
	NodeCapabilityExtractArchive
	NodeCapabilityRemoteDownload
	NodeCapability_CommunityPlaceholder
)

const (
	FileTypeFile FileType = iota
	FileTypeFolder
)

const (
	EntityTypeVersion EntityType = iota
	EntityTypeThumbnail
	EntityTypeLivePhoto
)

func FileTypeFromString(s string) FileType {
	switch s {
	case "file":
		return FileTypeFile
	case "folder":
		return FileTypeFolder
	}
	return -1
}

const (
	DavAccountReadOnly DavAccountOption = iota
	DavAccountProxy
	DavAccountDisableSysFiles
)

const (
	PolicyTypeLocal  = "local"
	PolicyTypeQiniu  = "qiniu"
	PolicyTypeUpyun  = "upyun"
	PolicyTypeOss    = "oss"
	PolicyTypeCos    = "cos"
	PolicyTypeS3     = "s3"
	PolicyTypeKs3    = "ks3"
	PolicyTypeOd     = "onedrive"
	PolicyTypeRemote = "remote"
	PolicyTypeObs    = "obs"
	// PolicyTypeLoadBalance distributes uploads across weighted child
	// policies. It is resolved to a concrete child policy before use and
	// never reaches a storage driver.
	PolicyTypeLoadBalance = "load_balance"
)

const (
	DownloaderProviderAria2       = DownloaderProvider("aria2")
	DownloaderProviderQBittorrent = DownloaderProvider("qbittorrent")
	DownloaderProviderYtDlp       = DownloaderProvider("ytdlp")
)

type (
	ViewerAction string
	ViewerType   string
)

const (
	ViewerActionView = "view"
	ViewerActionEdit = "edit"

	ViewerTypeBuiltin = "builtin"
	ViewerTypeWopi    = "wopi"
	ViewerTypeCustom  = "custom"
)

type (
	Viewer struct {
		ID                      string                             `json:"id"`
		Type                    ViewerType                         `json:"type"`
		DisplayName             string                             `json:"display_name"`
		Exts                    []string                           `json:"exts"`
		Url                     string                             `json:"url,omitempty"`
		Icon                    string                             `json:"icon,omitempty"`
		WopiActions             map[string]map[ViewerAction]string `json:"wopi_actions,omitempty"`
		Props                   map[string]string                  `json:"props,omitempty"`
		MaxSize                 int64                              `json:"max_size,omitempty"`
		Disabled                bool                               `json:"disabled,omitempty"`
		Templates               []NewFileTemplate                  `json:"templates,omitempty"`
		Platform                string                             `json:"platform,omitempty"`
		RequiredGroupPermission []GroupPermission                  `json:"required_group_permission,omitempty"`
	}
	ViewerGroup struct {
		Viewers []Viewer `json:"viewers"`
	}

	DefaultViewerMapping map[string]string

	NewFileTemplate struct {
		Ext         string `json:"ext"`
		DisplayName string `json:"display_name"`
	}
)

type (
	CustomPropsType string
	CustomProps     struct {
		ID      string          `json:"id"`
		Name    string          `json:"name"`
		Type    CustomPropsType `json:"type"`
		Max     int             `json:"max,omitempty"`
		Min     int             `json:"min,omitempty"`
		Default string          `json:"default,omitempty"`
		Options []string        `json:"options,omitempty"`
		Icon    string          `json:"icon,omitempty"`
	}
)

const (
	CustomPropsTypeText        = "text"
	CustomPropsTypeNumber      = "number"
	CustomPropsTypeBoolean     = "boolean"
	CustomPropsTypeSelect      = "select"
	CustomPropsTypeMultiSelect = "multi_select"
	CustomPropsTypeLink        = "link"
	CustomPropsTypeRating      = "rating"
)

const (
	ProfilePublicShareOnly = ShareLinksInProfileLevel("")
	ProfileAllShare        = ShareLinksInProfileLevel("all_share")
	ProfileHideShare       = ShareLinksInProfileLevel("hide_share")
	// ProfileSharePublic is the explicit "password-free shares only" choice.
	// The empty value (ProfilePublicShareOnly) doubles as "inherit the
	// site-wide default"; ProfileSharePublic lets a user force public-only
	// visibility even when the site default differs (#3390).
	ProfileSharePublic = ShareLinksInProfileLevel("public_share")
)

const (
	CipherAES256CTR Cipher = "aes-256-ctr"
)

const (
	ScopeProfile               = "profile"
	ScopeEmail                 = "email"
	ScopeOpenID                = "openid"
	ScopeOfflineAccess         = "offline_access"
	ScopeUserInfoRead          = "UserInfo.Read"
	ScopeUserInfoWrite         = "UserInfo.Write"
	ScopeUserSecurityInfoRead  = "UserSecurityInfo.Read"
	ScopeUserSecurityInfoWrite = "UserSecurityInfo.Write"
	ScopeWorkflowRead          = "Workflow.Read"
	ScopeWorkflowWrite         = "Workflow.Write"
	ScopeAdminRead             = "Admin.Read"
	ScopeAdminWrite            = "Admin.Write"
	ScopeFilesRead             = "Files.Read"
	ScopeFilesWrite            = "Files.Write"
	ScopeSharesRead            = "Shares.Read"
	ScopeSharesWrite           = "Shares.Write"
	ScopeFinanceRead           = "Finance.Read"
	ScopeFinanceWrite          = "Finance.Write"
	ScopeDavAccountRead        = "DavAccount.Read"
	ScopeDavAccountWrite       = "DavAccount.Write"
)
