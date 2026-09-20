package dbfs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/samber/lo"
	"math"
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
			WithError(fmt.Errorf("object with the same name but different type %v already exist", ancestor.Type()))
	}

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && !f.writePermitted(ancestor, NavigatorCapabilityCreateFile) {
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

		// Validate object name — folder segments keep the strict rule set
		// since their governing policy is ambiguous.
		if err := validateFileName(desired[i], nil); err != nil {
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
			f.emitFileCreated(ctx, ancestor)
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

func (f *DBFS) Rename(ctx context.Context, path *fs.URI, newName string) (fs.File, *fs.IndexDiff, error) {
	// Get navigator
	navigator, err := f.getNavigator(ctx, path, NavigatorCapabilityRenameFile, NavigatorCapabilityLockFile)
	if err != nil {
		return nil, nil, err
	}

	// Get target file
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get target file: %w", err)
	}
	if sourceIDMismatch(ctx, 0, target.Model.ID) {
		return nil, nil, fs.ErrModified.WithError(fmt.Errorf("source file no longer matches the expected file"))
	}
	oldName := target.Name()

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && !f.writePermitted(target, NavigatorCapabilityRenameFile) {
		return nil, nil, fs.ErrOwnerOnly
	}

	// Root folder cannot be modified
	if target.IsRootFolder() {
		return nil, nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot modify root folder"))
	}

	policy, err := f.getPreferredPolicy(ctx, target)
	if err != nil {
		return nil, nil, err
	}

	// Validate new name
	if err := validateFileName(newName, policy); err != nil {
		return nil, nil, fs.ErrIllegalObjectName.WithError(err)
	}

	// If target is a file, validate file extension

	if target.Type() == types.FileTypeFile {
		if err := validateExtension(newName, policy); err != nil {
			return nil, nil, fs.ErrIllegalObjectName.WithError(err)
		}

		if err := validateFileNameRegexp(newName, policy); err != nil {
			return nil, nil, fs.ErrIllegalObjectName.WithError(err)
		}
	}

	// Lock target
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationRename),
		&LockByPath{target.Uri(true), target, target.Type(), ""})
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, nil, err
	}

	// Rename target
	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	updated, err := fc.Rename(ctx, target.Model, newName)
	if err != nil {
		_ = inventory.Rollback(tx)
		if ent.IsConstraintError(err) {
			return nil, nil, fs.ErrFileExisted.WithError(err)
		}

		return nil, nil, serializer.NewError(serializer.CodeDBError, "failed to update file", err)
	}

	if target.Type() == types.FileTypeFile && !strings.EqualFold(filepath.Ext(newName), filepath.Ext(oldName)) {
		if err := fc.RemoveMetadata(ctx, target.Model, ThumbDisabledKey); err != nil {
			_ = inventory.Rollback(tx)
			return nil, nil, serializer.NewError(serializer.CodeDBError, "failed to remove disabled thumbnail mark", err)
		}
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, nil, serializer.NewError(serializer.CodeDBError, "Failed to commit rename change", err)
	}

	f.emitFileRenamed(ctx, target, newName)
	f.record(ctx, types.EventFileRename, activity.File(target.ID()), activity.Extra(map[string]any{"old_name": oldName, "new_name": newName}))

	originalMetadata := target.Metadata()
	newFile := target.Replace(updated)
	var diff *fs.IndexDiff
	if _, ok := originalMetadata[FullTextIndexKey]; ok {
		diff = &fs.IndexDiff{
			IndexToRename: []fs.IndexDiffRenameDetails{
				{
					Uri:      *newFile.Uri(false),
					FileID:   newFile.ID(),
					EntityID: newFile.PrimaryEntityID(),
				},
			},
		}
	}

	return newFile, diff, nil
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

		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && !f.writePermitted(target, NavigatorCapabilitySoftDelete) {
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

		// Save restore uri into metadata. A user-level retention overrides
		// the group default when set.
		retention := 0
		if g := inventory.EffectiveGroup(target.Owner()); g != nil && g.Settings != nil {
			retention = g.Settings.TrashRetention
		}
		if us := target.Owner().Settings; us != nil && us.TrashRetention > 0 {
			retention = us.TrashRetention
		}
		if err := fc.UpsertMetadata(ctx, target.Model, map[string]string{
			MetadataRestoreUri: target.Uri(true).String(),
			MetadataExpectedCollectTime: strconv.FormatInt(
				time.Now().Add(time.Duration(retention)*time.Second).Unix(),
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

	f.emitFileDeleted(ctx, targets...)
	for _, target := range targets {
		f.record(ctx, types.EventMoveToTrash, activity.File(target.ID()), activity.Extra(map[string]any{"uri": target.Uri(false).String()}))
	}

	return ae.Aggregate()
}

func (f *DBFS) Delete(ctx context.Context, path []*fs.URI, opts ...fs.Option) ([]fs.Entity, *fs.IndexDiff, error) {
	o := newDbfsOption()
	for _, opt := range opts {
		o.apply(opt)
	}

	var opt *types.EntityProps
	if o.UnlinkOnly {
		opt = &types.EntityProps{
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

		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !o.SysSkipSoftDelete && !ok && !f.writePermitted(target, NavigatorCapabilityDeleteFile) {
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
		return nil, nil, ae.Aggregate()
	}
	// Lock all targets
	lockTargets := lo.Map(targets, func(value *File, key int) *LockByPath {
		return &LockByPath{value.Uri(true), value, value.Type(), ""}
	})
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationDelete), lockTargets...)
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, nil, err
	}

	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	// Delete targets
	newStaleEntities, storageDiff, indexToDelete, err := f.deleteFiles(ctx, fileNavGroup, fc, opt)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, nil, serializer.NewError(serializer.CodeDBError, "failed to delete files", err)
	}

	tx.AppendStorageDiff(storageDiff)
	if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
		return nil, nil, serializer.NewError(serializer.CodeDBError, "Failed to commit delete change", err)
	}
	f.emitFileDeleted(ctx, targets...)
	for _, target := range targets {
		f.record(ctx, types.EventDeleteFile, activity.File(target.ID()), activity.Extra(map[string]any{"uri": target.Uri(false).String()}))
	}
	return newStaleEntities, &fs.IndexDiff{
		IndexToDelete: indexToDelete,
	}, ae.Aggregate()
}

