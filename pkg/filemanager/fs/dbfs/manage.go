package dbfs

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/samber/lo"
	"golang.org/x/tools/container/intsets"
)

func (f *DBFS) Create(ctx context.Context, path *fs.URI, fileType types.FileType, opts ...fs.Option) (fs.File, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	// Get navigator
	navigator, err := f.getNavigator(ctx, path, NavigatorCapabilityCreateFile, NavigatorCapabilityLockFile)
	if err != nil {
		return nil, err
	}

	// Get most recent ancestor
	var ancestor *File
	if o.ancestor != nil {
		ancestor = o.ancestor
	} else {
		ancestor, err = f.getFileByPath(ctx, navigator, path)
		if err != nil && !ent.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get ancestor: %w", err)
		}
	}

	if ancestor.Uri(false).IsSame(path, hashid.EncodeUserID(f.hasher, f.user.ID)) {
		if ancestor.Type() == fileType {
			if o.errOnConflict {
				return ancestor, fs.ErrFileExisted
			}

			// Target file already exist, return it.
			return ancestor, nil
		}

		// File with the same name but different type already exist
		return nil, fs.ErrFileExisted.
			WithError(fmt.Errorf("object with the same name but different type %q already exist", ancestor.Type()))
	}

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && ancestor.Owner().ID != f.user.ID {
		return nil, fs.ErrOwnerOnly
	}

	// Lock ancestor
	lockedPath := ancestor.RootUri().JoinRaw(path.PathTrimmed())
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationCreate),
		&LockByPath{lockedPath, ancestor, fileType, ""})
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, err
	}

	// For all ancestors in user's desired path, create folders if not exist
	existedElements := ancestor.Uri(false).Elements()
	desired := path.Elements()
	if (len(desired)-len(existedElements) > 1) && o.noChainedCreation {
		return nil, fs.ErrPathNotExist
	}

	for i := len(existedElements); i < len(desired); i++ {
		// Make sure parent is a folder
		if !ancestor.CanHaveChildren() {
			return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("parent must be a valid folder"))
		}

		// Validate object name
		if err := validateFileName(desired[i]); err != nil {
			return nil, fs.ErrIllegalObjectName.WithError(err)
		}

		if i < len(desired)-1 || fileType == types.FileTypeFolder {
			args := &inventory.CreateFolderParameters{
				Owner: ancestor.Model.OwnerID,
				Name:  desired[i],
			}

			// Apply options for last element
			if i == len(desired)-1 {
				if o.Metadata != nil {
					args.Metadata = o.Metadata
				}
				args.IsSymbolic = o.isSymbolicLink
			}

			// Create folder if it is not the last element or the target is a folder
			fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
			if err != nil {
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
			}

			newFolder, err := fc.CreateFolder(ctx, ancestor.Model, args)
			if err != nil {
				_ = inventory.Rollback(tx)
				return nil, fmt.Errorf("failed to create folder %q: %w", desired[i], err)
			}

			if err := inventory.Commit(tx); err != nil {
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit folder creation", err)
			}

			ancestor = newFile(ancestor, newFolder)
		} else {
			// valide file name
			policy, err := f.getPreferredPolicy(ctx, ancestor)
			if err != nil {
				return nil, err
			}

			if err := validateExtension(desired[i], policy); err != nil {
				return nil, fs.ErrIllegalObjectName.WithError(err)
			}

			if err := validateFileNameRegexp(desired[i], policy); err != nil {
				return nil, fs.ErrIllegalObjectName.WithError(err)
			}

			file, err := f.createFile(ctx, ancestor, desired[i], fileType, o)
			if err != nil {
				return nil, err
			}

			return file, nil
		}
	}

	return ancestor, nil
}

