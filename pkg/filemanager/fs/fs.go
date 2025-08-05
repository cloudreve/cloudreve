package fs

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gofrs/uuid"
)

type FsCapability int

const (
	FsCapabilityList = FsCapability(iota)
)

var (
	ErrDirectLinkInvalid    = serializer.NewError(serializer.CodeNotFound, "Direct link invalid", nil)
	ErrUnknownPolicyType    = serializer.NewError(serializer.CodeInternalSetting, "Unknown policy type", nil)
	ErrPathNotExist         = serializer.NewError(serializer.CodeParentNotExist, "Path not exist", nil)
	ErrFileDeleted          = serializer.NewError(serializer.CodeFileDeleted, "File deleted", nil)
	ErrEntityNotExist       = serializer.NewError(serializer.CodeEntityNotExist, "Entity not exist", nil)
	ErrFileExisted          = serializer.NewError(serializer.CodeObjectExist, "Object existed", nil)
	ErrNotSupportedAction   = serializer.NewError(serializer.CodeNoPermissionErr, "Not supported action", nil)
	ErrLockConflict         = serializer.NewError(serializer.CodeLockConflict, "Lock conflict", nil)
	ErrLockExpired          = serializer.NewError(serializer.CodeLockConflict, "Lock expired", nil)
	ErrModified             = serializer.NewError(serializer.CodeConflict, "Object conflict", nil)
	ErrIllegalObjectName    = serializer.NewError(serializer.CodeIllegalObjectName, "Invalid object name", nil)
	ErrFileSizeTooBig       = serializer.NewError(serializer.CodeFileTooLarge, "File is too large", nil)
	ErrInsufficientCapacity = serializer.NewError(serializer.CodeInsufficientCapacity, "Insufficient capacity", nil)
	ErrStaleVersion         = serializer.NewError(serializer.CodeStaleVersion, "File is updated during your edit", nil)
	ErrOwnerOnly            = serializer.NewError(serializer.CodeOwnerOnly, "Only owner or administrator can perform this action", nil)
	ErrArchiveSrcSizeTooBig = ErrFileSizeTooBig.WithError(fmt.Errorf("total size of to-be compressed file exceed group limit (%w)", queue.CriticalErr))
)

