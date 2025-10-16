package dbfs

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/samber/lo"
)

func (f *DBFS) PatchProps(ctx context.Context, uri *fs.URI, props *types.FileProps, delete bool) error {
	navigator, err := f.getNavigator(ctx, uri, NavigatorCapabilityModifyProps, NavigatorCapabilityLockFile)
	if err != nil {
		return err
	}

	target, err := f.getFileByPath(ctx, navigator, uri)
	if err != nil {
		return fmt.Errorf("failed to get target file: %w", err)
	}

	if target.OwnerID() != f.user.ID && !f.user.EnforceGroupPermission(types.GroupPermissionIsAdmin) {
		return fs.ErrOwnerOnly.WithError(fmt.Errorf("only file owner can modify file props"))
	}

	// Lock target
	lr := &LockByPath{target.Uri(true), target, target.Type(), ""}
	ls, err := f.acquireByPath(ctx, -1, f.user, true, fs.LockApp(fs.ApplicationUpdateMetadata), lr)
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return err
	}

	currentProps := target.Model.Props
	if currentProps == nil {
		currentProps = &types.FileProps{}
	}

	if props.View != nil {
		if delete {
			currentProps.View = nil
		} else {
			currentProps.View = props.View
		}
	}

	if _, err := f.fileClient.UpdateProps(ctx, target.Model, currentProps); err != nil {
		return serializer.NewError(serializer.CodeDBError, "failed to update file props", err)
	}

	return nil
}

func (f *DBFS) PatchMetadata(ctx context.Context, path []*fs.URI, metas ...fs.MetadataPatch) error {
	ae := serializer.NewAggregateError()
	targets := make([]*File, 0, len(path))
	for _, p := range path {
		navigator, err := f.getNavigator(ctx, p, NavigatorCapabilityUpdateMetadata, NavigatorCapabilityLockFile)
		if err != nil {
			ae.Add(p.String(), err)
			continue
		}

		target, err := f.getFileByPath(ctx, navigator, p)
		if err != nil {
			ae.Add(p.String(), fmt.Errorf("failed to get target file: %w", err))
			continue
		}

		// Require Update permission
		if _, ok := ctx.Value(ByPassOwnerCheckCtxKey{}).(bool); !ok && target.OwnerID() != f.user.ID {
			return fs.ErrOwnerOnly.WithError(fmt.Errorf("permission denied"))
		}

		if target.IsRootFolder() {
			ae.Add(p.String(), fs.ErrNotSupportedAction.WithError(fmt.Errorf("cannot move root folder")))
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
	ls, err := f.acquireByPath(ctx, -1, f.user, true, fs.LockApp(fs.ApplicationUpdateMetadata), lockTargets...)
	defer func() { _ = f.Release(ctx, ls) }()
	if err != nil {
		return err
	}

	metadataMap := make(map[string]string)
	privateMap := make(map[string]bool)
	deleted := make([]string, 0)
	updateModifiedAt := false
	for _, meta := range metas {
		if meta.Remove {
			deleted = append(deleted, meta.Key)
			continue
		}
		metadataMap[meta.Key] = meta.Value
		if meta.Private {
			privateMap[meta.Key] = meta.Private
		}
		if meta.UpdateModifiedAt {
			updateModifiedAt = true
		}
	}

	fc, tx, ctx, err := inventory.WithTx(ctx, f.fileClient)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	for _, target := range targets {
		if err := fc.UpsertMetadata(ctx, target.Model, metadataMap, privateMap); err != nil {
			_ = inventory.Rollback(tx)
			return fmt.Errorf("failed to upsert metadata: %w", err)
		}

		if len(deleted) > 0 {
			if err := fc.RemoveMetadata(ctx, target.Model, deleted...); err != nil {
				_ = inventory.Rollback(tx)
				return fmt.Errorf("failed to remove metadata: %w", err)
			}
		}

		if updateModifiedAt {
			if err := fc.UpdateModifiedAt(ctx, target.Model, time.Now()); err != nil {
				_ = inventory.Rollback(tx)
				return fmt.Errorf("failed to update file modified at: %w", err)
			}
		}
	}

	if err := inventory.Commit(tx); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to commit metadata change", err)
	}

	return ae.Aggregate()
}
