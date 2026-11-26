package eventhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
)

type Subscriber interface {
	ID() string
	Ch() chan *Event
	Publish(evt Event)
	Stop()
	Buffer() []*Event
	// Owner returns the owner of the subscriber.
	Owner(ctx context.Context) (*ent.User, error)
	// Online returns whether the subscriber is online.
	Online() bool
	// OfflineSince returns when the subscriber went offline.
	// Returns zero time if the subscriber is online.
	OfflineSince() time.Time
}

const (
	userCacheTTL        = 1 * time.Hour
	subscriberIOTimeout = 30 * time.Second
)

type subscriber struct {
	mu            sync.Mutex
	userClient    inventory.UserClient
	fsEventClient inventory.FsEventClient

	id  string
	uid int
	ch  chan *Event

	// Online status
	online       bool
	offlineSince time.Time

	// Debounce buffer for pending events
	buffer        []*Event
	timer         *time.Timer
	offlineMaxAge time.Duration
	debounceDelay time.Duration

	// Owner info
	ownerCached  *ent.User
	cachedAt     time.Time
	ownerVersion uint64

	// Close signal
	closed   bool
	closedCh chan struct{}
}

func newSubscriber(ctx context.Context, id string, userClient inventory.UserClient, fsEventClient inventory.FsEventClient, maxAge, debounceDelay time.Duration) (*subscriber, error) {
	user := inventory.UserFromContext(ctx)
	if user == nil || inventory.IsAnonymousUser(user) {
		return nil, errors.New("user not found")
	}

	return &subscriber{
		id:            id,
		ch:            make(chan *Event, bufSize),
		userClient:    userClient,
		fsEventClient: fsEventClient,

		ownerCached:   user,
		uid:           user.ID,
		cachedAt:      time.Now(),
		online:        true,
		closedCh:      make(chan struct{}),
		offlineMaxAge: maxAge,
		debounceDelay: debounceDelay,
	}, nil
}

func (s *subscriber) ID() string {
	return s.id
}

func (s *subscriber) Ch() chan *Event {
	return s.ch
}

func (s *subscriber) Online() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.online
}

func (s *subscriber) OfflineSince() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offlineSince
}

func (s *subscriber) Owner(ctx context.Context) (*ent.User, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrEventHubClosed
	}
	if s.ownerCached != nil && time.Since(s.cachedAt) <= userCacheTTL {
		owner := s.ownerCached
		s.mu.Unlock()
		return owner, nil
	}
	version := s.ownerVersion
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, subscriberIOTimeout)
	defer cancel()
	users, tx := inventory.InheritTx(ctx, s.userClient)
	owner, err := users.GetLoginUserByID(ctx, s.uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get login user: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrEventHubClosed
	}
	// Do not retain transaction-local data or undo a concurrent reconnect's invalidation.
	if tx == nil && version == s.ownerVersion && (s.ownerCached == nil || time.Since(s.cachedAt) > userCacheTTL) {
		s.ownerCached = owner
		s.cachedAt = time.Now()
	}
	return owner, nil
}

// Publish adds an event to the buffer and starts/resets the debounce timer.
// Events will be flushed to the channel after the debounce delay.
// If the subscriber is offline, events are kept in the buffer only.
func (s *subscriber) Publish(evt Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.publishLocked(evt)
}

// publishLocked adds an event to the buffer and manages the debounce timer.
// Caller must hold s.mu.
func (s *subscriber) publishLocked(evt Event) {
	// Add event to buffer
	s.buffer = append(s.buffer, &evt)

	// Reset or start the debounce timer
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(s.debounceDelay, s.flush)
}

// flush sends all buffered events to the channel.
// Called by the debounce timer.
func (s *subscriber) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flushLocked(context.Background())
}

// flushLocked sends all buffered events to the channel.
// Caller must hold s.mu.
func (s *subscriber) flushLocked(ctx context.Context) {
	if len(s.buffer) == 0 || s.closed {
		return
	}

	if !s.online {
		ctx, cancel := context.WithTimeout(ctx, subscriberIOTimeout)
		defer cancel()
		// The subscriber ID is already bound to this immutable user ID.
		if err := s.fsEventClient.Create(ctx, s.uid, uuid.FromStringOrNil(s.id), lo.Map(s.buffer, func(item *Event, index int) string {
			res, _ := json.Marshal(item)
			return string(res)
		})...); err != nil {
			logging.FromContext(ctx).Error("Failed to persist subscriber events: %s", err)
			return
		}
	} else {
		// TODO: implement event merging logic here
		// For now, send all buffered events individually
		debouncedEvents := DebounceEvents(s.buffer)
		for _, evt := range debouncedEvents {
			select {
			case s.ch <- evt:
			default:
				// Non-blocking send; drop if subscriber is slow
			}
		}
	}

	// Clear the buffer
	s.buffer = nil
	s.timer = nil
}

// Stop cancels any pending debounce timer and flushes remaining events.
// Should be called before closing the subscriber.
func (s *subscriber) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	// Flush any remaining events before stopping
	s.flushLocked(context.Background())
}

// setOnline marks the subscriber as online and flushes any buffered events.
func (s *subscriber) setOnline(ctx context.Context) {
	l := logging.FromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.online = true
	s.ownerCached = nil
	s.ownerVersion++
	s.offlineSince = time.Time{}

	// Retrieve events from inventory
	ctx, cancel := context.WithTimeout(ctx, subscriberIOTimeout)
	defer cancel()
	events, err := s.fsEventClient.TakeBySubscriber(ctx, uuid.FromStringOrNil(s.id), s.uid)
	if err != nil {
		l.Error("Failed to get events from inventory: %s", err)
		return
	}

	// Append events to buffer
	for _, event := range events {
		var eventParsed Event
		err := json.Unmarshal([]byte(event.Event), &eventParsed)
		if err != nil {
			l.Error("Failed to unmarshal event: %s", err)
			continue
		}
		s.buffer = append(s.buffer, &eventParsed)
	}

	// Flush buffered events if any
	if len(s.buffer) > 0 {
		if s.timer != nil {
			s.timer.Stop()
		}
		s.timer = time.AfterFunc(s.debounceDelay, s.flush)
	}
}

// setOffline marks the subscriber as offline.
func (s *subscriber) setOffline() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.online = false
	s.offlineSince = time.Now()

	// Stop the timer, events will be kept in buffer
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	// flush the buffer
	s.flushLocked(context.Background())
}

// close permanently closes the subscriber.
func (s *subscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	// Delete the FsEvent
	ctx, cancel := context.WithTimeout(context.Background(), subscriberIOTimeout)
	defer cancel()
	if err := s.fsEventClient.DeleteBySubscriber(ctx, uuid.FromStringOrNil(s.id)); err != nil {
		logging.FromContext(ctx).Error("Failed to delete subscriber events: %s", err)
	}

	// Signal close and close the channel
	close(s.closedCh)
	close(s.ch)
	s.buffer = nil
}

// isClosed returns whether the subscriber is closed.
func (s *subscriber) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// shouldExpire returns whether the subscriber should be expired (offline for too long).
func (s *subscriber) shouldExpire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.online && !s.offlineSince.IsZero() && time.Since(s.offlineSince) > s.offlineMaxAge
}

// Buffer returns a copy of the current buffered events.
// Useful for debugging or implementing custom merging logic.
func (s *subscriber) Buffer() []*Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.buffer) == 0 {
		return nil
	}

	// Return a copy to avoid data races
	buf := make([]*Event, len(s.buffer))
	copy(buf, s.buffer)
	return buf
}