type (
	FileSystem interface {
		LockSystem
		UploadManager
		FileManager
		// Recycle recycles a DBFS and its generated resources.
		Recycle()
		// Capacity returns the storage capacity of the filesystem.
		Capacity(ctx context.Context, u *ent.User) (*Capacity, error)
		// CheckCapability checks if the filesystem supports given capability.
		CheckCapability(ctx context.Context, uri *URI, opts ...Option) error
		// StaleEntities returns all stale entities of given IDs. If no ID is given, all
		// potential stale entities will be returned.
		StaleEntities(ctx context.Context, entities ...int) ([]Entity, error)
		// AllFilesInTrashBin returns all files in trash bin, despite owner.
		AllFilesInTrashBin(ctx context.Context, opts ...Option) (*ListFileResult, error)
		// Walk walks through all files under given path with given depth limit.
		Walk(ctx context.Context, path *URI, depth int, walk WalkFunc, opts ...Option) error
		// SharedAddressTranslation translates a path that potentially contain shared symbolic to a real address.
		SharedAddressTranslation(ctx context.Context, path *URI, opts ...Option) (File, *URI, error)
		// ExecuteNavigatorHooks executes hooks of given type on a file for navigator based custom hooks.
		ExecuteNavigatorHooks(ctx context.Context, hookType HookType, file File) error
	}

	FileManager interface {
		// Get returns a file by its path.
		Get(ctx context.Context, path *URI, opts ...Option) (File, error)
		// Create creates a file.
		Create(ctx context.Context, path *URI, fileType types.FileType, opts ...Option) (File, error)
		// List lists files under give path.
		List(ctx context.Context, path *URI, opts ...Option) (File, *ListFileResult, error)
		// Rename renames a file.
		Rename(ctx context.Context, path *URI, newName string) (File, error)
		// Move moves files to dst.
		MoveOrCopy(ctx context.Context, path []*URI, dst *URI, isCopy bool) error
		// Delete performs hard-delete for given paths, return newly generated stale entities in this delete operation.
		Delete(ctx context.Context, path []*URI, opts ...Option) ([]Entity, error)
		// GetEntitiesFromFileID returns all entities of a given file.
		GetEntity(ctx context.Context, entityID int) (Entity, error)
		// UpsertMetadata update or insert metadata of a file.
		PatchMetadata(ctx context.Context, path []*URI, metas ...MetadataPatch) error
		// SoftDelete moves given files to trash bin.
		SoftDelete(ctx context.Context, path ...*URI) error
		// Restore restores given files from trash bin to its original location.
		Restore(ctx context.Context, path ...*URI) error
		// VersionControl performs version control on given file.
		//  - `delete` is false: set version as current version;
		//  - `delete` is true: delete version.
		VersionControl(ctx context.Context, path *URI, versionId int, delete bool) error
		// GetFileFromDirectLink gets a file from a direct link.
		GetFileFromDirectLink(ctx context.Context, dl *ent.DirectLink) (File, error)
		// TraverseFile traverses a file to its root file, return the file with linked root.
		TraverseFile(ctx context.Context, fileID int) (File, error)
		// PatchProps patches the props of a file.
		PatchProps(ctx context.Context, uri *URI, props *types.FileProps, delete bool) error
	}

	UploadManager interface {
		// PrepareUpload prepares an upload session. It performs validation on upload request and returns a placeholder
		// file if needed.
		PrepareUpload(ctx context.Context, req *UploadRequest, opts ...Option) (*UploadSession, error)
		// CompleteUpload completes an upload session.
		CompleteUpload(ctx context.Context, session *UploadSession) (File, error)
		// CancelUploadSession cancels an upload session. Delete the placeholder file if no other entity is created.
		CancelUploadSession(ctx context.Context, path *URI, sessionID string, session *UploadSession) ([]Entity, error)
		// PreValidateUpload pre-validates an upload request.
		PreValidateUpload(ctx context.Context, dst *URI, files ...PreValidateFile) error
	}

	LockSystem interface {
		// ConfirmLock confirms if a lock token is valid on given URI.
		ConfirmLock(ctx context.Context, ancestor File, uri *URI, token ...string) (func(), LockSession, error)
		// Lock locks a file. If zeroDepth is true, only the file itself will be locked. Ancestor is closest ancestor
		// of the file that will be locked, if the given uri is an existing file, ancestor will be itself.
		// `token` is optional and can be used if the requester need to explicitly specify a token.
		Lock(ctx context.Context, d time.Duration, requester *ent.User, zeroDepth bool, application lock.Application,
			uri *URI, token string) (LockSession, error)
		// Unlock unlocks files by given tokens.
		Unlock(ctx context.Context, tokens ...string) error
		// Refresh refreshes a lock.
		Refresh(ctx context.Context, d time.Duration, token string) (lock.LockDetails, error)
	}

	StatelessUploadManager interface {
		// PrepareUpload prepares the upload on the node.
		PrepareUpload(ctx context.Context, args *StatelessPrepareUploadService) (*StatelessPrepareUploadResponse, error)
		// CompleteUpload completes the upload on the node.
		CompleteUpload(ctx context.Context, args *StatelessCompleteUploadService) error
		// OnUploadFailed handles the failed upload on the node.
		OnUploadFailed(ctx context.Context, args *StatelessOnUploadFailedService) error
		// CreateFile creates a file on the node.
		CreateFile(ctx context.Context, args *StatelessCreateFileService) error
	}

	WalkFunc func(file File, level int) error

	File interface {
		IsNil() bool
		ID() int
		Name() string
		DisplayName() string
		Ext() string
		Type() types.FileType
		Size() int64
		UpdatedAt() time.Time
		CreatedAt() time.Time
		Metadata() map[string]string
		// Uri returns the URI of the file.
		Uri(isRoot bool) *URI
		Owner() *ent.User
		OwnerID() int
		// RootUri return the URI of the user root file under owner's view.
		RootUri() *URI
		Entities() []Entity
		PrimaryEntity() Entity
		PrimaryEntityID() int
		Shared() bool
		IsSymbolic() bool
		PolicyID() (id int)
		ExtendedInfo() *FileExtendedInfo
		FolderSummary() *FolderSummary
		Capabilities() *boolset.BooleanSet
		IsRootFolder() bool
		View() *types.ExplorerView
	}

	Entities []Entity
	Entity   interface {
		ID() int
		Type() types.EntityType
		Size() int64
		UpdatedAt() time.Time
		CreatedAt() time.Time
		Source() string
		ReferenceCount() int
		PolicyID() int
		UploadSessionID() *uuid.UUID
		CreatedBy() *ent.User
		Model() *ent.Entity
	}

	FileExtendedInfo struct {
		StoragePolicy         *ent.StoragePolicy
		StorageUsed           int64
		Shares                []*ent.Share
		EntityStoragePolicies map[int]*ent.StoragePolicy
		View                  *types.ExplorerView
		DirectLinks           []*ent.DirectLink
	}

	FolderSummary struct {
		Size         int64     `json:"size"`
		Files        int       `json:"files"`
		Folders      int       `json:"folders"`
		Completed    bool      `json:"completed"` // whether the size calculation is completed
		CalculatedAt time.Time `json:"calculated_at"`
	}

	MetadataPatch struct {
		Key              string `json:"key" binding:"required"`
		Value            string `json:"value"`
		Private          bool   `json:"private" binding:"ne=true"`
		Remove           bool   `json:"remove"`
		UpdateModifiedAt bool   `json:"-"`
	}

	// ListFileResult result of listing files.
	ListFileResult struct {
		Files                 []File
		Parent                File
		Pagination            *inventory.PaginationResults
		Props                 *NavigatorProps
		ContextHint           *uuid.UUID
		RecursionLimitReached bool
		MixedType             bool
		SingleFileView        bool
		StoragePolicy         *ent.StoragePolicy
		View                  *types.ExplorerView
	}

	// NavigatorProps is the properties of current filesystem.
	NavigatorProps struct {
		// Supported capabilities of the navigator.
		Capability *boolset.BooleanSet `json:"capability"`
		// MaxPageSize is the maximum page size of the navigator.
		MaxPageSize int `json:"max_page_size"`
		// OrderByOptions is the supported order by options of the navigator.
		OrderByOptions []string `json:"order_by_options"`
		// OrderDirectionOptions is the supported order direction options of the navigator.
		OrderDirectionOptions []string `json:"order_direction_options"`
	}

	// UploadCredential for uploading files in client side.
	UploadCredential struct {
		SessionID      string   `json:"session_id"`
		ChunkSize      int64    `json:"chunk_size"` // 分块大小，0 为部分快
		Expires        int64    `json:"expires"`    // 上传凭证过期时间， Unix 时间戳
		UploadURLs     []string `json:"upload_urls,omitempty"`
		Credential     string   `json:"credential,omitempty"`
		UploadID       string   `json:"uploadID,omitempty"`
		Callback       string   `json:"callback,omitempty"` // 回调地址
		Uri            string   `json:"uri,omitempty"`      // 存储路径
		AccessKey      string   `json:"ak,omitempty"`
		KeyTime        string   `json:"keyTime,omitempty"` // COS用有效期
		CompleteURL    string   `json:"completeURL,omitempty"`
		StoragePolicy  *ent.StoragePolicy
		CallbackSecret string `json:"callback_secret,omitempty"`
		MimeType       string `json:"mime_type,omitempty"`     // Expected mimetype
		UploadPolicy   string `json:"upload_policy,omitempty"` // Upyun upload policy
	}

	// UploadSession stores the information of an upload session, used in server side.
	UploadSession struct {
		UID            int // 发起者
		Policy         *ent.StoragePolicy
		FileID         int    // ID of the placeholder file
		EntityID       int    // ID of the new entity
		Callback       string // 回调 URL 地址
		CallbackSecret string // Callback secret
		UploadID       string // Multi-part upload ID
		UploadURL      string
		Credential     string
		ChunkSize      int64
		SentinelTaskID int
		NewFileCreated bool // If new file is created for this session
		Importing      bool // If the upload is importing from another file

		LockToken string // Token of the locked placeholder file
		Props     *UploadProps
	}

	// UploadProps properties of an upload session/request.
	UploadProps struct {
		Uri                    *URI
		Size                   int64
		UploadSessionID        string
		PreferredStoragePolicy int
		SavePath               string
		LastModified           *time.Time
		MimeType               string
		Metadata               map[string]string
		PreviousVersion        string
		// EntityType is the type of the entity to be created. If not set, a new file will be created
		// with a default version entity. This will be set in update request for existing files.
		EntityType *types.EntityType
		ExpireAt   time.Time
	}

	// FsOption options for underlying file system.
	FsOption struct {
		Page               int // Page number when listing files.
		PageSize           int // Size of pages when listing files.
		OrderBy            string
		OrderDirection     string
		UploadRequest      *UploadRequest
		UnlinkOnly         bool
		UploadSession      *UploadSession
		DownloadSpeed      int64
		IsDownload         bool
		Expire             *time.Time
		Entity             Entity
		IsThumb            bool
		EntityType         *types.EntityType
		EntityTypeNil      bool
		SkipSoftDelete     bool
		SysSkipSoftDelete  bool
		Metadata           map[string]string
		ArchiveCompression bool
		ProgressFunc
		MaxArchiveSize  int64
		DryRun          CreateArchiveDryRunFunc
		Policy          *ent.StoragePolicy
		Node            StatelessUploadManager
		StatelessUserID int
		NoCache         bool
	}

	// Option 发送请求的额外设置
	Option interface {
		Apply(any)
	}

	OptionFunc func(*FsOption)

	// Ctx keys used to detect user canceled operation.
	UserCancelCtx struct{}
	GinCtx        struct{}

	// Capacity describes the capacity of a filesystem.
	Capacity struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
	}

	FileCapacity int

	LockSession interface {
		LastToken() string
	}

	HookType int

	CreateArchiveDryRunFunc func(name string, e Entity)

	StatelessPrepareUploadService struct {
		UploadRequest *UploadRequest `json:"upload_request" binding:"required"`
		UserID        int            `json:"user_id"`
	}
	StatelessCompleteUploadService struct {
		UploadSession *UploadSession `json:"upload_session" binding:"required"`
		UserID        int            `json:"user_id"`
	}
	StatelessOnUploadFailedService struct {
		UploadSession *UploadSession `json:"upload_session" binding:"required"`
		UserID        int            `json:"user_id"`
	}
	StatelessCreateFileService struct {
		Path   string         `json:"path" binding:"required"`
		Type   types.FileType `json:"type" binding:"required"`
		UserID int            `json:"user_id"`
	}
	StatelessPrepareUploadResponse struct {
		Session *UploadSession
		Req     *UploadRequest
	}

	PrepareRelocateRes struct {
		Entities  map[int]*RelocateEntity `json:"entities,omitempty"`
		LockToken string                  `json:"lock_token,omitempty"`
		Policy    *ent.StoragePolicy      `json:"policy,omitempty"`
	}

	RelocateEntity struct {
		SrcEntity                *ent.Entity `json:"src_entity"`
		FileUri                  *URI        `json:"file_uri,omitempty"`
		NewSavePath              string      `json:"new_save_path"`
		ParentFiles              []int       `json:"parent_files"`
		PrimaryEntityParentFiles []int       `json:"primary_entity_parent_files"`
	}

	PreValidateFile struct {
		Name     string
		Size     int64
		OmitName bool // if true, file name will not be validated
	}

	PhysicalObject struct {
		Name         string    `json:"name"`
		Source       string    `json:"source"`
		RelativePath string    `json:"relative_path"`
		Size         int64     `json:"size"`
		IsDir        bool      `json:"is_dir"`
		LastModify   time.Time `json:"last_modify"`
	}
)