func (f *DBFS) Rename(ctx context.Context, path *fs.URI, newName string) (fs.File, error) {
	// Get navigator
	navigator, err := f.getNavigator(ctx, path, NavigatorCapabilityRenameFile, NavigatorCapabilityLockFile)
	if err != nil {
		return nil, err
	}

	// Get target file
	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get target file: %w", err)
	}
	oldName := target.Name()

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.Owner().ID != f.user.ID {
		return nil, fs.ErrOwnerOnly
	}

	// Root folder cannot be modified
	if target.IsRootFolder() {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot modify root folder"))
	}

	// Validate new name
	if err := validateFileName(newName); err != nil {
		return nil, fs.ErrIllegalObjectName.WithError(err)
	}

	// If target is a file, validate file extension
	policy, err := f.getPreferredPolicy(ctx, target)
	if err != nil {
		return nil, err
	}

	if target.Type() == types.FileTypeFile {
		if err := validateExtension(newName, policy); err != nil {
			return nil, fs.ErrIllegalObjectName.WithError(err)
		}

		if err := validateFileNameRegexp(newName, policy); err != nil {
			return nil, fs.ErrIllegalObjectName.WithError(err)
		}
	}

	// Lock target
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationRename),
		&LockByPath{target.Uri(true), target, target.Type(), ""})
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, err
	}

	// Rename target
	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	updated, err := fc.Rename(ctx, target.Model, newName)
	if err != nil {
		_ = inventory.Rollback(tx)
		if ent.IsConstraintError(err) {
			return nil, fs.ErrFileExisted.WithError(err)
		}

		return nil, serializer.NewError(serializer.CodeDBError, "failed to update file", err)
	}

	if target.Type() == types.FileTypeFile && !strings.EqualFold(filepath.Ext(newName), filepath.Ext(oldName)) {
		if err := fc.RemoveMetadata(ctx, target.Model, ThumbDisabledKey); err != nil {
			_ = inventory.Rollback(tx)
			return nil, serializer.NewError(serializer.CodeDBError, "failed to remove disabled thumbnail mark", err)
		}
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit rename change", err)
	}

	return target.Replace(updated), nil
}

func (f *DBFS) SoftDelete(ctx context.Context, path ...*fs.URI) error {
	ae := serializer.NewAggregateError()
	targets := make([]*File, 0, len(path))
	for _, p := range path {
		// Get navigator
		navigator, err := f.getNavigator(ctx, p, NavigatorCapabilitySoftDelete)
		if err != nil {
			ae.Add(p.String(), err)
			continue
		}

		// Get target file
		target, err := f.getFileByPath(ctx, navigator, p)
		if err != nil {
			ae.Add(p.String(), fmt.Errorf("failed to get target file: %w", err))
			continue
		}

		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.Owner().ID != f.user.ID {
			ae.Add(p.String(), fs.ErrOwnerOnly.WithError(fmt.Errorf("only file owner can delete file without trash bin")))
			continue
		}

		// Root folder cannot be deleted
		if target.IsRootFolder() {
			ae.Add(p.String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot delete root folder")))
			continue
		}

		targets = append(targets, target)
	}

	if len(targets) == 0 {
		return ae.Aggregate()
	}
	// Lock all targets
	lockTargets := lo.Map(targets, func(value *File, key int) *LockByPath {
		return &LockByPath{value.Uri(true), value, value.Type(), ""}
	})
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationSoftDelete), lockTargets...)
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return err
	}

	// Start transaction to soft-delete files
	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	for _, target := range targets {
		// Perform soft-delete
		if err := fc.SoftDelete(ctx, target.Model); err != nil {
			_ = inventory.Rollback(tx)
			return serializer.NewError(serializer.CodeDBError, "failed to soft-delete file", err)
		}

		// Save restore uri into metadata
		if err := fc.UpsertMetadata(ctx, target.Model, map[string]string{
			MetadataRestoreUri: target.Uri(true).String(),
			MetadataExpectedCollectTime: strconv.FormatInt(
				time.Now().Add(time.Duration(target.Owner().GroupMaxTrashRetention())*time.Second).Unix(),
				10),
		}, nil); err != nil {
			_ = inventory.Rollback(tx)
			return serializer.NewError(serializer.CodeDBError, "failed to update metadata", err)
		}
	}

	// Commit transaction
	if err := inventory.Commit(tx); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to commit soft-delete change", err)
	}

	return ae.Aggregate()
}

