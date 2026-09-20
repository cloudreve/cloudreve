package dbfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

var (
	ErrShareNotFound          = serializer.NewError(serializer.CodeNotFound, "Shared file does not exist", nil)
	ErrNotPurchased           = serializer.NewError(serializer.CodePurchaseRequired, "You need to purchased this share", nil)
	ErrShareDownloadDisabled  = serializer.NewError(serializer.CodeNoPermissionErr, "Download is disabled for this share", nil)
	ErrShareOperationDisabled = serializer.NewError(serializer.CodeNoPermissionErr, "This operation is not allowed for this share", nil)
)

const (
	PurchaseTicketHeader = constants.CrHeaderPrefix + "Purchase-Ticket"
)

// shareNavigatorCapability is the static capability superset of the share
// file system, populated in init() alongside the other navigator sets. The
// navigator-level gate in getNavigator runs before the share is resolved, so
// it must admit every action a share may grant. The effective per-share
// permission is enforced after share resolution via the capability set
// stamped onto resolved files (see Capabilities) and writePermitted.
var shareNavigatorCapability = &boolset.BooleanSet{}

// NewShareNavigator creates a navigator for user's "shared" file system.
func NewShareNavigator(u *ent.User, fileClient inventory.FileClient, shareClient inventory.ShareClient,
	aclClient inventory.AclClient, vasClient inventory.VasClient, l logging.Logger, config *setting.DBFS, hasher hashid.Encoder) Navigator {
	n := &shareNavigator{
		user:        u,
		l:           l,
		fileClient:  fileClient,
		shareClient: shareClient,
		aclClient:   aclClient,
		vasClient:   vasClient,
		config:      config,
	}
	n.baseNavigator = newBaseNavigator(fileClient, defaultFilter, u, hasher, config)
	return n
}

type (
	shareNavigator struct {
		l           logging.Logger
		user        *ent.User
		fileClient  inventory.FileClient
		shareClient inventory.ShareClient
		aclClient   inventory.AclClient
		config      *setting.DBFS

		*baseNavigator
		shareRoot       *File
		singleFileShare bool
		multiFileShare  bool
		ownerRoot       *File
		share           *ent.Share
		owner           *ent.User
		// aclCaps carries the unioned permission bits of ACL entries on the
		// shared file matching the acting user. nil means no entry matched —
		// capabilities then fall back to share props.
		aclCaps        *boolset.BooleanSet
		vasClient      inventory.VasClient
		disableRecycle bool
		persist        func()
		// sharePaid is resolved at Root() and persisted with the navigator
		// state: free shares, the owner, and buyers/ticket holders are true.
		sharePaid bool
	}

	shareNavigatorState struct {
		ShareRoot       *File
		OwnerRoot       *File
		SingleFileShare bool
		MultiFileShare  bool
		Share           *ent.Share
		Owner           *ent.User
		AclCaps         *boolset.BooleanSet
		SharePaid       bool
	}
)

func (n *shareNavigator) PersistState(kv cache.Driver, key string) {
	n.disableRecycle = true
	n.persist = func() {
		kv.Set(key, shareNavigatorState{
			ShareRoot:       n.shareRoot,
			OwnerRoot:       n.ownerRoot,
			SingleFileShare: n.singleFileShare,
			MultiFileShare:  n.multiFileShare,
			Share:           n.share,
			Owner:           n.owner,
			AclCaps:         n.aclCaps,
			SharePaid:       n.sharePaid,
		}, ContextHintTTL)
	}
}

func (n *shareNavigator) RestoreState(s State) error {
	n.disableRecycle = true
	if state, ok := s.(shareNavigatorState); ok {
		n.shareRoot = state.ShareRoot
		n.ownerRoot = state.OwnerRoot
		n.singleFileShare = state.SingleFileShare
		n.multiFileShare = state.MultiFileShare
		n.share = state.Share
		n.aclCaps = state.AclCaps
		n.owner = state.Owner
		n.sharePaid = state.SharePaid
		return nil
	}

	return fmt.Errorf("invalid state type: %T", s)
}

func (n *shareNavigator) Recycle() {
	if n.persist != nil {
		n.persist()
		n.persist = nil
	}

	if !n.disableRecycle {
		if n.ownerRoot != nil {
			n.ownerRoot.Recycle()
		} else if n.shareRoot != nil {
			n.shareRoot.Recycle()
		}
	}
}

