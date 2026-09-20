package inventory

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/ssobinding"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
)

type (
	SsoBindingClient interface {
		TxOperator
		// Get returns the binding for (provider, subject).
		Get(ctx context.Context, provider, subject string) (*ent.SsoBinding, error)
		// ListByUser returns all bindings owned by the given user.
		ListByUser(ctx context.Context, userID int) ([]*ent.SsoBinding, error)
		// Bind links (provider, subject) to a user. An existing binding for
		// the same (user, provider) is moved to the new subject; a binding
		// for the same (provider, subject) owned by a different user is
		// treated as a conflict.
		Bind(ctx context.Context, userID int, provider, subject string) (*ent.SsoBinding, error)
		// Unbind removes the user's binding at the given provider. Returns
		// nil when no binding exists.
		Unbind(ctx context.Context, userID int, provider string) error
	}
)

// SsoProvider enumerates the external sign-in providers that can appear in
// sso_binding rows.
const (
	SsoProviderQQ     = "qq"
	SsoProviderWeChat = "wechat"
)

var (
	ErrSsoBindingConflict = &BindingConflictError{}
)

type BindingConflictError struct{}

func (e *BindingConflictError) Error() string {
	return "external account is already linked to a different user"
}

func NewSsoBindingClient(client *ent.Client, dbType conf.DBType) SsoBindingClient {
	return &ssoBindingClient{
		client:      client,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type ssoBindingClient struct {
	maxSQlParam int
	client      *ent.Client
}

func (c *ssoBindingClient) SetClient(newClient *ent.Client) TxOperator {
	return &ssoBindingClient{client: newClient, maxSQlParam: c.maxSQlParam}
}

func (c *ssoBindingClient) GetClient() *ent.Client {
	return c.client
}

func (c *ssoBindingClient) Get(ctx context.Context, provider, subject string) (*ent.SsoBinding, error) {
	return c.client.SsoBinding.Query().
		Where(ssobinding.Provider(provider), ssobinding.Subject(subject)).
		Only(ctx)
}

func (c *ssoBindingClient) ListByUser(ctx context.Context, userID int) ([]*ent.SsoBinding, error) {
	return c.client.SsoBinding.Query().
		Where(ssobinding.UserID(userID)).
		All(ctx)
}

func (c *ssoBindingClient) Bind(ctx context.Context, userID int, provider, subject string) (*ent.SsoBinding, error) {
	if existing, err := c.Get(ctx, provider, subject); err == nil {
		if existing.UserID != userID {
			return nil, ErrSsoBindingConflict
		}
		return existing, nil
	} else if !ent.IsNotFound(err) {
		return nil, err
	}

	// One binding per (user, provider): re-bind moves the row to the new
	// subject instead of stacking a second account.
	existing, err := c.client.SsoBinding.Query().
		Where(ssobinding.UserID(userID), ssobinding.Provider(provider)).
		Only(ctx)
	if err == nil {
		return c.client.SsoBinding.UpdateOne(existing).
			SetSubject(subject).
			Save(ctx)
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}

	return c.client.SsoBinding.Create().
		SetUserID(userID).
		SetProvider(provider).
		SetSubject(subject).
		Save(ctx)
}

func (c *ssoBindingClient) Unbind(ctx context.Context, userID int, provider string) error {
	_, err := c.client.SsoBinding.Delete().
		Where(ssobinding.UserID(userID), ssobinding.Provider(provider)).
		Exec(ctx)
	return err
}
