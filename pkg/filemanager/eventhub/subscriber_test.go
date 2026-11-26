package eventhub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	_ "github.com/cloudreve/Cloudreve/v4/ent/runtime"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

type subscriberTestUsers struct {
	inventory.UserClient
	get func(context.Context, int) (*ent.User, error)
}

func (u subscriberTestUsers) GetLoginUserByID(ctx context.Context, id int) (*ent.User, error) {
	return u.get(ctx, id)
}

type subscriberTestEvents struct {
	inventory.FsEventClient
	create func(context.Context, int, uuid.UUID, ...string) error
}

func (e subscriberTestEvents) Create(ctx context.Context, uid int, id uuid.UUID, events ...string) error {
	return e.create(ctx, uid, id, events...)
}
func (subscriberTestEvents) DeleteBySubscriber(context.Context, uuid.UUID) error { return nil }
func (subscriberTestEvents) TakeBySubscriber(context.Context, uuid.UUID, int) ([]*ent.FsEvent, error) {
	return nil, nil
}

func TestOwnerLookupDoesNotHoldSubscriberLock(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	releaseOwner := sync.OnceFunc(func() { close(release) })
	u := &ent.User{ID: 1}
	users := subscriberTestUsers{get: func(ctx context.Context, _ int) (*ent.User, error) {
		close(entered)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return u, nil
		}
	}}
	subCtx := context.WithValue(context.Background(), inventory.UserCtx{}, u)
	s, err := newSubscriber(subCtx, uuid.Must(uuid.NewV4()).String(), users, subscriberTestEvents{}, time.Hour, time.Hour)
	require.NoError(t, err)
	s.ownerCached = nil
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		releaseOwner()
		s.close()
	}()
	done := make(chan error, 1)
	go func() {
		_, err := s.Owner(ctx)
		done <- err
	}()
	<-entered

	available := make(chan bool, 1)
	go func() {
		s.Publish(Event{Type: EventTypeCreate, FileID: "file"})
		available <- s.Online()
	}()
	select {
	case online := <-available:
		require.True(t, online)
	case <-time.After(time.Second):
		t.Fatal("owner database lookup holds the subscriber mutex")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("owner lookup did not honor cancellation")
	}
	require.Nil(t, s.ownerCached)
}

func TestOwnerLookupPreservesReconnectInvalidation(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	releaseOwner := sync.OnceFunc(func() { close(release) })
	u := &ent.User{ID: 1, Nick: "old"}
	fresh := &ent.User{ID: 1, Nick: "fresh"}
	calls := 0
	users := subscriberTestUsers{get: func(context.Context, int) (*ent.User, error) {
		calls++
		if calls == 1 {
			close(entered)
			<-release
			return u, nil
		}
		return fresh, nil
	}}
	subCtx := context.WithValue(context.Background(), inventory.UserCtx{}, u)
	s, err := newSubscriber(subCtx, uuid.Must(uuid.NewV4()).String(), users, subscriberTestEvents{}, time.Hour, time.Hour)
	require.NoError(t, err)
	defer func() {
		releaseOwner()
		s.close()
	}()
	s.ownerCached = nil
	done := make(chan error, 1)
	go func() {
		_, err := s.Owner(context.Background())
		done <- err
	}()
	<-entered
	reconnected := make(chan struct{})
	go func() {
		s.setOnline(context.Background())
		close(reconnected)
	}()
	select {
	case <-reconnected:
	case <-time.After(time.Second):
		t.Fatal("owner lookup blocks subscriber reconnect")
	}
	releaseOwner()
	require.NoError(t, <-done)
	require.Nil(t, s.ownerCached)
	owner, err := s.Owner(context.Background())
	require.NoError(t, err)
	require.Same(t, fresh, owner)
	require.Equal(t, 2, calls)
}

func TestOwnerDoesNotCacheTransactionState(t *testing.T) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:subscriber-owner-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Schema.Create(ctx))
	group, err := client.Group.Create().SetName("test").SetPermissions(&boolset.BooleanSet{}).Save(ctx)
	require.NoError(t, err)
	u, err := client.User.Create().SetEmail("owner@example.com").SetNick("original").SetGroupID(group.ID).Save(ctx)
	require.NoError(t, err)
	users := inventory.NewUserClient(client)
	s, err := newSubscriber(context.WithValue(ctx, inventory.UserCtx{}, u), uuid.Must(uuid.NewV4()).String(),
		users, subscriberTestEvents{}, time.Hour, time.Hour)
	require.NoError(t, err)
	defer s.close()
	s.ownerCached = nil

	txUsers, tx, txCtx, err := inventory.WithTx(ctx, users)
	require.NoError(t, err)
	defer inventory.Rollback(tx)
	_, err = txUsers.UpdateNickname(txCtx, u, "uncommitted")
	require.NoError(t, err)
	owner, err := s.Owner(txCtx)
	require.NoError(t, err, "owner lookup must reuse the single transaction connection")
	require.Equal(t, "uncommitted", owner.Nick)
	require.Nil(t, s.ownerCached)
	require.NoError(t, inventory.Rollback(tx))

	owner, err = s.Owner(ctx)
	require.NoError(t, err)
	require.Equal(t, "original", owner.Nick)
	require.Same(t, owner, s.ownerCached)
}

func TestOfflineFlushPreservesEventsOnFailure(t *testing.T) {
	u := &ent.User{ID: 7}
	attempts := 0
	events := subscriberTestEvents{create: func(ctx context.Context, uid int, _ uuid.UUID, batch ...string) error {
		require.Equal(t, u.ID, uid)
		require.Len(t, batch, 1)
		_, bounded := ctx.Deadline()
		require.True(t, bounded)
		attempts++
		if attempts == 1 {
			return errors.New("temporary database failure")
		}
		return nil
	}}
	s, err := newSubscriber(context.WithValue(context.Background(), inventory.UserCtx{}, u),
		uuid.Must(uuid.NewV4()).String(), subscriberTestUsers{}, events, time.Hour, time.Hour)
	require.NoError(t, err)
	defer s.close()
	s.ownerCached = nil
	s.Publish(Event{Type: EventTypeCreate, FileID: "file"})
	s.setOffline()
	require.Len(t, s.Buffer(), 1)
	s.Stop()
	require.Empty(t, s.Buffer())
	require.Equal(t, 2, attempts)
}