func (n *shareNavigator) Root(ctx context.Context, path *fs.URI) (*File, error) {
	ctx = context.WithValue(ctx, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFiles{}, true)
	share, err := n.shareClient.GetByHashID(ctx, path.ID(hashid.EncodeUserID(n.hasher, n.user.ID)))
	if err != nil {
		return nil, ErrShareNotFound.WithError(err)
	}

	if err := inventory.IsValidShare(share); err != nil {
		return nil, ErrShareNotFound.WithError(err)
	}

	n.owner = share.Edges.User

	// Check password
	if share.Password != "" && share.Password != path.Password() {
		return nil, ErrShareIncorrectPassword
	}

	// Share must be assigned before capabilities are derived from its props.
	n.share = share
	n.sharePaid = n.checkSharePaid(ctx, share)

	// Resolve per-file ACL entries for non-owner visitors; matched rows
	// replace the share-props capability set for this user.
	if n.aclClient != nil && n.user.ID != n.owner.ID {
		if caps, err := n.aclClient.EffectivePermissions(ctx, share.Edges.File.ID, n.user); err == nil {
			n.aclCaps = caps
		}
	}

	var ownerRoot *File
	if len(share.Edges.Files) > 0 {
		// Multi-file share: a synthetic folder root unions every linked
		// file. The anchor file edge still drives validity, ACL and the
		// paid gate.
		n.multiFileShare = true
		n.shareRoot = newFile(nil, &ent.File{
			Type: int(types.FileTypeFolder),
			Name: share.Edges.File.Name,
		})
		n.shareRoot.Path[pathIndexUser] = path.Root()
		n.shareRoot.OwnerModel = n.owner
		n.shareRoot.IsUserRoot = true
		n.shareRoot.disableView = (share.Props == nil || !share.Props.ShareView) && n.user.ID != n.owner.ID
		n.shareRoot.CapabilitiesBs = n.Capabilities(false).Capability

		for _, m := range share.Edges.Files {
			if m.FileChildren == 0 {
				// Deleted or trashed — excluded like IsValidShare.
				continue
			}
			child, realRoot, err := n.linkSharedFile(ctx, m)
			if err != nil || realRoot.Name() != inventory.RootFolderName {
				delete(n.shareRoot.Children, m.Name)
				continue
			}
			_ = child
			if ownerRoot == nil {
				ownerRoot = realRoot
			}
		}
		if ownerRoot == nil {
			return nil, ErrShareNotFound
		}
	} else {
		// Share permission setting should overwrite root folder's permission
		n.shareRoot = newFile(nil, share.Edges.File)

		// Find the user side root of the file.
		ownerRoot, err = n.findRoot(ctx, n.shareRoot)
		if err != nil {
			return nil, err
		}

		if n.shareRoot.Type() == types.FileTypeFile {
			n.singleFileShare = true
			n.shareRoot = n.shareRoot.Parent
		}

		n.shareRoot.Path[pathIndexUser] = path.Root()
		n.shareRoot.OwnerModel = n.owner
		n.shareRoot.IsUserRoot = true
		n.shareRoot.disableView = (share.Props == nil || !share.Props.ShareView) && n.user.ID != n.owner.ID
		n.shareRoot.CapabilitiesBs = n.Capabilities(false).Capability

		// Check if any ancestors is deleted
		if ownerRoot.Name() != inventory.RootFolderName {
			return nil, ErrShareNotFound
		}
	}

	if n.user.ID != n.owner.ID && !n.user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionShareDownload)) {
		if inventory.IsAnonymousUser(n.user) {
			return nil, serializer.NewError(
				serializer.CodeAnonymouseAccessDenied,
				fmt.Sprintf("You don't have permission to access share links"),
				err,
			)
		}

		return nil, serializer.NewError(
			serializer.CodeNoPermissionErr,
			fmt.Sprintf("You don't have permission to access share links"),
			err,
		)
	}

	n.ownerRoot = ownerRoot
	n.ownerRoot.Path[pathIndexRoot] = newMyIDUri(hashid.EncodeUserID(n.hasher, n.owner.ID))
	return n.shareRoot, nil
}