const (
	FileCapacityPreview FileCapacity = iota
	FileCapacityEnter
	FileCapacityDownload
	FileCapacityRename
	FileCapacityCopy
	FileCapacityMove
)

const (
	HookTypeBeforeDownload = HookType(iota)
)

func (p *UploadProps) Copy() *UploadProps {
	newProps := *p
	return &newProps
}

func (f OptionFunc) Apply(o any) {
	f(o.(*FsOption))
}

// ==================== FS Options ====================

// WithUploadSession sets upload session for manager.
func WithUploadSession(s *UploadSession) Option {
	return OptionFunc(func(o *FsOption) {
		o.UploadSession = s
	})
}

// WithPageSize limit items in a page for listing files.
func WithPageSize(s int) Option {
	return OptionFunc(func(o *FsOption) {
		o.PageSize = s
	})
}

// WithPage set page number for listing files.
func WithPage(p int) Option {
	return OptionFunc(func(o *FsOption) {
		o.Page = p
	})
}

// WithOrderBy set order by for listing files.
func WithOrderBy(p string) Option {
	return OptionFunc(func(o *FsOption) {
		o.OrderBy = p
	})
}

// WithOrderDirection set order direction for listing files.
func WithOrderDirection(p string) Option {
	return OptionFunc(func(o *FsOption) {
		o.OrderDirection = p
	})
}

