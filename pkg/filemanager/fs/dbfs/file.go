package dbfs

import (
	"encoding/gob"
	"path"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/samber/lo"
)

func init() {
	gob.Register(File{})
	gob.Register(shareNavigatorState{})
	gob.Register(map[string]*File{})
	gob.Register(map[int]*File{})
}

var (
	filePool = &sync.Pool{
		New: func() any {
			return &File{
				Children: make(map[string]*File),
			}
		},
	}
)

func getDefaultView() *types.ExplorerView {
	return &types.ExplorerView{
		PageSize:  defaultPageSize,
		View:      "grid",
		Thumbnail: true,
	}
}

type (
	File struct {
		Model             *ent.File
		Children          map[string]*File
		Parent            *File
		Path              [2]*fs.URI
		OwnerModel        *ent.User
		IsUserRoot        bool
		CapabilitiesBs    *boolset.BooleanSet
		FileExtendedInfo  *fs.FileExtendedInfo
		FileFolderSummary *fs.FolderSummary

		disableView bool
		mu          *sync.Mutex
	}
)

const (
	MetadataSysPrefix           = "sys:"
	MetadataUploadSessionPrefix = MetadataSysPrefix + "upload_session"
	MetadataUploadSessionID     = MetadataUploadSessionPrefix + "_id"
	MetadataSharedRedirect      = MetadataSysPrefix + "shared_redirect"
	MetadataRestoreUri          = MetadataSysPrefix + "restore_uri"
	MetadataExpectedCollectTime = MetadataSysPrefix + "expected_collect_time"
	MetadataSharedOwner         = MetadataSysPrefix + "shared_owner"
	// MetadataPreferredPolicy stores the hashid-encoded storage policy a
	// directory prefers for new uploads inside it.
	MetadataPreferredPolicy = MetadataSysPrefix + "preferred_policy"

	ThumbMetadataPrefix = "thumb:"
	ThumbDisabledKey    = ThumbMetadataPrefix + "disabled"

	FullTextIndexKey = MetadataSysPrefix + "fulltext_index"

	pathIndexRoot = 0
	pathIndexUser = 1
)

func (f *File) Name() string {
	return f.Model.Name
}

func (f *File) IsNil() bool {
	return f == nil
}

func (f *File) DisplayName() string {
	if uri, ok := f.Metadata()[MetadataRestoreUri]; ok {
		restoreUri, err := fs.NewUriFromString(uri)
		if err != nil {
			return f.Name()
		}

		return path.Base(restoreUri.Path())
	}

	return f.Name()
}

func (f *File) CanHaveChildren() bool {
	return f.Type() == types.FileTypeFolder && !f.IsSymbolic()
}

func (f *File) Ext() string {
	return util.Ext(f.Name())
}

func (f *File) ID() int {
	return f.Model.ID
}

func (f *File) IsSymbolic() bool {
	return f.Model.IsSymbolic
}

func (f *File) Type() types.FileType {
	return types.FileType(f.Model.Type)
}

func (f *File) Size() int64 {
	return f.Model.Size
}

func (f *File) SizeUsed() int64 {
	return lo.SumBy(f.Entities(), func(item fs.Entity) int64 {
		return item.Size()
	})
}

func (f *File) InTrashBin() bool {
	return f.Root().Name() != inventory.RootFolderName
}

func (f *File) UpdatedAt() time.Time {
	return f.Model.UpdatedAt
}

func (f *File) CreatedAt() time.Time {
	return f.Model.CreatedAt
}

func (f *File) ExtendedInfo() *fs.FileExtendedInfo {
	return f.FileExtendedInfo
}

func (f *File) Owner() *ent.User {
	parent := f
	for parent != nil {
		if parent.OwnerModel != nil {
			return parent.OwnerModel
		}
		parent = parent.Parent
	}

	return nil
}

func (f *File) OwnerID() int {
	return f.Model.OwnerID
}

func (f *File) Shared() bool {
	return len(f.Model.Edges.Shares) > 0
}

func (f *File) Metadata() map[string]string {
	if f.Model.Edges.Metadata == nil {
		return nil
	}
	return lo.Associate(f.Model.Edges.Metadata, func(item *ent.Metadata) (string, string) {
		return item.Name, item.Value
	})
}

// Uri returns the URI of the file.
// If isRoot is true, the URI will be returned from owner's view.
// Otherwise, the URI will be returned from user's view.
func (f *File) Uri(isRoot bool) *fs.URI {
	index := 1
	if isRoot {
		index = 0
	}
	if f.Path[index] != nil || f.Parent == nil {
		return f.Path[index]
	}

	// Find the root file
	elements := make([]string, 0)
	parent := f
	for parent.Parent != nil && parent.Path[index] == nil {
		elements = append([]string{parent.Name()}, elements...)
		parent = parent.Parent
	}

	if parent.Path[index] == nil {
		return nil
	}

	return parent.Path[index].Join(elements...)
}

