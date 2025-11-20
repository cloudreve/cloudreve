package dbfs

import (
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
)

type dbfsOption struct {
	*fs.FsOption
	loadFolderSummary          bool
	extendedInfo               bool
	loadFilePublicMetadata     bool
	loadFileShareIfOwned       bool
	loadEntityUser             bool
	loadFileEntities           bool
	useCursorPagination        bool
	pageToken                  string
	preferredStoragePolicy     *ent.StoragePolicy
	errOnConflict              bool
	previousVersion            string
	removeStaleEntities        bool
	requiredCapabilities       []NavigatorCapability
	generateContextHint        bool
	isSymbolicLink             bool
	noChainedCreation          bool
	streamListResponseCallback func(parent fs.File, file []fs.File)
	ancestor                   *File
	notRoot                    bool
	encryptMetadata            *types.EncryptMetadata
}

func newDbfsOption() *dbfsOption {
	return &dbfsOption{
		FsOption: &fs.FsOption{},
	}
}

func (o *dbfsOption) apply(opt fs.Option) {
	if fsOpt, ok := opt.(fs.OptionFunc); ok {
		fsOpt.Apply(o.FsOption)
	} else if dbfsOpt, ok := opt.(optionFunc); ok {
		dbfsOpt.Apply(o)
	}
}

type optionFunc func(*dbfsOption)

func (f optionFunc) Apply(o any) {
	if dbfsO, ok := o.(*dbfsOption); ok {
		f(dbfsO)
	}
}

// WithEncryptMetadata sets the encrypt metadata for the upload operation.
func WithEncryptMetadata(encryptMetadata *types.EncryptMetadata) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.encryptMetadata = encryptMetadata
	})
}

// WithFilePublicMetadata enables loading file public metadata.
func WithFilePublicMetadata() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.loadFilePublicMetadata = true
	})
}

// WithNotRoot force the get result cannot be a root folder
func WithNotRoot() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.notRoot = true
	})
}

// WithContextHint enables generating context hint for the list operation.
func WithContextHint() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.generateContextHint = true
	})
}

// WithFileEntities enables loading file entities.
func WithFileEntities() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.loadFileEntities = true
	})
}

// WithCursorPagination enables cursor pagination for the list operation.
func WithCursorPagination(pageToken string) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.useCursorPagination = true
		o.pageToken = pageToken
	})
}

// WithPreferredStoragePolicy sets the preferred storage policy for the upload operation.
func WithPreferredStoragePolicy(policy *ent.StoragePolicy) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.preferredStoragePolicy = policy
	})
}

// WithErrorOnConflict sets to throw error on conflict for the create operation.
func WithErrorOnConflict() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.errOnConflict = true
	})
}

// WithPreviousVersion sets the previous version for the update operation.
func WithPreviousVersion(version string) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.previousVersion = version
	})
}

// WithRemoveStaleEntities sets to remove stale entities for the update operation.
func WithRemoveStaleEntities() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.removeStaleEntities = true
	})
}

// WithRequiredCapabilities sets the required capabilities for operations.
func WithRequiredCapabilities(capabilities ...NavigatorCapability) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.requiredCapabilities = capabilities
	})
}

// WithNoChainedCreation sets to disable chained creation for the create operation. This
// will require parent folder existed before creating new files under it.
func WithNoChainedCreation() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.noChainedCreation = true
	})
}

// WithFileShareIfOwned enables loading file share link if the file is owned by the user.
func WithFileShareIfOwned() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.loadFileShareIfOwned = true
	})
}

// WithStreamListResponseCallback sets the callback for handling stream list response.
func WithStreamListResponseCallback(callback func(parent fs.File, file []fs.File)) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.streamListResponseCallback = callback
	})
}

// WithSymbolicLink sets the file is a symbolic link.
func WithSymbolicLink() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.isSymbolicLink = true
	})
}

// WithExtendedInfo enables loading extended info for the file.
func WithExtendedInfo() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.extendedInfo = true
	})
}

// WithLoadFolderSummary enables loading folder summary.
func WithLoadFolderSummary() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.loadFolderSummary = true
	})
}

// WithEntityUser enables loading entity user.
func WithEntityUser() fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.loadEntityUser = true
	})
}

// WithAncestor sets most recent ancestor for creating files
func WithAncestor(f *File) fs.Option {
	return optionFunc(func(o *dbfsOption) {
		o.ancestor = f
	})
}