// WithUploadRequest set upload request for uploading files.
func WithUploadRequest(p *UploadRequest) Option {
	return OptionFunc(func(o *FsOption) {
		o.UploadRequest = p
	})
}

// WithProgressFunc set progress function for manager.
func WithProgressFunc(p ProgressFunc) Option {
	return OptionFunc(func(o *FsOption) {
		o.ProgressFunc = p
	})
}

// WithUnlinkOnly set unlink only for unlinking files.
func WithUnlinkOnly(p bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.UnlinkOnly = p
	})
}

// WithDownloadSpeed sets download speed limit for manager.
func WithDownloadSpeed(speed int64) Option {
	return OptionFunc(func(o *FsOption) {
		o.DownloadSpeed = speed
	})
}

func WithIsDownload(b bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.IsDownload = b
	})
}

// WithSysSkipSoftDelete sets whether to skip soft delete without checking
// file ownership.
func WithSysSkipSoftDelete(b bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.SysSkipSoftDelete = b
	})
}

// WithNoCache sets whether to disable cache for entity's URL.
func WithNoCache(b bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.NoCache = b
	})
}

// WithUrlExpire sets expire time for entity's URL.
func WithUrlExpire(t *time.Time) Option {
	return OptionFunc(func(o *FsOption) {
		o.Expire = t
	})
}