func (f *DBFS) Delete(ctx context.Context, path []*fs.URI, opts ...fs.Option) ([]fs.Entity, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	var opt *types.EntityRecycleOption
	if o.UnlinkOnly {
		opt = &types.EntityRecycleOption{
			UnlinkOnly: true,
		}
	}

	ae := serializer.NewAggregateError()
	fileNavGroup := make(map[Navigator][]*File)
	ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)

	for _, p := range path {
		// Get navigator
		navigator, err := f.getNavigator(ctx, p, NavigatorCapabilityDeleteFile, NavigatorCapabilityLockFile)
		if err != nil {
			ae.Add(p.String(), err)
			continue
		}

		// Get target file
		target, err := f.getFileByPath(ctx, navigator, p)
		if err != nil {
			ae.Add(p.String(), fmt.Errorf("failed to get target file: %w", err))
			continue
		}

		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !o.SysSkipSoftDelete && !ok && target.Owner().ID != f.user.ID {
			ae.Add(p.String(), fs.ErrOwnerOnly)
			continue
		}

		// Root folder cannot be deleted
		if target.IsRootFolder() {
			ae.Add(p.String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot delete root folder")))
			continue
		}

		if _, ok := fileNavGroup[navigator]; !ok {
			fileNavGroup[navigator] = make([]*File, 0)
		}
		fileNavGroup[navigator] = append(fileNavGroup[navigator], target)
	}

	targets := lo.Flatten(lo.Values(fileNavGroup))
	if len(targets) == 0 {
		return nil, ae.Aggregate()
	}
	// Lock all targets
	lockTargets := lo.Map(targets, func(value *File, key int) *LockByPath {
		return &LockByPath{value.Uri(true), value, value.Type(), ""}
	})
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationDelete), lockTargets...)
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, err
	}

	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	// Delete targets
	newStaleEntities, storageDiff, err := f.deleteFiles(ctx, fileNavGroup, fc, opt)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "failed to delete files", err)
	}

	tx.AppendStorageDiff(storageDiff)
	if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit delete change", err)
	}

	return newStaleEntities, ae.Aggregate()
}

func (f *DBFS) VersionControl(ctx context.Context, path *fs.URI, versionId int, delete bool) error {
	// Get navigator
	navigator, err := f.getNavigator(ctx, path, NavigatorCapabilityVersionControl)
	if err != nil {
		return err
	}

	// Get target file
	ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return fmt.Errorf("failed to get target file: %w", err)
	}

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.Owner().ID != f.user.ID {
		return fs.ErrOwnerOnly
	}

	// Target must be a file
	if target.Type() != types.FileTypeFile {
		return fs.ErrNotSupportedAction.WithError(fmt.Errorf("target must be a valid file"))
	}

	// Lock file
	ls, err := f.acquireByPath(ctx, -1, f.user, true, fs.LockApp(fs.ApplicationVersionControl),
		&LockByPath{target.Uri(true), target, target.Type(), ""})
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return err
	}

	if delete {
		storageDiff, err := f.deleteEntity(ctx, target, versionId)
		if err != nil {
			return err
		}

		if err := f.userClient.ApplyStorageDiff(ctx, storageDiff); err != nil {
			f.l.Error("Failed to apply storage diff after deleting version: %s", err)
		}
		return nil
	} else {
		return f.setCurrentVersion(ctx, target, versionId)
	}
}