func (n *shareNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	if n.shareRoot == nil {
		root, err := n.Root(ctx, path)
		if err != nil {
			return nil, err
		}

		n.shareRoot = root
	}

	current, lastAncestor := n.shareRoot, n.shareRoot
	elements := path.Elements()

	// Multi-file share: the first path element must be one of the linked
	// files; deeper elements walk into it like a normal folder share.
	if n.multiFileShare && len(elements) > 0 {
		first, ok := n.shareRoot.Children[elements[0]]
		if !ok {
			// Restored navigators may carry an empty child map — repopulate
			// from the linked file set before failing.
			if _, err := n.latestSharedFiles(ctx); err == nil {
				first, ok = n.shareRoot.Children[elements[0]]
			}
		}
		if !ok {
			return nil, fs.ErrPathNotExist
		}

		current = first
		var err error
		for index := 1; index < len(elements); index++ {
			lastAncestor = current
			current, err = n.walkNext(ctx, current, elements[index], index == len(elements)-1)
			if err != nil {
				return lastAncestor, fmt.Errorf("failed to walk into %q: %w", elements[index], err)
			}
		}

		return current, nil
	}

	// If target is root of single file share, the root itself is the target.
	if len(elements) == 1 && n.singleFileShare {
		file, err := n.latestSharedSingleFile(ctx)
		if err != nil {
			return nil, err
		}

		if len(elements) == 1 && file.Name() != elements[0] {
			return nil, fs.ErrPathNotExist
		}

		return file, nil
	}

	var err error
	for index, element := range elements {
		lastAncestor = current
		current, err = n.walkNext(ctx, current, element, index == len(elements)-1)
		if err != nil {
			return lastAncestor, fmt.Errorf("failed to walk into %q: %w", element, err)
		}
	}

	return current, nil
}

func (n *shareNavigator) walkNext(ctx context.Context, root *File, next string, isLeaf bool) (*File, error) {
	nextFile, err := n.baseNavigator.walkNext(ctx, root, next, isLeaf)
	if err != nil {
		return nil, err
	}

	return nextFile, nil
}

func (n *shareNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	if n.singleFileShare {
		file, err := n.latestSharedSingleFile(ctx)
		if err != nil {
			return nil, err
		}

		return &ListResult{
			Files:          []*File{file},
			Pagination:     &inventory.PaginationResults{},
			SingleFileView: true,
		}, nil
	}

	// Drop-box shares accept uploads but never list existing content.
	if n.share != nil && n.share.Props != nil && n.share.Props.UploadOnly {
		return &ListResult{
			Files:      []*File{},
			Pagination: &inventory.PaginationResults{},
		}, nil
	}

	// The synthetic root of a multi-file share lists the union of linked
	// files instead of any real folder's children.
	if n.multiFileShare && (parent == nil || parent.Model == nil || parent.Model.ID == 0) {
		files, err := n.latestSharedFiles(ctx)
		if err != nil {
			return nil, err
		}

		return &ListResult{
			Files:      files,
			Pagination: &inventory.PaginationResults{},
		}, nil
	}

	res, err := n.baseNavigator.children(ctx, parent, args)
	if err != nil {
		return nil, err
	}

	// Shares with a rendered readme can keep the file itself out of the
	// listing; it stays reachable by direct path for the readme viewer.
	if n.share != nil && n.share.Props != nil && n.share.Props.ShowReadMe && n.share.Props.HideReadMe {
		res.Files = lo.Filter(res.Files, func(f *File, _ int) bool {
			return !readMeFileNames[strings.ToUpper(f.Name())]
		})
	}
	return res, nil
}

// readMeFileNames mirrors the frontend's detection priority list; entries
// are uppercase for case-insensitive matching.
var readMeFileNames = map[string]bool{
	"README.MD":  true,
	"README.TXT": true,
}

// linkSharedFile attaches a linked file of a multi-file share under the
// synthetic root, then rebuilds its real parent chain so Uri(true)
// resolves to the owner's filesystem. The returned root is the file's
// user-side root; callers validate it against RootFolderName.
func (n *shareNavigator) linkSharedFile(ctx context.Context, m *ent.File) (*File, *File, error) {
	child := newFile(n.shareRoot, m)
	child.OwnerModel = n.owner

	realRoot, err := n.findRoot(ctx, child)
	if err != nil {
		return nil, nil, err
	}

	realRoot.Path[pathIndexRoot] = newMyIDUri(hashid.EncodeUserID(n.hasher, n.owner.ID))
	return child, realRoot, nil
}

// latestSharedFiles reloads every linked file of a multi-file share so
// deleted entries drop out of the listing. Each result is also registered
// under shareRoot.Children for To() resolution.
func (n *shareNavigator) latestSharedFiles(ctx context.Context) ([]*File, error) {
	files := make([]*File, 0, len(n.share.Edges.Files))
	for _, m := range n.share.Edges.Files {
		file, err := n.fileClient.GetByID(ctx, m.ID)
		if err != nil {
			continue
		}

		child, realRoot, err := n.linkSharedFile(ctx, file)
		if err != nil || realRoot.Name() != inventory.RootFolderName {
			delete(n.shareRoot.Children, file.Name)
			continue
		}

		files = append(files, child)
	}

	return files, nil
}