// WithEntity sets entity for manager.
func WithEntity(e Entity) Option {
	return OptionFunc(func(o *FsOption) {
		o.Entity = e
	})
}

// WithPolicy sets storage policy overwrite for manager.
func WithPolicy(p *ent.StoragePolicy) Option {
	return OptionFunc(func(o *FsOption) {
		o.Policy = p
	})
}

// WithUseThumb sets whether entity's URL is used for thumbnail.
func WithUseThumb(b bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.IsThumb = b
	})
}

// WithEntityType sets entity type for manager.
func WithEntityType(t types.EntityType) Option {
	return OptionFunc(func(o *FsOption) {
		o.EntityType = &t
	})
}

// WithNoEntityType sets entity type to nil for manager.
func WithNoEntityType() Option {
	return OptionFunc(func(o *FsOption) {
		o.EntityTypeNil = true
	})
}

// WithSkipSoftDelete sets whether to skip soft delete.
func WithSkipSoftDelete(b bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.SkipSoftDelete = b
	})
}

// WithMetadata sets metadata for file creation.
func WithMetadata(m map[string]string) Option {
	return OptionFunc(func(o *FsOption) {
		o.Metadata = m
	})
}

// WithArchiveCompression sets whether to compress files in archive.
func WithArchiveCompression(b bool) Option {
	return OptionFunc(func(o *FsOption) {
		o.ArchiveCompression = b
	})
}

// WithMaxArchiveSize sets maximum size of to be archived file or to-be decompressed
// size, 0 for unlimited.
func WithMaxArchiveSize(s int64) Option {
	return OptionFunc(func(o *FsOption) {
		o.MaxArchiveSize = s
	})
}

// WithDryRun sets whether to perform dry run.
func WithDryRun(b CreateArchiveDryRunFunc) Option {
	return OptionFunc(func(o *FsOption) {
		o.DryRun = b
	})
}

// WithNode sets node for stateless upload manager.
func WithNode(n StatelessUploadManager) Option {
	return OptionFunc(func(o *FsOption) {
		o.Node = n
	})
}

// WithStatelessUserID sets stateless user ID for manager.
func WithStatelessUserID(id int) Option {
	return OptionFunc(func(o *FsOption) {
		o.StatelessUserID = id
	})
}

type WriteMode int

const (
	ModeNone      WriteMode = 0x00000
	ModeOverwrite WriteMode = 0x00001
	// Deprecated
	ModeNop WriteMode = 0x00004
)

type (
	ProgressFunc  func(current, diff int64, total int64)
	UploadRequest struct {
		Props *UploadProps

		Mode         WriteMode
		File         io.ReadCloser `json:"-"`
		Seeker       io.Seeker     `json:"-"`
		Offset       int64
		ProgressFunc `json:"-"`

		ImportFrom *PhysicalObject `json:"-"`
		read       int64
	}
)