func (f *DBFS) Restore(ctx context.Context, path ...*fs.URI) error {
	ae := serializer.NewAggregateError()
	targets := make([]*File, 0, len(path))
	ctx = context.WithValue(ctx, inventory.LoadFilePublicMetadata{}, true)

	for _, p := range path {
		// Get navigator
		navigator, err := f.getNavigator(ctx, p, NavigatorCapabilityRestore)
		if err != nil {
			ae.Add(p.String(), err)
			continue
		}

		// Get target file
		target, err := f.getFileByPath(ctx, navigator, p)
		if err != nil {
			ae.Add(p.String(), fmt.Errorf("failed to get file: %w", err))
			continue
		}

		targets = append(targets, target)
	}

	if len(targets) == 0 {
		return ae.Aggregate()
	}

	allTrashUriStr := lo.FilterMap(targets, func(t *File, key int) ([]*fs.URI, bool) {
		if restoreUri, ok := t.Metadata()[MetadataRestoreUri]; ok {
			srcUrl, err := fs.NewUriFromString(restoreUri)
			if err != nil {
				ae.Add(t.Uri(false).String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("invalid restore uri: %w", err)))
				return nil, false
			}

			return []*fs.URI{t.Uri(false), srcUrl.DirUri()}, true
		}

		ae.Add(t.Uri(false).String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot restore file without required metadata mark")))
		return nil, false
	})

	// Copy each file to its original location
	for _, uris := range allTrashUriStr {
		if err := f.MoveOrCopy(ctx, []*fs.URI{uris[0]}, uris[1], false); err != nil {
			if !ae.Merge(err) {
				ae.Add(uris[0].String(), err)
			}
		}
	}

	return ae.Aggregate()

}

func (f *DBFS) MoveOrCopy(ctx context.Context, path []*fs.URI, dst *fs.URI, isCopy bool) error {
	targets := make([]*File, 0, len(path))
	dstNavigator, err := f.getNavigator(ctx, dst, NavigatorCapabilityLockFile)
	if err != nil {
		return err
	}

	// Get destination file
	destination, err := f.getFileByPath(ctx, dstNavigator, dst)
	if err != nil {
		return fmt.Errorf("faield to get destination folder: %w", err)
	}

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && destination.Owner().ID != f.user.ID {
		return fs.ErrOwnerOnly
	}

	// Target must be a folder
	if !destination.CanHaveChildren() {
		return fs.ErrNotSupportedAction.WithError(fmt.Errorf("destination must be a valid folder"))
	}

	ae := serializer.NewAggregateError()
	fileNavGroup := make(map[Navigator][]*File)
	dstRootPath := destination.Uri(true)
	ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)

	for _, p := range path {
		// Get navigator
		navigator, err := f.getNavigator(ctx, p, NavigatorCapabilityLockFile)
		if err != nil {
			ae.Add(p.String(), err)
			continue
		}

		// Check fs capability
		if !canMoveOrCopyTo(p, dst, isCopy) {
			ae.Add(p.String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot move or copy file form %s to %s", p.String(), dst.String())))
			continue
		}

		// Get target file
		target, err := f.getFileByPath(ctx, navigator, p)
		if err != nil {
			ae.Add(p.String(), fmt.Errorf("failed to get file: %w", err))
			continue
		}

		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.Owner().ID != f.user.ID {
			ae.Add(p.String(), fs.ErrOwnerOnly)
			continue
		}

		// Root folder cannot be moved or copied
		if target.IsRootFolder() {
			ae.Add(p.String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot move root folder")))
			continue
		}

		// Cannot move or copy folder to its descendant
		if target.Type() == types.FileTypeFolder &&
			dstRootPath.EqualOrIsDescendantOf(target.Uri(true), hashid.EncodeUserID(f.hasher, f.user.ID)) {
			ae.Add(p.String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot move or copy folder to itself or its descendant")))
			continue
		}

		targets = append(targets, target)
		if isCopy {
			if _, ok := fileNavGroup[navigator]; !ok {
				fileNavGroup[navigator] = make([]*File, 0)
			}
			fileNavGroup[navigator] = append(fileNavGroup[navigator], target)
		}
	}

	if len(targets) > 0 {
		// Lock all targets
		lockTargets := lo.Map(targets, func(value *File, key int) *LockByPath {
			return &LockByPath{value.Uri(true), value, value.Type(), ""}
		})

		// Lock destination
		dstBase := destination.Uri(true)
		dstLockTargets := lo.Map(targets, func(value *File, key int) *LockByPath {
			return &LockByPath{dstBase.Join(value.Name()), destination, value.Type(), ""}
		})
		allLockTargets := make([]*LockByPath, 0, len(targets)*2)
		if !isCopy {
			// For moving files from trash bin, also lock the dst with restored name.
			dstRestoreTargets := lo.FilterMap(targets, func(value *File, key int) (*LockByPath, bool) {
				if _, ok := value.Metadata()[MetadataRestoreUri]; ok {
					return &LockByPath{dstBase.Join(value.DisplayName()), destination, value.Type(), ""}, true
				}
				return nil, false
			})
			allLockTargets = append(allLockTargets, lockTargets...)
			allLockTargets = append(allLockTargets, dstRestoreTargets...)
		}
		allLockTargets = append(allLockTargets, dstLockTargets...)
		ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationMoveCopy), allLockTargets...)
		defer func() { _ = f.Release(ctx, ls) }()
		if err != nil {
			return err
		}

		// Start transaction to move files
		fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
		}

		var (
			storageDiff inventory.StorageDiff
		)
		if isCopy {
			_, storageDiff, err = f.copyFiles(ctx, fileNavGroup, destination, fc)
		} else {
			storageDiff, err = f.moveFiles(ctx, targets, destination, fc, dstNavigator)
		}

		if err != nil {
			_ = inventory.Rollback(tx)
			return err
		}

		tx.AppendStorageDiff(storageDiff)
		if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to commit move change", err)
		}

		// TODO: after move, dbfs cache should be cleared
	}

	return ae.Aggregate()
}