func (f *DBFS) VersionControl(ctx context.Context, path *fs.URI, versionId int, delete bool) (*fs.IndexDiff, error) {
	// Get navigator
	navigator, err := f.getNavigator(ctx, path, NavigatorCapabilityVersionControl)
	if err != nil {
		return nil, err
	}

	// Get target file
	ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	if !delete {
		ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	}
	target, err := f.getFileByPath(ctx, navigator, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get target file: %w", err)
	}

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.Owner().ID != f.user.ID {
		return nil, fs.ErrOwnerOnly
	}

	// Target must be a file
	if target.Type() != types.FileTypeFile {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("target must be a valid file"))
	}

	// Lock file
	ls, err := f.acquireByPath(ctx, -1, f.user, true, fs.LockApp(fs.ApplicationVersionControl),
		&LockByPath{target.Uri(true), target, target.Type(), ""})
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, err
	}

	if delete {
		storageDiff, err := f.deleteEntity(ctx, target, versionId)
		if err != nil {
			return nil, err
		}

		if err := f.userClient.ApplyStorageDiff(ctx, storageDiff); err != nil {
			f.l.Error("Failed to apply storage diff after deleting version: %s", err)
		}
		return nil, nil
	} else {
		if err := f.setCurrentVersion(ctx, target, versionId); err != nil {
			return nil, err
		}

		if _, ok := target.Metadata()[FullTextIndexKey]; ok {
			return &fs.IndexDiff{
				IndexToUpdate: []fs.IndexDiffUpdateDetails{
					{
						Uri:      *target.Uri(false),
						FileID:   target.ID(),
						OwnerID:  target.Owner().ID,
						EntityID: versionId,
					},
				},
			}, nil
		}

		return nil, nil
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
		if _, err := f.MoveOrCopy(ctx, []*fs.URI{uris[0]}, uris[1], false); err != nil {
			if !ae.Merge(err) {
				ae.Add(uris[0].String(), err)
			}
		}
	}

	return ae.Aggregate()

}

