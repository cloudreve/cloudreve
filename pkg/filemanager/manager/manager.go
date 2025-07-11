package manager

import (
	"context"
	"io"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

var (
	ErrUnknownPolicyType = serializer.NewError(serializer.CodeInternalSetting, "Unknown policy type", nil)
)

const (
	UploadSessionCachePrefix = "callback_"
	// Ctx key for upload session
	UploadSessionCtx = "uploadSession"
)

type (
	FileOperation interface {
		// Get gets file object by given path
		Get(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, error)
		// List lists files under given path
		List(ctx context.Context, path *fs.URI, args *ListArgs) (fs.File, *fs.ListFileResult, error)
		// Create creates a file or directory
		Create(ctx context.Context, path *fs.URI, fileType types.FileType, opt ...fs.Option) (fs.File, error)
		// Rename renames a file or directory
		Rename(ctx context.Context, path *fs.URI, newName string) (fs.File, error)
		// Delete deletes a group of file or directory. UnlinkOnly indicates whether to delete file record in DB only.
		Delete(ctx context.Context, path []*fs.URI, opts ...fs.Option) error
		// Restore restores a group of files
		Restore(ctx context.Context, path ...*fs.URI) error
		// MoveOrCopy moves or copies a group of files
		MoveOrCopy(ctx context.Context, src []*fs.URI, dst *fs.URI, isCopy bool) error
		// Update puts file content. If given file does not exist, it will create a new one.
		Update(ctx context.Context, req *fs.UploadRequest, opts ...fs.Option) (fs.File, error)
		// Walk walks through given path
		Walk(ctx context.Context, path *fs.URI, depth int, f fs.WalkFunc, opts ...fs.Option) error
		// UpsertMedata update or insert metadata of given file
		PatchMedata(ctx context.Context, path []*fs.URI, data ...fs.MetadataPatch) error
		// CreateViewerSession creates a viewer session for given file
		CreateViewerSession(ctx context.Context, uri *fs.URI, version string, viewer *types.Viewer) (*ViewerSession, error)
		// TraverseFile traverses a file to its root file, return the file with linked root.
		TraverseFile(ctx context.Context, fileID int) (fs.File, error)
	}

	FsManagement interface {
		// SharedAddressTranslation translates shared symbolic address to real address. If path does not exist,
		// most recent existing parent directory will be returned.
		SharedAddressTranslation(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, *fs.URI, error)
		// Capacity gets capacity of current file system
		Capacity(ctx context.Context) (*fs.Capacity, error)
		// CheckIfCapacityExceeded checks if given user's capacity exceeded, and send notification email
		CheckIfCapacityExceeded(ctx context.Context) error
		// LocalDriver gets local driver for operating local files.
		LocalDriver(policy *ent.StoragePolicy) driver.Handler
		// CastStoragePolicyOnSlave check if given storage policy need to be casted to another.
		// It is used on slave node, when local policy need to cast to remote policy;
		// Remote policy with same node ID can be casted to local policy.
		CastStoragePolicyOnSlave(ctx context.Context, policy *ent.StoragePolicy) *ent.StoragePolicy
		// GetStorageDriver gets storage driver for given policy
		GetStorageDriver(ctx context.Context, policy *ent.StoragePolicy) (driver.Handler, error)
		// PatchView patches the view setting of a file
		PatchView(ctx context.Context, uri *fs.URI, view *types.ExplorerView) error
	}

	ShareManagement interface {
		// CreateShare creates a share link for given path
		CreateOrUpdateShare(ctx context.Context, path *fs.URI, args *CreateShareArgs) (*ent.Share, error)
	}

	Archiver interface {
		CreateArchive(ctx context.Context, uris []*fs.URI, writer io.Writer, opts ...fs.Option) (int, error)
	}

	FileManager interface {
		fs.LockSystem
		FileOperation
		EntityManagement
		UploadManagement
		FsManagement
		ShareManagement
		Archiver

		// Recycle reset current FileManager object and put back to resource pool
		Recycle()
	}

	// GetEntityUrlArgs single args to get entity url
	GetEntityUrlArgs struct {
		URI               *fs.URI
		PreferredEntityID string
	}

	// CreateShareArgs args to create share link
	CreateShareArgs struct {
		ExistedShareID  int
		IsPrivate       bool
		Password        string
		RemainDownloads int
		Expire          *time.Time
		ShareView       bool
		ShowReadMe      bool
	}
)

type manager struct {
	user         *ent.User
	l            logging.Logger
	fs           fs.FileSystem
	settings     setting.Provider
	kv           cache.Driver
	config       conf.ConfigProvider
	stateless    bool
	auth         auth.Auth
	hasher       hashid.Encoder
	policyClient inventory.StoragePolicyClient

	dep dependency.Dep
}

func NewFileManager(dep dependency.Dep, u *ent.User) FileManager {
	config := dep.ConfigProvider()
	if config.System().Mode == conf.SlaveMode || u == nil {
		return newStatelessFileManager(dep)
	}
	return &manager{
		l:        dep.Logger(),
		user:     u,
		settings: dep.SettingProvider(),
		fs: dbfs.NewDatabaseFS(u, dep.FileClient(), dep.ShareClient(), dep.Logger(), dep.LockSystem(),
			dep.SettingProvider(), dep.StoragePolicyClient(), dep.HashIDEncoder(), dep.UserClient(), dep.KV(), dep.NavigatorStateKV(), dep.DirectLinkClient()),
		kv:           dep.KV(),
		config:       config,
		auth:         dep.GeneralAuth(),
		hasher:       dep.HashIDEncoder(),
		policyClient: dep.StoragePolicyClient(),
		dep:          dep,
	}
}

func newStatelessFileManager(dep dependency.Dep) FileManager {
	return &manager{
		l:         dep.Logger(),
		settings:  dep.SettingProvider(),
		kv:        dep.KV(),
		config:    dep.ConfigProvider(),
		stateless: true,
		auth:      dep.GeneralAuth(),
		dep:       dep,
		hasher:    dep.HashIDEncoder(),
	}
}

func (m *manager) Recycle() {
	if m.fs != nil {
		m.fs.Recycle()
	}
}

func newOption() *fs.FsOption {
	return &fs.FsOption{}
}