// View returns the view setting of the file, can be inherited from parent.
func (f *File) View() *types.ExplorerView {
	// If owner has disabled view sync, return nil
	owner := f.Owner()
	if owner != nil && owner.Settings != nil && owner.Settings.DisableViewSync {
		return nil
	}

	// If navigator has disabled view sync, return nil
	userRoot := f.UserRoot()
	if userRoot == nil || userRoot.disableView {
		return nil
	}

	current := f
	for current != nil {
		if current.Model.Props != nil && current.Model.Props.View != nil {
			return current.Model.Props.View
		}
		current = current.Parent
	}

	return getDefaultView()
}

// UserRoot return the root file from user's view.
func (f *File) UserRoot() *File {
	root := f
	for root != nil && !root.IsUserRoot {
		root = root.Parent
	}

	return root
}

// Root return the root file from owner's view.
func (f *File) Root() *File {
	root := f
	for root.Parent != nil {
		root = root.Parent
	}

	return root
}

// RootUri return the URI of the user root file under owner's view.
func (f *File) RootUri() *fs.URI {
	return f.UserRoot().Uri(true)
}

func (f *File) Replace(model *ent.File) *File {
	f.mu.Lock()
	delete(f.Parent.Children, f.Model.Name)
	f.mu.Unlock()

	defer f.Recycle()
	replaced := newFile(f.Parent, model)
	if f.IsRootFile() {
		// If target is a root file, the user path should remain the same.
		replaced.Path[pathIndexUser] = f.Path[pathIndexUser]
	}

	return replaced
}

// Ancestors return all ancestors of the file, until the owner root is reached.
func (f *File) Ancestors() []*File {
	return f.AncestorsChain()[1:]
}

// AncestorsChain return all ancestors of the file (including itself), until the owner root is reached.
func (f *File) AncestorsChain() []*File {
	ancestors := make([]*File, 0)
	parent := f
	for parent != nil {
		ancestors = append(ancestors, parent)
		parent = parent.Parent
	}

	return ancestors
}

func (f *File) PolicyID() int {
	root := f
	return root.Model.StoragePolicyFiles
}

// IsRootFolder return true if the file is the root folder under user's view.
func (f *File) IsRootFolder() bool {
	return f.Type() == types.FileTypeFolder && f.IsRootFile()
}

// IsRootFile return true if the file is the root file under user's view.
func (f *File) IsRootFile() bool {
	uri := f.Uri(false)
	p := uri.Path()
	return f.Model.Name == inventory.RootFolderName || p == fs.Separator || p == ""
}

func (f *File) Entities() []fs.Entity {
	return lo.Map(f.Model.Edges.Entities, func(item *ent.Entity, index int) fs.Entity {
		return fs.NewEntity(item)
	})
}

func (f *File) PrimaryEntity() fs.Entity {
	primary, _ := lo.Find(f.Model.Edges.Entities, func(item *ent.Entity) bool {
		return item.Type == int(types.EntityTypeVersion) && item.ID == f.Model.PrimaryEntity
	})
	if primary != nil {
		return fs.NewEntity(primary)
	}

	return fs.NewEmptyEntity(f.Owner())
}

func (f *File) PrimaryEntityID() int {
	return f.Model.PrimaryEntity
}

func (f *File) FolderSummary() *fs.FolderSummary {
	return f.FileFolderSummary
}

func (f *File) Capabilities() *boolset.BooleanSet {
	return f.CapabilitiesBs
}

func newFile(parent *File, model *ent.File) *File {
	f := filePool.Get().(*File)
	f.Model = model

	if parent != nil {
		f.Parent = parent
		parent.mu.Lock()
		parent.Children[model.Name] = f
		if parent.Path[pathIndexUser] != nil {
			f.Path[pathIndexUser] = parent.Path[pathIndexUser].Join(model.Name)
		}

		if parent.Path[pathIndexRoot] != nil {
			f.Path[pathIndexRoot] = parent.Path[pathIndexRoot].Join(model.Name)
		}

		f.CapabilitiesBs = parent.CapabilitiesBs
		f.mu = parent.mu
		parent.mu.Unlock()
	} else {
		f.mu = &sync.Mutex{}
	}

	return f
}

func newParentFile(parent *ent.File, child *File) *File {
	newParent := newFile(nil, parent)
	newParent.Children[child.Name()] = child
	child.Parent = newParent
	newParent.mu = child.mu
	return newParent
}

func (f *File) Recycle() {
	for _, child := range f.Children {
		child.Recycle()
	}

	f.Model = nil
	f.Children = make(map[string]*File)
	f.Path[0] = nil
	f.Path[1] = nil
	f.Parent = nil
	f.OwnerModel = nil
	f.IsUserRoot = false
	f.CapabilitiesBs = nil
	f.FileExtendedInfo = nil
	f.FileFolderSummary = nil
	f.disableView = false
	f.mu = nil

	filePool.Put(f)
}