func (file *UploadRequest) Read(p []byte) (n int, err error) {
	if file.File != nil {
		n, err = file.File.Read(p)
		file.read += int64(n)
		if file.ProgressFunc != nil {
			file.ProgressFunc(file.read, int64(n), file.Props.Size)
		}

		return
	}

	return 0, io.EOF
}

func (file *UploadRequest) Close() error {
	if file.File != nil {
		return file.File.Close()
	}

	return nil
}

func (file *UploadRequest) Seek(offset int64, whence int) (int64, error) {
	if file.Seekable() {
		previous := file.read
		o, err := file.Seeker.Seek(offset, whence)
		file.read = o
		if file.ProgressFunc != nil {
			file.ProgressFunc(o, file.read-previous, file.Props.Size)
		}
		return o, err
	}

	return 0, errors.New("no seeker")
}

func (file *UploadRequest) Seekable() bool {
	return file.Seeker != nil
}

func init() {
	gob.Register(UploadSession{})
	gob.Register(FolderSummary{})
}

type ApplicationType string

const (
	ApplicationCreate         ApplicationType = "create"
	ApplicationRename         ApplicationType = "rename"
	ApplicationSetPermission  ApplicationType = "setPermission"
	ApplicationMoveCopy       ApplicationType = "moveCopy"
	ApplicationUpload         ApplicationType = "upload"
	ApplicationUpdateMetadata ApplicationType = "updateMetadata"
	ApplicationDelete         ApplicationType = "delete"
	ApplicationSoftDelete     ApplicationType = "softDelete"
	ApplicationDAV            ApplicationType = "dav"
	ApplicationVersionControl ApplicationType = "versionControl"
	ApplicationViewer         ApplicationType = "viewer"
	ApplicationMount          ApplicationType = "mount"
	ApplicationRelocate       ApplicationType = "relocate"
)

func LockApp(a ApplicationType) lock.Application {
	return lock.Application{Type: string(a)}
}

type LockSessionCtxKey struct{}

// LockSessionToContext stores lock session to context.
func LockSessionToContext(ctx context.Context, session LockSession) context.Context {
	return context.WithValue(ctx, LockSessionCtxKey{}, session)
}

func FindDesiredEntity(file File, version string, hasher hashid.Encoder, entityType *types.EntityType) (bool, Entity) {
	if version == "" {
		return true, file.PrimaryEntity()
	}

	requestedVersion, err := hasher.Decode(version, hashid.EntityID)
	if err != nil {
		return false, nil
	}

	hasVersions := false
	for _, entity := range file.Entities() {
		if entity.Type() == types.EntityTypeVersion {
			hasVersions = true
		}

		if entity.ID() == requestedVersion && (entityType == nil || *entityType == entity.Type()) {
			return true, entity
		}
	}

	// Happy path for: File has no versions, requested version is empty entity
	if !hasVersions && requestedVersion == 0 {
		return true, file.PrimaryEntity()
	}

	return false, nil
}

type DbEntity struct {
	model *ent.Entity
}

func NewEntity(model *ent.Entity) Entity {
	return &DbEntity{model: model}
}

func (e *DbEntity) ID() int {
	return e.model.ID
}

func (e *DbEntity) Type() types.EntityType {
	return types.EntityType(e.model.Type)
}

func (e *DbEntity) Size() int64 {
	return e.model.Size
}

func (e *DbEntity) UpdatedAt() time.Time {
	return e.model.UpdatedAt
}

func (e *DbEntity) CreatedAt() time.Time {
	return e.model.CreatedAt
}

func (e *DbEntity) CreatedBy() *ent.User {
	return e.model.Edges.User
}

func (e *DbEntity) Source() string {
	return e.model.Source
}

func (e *DbEntity) ReferenceCount() int {
	return e.model.ReferenceCount
}

func (e *DbEntity) PolicyID() int {
	return e.model.StoragePolicyEntities
}

func (e *DbEntity) UploadSessionID() *uuid.UUID {
	return e.model.UploadSessionID
}

func (e *DbEntity) Model() *ent.Entity {
	return e.model
}

func NewEmptyEntity(u *ent.User) Entity {
	return &DbEntity{
		model: &ent.Entity{
			UpdatedAt:      time.Now(),
			ReferenceCount: 1,
			CreatedAt:      time.Now(),
			Edges: ent.EntityEdges{
				User: u,
			},
		},
	}
}