func (f *DBFS) GetFileFromDirectLink(ctx context.Context, dl *ent.DirectLink) (fs.File, error) {
	fileModel, err := dl.Edges.FileOrErr()
	if err != nil {
		return nil, err
	}

	owner, err := fileModel.Edges.OwnerOrErr()
	if err != nil {
		return nil, err
	}

	// File owner must be active
	if owner.Status != user.StatusActive {
		return nil, fs.ErrDirectLinkInvalid.WithError(fmt.Errorf("file owner is not active"))
	}

	file := newFile(nil, fileModel)

	// Traverse to the root file
	baseNavigator := newBaseNavigator(f.fileClient, defaultFilter, f.user, f.hasher, f.settingClient.DBFS(ctx))
	root, err := baseNavigator.findRoot(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("failed to find root file: %w", err)
	}

	if root.Name() != inventory.RootFolderName {
		return nil, serializer.NewError(serializer.CodeNotFound, "direct link not found", err)
	}

	return file, nil
}

func (f *DBFS) TraverseFile(ctx context.Context, fileID int) (fs.File, error) {
	fileModel, err := f.fileClient.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if fileModel.OwnerID != f.user.ID && !f.user.EnforceGroupPermission(types.GroupPermissionIsAdmin) {
		return nil, fs.ErrOwnerOnly.WithError(fmt.Errorf("only file owner can traverse file's uri"))
	}

	file := newFile(nil, fileModel)

	// Traverse to the root file
	baseNavigator := newBaseNavigator(f.fileClient, defaultFilter, f.user, f.hasher, f.settingClient.DBFS(ctx))
	root, err := baseNavigator.findRoot(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("failed to find root file: %w", err)
	}

	rootUri := newMyUri()
	if fileModel.OwnerID != f.user.ID {
		rootUri = newMyIDUri(hashid.EncodeUserID(f.hasher, fileModel.OwnerID))
	}

	if root.Name() != inventory.RootFolderName {
		rootUri = newTrashUri(root.Name())
	}

	root.Path[pathIndexRoot] = rootUri
	root.Path[pathIndexUser] = rootUri

	return file, nil
}