func (n *shareNavigator) latestSharedSingleFile(ctx context.Context) (*File, error) {
	if n.singleFileShare {
		file, err := n.fileClient.GetByID(ctx, n.share.Edges.File.ID)
		if err != nil {
			return nil, err
		}

		f := newFile(n.shareRoot, file)
		f.OwnerModel = n.shareRoot.OwnerModel

		return f, nil
	}

	return nil, fs.ErrPathNotExist
}

func (n *shareNavigator) Capabilities(isSearching bool) *fs.NavigatorProps {
	res := baseNavigatorProps(shareNavigatorCapability, n.config.MaxPageSize)

	// Once the share is resolved, narrow capabilities to what its props grant.
	// This set is stamped onto resolved files and consulted by writePermitted.
	if n.share != nil {
		res.Capability = n.shareCapabilities()
	}

	if isSearching {
		res.OrderByOptions = nil
		res.OrderDirectionOptions = nil
	}

	return res
}

// shareCapabilities derives the effective capability set from share props.
func (n *shareNavigator) shareCapabilities() *boolset.BooleanSet {
	var bs *boolset.BooleanSet
	// Matched ACL entries fully define a non-owner visitor's capabilities;
	// when no entry matched (nil) share props apply as the link default.
	if n.aclCaps != nil && n.owner != nil && n.user.ID != n.owner.ID {
		bs = aclPermsToCapabilities(n.aclCaps)
	} else {
		bs = n.propsCapabilities()
	}

	// Multi-file shares are a read/download union: write targets are
	// ambiguous at the synthetic root, which owns no real folder row.
	if n.multiFileShare {
		boolset.Sets(map[NavigatorCapability]bool{
			NavigatorCapabilityUploadFile:     false,
			NavigatorCapabilityCreateFile:     false,
			NavigatorCapabilityLockFile:       false,
			NavigatorCapabilityRenameFile:     false,
			NavigatorCapabilityDeleteFile:     false,
			NavigatorCapabilitySoftDelete:     false,
			NavigatorCapabilityUpdateMetadata: false,
		}, bs)
	}

	n.stripUnpaid(bs)
	return bs
}

// propsCapabilities maps share props to the default link capability set.
func (n *shareNavigator) propsCapabilities() *boolset.BooleanSet {
	bs := &boolset.BooleanSet{}
	boolset.Sets(map[NavigatorCapability]bool{
		NavigatorCapabilityListChildren:  true,
		NavigatorCapabilityDownloadFile:  true,
		NavigatorCapabilityEnterFolder:   true,
		NavigatorCapabilityInfo:          true,
		NavigatorCapabilityGenerateThumb: true,
	}, bs)

	props := n.share.Props
	if props == nil {
		return bs
	}

	// Drop-box shares accept uploads but deny any read or edit of existing
	// content; UploadOnly implies upload access and wins over edit grants.
	// It is folder-only: on a single-file share it would strip download from
	// the shared file itself.
	if props.UploadOnly && !n.singleFileShare {
		boolset.Sets(map[NavigatorCapability]bool{
			NavigatorCapabilityListChildren: false,
			NavigatorCapabilityDownloadFile: false,
			NavigatorCapabilityUploadFile:   true,
			NavigatorCapabilityCreateFile:   true,
			NavigatorCapabilityLockFile:     true,
			NavigatorCapabilityRenameFile:   false,
			NavigatorCapabilityDeleteFile:   false,
			NavigatorCapabilitySoftDelete:   false,
		}, bs)
		return bs
	}

	// PreviewOnly keeps DownloadFile so viewers can fetch entities; the
	// download action itself is denied in ExecuteHook when the request is an
	// explicit download.
	if props.AllowUpload || props.AllowEdit {
		boolset.Set(int(NavigatorCapabilityUploadFile), true, bs)
		boolset.Set(int(NavigatorCapabilityCreateFile), true, bs)
		boolset.Set(int(NavigatorCapabilityLockFile), true, bs)
	}
	if props.AllowEdit {
		boolset.Set(int(NavigatorCapabilityRenameFile), true, bs)
		boolset.Set(int(NavigatorCapabilityDeleteFile), true, bs)
		boolset.Set(int(NavigatorCapabilitySoftDelete), true, bs)
	}

	return bs
}

// stripUnpaid removes download/thumbnail capabilities for visitors who have
// not paid for a priced share; listing stays so the paywall can render.
func (n *shareNavigator) stripUnpaid(bs *boolset.BooleanSet) {
	if n.share != nil && n.share.PricePoints > 0 && !n.sharePaid {
		boolset.Set(int(NavigatorCapabilityDownloadFile), false, bs)
		boolset.Set(int(NavigatorCapabilityGenerateThumb), false, bs)
	}
}