func (f *DBFS) MoveOrCopy(ctx context.Context, path []*fs.URI, dst *fs.URI, isCopy bool) (*fs.IndexDiff, error) {
	targets := make([]*File, 0, len(path))
	dstNavigator, err := f.getNavigator(ctx, dst, NavigatorCapabilityLockFile)
	if err != nil {
		return nil, err
	}

	// Get destination file
	destination, err := f.getFileByPath(ctx, dstNavigator, dst)
	if err != nil {
		return nil, fmt.Errorf("faield to get destination folder: %w", err)
	}

	if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && !f.writePermitted(destination, NavigatorCapabilityCreateFile) {
		return nil, fs.ErrOwnerOnly
	}

	// Target must be a folder
	if !destination.CanHaveChildren() {
		return nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("destination must be a valid folder"))
	}

	ae := serializer.NewAggregateError()
	fileNavGroup := make(map[Navigator][]*File)
	navOf := make(map[*File]Navigator)
	dstRootPath := destination.Uri(true)
	ctx = context.WithValue(ctx, inventory.LoadFileEntity{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)

	for i, p := range path {
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

		if sourceIDMismatch(ctx, i, target.Model.ID) {
			ae.Add(p.String(), fs.ErrModified.WithError(fmt.Errorf("source file no longer matches the expected file")))
			continue
		}

		// Copy reads the source, move deletes it.
		requiredSrcCap := NavigatorCapabilityDownloadFile
		if !isCopy {
			requiredSrcCap = NavigatorCapabilityDeleteFile
		}
		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok &&
			!f.writePermitted(target, requiredSrcCap) {
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
		navOf[target] = navigator
		if isCopy {
			if _, ok := fileNavGroup[navigator]; !ok {
				fileNavGroup[navigator] = make([]*File, 0)
			}
			fileNavGroup[navigator] = append(fileNavGroup[navigator], target)
		}
	}

	// Resolve name conflicts against existing destination children before
	// locking: skip drops the colliding target, overwrite deletes the
	// colliding destination object first (#3159).
	if mode := moveConflictMode(ctx); mode != "" && len(targets) > 0 {
		targets, fileNavGroup = f.resolveMoveConflicts(ctx, targets, navOf, destination, isCopy, mode, ae)
	}

	indexDiff := &fs.IndexDiff{}
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
			return nil, err
		}

		// Start transaction to move files
		fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
		}

		var (
			copiedNewTargetsMap map[int]*ent.File
			storageDiff         inventory.StorageDiff
			indexDiffBatch      *fs.IndexDiff
		)
		if isCopy {
			copiedNewTargetsMap, storageDiff, indexDiffBatch, err = f.copyFiles(ctx, fileNavGroup, destination, fc)
		} else {
			storageDiff, indexDiffBatch, err = f.moveFiles(ctx, targets, destination, fc, dstNavigator)
		}

		if err != nil {
			_ = inventory.Rollback(tx)
			if errors.Is(err, fs.ErrInsufficientCapacity) {
				extra := map[string]any{"dst": destination.Uri(true).String()}
				if capacity, cerr := f.Capacity(ctx, destination.Owner()); cerr == nil {
					extra["capacity"] = capacity.Total
				}
				f.record(ctx, types.EventUserExceedQuotaNotified, activity.Extra(extra))
			}
			return nil, err
		}

		indexDiff.Merge(indexDiffBatch)
		tx.AppendStorageDiff(storageDiff)
		if err := inventory.CommitWithStorageDiff(ctx, tx, f.l, f.userClient); err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit move change", err)
		}

		for _, target := range targets {
			if isCopy {
				copied := copiedNewTargetsMap[target.ID()]
				f.emitFileCreated(ctx, newFile(destination, copied))
				dst := destination.Uri(false).String()
				f.record(ctx, types.EventCopyTo, activity.File(copied.ID), activity.Extra(map[string]any{"dst": dst}))
				f.record(ctx, types.EventCopyFrom, activity.File(target.ID()), activity.Extra(map[string]any{"dst": dst}))
			} else {
				f.emitFileMoved(ctx, target, destination)
				f.record(ctx, types.EventMoveTo, activity.File(target.ID()), activity.Extra(map[string]any{"dst": destination.Uri(false).String()}))
			}
		}

		// TODO: after move, dbfs cache should be cleared
	}

	return indexDiff, ae.Aggregate()
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

	// Check the owner's current direct-link permission.
	group := inventory.EffectiveGroup(owner)
	if group == nil || group.Settings == nil || group.Settings.SourceBatchSize <= 0 {
		return nil, fs.ErrDirectLinkInvalid
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

	// Files inside a private space are never served through direct links;
	// there is no session to carry an unlock state.
	if inside, err := f.fileInVault(ctx, file, f.vaultRootID(ctx, fileModel.OwnerID)); err != nil {
		return nil, err
	} else if inside {
		return nil, fs.ErrDirectLinkInvalid
	}

	return file, nil
}

