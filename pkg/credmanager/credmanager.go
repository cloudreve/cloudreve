package credmanager

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
)

type (
	// CredManager is a centralized for all Oauth tokens that requires periodic refresh
	// It is primarily used by OneDrive storage policy.
	CredManager interface {
		// Obtain gets a credential from the manager, refresh it if it's expired
		Obtain(ctx context.Context, key string) (Credential, error)
		// Upsert inserts or updates a credential in the manager
		Upsert(ctx context.Context, cred ...Credential) error
		RefreshAll(ctx context.Context)
	}

	Credential interface {
		String() string
		Refresh(ctx context.Context) (Credential, error)
		Key() string
		Expiry() time.Time
		RefreshedAt() *time.Time
	}
)

func init() {
	gob.Register(CredentialResponse{})
}

func New(kv cache.Driver) CredManager {
	return &credManager{
		kv:    kv,
		locks: make(map[string]*sync.Mutex),
	}
}

type (
	credManager struct {
		kv cache.Driver
		mu sync.RWMutex

		locks map[string]*sync.Mutex
	}
)

var (
	ErrNotFound = errors.New("credential not found")
)

func (m *credManager) Upsert(ctx context.Context, cred ...Credential) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	l := logging.FromContext(ctx)
	for _, c := range cred {
		l.Info("CredManager: Upsert credential for key %q...", c.Key())
		if err := m.kv.Set(c.Key(), c, 0); err != nil {
			return fmt.Errorf("failed to update credential in KV for key %q: %w", c.Key(), err)
		}

		if _, ok := m.locks[c.Key()]; !ok {
			m.locks[c.Key()] = &sync.Mutex{}
		}
	}

	return nil
}

func (m *credManager) Obtain(ctx context.Context, key string) (Credential, error) {
	m.mu.RLock()
	itemRaw, ok := m.kv.Get(key)
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("credential not found for key %q: %w", key, ErrNotFound)
	}

	l := logging.FromContext(ctx)

	item := itemRaw.(Credential)
	if _, ok := m.locks[key]; !ok {
		m.locks[key] = &sync.Mutex{}
	}
	m.locks[key].Lock()
	defer m.locks[key].Unlock()
	m.mu.RUnlock()

	if item.Expiry().After(time.Now()) {
		// Credential is still valid
		return item, nil
	}

	// Credential is expired, refresh it
	l.Info("Refreshing credential for key %q...", key)
	newCred, err := item.Refresh(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh credential for key %q: %w", key, err)
	}

	l.Info("New credential for key %q is obtained, expire at %s", key, newCred.Expiry().String())
	if err := m.kv.Set(key, newCred, 0); err != nil {
		return nil, fmt.Errorf("failed to update credential in KV for key %q: %w", key, err)
	}

	return newCred, nil
}

func (m *credManager) RefreshAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	l := logging.FromContext(ctx)
	for key := range m.locks {
		l.Info("Refreshing credential for key %q...", key)
		m.locks[key].Lock()
		defer m.locks[key].Unlock()

		itemRaw, ok := m.kv.Get(key)
		if !ok {
			l.Warning("Credential not found for key %q", key)
			continue
		}

		item := itemRaw.(Credential)
		newCred, err := item.Refresh(ctx)
		if err != nil {
			l.Warning("Failed to refresh credential for key %q: %s", key, err)
			continue
		}

		l.Info("New credential for key %q is obtained, expire at %s", key, newCred.Expiry().String())
		if err := m.kv.Set(key, newCred, 0); err != nil {
			l.Warning("Failed to update credential in KV for key %q: %s", key, err)
		}
	}
}

type (
	slaveCredManager struct {
		kv     cache.Driver
		client request.Client
	}

	CredentialResponse struct {
		Token    string    `json:"token"`
		ExpireAt time.Time `json:"expire_at"`
	}
)

func NewSlaveManager(kv cache.Driver, config conf.ConfigProvider) CredManager {
	return &slaveCredManager{
		kv: kv,
		client: request.NewClient(
			config,
			request.WithCredential(auth.HMACAuth{SecretKey: []byte(config.Slave().Secret)}, int64(config.Slave().SignatureTTL)),
		),
	}
}

func (c CredentialResponse) String() string {
	return c.Token
}

func (c CredentialResponse) Refresh(ctx context.Context) (Credential, error) {
	return c, nil
}

func (c CredentialResponse) Key() string {
	return ""
}

func (c CredentialResponse) Expiry() time.Time {
	return c.ExpireAt
}

func (c CredentialResponse) RefreshedAt() *time.Time {
	return nil
}

func (m *slaveCredManager) Upsert(ctx context.Context, cred ...Credential) error {
	return nil
}

func (m *slaveCredManager) Obtain(ctx context.Context, key string) (Credential, error) {
	itemRaw, ok := m.kv.Get(key)
	if !ok {
		return m.requestCredFromMaster(ctx, key)
	}

	return itemRaw.(Credential), nil
}

// No op on slave node
func (m *slaveCredManager) RefreshAll(ctx context.Context) {}

func (m *slaveCredManager) requestCredFromMaster(ctx context.Context, key string) (Credential, error) {
	l := logging.FromContext(ctx)
	l.Info("SlaveCredManager: Requesting credential for key %q from master...", key)

	requestDst := routes.MasterGetCredentialUrl(cluster.MasterSiteUrlFromContext(ctx), key)
	resp, err := m.client.Request(
		http.MethodGet,
		requestDst.String(),
		nil,
		request.WithContext(ctx),
		request.WithLogger(l),
		request.WithSlaveMeta(cluster.NodeIdFromContext(ctx)),
		request.WithCorrelationID(),
	).CheckHTTPResponse(http.StatusOK).DecodeResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to request credential from master: %w", err)
	}

	cred := &CredentialResponse{}
	resp.DecodeGob(&cred)

	if err := m.kv.Set(key, *cred, max(int(time.Until(cred.Expiry()).Seconds()), 1)); err != nil {
		return nil, fmt.Errorf("failed to update credential in KV for key %q: %w", key, err)
	}

	return cred, nil
}
