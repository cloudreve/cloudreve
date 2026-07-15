package inventory

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
)

type TxOperator interface {
	SetClient(newClient *ent.Client) TxOperator
	GetClient() *ent.Client
}

type (
	Tx struct {
		tx          *ent.Tx
		parent      *Tx
		inherited   bool
		finished    bool
		storageDiff StorageDiff
	}

	// TxCtx is the context key for inherited transaction
	TxCtx struct{}
)

// ReserveStorage atomically adds `size` bytes to user `uid`'s storage inside
// this transaction, enforcing storage + size <= maxTotal at the database
// level (when maxTotal > 0). This is the correct API for grabbing the
// per-user storage quota:
//
//   - It MUST be called as the first write of the transaction (or at least
//     before any other WRITE to files/entities/etc.). That way the users-row
//     X lock is acquired before any INSERT into files/entities, so
//     concurrent uploads for the same owner serialize on the users row from
//     the start and cannot deadlock via foreign-key S locks on the shared
//     parent folder / storage-policy rows.
//   - A matching compensating negative diff is appended to storageDiff so
//     the pre-reservation is netted against the positive diff emitted later
//     by CreateEntity / fc.Copy / etc. No amount is double-applied.
//   - On failure (quota exceeded or any other DB error) the caller MUST
//     Rollback the transaction: any placeholder writes done afterwards
//     would otherwise be committed without a matching storage reservation.
//
// It is safe to call this from an inherited-tx wrapper (returned by
// InheritTx) — AppendStorageDiff routes to the root tx.
func (t *Tx) ReserveStorage(ctx context.Context, uc UserClient, uid int, size, maxTotal int64) error {
	if size <= 0 {
		return nil
	}

	txUc, _ := InheritTx(ctx, uc)
	if err := txUc.ReserveStorage(ctx, uid, size, maxTotal); err != nil {
		return err
	}

	// Net out the pre-reservation against any positive diff that
	// CreateEntity / fc.Copy / ... will append later in the same tx.
	t.AppendStorageDiff(StorageDiff{uid: -size})
	return nil
}

// AppendStorageDiff appends the given storage diff to the transaction.
func (t *Tx) AppendStorageDiff(diff StorageDiff) {
	root := t
	for root.inherited {
		root = root.parent
	}

	if root.storageDiff == nil {
		root.storageDiff = diff
	} else {
		root.storageDiff.Merge(diff)
	}
}

// WithTx wraps the given inventory client with a transaction.
func WithTx[T TxOperator](ctx context.Context, c T) (T, *Tx, context.Context, error) {
	var txClient *ent.Client
	var txWrapper *Tx

	if txInherited, ok := ctx.Value(TxCtx{}).(*Tx); ok && !txInherited.finished {
		txWrapper = &Tx{inherited: true, tx: txInherited.tx, parent: txInherited}
	} else {
		tx, err := c.GetClient().Tx(ctx)
		if err != nil {
			return c, nil, ctx, fmt.Errorf("failed to create transaction: %w", err)
		}

		txWrapper = &Tx{inherited: false, tx: tx}
		ctx = context.WithValue(ctx, TxCtx{}, txWrapper)
	}

	txClient = txWrapper.tx.Client()
	return c.SetClient(txClient).(T), txWrapper, ctx, nil
}

// InheritTx wraps the given inventory client with a transaction.
// If the transaction is already in the context, it will be inherited.
// Otherwise, original client will be returned.
func InheritTx[T TxOperator](ctx context.Context, c T) (T, *Tx) {
	var txClient *ent.Client
	var txWrapper *Tx

	if txInherited, ok := ctx.Value(TxCtx{}).(*Tx); ok && !txInherited.finished {
		txWrapper = &Tx{inherited: true, tx: txInherited.tx, parent: txInherited}
		txClient = txWrapper.tx.Client()
		return c.SetClient(txClient).(T), txWrapper
	}

	return c, nil
}

func Rollback(tx *Tx) error {
	if !tx.inherited {
		tx.finished = true
		return tx.tx.Rollback()
	}

	return nil
}

func commit(tx *Tx) (bool, error) {
	if !tx.inherited {
		tx.finished = true
		return true, tx.tx.Commit()
	}
	return false, nil
}

func Commit(tx *Tx) error {
	_, err := commit(tx)
	return err
}

// CommitWithStorageDiff commits the transaction and applies the accumulated
// storage diff. Only the outermost (non-inherited) transaction performs the
// commit and the storage mutation.
//
// Quota enforcement is done UPFRONT via Tx.ReserveStorage, not here. This
// function's role is to (a) commit the tx, and (b) apply any residual
// storage diff produced by CreateEntity / CapEntities / etc. Tx.ReserveStorage
// pre-emits a matching negative diff, so pre-reserved sizes are netted out
// against the positive diff CreateEntity appends later. Residual is applied
// AFTER commit via ApplyStorageDiff (auto-commit, safe to retry on
// deadlock).
func CommitWithStorageDiff(ctx context.Context, tx *Tx, l logging.Logger, uc UserClient) error {
	committed, err := commit(tx)
	if err != nil {
		return err
	}

	if !committed {
		return nil
	}

	if err := uc.ApplyStorageDiff(ctx, tx.storageDiff); err != nil {
		l.Error("Failed to apply storage diff", "error", err)
	}

	return nil
}