func (f *DBFS) TraverseFile(ctx context.Context, fileID int) (fs.File, error) {
	fileModel, err := f.fileClient.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if fileModel.OwnerID != f.user.ID && !inventory.EffectiveGroup(f.user).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
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

	if err := f.requireVaultAccess(ctx, file); err != nil {
		return nil, err
	}

	navigator, err := f.getNavigator(ctx, rootUri)
	if err != nil {
		return nil, fmt.Errorf("failed to get navigator for root file: %w", err)
	}

	root.Path[pathIndexRoot] = rootUri
	root.Path[pathIndexUser] = rootUri
	file.CapabilitiesBs = navigator.Capabilities(false).Capability

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

	f.emitFileModified(ctx, target)
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

	if err := fc.SetPrimaryEntity(ctx, target.Model, targetVersion.Model()); err != nil {
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

	f.emitFileModified(ctx, target)
	return nil
}

func (f *DBFS) deleteFiles(ctx context.Context, targets map[Navigator][]*File, fc inventory.FileClient, opt *types.EntityProps) ([]fs.Entity, inventory.StorageDiff, []int, error) {
	if inventory.EffectiveGroup(f.user) == nil {
		return nil, nil, nil, fmt.Errorf("user group not loaded")
	}
	allStaleEntities := make([]fs.Entity, 0, len(targets))
	storageDiff := make(inventory.StorageDiff)
	indexToDelete := make([]int, 0)
	for n, files := range targets {
		// Let navigator use tx
		reset, err := n.FollowTx(ctx)
		if err != nil {
			return nil, nil, nil, err
		}

		defer reset()

		// Walk the tree and delete in bounded batches: accumulating the
		// whole tree before deleting materializes every file model +
		// entity edges at once and OOMs on large trees (upstream #3574).
		// Folders are deferred to a single delete after the walk — child
		// listing resolves via HasParentWith, so parent rows must stay
		// alive until their level has been fetched.
		folderModels := make([]*ent.File, 0, 64)
		if err := n.Walk(ctx, files, math.MaxInt, math.MaxInt, func(targets []*File, level int) error {
			indexToDelete = append(indexToDelete, lo.Map(targets, func(item *File, index int) int {
				return item.ID()
			})...)

			fileModels := make([]*ent.File, 0, len(targets))
			for _, item := range targets {
				if item.Model.Type == int(types.FileTypeFolder) && !item.IsSymbolic() {
					folderModels = append(folderModels, item.Model)
					continue
				}
				fileModels = append(fileModels, item.Model)
			}

			if len(fileModels) == 0 {
				return nil
			}
			staleEntities, diff, err := fc.Delete(ctx, fileModels, opt)
			if err != nil {
				return fmt.Errorf("failed to delete files: %w", err)
			}
			storageDiff.Merge(diff)
			allStaleEntities = append(allStaleEntities, lo.Map(staleEntities, func(item *ent.Entity, index int) fs.Entity {
				return fs.NewEntity(item)
			})...)
			return nil
		}); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to walk files: %w", err)
		}

		if len(folderModels) > 0 {
			staleEntities, diff, err := fc.Delete(ctx, folderModels, opt)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to delete folders: %w", err)
			}
			storageDiff.Merge(diff)
			allStaleEntities = append(allStaleEntities, lo.Map(staleEntities, func(item *ent.Entity, index int) fs.Entity {
				return fs.NewEntity(item)
			})...)
		}
	}

	return allStaleEntities, storageDiff, indexToDelete, nil
}

