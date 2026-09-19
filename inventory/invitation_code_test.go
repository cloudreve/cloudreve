package inventory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/enttest"
	"github.com/stretchr/testify/require"
)

func newInvitationCodeClient(t *testing.T) (*ent.Client, InvitationCodeClient) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return client, NewInvitationCodeClient(client, "sqlite3", nil)
}

func TestInvitationCodeConsume(t *testing.T) {
	ctx := context.Background()
	client, c := newInvitationCodeClient(t)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	client.InvitationCode.Create().SetCode("expired").SetExpiresAt(past).SaveX(ctx)
	client.InvitationCode.Create().SetCode("once").SetMaxUses(1).SaveX(ctx)
	client.InvitationCode.Create().SetCode("group").SetMaxUses(0).SetGroupID(7).SaveX(ctx)

	_, err := c.Consume(ctx, "missing")
	require.ErrorIs(t, err, ErrInvitationCodeNotFound)

	_, err = c.Consume(ctx, "expired")
	require.ErrorIs(t, err, ErrInvitationCodeExpired)

	// Unlimited code returns its group assignment.
	got, err := c.Consume(ctx, "group")
	require.NoError(t, err)
	require.Equal(t, 7, got.GroupID)

	// Single-use code exhausts.
	_, err = c.Consume(ctx, "once")
	require.NoError(t, err)
	_, err = c.Consume(ctx, "once")
	require.ErrorIs(t, err, ErrInvitationCodeExhausted)

	// A code that expires later but is already full reports exhausted.
	client.InvitationCode.Create().SetCode("full").SetMaxUses(1).SetUsedCount(1).SetExpiresAt(future).SaveX(ctx)
	_, err = c.Consume(ctx, "full")
	require.ErrorIs(t, err, ErrInvitationCodeExhausted)
}

func TestInvitationCodeConsumeConcurrent(t *testing.T) {
	ctx := context.Background()
	_, c := newInvitationCodeClient(t)
	_, err := c.Create(ctx, &CreateInvitationCodeParams{Code: "race", MaxUses: 1})
	require.NoError(t, err)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Consume(ctx, "race")
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		} else {
			require.ErrorIs(t, err, ErrInvitationCodeExhausted)
		}
	}
	require.Equal(t, 1, succeeded, "exactly one consume may win the last use")
}

func TestInvitationCodeConsumeRollback(t *testing.T) {
	ctx := context.Background()
	client, c := newInvitationCodeClient(t)
	_, err := c.Create(ctx, &CreateInvitationCodeParams{Code: "rollback", MaxUses: 5})
	require.NoError(t, err)

	txClient, tx, txCtx, err := WithTx(ctx, c)
	require.NoError(t, err)
	_, err = txClient.Consume(txCtx, "rollback")
	require.NoError(t, err)
	require.NoError(t, Rollback(tx))

	record := client.InvitationCode.Query().FirstX(ctx)
	require.Equal(t, 0, record.UsedCount, "rolled-back consume must not persist the use")
}

func TestInvitationCodeList(t *testing.T) {
	ctx := context.Background()
	_, c := newInvitationCodeClient(t)

	for _, code := range []string{"alpha-1", "alpha-2", "beta-1"} {
		_, err := c.Create(ctx, &CreateInvitationCodeParams{Code: code})
		require.NoError(t, err)
	}

	res, err := c.List(ctx, &ListInvitationCodeArgs{
		PaginationArgs: &PaginationArgs{PageSize: 10},
		Keyword:        "alpha",
	})
	require.NoError(t, err)
	require.Len(t, res.Codes, 2)
	require.Empty(t, res.NextPageToken)

	// Newest first.
	require.Equal(t, "alpha-2", res.Codes[0].Code)
}