func (f *DBFS) deleteEntity(ctx context.Context, target *File, entityId int) (inventory.StorageDiff, error) {
	if target.PrimaryEntityID() == entityId {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot delete current version"))
	}

	targetVersion, found := lo.Find(target.Entities(), func(item fs.Entity) bool {
		return item.ID() == entityId
	})
	if !found {
		return nil, fs.ErrEntityNotExist.WithError(fmt.Errorf("version not found"))
	}

	diff, err := f.fileClient.UnlinkEntity(ctx, targetVersion.Model(), target.Model, target.Owner())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to unlink entity", err)
	}

	if targetVersion.UploadSessionID() != nil {
		err = f.fileClient.RemoveMetadata(ctx, target.Model, MetadataUploadSessionID)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to remove upload session metadata", err)
		}
	}
	return diff, nil
}

func (f *DBFS) setCurrentVersion(ctx context.Context, target *File, versionId int) error {
	if target.PrimaryEntityID() == versionId {
		return nil
	}

	targetVersion, found := lo.Find(target.Entities(), func(item fs.Entity) bool {
		return item.ID() == versionId && item.Type() == types.EntityTypeVersion && item.UploadSessionID() == nil
	})
	if !found {
		return fs.ErrEntityNotExist.WithError(fmt.Errorf("version not found"))
	}

	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	if err := fc.SetPrimaryEntity(ctx, target.Model, targetVersion.ID()); err != nil {
		_ = inventory.Rollback(tx)
		return serializer.NewError(serializer.CodeDBError, "Failed to set primary entity", err)
	}

	// Cap thumbnail entities
	diff, err := fc.CapEntities(ctx, target.Model, target.Owner(), 0, types.EntityTypeThumbnail)
	if err != nil {
		_ = inventory.Rollback(tx)
		return serializer.NewError(serializer.CodeDBError, "Failed to cap thumbnail entities", err)
	}

	tx.AppendStorageDiff(diff)
	if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to commit set current version", err)
	}

	return nil
}

func (f *DBFS) deleteFiles(ctx context.Context, targets map[Navigator][]*File, fc inventory.FileClient, opt *types.EntityRecycleOption) ([]fs.Entity, inventory.StorageDiff, error) {
	if f.user.Edges.Group == nil {
		return nil, nil, fmt.Errorf("user group not loaded")
	}
	allStaleEntities := make([]fs.Entity, 0, len(targets))
	storageDiff := make(inventory.StorageDiff)
	for n, files := range targets {
		// Let navigator use tx
		reset, err := n.FollowTx(ctx)
		if err != nil {
			return nil, nil, err
		}

		defer reset()

		// List all files to be deleted
		toBeDeletedFiles := make([]*File, 0, len(files))
		if err := n.Walk(ctx, files, intsets.MaxInt, intsets.MaxInt, func(targets []*File, level int) error {
			toBeDeletedFiles = append(toBeDeletedFiles, targets...)
			return nil
		}); err != nil {
			return nil, nil, fmt.Errorf("failed to walk files: %w", err)
		}

		// Delete files
		staleEntities, diff, err := fc.Delete(ctx, lo.Map(toBeDeletedFiles, func(item *File, index int) *ent.File {
			return item.Model
		}), opt)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete files: %w", err)
		}
		storageDiff.Merge(diff)
		allStaleEntities = append(allStaleEntities, lo.Map(staleEntities, func(item *ent.Entity, index int) fs.Entity {
			return fs.NewEntity(item)
		})...)
	}

	return allStaleEntities, storageDiff, nil
}

