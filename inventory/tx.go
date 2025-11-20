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

// CommitWithStorageDiff commits the transaction and applies the storage diff, only if the transaction is not inherited.
func CommitWithStorageDiff(ctx context.Context, tx *Tx, l logging.Logger, uc UserClient) error {
	commited, err := commit(tx)
	if err != nil {
		return err
	}

	if !commited {
		return nil
	}

	if err := uc.ApplyStorageDiff(ctx, tx.storageDiff); err != nil {
		l.Error("Failed to apply storage diff", "error", err)
	}

	return nil
}
