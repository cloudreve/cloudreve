package dbfs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

// ReceiveFile is deliberately separate from MoveOrCopy: only administrators may
// use a source file belonging to another user, and the destination must belong
// to the authenticated administrator. File contents never leave their provider.
func (f *DBFS) ReceiveFile(ctx context.Context, fileID int, dst *fs.URI, name string) (fs.File, *fs.IndexDiff, error) {
	if f.user == nil || f.user.Edges.Group == nil ||
		!f.user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, nil, ErrPermissionDenied
	}
	uid := hashid.EncodeUserID(f.hasher, f.user.ID)
	if dst == nil || dst.FileSystem() != constants.FileSystemMy || dst.ID(uid) != uid {
		return nil, nil, ErrPermissionDenied
	}

	dstNav, err := f.getNavigator(ctx, dst, NavigatorCapabilityLockFile)
	if err != nil {
		return nil, nil, err
	}
	destination, err := f.getFileByPath(ctx, dstNav, dst)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get receiving folder: %w", err)
	}
	if destination.OwnerID() != f.user.ID || destination.InTrashBin() || !destination.CanHaveChildren() {
		return nil, nil, fs.ErrNotSupportedAction
	}

	traversed, err := f.TraverseFile(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to locate source file: %w", err)
	}
	source := traversed.(*File)
	sourceRoot := source.Root()
	defer sourceRoot.Recycle()
	// TraverseFile does not enter the source filesystem. Mark its linked root
	// explicitly so lock conflict reporting can safely describe this file.
	sourceRoot.IsUserRoot = true
	if source.InTrashBin() || source.Type() != types.FileTypeFile || source.IsSymbolic() {
		return nil, nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("source must be a live regular file"))
	}

	if name == "" {
		name = source.Name()
	}
	if err := validateFileName(name); err != nil {
		return nil, nil, fs.ErrIllegalObjectName.WithError(err)
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return nil, nil, fs.ErrIllegalObjectName.WithError(fmt.Errorf("name contains control characters"))
	}

	// Use the same lock namespace as normal uploads, deletes and moves. Hold both
	// source and destination locks until the copy transaction has committed.
	ls, err := f.acquireByPath(ctx, -1, f.user, false, fs.LockApp(fs.ApplicationMoveCopy),
		&LockByPath{source.Uri(true), source, source.Type(), ""},
		&LockByPath{destination.Uri(true).Join(name), destination, source.Type(), ""},
	)
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return nil, nil, err
	}

	fc, tx, txCtx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = inventory.Rollback(tx) }()
	txCtx = context.WithValue(txCtx, inventory.LoadFileEntity{}, true)
	txCtx = context.WithValue(txCtx, inventory.LoadFileMetadata{}, true)
	// Reload after acquiring locks, so a concurrent upload completed immediately
	// before the lock cannot leave us copying a stale or incomplete version.
	model, err := fc.GetByID(txCtx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if model.OwnerID != source.OwnerID() || model.FileChildren != source.Model.FileChildren ||
		model.Name != source.Name() || model.Type != int(types.FileTypeFile) || model.IsSymbolic {
		return nil, nil, fs.ErrModified
	}
	primaryComplete := false
	var size int64
	for _, blob := range model.Edges.Entities {
		if blob.UploadSessionID != nil {
			continue
		}
		size += blob.Size
		if blob.ID == model.PrimaryEntity && blob.Type == int(types.EntityTypeVersion) {
			primaryComplete = true
		}
	}
	if !primaryComplete {
		return nil, nil, fs.ErrNotSupportedAction.WithError(fmt.Errorf("source upload is not complete"))
	}
	capacity, err := f.Capacity(txCtx, f.user)
	if err != nil {
		return nil, nil, err
	}
	if err := f.validateUserCapacityRaw(txCtx, size, capacity); err != nil {
		return nil, nil, err
	}
	// Reserve quota as the first database write. This serializes concurrent
	// receives/uploads for the destination owner and rolls back with the copy.
	if err := tx.ReserveStorage(txCtx, f.userClient, f.user.ID, size, capacity.Total); err != nil {
		if errors.Is(err, inventory.ErrInsufficientCapacity) {
			return nil, nil, fs.ErrInsufficientCapacity
		}
		return nil, nil, err
	}

	// Set the destination name on a shallow copy before INSERT. This keeps the
	// source untouched and checks final-name conflicts in the copy transaction.
	copyModel := *model
	copyModel.Name = name
	copied, storageDiff, err := fc.Copy(txCtx, &inventory.CopyParameter{
		Files: []*ent.File{&copyModel},
		DstMap: map[int][]*ent.File{model.FileChildren: lo.Map(destination.AncestorsChain(), func(file *File, _ int) *ent.File {
			return file.Model
		})},
		ExcludedMetadataKeys: []string{FullTextIndexKey},
	})
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, nil, fs.ErrFileExisted.WithError(err)
		}
		return nil, nil, err
	}
	tx.AppendStorageDiff(storageDiff)
	newModel, err := fc.GetByID(txCtx, copied[model.ID][0].ID)
	if err != nil {
		return nil, nil, err
	}
	if err := inventory.CommitWithStorageDiff(txCtx, tx, f.l, f.userClient); err != nil {
		return nil, nil, err
	}

	received := newFile(destination, newModel)
	f.emitFileCreated(ctx, received)
	var diff *fs.IndexDiff
	for _, metadata := range model.Edges.Metadata {
		if metadata.Name == FullTextIndexKey {
			diff = &fs.IndexDiff{IndexToCopy: []fs.IndexDiffCopyDetails{{
				OriginalFileID: model.ID,
				FileID:         received.ID(),
				Uri:            *received.Uri(false),
				EntityID:       received.PrimaryEntityID(),
				OwnerID:        f.user.ID,
			}}}
			break
		}
	}
	return received, diff, nil
}