func (f *DBFS) copyFiles(ctx context.Context, targets map[Navigator][]*File, destination *File, fc inventory.FileClient) (map[int]*ent.File, inventory.StorageDiff, error) {
	if f.user.Edges.Group == nil {
		return nil, nil, fmt.Errorf("user group not loaded")
	}
	limit := f.user.GroupMaxWalkedFiles()
	capacity, err := f.Capacity(ctx, destination.Owner())
	if err != nil {
		return nil, nil, fmt.Errorf("copy files: failed to destination owner capacity: %w", err)
	}

	dstAncestors := lo.Map(destination.AncestorsChain(), func(item *File, index int) *ent.File {
		return item.Model
	})

	// newTargetsMap is the map of between new target files in first layer, and its src file ID.
	newTargetsMap := make(map[int]*ent.File)
	storageDiff := make(inventory.StorageDiff)
	var diff inventory.StorageDiff
	for n, files := range targets {
		initialDstMap := make(map[int][]*ent.File)
		for _, file := range files {
			initialDstMap[file.Model.FileChildren] = dstAncestors
		}

		firstLayer := true
		// Let navigator use tx
		reset, err := n.FollowTx(ctx)
		if err != nil {
			return nil, nil, err
		}

		defer reset()

		if err := n.Walk(ctx, files, limit, intsets.MaxInt, func(targets []*File, level int) error {
			// check capacity for each file
			sizeTotal := int64(0)
			for _, file := range targets {
				sizeTotal += file.SizeUsed()
			}

			if err := f.validateUserCapacityRaw(ctx, sizeTotal, capacity); err != nil {
				return fs.ErrInsufficientCapacity
			}

			limit -= len(targets)
			initialDstMap, diff, err = fc.Copy(ctx, lo.Map(targets, func(item *File, index int) *ent.File {
				return item.Model
			}), initialDstMap)
			if err != nil {
				if ent.IsConstraintError(err) {
					return fs.ErrFileExisted.WithError(err)
				}

				return serializer.NewError(serializer.CodeDBError, "Failed to copy files", err)
			}

			storageDiff.Merge(diff)

			if firstLayer {
				for k, v := range initialDstMap {
					newTargetsMap[k] = v[0]
				}
			}

			capacity.Used += sizeTotal
			firstLayer = false

			return nil
		}); err != nil {
			return nil, nil, fmt.Errorf("failed to walk files: %w", err)
		}
	}

	return newTargetsMap, storageDiff, nil
}

func (f *DBFS) moveFiles(ctx context.Context, targets []*File, destination *File, fc inventory.FileClient, n Navigator) (inventory.StorageDiff, error) {
	models := lo.Map(targets, func(value *File, key int) *ent.File {
		return value.Model
	})

	// Change targets' parent
	if err := fc.SetParent(ctx, models, destination.Model); err != nil {
		if ent.IsConstraintError(err) {
			return nil, fs.ErrFileExisted.WithError(err)
		}

		return nil, serializer.NewError(serializer.CodeDBError, "Failed to move file", err)
	}

	var (
		storageDiff inventory.StorageDiff
	)

	// For files moved out from trash bin
	for _, file := range targets {
		if _, ok := file.Metadata()[MetadataRestoreUri]; !ok {
			continue
		}

		// renaming it to its original name
		if _, err := fc.Rename(ctx, file.Model, file.DisplayName()); err != nil {
			if ent.IsConstraintError(err) {
				return nil, fs.ErrFileExisted.WithError(err)
			}

			return storageDiff, serializer.NewError(serializer.CodeDBError, "Failed to rename file from trash bin to its original name", err)
		}

		// Remove trash bin metadata
		if err := fc.RemoveMetadata(ctx, file.Model, MetadataRestoreUri, MetadataExpectedCollectTime); err != nil {
			return storageDiff, serializer.NewError(serializer.CodeDBError, "Failed to remove trash related metadata", err)
		}
	}

	return storageDiff, nil
}