func (f *DBFS) copyFiles(ctx context.Context, targets map[Navigator][]*File, destination *File, fc inventory.FileClient) (map[int]*ent.File, inventory.StorageDiff, *fs.IndexDiff, error) {
	if inventory.EffectiveGroup(f.user) == nil {
		return nil, nil, nil, fmt.Errorf("user group not loaded")
	}
	limit := max(inventory.EffectiveGroup(f.user).Settings.MaxWalkedFiles, 1)
	capacity, err := f.Capacity(ctx, destination.Owner())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("copy files: failed to destination owner capacity: %w", err)
	}

	// Baseline usage of the destination policy: new entities are written in
	// this transaction, invisible to the usage query, so copied batches are
	// accumulated locally.
	dstPolicy, err := f.getPreferredPolicy(ctx, destination)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("copy files: failed to get destination storage policy: %w", err)
	}
	var dstPolicyUsed int64
	if dstPolicy.Settings.MaxTotalSize > 0 {
		var used int
		_, used, err = f.fileClient.CountEntityByStoragePolicyID(ctx, dstPolicy.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("copy files: failed to get storage policy usage: %w", err)
		}
		dstPolicyUsed = int64(used)
	}

	dstAncestors := lo.Map(destination.AncestorsChain(), func(item *File, index int) *ent.File {
		return item.Model
	})

	// newTargetsMap is the map of between new target files in first layer, and its src file ID.
	newTargetsMap := make(map[int]*ent.File)
	storageDiff := make(inventory.StorageDiff)
	indexToCopy := make([]fs.IndexDiffCopyDetails, 0)
	for n, files := range targets {
		initialDstMap := make(map[int][]*ent.File)
		for _, file := range files {
			initialDstMap[file.Model.FileChildren] = dstAncestors
		}

		if err := n.Walk(ctx, files, limit, math.MaxInt, func(targets []*File, level int) error {
			// check capacity for each file
			sizeTotal := int64(0)
			for _, file := range targets {
				sizeTotal += file.SizeUsed()
			}

			if err := f.validateUserCapacityRaw(ctx, sizeTotal, capacity); err != nil {
				return fs.ErrInsufficientCapacity
			}
			if err := f.validatePolicyCapacityRaw(sizeTotal, dstPolicy, dstPolicyUsed); err != nil {
				return err
			}
			dstPolicyUsed += sizeTotal

			limit -= len(targets)
			newDstMap, diff, err := fc.Copy(ctx, &inventory.CopyParameter{
				Files: lo.Map(targets, func(item *File, index int) *ent.File {
					return item.Model
				}),
				ExcludedMetadataKeys: []string{FullTextIndexKey},
				DstMap:               initialDstMap,
			})
			if err != nil {
				if ent.IsConstraintError(err) {
					return fs.ErrFileExisted.WithError(err)
				}

				return serializer.NewError(serializer.CodeDBError, "Failed to copy files", err)
			}

			// Walk emits each level in bounded pages, so dst mappings must
			// accumulate across callbacks — children of a folder copied in
			// an earlier page may arrive in any later page.
			for k, v := range newDstMap {
				initialDstMap[k] = v
			}
			if level == 0 {
				for k, v := range newDstMap {
					newTargetsMap[k] = v[0]
				}
			}

			storageDiff.Merge(diff)

			for _, file := range targets {
				if _, ok := file.Metadata()[FullTextIndexKey]; ok {
					// initialDstMap holds src->dst entries for every file
					// copied so far, across all levels and pages.
					copiedChain, ok := initialDstMap[file.ID()]
					if !ok || len(copiedChain) == 0 {
						continue
					}
					copiedFile := copiedChain[0]
					indexToCopy = append(indexToCopy, fs.IndexDiffCopyDetails{
						OriginalFileID: file.ID(),
						FileID:         copiedFile.ID,
						Uri:            *destination.Uri(false).Join(file.Name()),
						EntityID:       copiedFile.PrimaryEntity,
						OwnerID:        destination.OwnerID(),
					})
				}
			}

			capacity.Used += sizeTotal

			return nil
		}); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to walk files: %w", err)
		}
	}

	var indexDiff *fs.IndexDiff
	if len(indexToCopy) > 0 {
		indexDiff = &fs.IndexDiff{
			IndexToCopy: indexToCopy,
		}
	}

	return newTargetsMap, storageDiff, indexDiff, nil
}