// aclPermsToCapabilities maps ACL permission bits (read/create/update/delete)
// to the navigator capability set granted through a share link.
func aclPermsToCapabilities(perms *boolset.BooleanSet) *boolset.BooleanSet {
	bs := &boolset.BooleanSet{}
	if perms.Enabled(int(types.AclPermRead)) {
		boolset.Sets(map[NavigatorCapability]bool{
			NavigatorCapabilityListChildren:  true,
			NavigatorCapabilityDownloadFile:  true,
			NavigatorCapabilityEnterFolder:   true,
			NavigatorCapabilityInfo:          true,
			NavigatorCapabilityGenerateThumb: true,
		}, bs)
	}
	if perms.Enabled(int(types.AclPermCreate)) {
		boolset.Sets(map[NavigatorCapability]bool{
			NavigatorCapabilityUploadFile:  true,
			NavigatorCapabilityCreateFile:  true,
			NavigatorCapabilityLockFile:    true,
			NavigatorCapabilityEnterFolder: true,
		}, bs)
	}
	if perms.Enabled(int(types.AclPermUpdate)) {
		boolset.Sets(map[NavigatorCapability]bool{
			NavigatorCapabilityRenameFile:     true,
			NavigatorCapabilityUpdateMetadata: true,
			NavigatorCapabilityUploadFile:     true,
			NavigatorCapabilityCreateFile:     true,
			NavigatorCapabilityLockFile:       true,
		}, bs)
	}
	if perms.Enabled(int(types.AclPermDelete)) {
		boolset.Sets(map[NavigatorCapability]bool{
			NavigatorCapabilityDeleteFile: true,
			NavigatorCapabilitySoftDelete: true,
		}, bs)
	}
	return bs
}

func (n *shareNavigator) FollowTx(ctx context.Context) (func(), error) {
	oldBase := n.baseNavigator.fileClient
	revertFile, err := followTxClients(ctx, &n.fileClient)
	if err != nil {
		return nil, err
	}

	revertShare, err := followTxClients(ctx, &n.shareClient)
	if err != nil {
		revertFile()
		return nil, err
	}

	n.baseNavigator.fileClient = n.fileClient
	return func() {
		revertShare()
		revertFile()
		n.baseNavigator.fileClient = oldBase
	}, nil
}

func (n *shareNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	switch hookType {
	case fs.HookTypeBeforeDownload:
		// Priced shares deny every entity fetch — previews included — until
		// the visitor holds a purchase or a valid resume ticket.
		if n.share != nil && n.share.PricePoints > 0 && !n.sharePaid {
			return ErrNotPurchased
		}
		// Preview-only shares deny explicit downloads but still allow
		// entity fetches for inline viewers.
		if n.share != nil && n.share.Props != nil && n.share.Props.PreviewOnly {
			if isDownload, _ := ctx.Value(IsDownloadCtxKey{}).(bool); isDownload {
				return ErrShareDownloadDisabled
			}
		}
		if err := n.shareClient.Downloaded(ctx, n.share); err != nil {
			n.l.Warning("Failed to increase share download count: %s", err)
		}
	}
	return nil
}

// checkSharePaid resolves whether the acting user may fetch entities of a
// priced share. Free shares, the owner, existing buyers, and holders of a
// valid resume ticket pass; everyone else — including anonymous users
// without a ticket — is denied.
func (n *shareNavigator) checkSharePaid(ctx context.Context, share *ent.Share) bool {
	if share.PricePoints <= 0 || n.user.ID == share.Edges.User.ID {
		return true
	}
	// Groups with the share-free bit (staff/VIP) bypass the paywall.
	if n.user.Edges.Group != nil &&
		n.user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionShareFree)) {
		return true
	}
	if n.vasClient == nil {
		return false
	}
	if ticket, ok := ctx.Value(PurchaseTicketCtxKey{}).(string); ok && ticket != "" {
		if _, err := n.vasClient.SharePurchaseByTicket(ctx, share.ID, ticket); err == nil {
			return true
		}
	}
	if inventory.IsAnonymousUser(n.user) {
		return false
	}
	if _, err := n.vasClient.SharePurchase(ctx, share.ID, n.user.ID); err == nil {
		return true
	}
	return false
}

func (n *shareNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	return n.baseNavigator.walk(ctx, levelFiles, limit, depth, f)
}

func (n *shareNavigator) GetView(ctx context.Context, file *File) *types.ExplorerView {
	return file.View()
}
