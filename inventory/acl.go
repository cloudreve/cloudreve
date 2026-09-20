package inventory

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/aclentry"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/samber/lo"
)

type (
	AclClient interface {
		TxOperator
		// List returns all ACL entries for a file, ordered by subject type.
		List(ctx context.Context, fileID int) ([]*ent.AclEntry, error)
		// Upsert creates or updates the entry for (file, subject).
		Upsert(ctx context.Context, params *UpsertAclEntryParams) (*ent.AclEntry, error)
		// Delete removes an ACL entry scoped to the given file.
		Delete(ctx context.Context, fileID, id int) error
		// EffectivePermissions unions the permission bits of all entries on
		// fileID matching the acting user. Anonymous users (id 0) only match
		// the anonymous tier; authenticated users match their explicit user
		// and group rows plus the everyone tier. Returns nil when no entry
		// matches — callers then fall back to share-level defaults.
		EffectivePermissions(ctx context.Context, fileID int, user *ent.User) (*boolset.BooleanSet, error)
	}

	UpsertAclEntryParams struct {
		FileID      int
		SubjectType aclentry.SubjectType
		SubjectID   int
		Permissions *boolset.BooleanSet
	}
)

func NewAclClient(client *ent.Client, dbType conf.DBType) AclClient {
	return &aclClient{
		client:      client,
		maxSQlParam: sqlParamLimit(dbType),
	}
}

type aclClient struct {
	maxSQlParam int
	client      *ent.Client
}

func (c *aclClient) SetClient(newClient *ent.Client) TxOperator {
	return &aclClient{client: newClient, maxSQlParam: c.maxSQlParam}
}

func (c *aclClient) GetClient() *ent.Client {
	return c.client
}

func (c *aclClient) List(ctx context.Context, fileID int) ([]*ent.AclEntry, error) {
	return c.client.AclEntry.Query().
		Where(aclentry.FileID(fileID)).
		Order(aclentry.BySubjectType(), aclentry.BySubjectID()).
		All(ctx)
}

func (c *aclClient) Upsert(ctx context.Context, params *UpsertAclEntryParams) (*ent.AclEntry, error) {
	if params.Permissions == nil {
		params.Permissions = &boolset.BooleanSet{}
	}

	id, err := c.client.AclEntry.Create().
		SetFileID(params.FileID).
		SetSubjectType(params.SubjectType).
		SetSubjectID(params.SubjectID).
		SetPermissions(params.Permissions).
		OnConflictColumns(aclentry.FieldFileID, aclentry.FieldSubjectType, aclentry.FieldSubjectID).
		UpdateNewValues().
		ID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert ACL entry: %w", err)
	}

	return c.client.AclEntry.Get(ctx, id)
}

func (c *aclClient) Delete(ctx context.Context, fileID, id int) error {
	affected, err := c.client.AclEntry.Delete().
		Where(aclentry.ID(id), aclentry.FileID(fileID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete ACL entry: %w", err)
	}
	if affected == 0 {
		return &ent.NotFoundError{}
	}
	return nil
}

func (c *aclClient) EffectivePermissions(ctx context.Context, fileID int, user *ent.User) (*boolset.BooleanSet, error) {
	entries, err := c.client.AclEntry.Query().
		Where(aclentry.FileID(fileID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query ACL entries: %w", err)
	}
	if len(entries) == 0 {
		return nil, nil
	}

	if IsAnonymousUser(user) {
		for _, e := range entries {
			if e.SubjectType == aclentry.SubjectTypeAnonymous {
				if e.Permissions == nil {
					return &boolset.BooleanSet{}, nil
				}
				return e.Permissions, nil
			}
		}
		return nil, nil
	}

	groupIDs := GroupIDsOf(user)

	res := &boolset.BooleanSet{}
	matched := false
	for _, e := range entries {
		hit := false
		switch e.SubjectType {
		case aclentry.SubjectTypeEveryone:
			hit = true
		case aclentry.SubjectTypeUser:
			hit = e.SubjectID == user.ID
		case aclentry.SubjectTypeGroup:
			hit = lo.Contains(groupIDs, e.SubjectID)
		}
		if !hit {
			continue
		}
		matched = true
		if e.Permissions == nil {
			continue
		}
		for i := 0; i <= int(types.AclPermDelete); i++ {
			if e.Permissions.Enabled(i) {
				boolset.Set(i, true, res)
			}
		}
	}
	if !matched {
		return nil, nil
	}
	return res, nil
}