func (f *DBFS) moveFiles(ctx context.Context, targets []*File, destination *File, fc inventory.FileClient, n Navigator) (inventory.StorageDiff, *fs.IndexDiff, error) {
	models := lo.Map(targets, func(value *File, key int) *ent.File {
		return value.Model
	})

	// Change targets' parent
	if err := fc.SetParent(ctx, models, destination.Model); err != nil {
		if ent.IsConstraintError(err) {
			return nil, nil, fs.ErrFileExisted.WithError(err)
		}

		return nil, nil, serializer.NewError(serializer.CodeDBError, "Failed to move file", err)
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
				return nil, nil, fs.ErrFileExisted.WithError(err)
			}

			return storageDiff, nil, serializer.NewError(serializer.CodeDBError, "Failed to rename file from trash bin to its original name", err)
		}

		// Remove trash bin metadata
		if err := fc.RemoveMetadata(ctx, file.Model, MetadataRestoreUri, MetadataExpectedCollectTime); err != nil {
			return storageDiff, nil, serializer.NewError(serializer.CodeDBError, "Failed to remove trash related metadata", err)
		}
	}

	return storageDiff, nil, nil
}

// resolveMoveConflicts filters or replaces targets whose names collide
// with existing destination children (#3159). "skip" drops the colliding
// target with a per-file ErrFileExisted; "overwrite" deletes the
// colliding destination object through the regular Delete path first.
func (f *DBFS) resolveMoveConflicts(ctx context.Context, targets []*File, navOf map[*File]Navigator, destination *File, isCopy bool, mode string, ae *serializer.AggregateError) ([]*File, map[Navigator][]*File) {
	surviving := make([]*File, 0, len(targets))
	group := make(map[Navigator][]*File)
	dstBase := destination.Uri(true)

	for _, target := range targets {
		dstName := target.Name()
		if !isCopy {
			if _, ok := target.Metadata()[MetadataRestoreUri]; ok {
				dstName = target.DisplayName()
			}
		}

		_, err := f.fileClient.GetChildFile(ctx, destination.Model, destination.OwnerID(), dstName, false)
		if err != nil && !ent.IsNotFound(err) {
			ae.Add(target.Uri(true).String(), fmt.Errorf("failed to check destination conflict: %w", err))
			continue
		}

		if err == nil {
			// Destination already holds an object under the same name.
			if mode == MoveConflictSkip {
				ae.Add(target.Uri(true).String(), fs.ErrFileExisted)
				continue
			}
			if _, _, err := f.Delete(ctx, []*fs.URI{dstBase.Join(dstName)}); err != nil {
				ae.Add(target.Uri(true).String(), err)
				continue
			}
		}

		surviving = append(surviving, target)
		if isCopy {
			nav := navOf[target]
			group[nav] = append(group[nav], target)
		}
	}

	return surviving, group
}
