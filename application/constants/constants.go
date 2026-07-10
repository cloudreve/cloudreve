package constants

// These values will be injected at build time, DO NOT EDIT.

// BackendVersion 当前后端版本号
// Bumped to 4.14.1 in this fork to trigger the schema migration that creates the
// group⇄storage_policy allowed-set join table and backfills it (see inventory/migration.go).
// Bumped to 4.14.2 (APP-101) to trigger the migration that creates the
// media_process_task table and seeds the media-compression default settings.
var BackendVersion = "4.14.2"

// IsPro 是否为Pro版本
var IsPro = "false"

var IsProBool = IsPro == "true"

// LastCommit 最后commit id
var LastCommit = "000000"

const (
	APIPrefix      = "/api/v4"
	APIPrefixSlave = "/api/v4/slave"
	CrHeaderPrefix = "X-Cr-"
)

const CloudreveScheme = "cloudreve"

type (
	FileSystemType string
)

const (
	FileSystemMy           = FileSystemType("my")
	FileSystemShare        = FileSystemType("share")
	FileSystemTrash        = FileSystemType("trash")
	FileSystemSharedWithMe = FileSystemType("shared_with_me")
	FileSystemUnknown      = FileSystemType("unknown")
)
