package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/group"
	"github.com/cloudreve/Cloudreve/v4/ent/groupmembership"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/samber/lo"
)

type (
	// Ctx keys for eager loading options.
	LoadGroupPolicy struct{}

	// LoadGroupAllowedPolicies eagerly loads the group's allowed_policies M2M
	// edge.
	LoadGroupAllowedPolicies struct{}
)

const (
	AnonymousGroupID = 3
)

type (
	GroupClient interface {
		TxOperator
		// AnonymousGroup returns the anonymous group.
		AnonymousGroup(ctx context.Context) (*ent.Group, error)
		// ListAll returns all groups.
		ListAll(ctx context.Context) ([]*ent.Group, error)
		// GetByID returns the group by id.
		GetByID(ctx context.Context, id int) (*ent.Group, error)
		// ListGroups returns a list of groups with pagination.
		ListGroups(ctx context.Context, args *ListGroupParameters) (*ListGroupResult, error)
		// CountUsers returns the number of users in the group.
		CountUsers(ctx context.Context, id int) (int, error)
		// Upsert upserts a group.
		Upsert(ctx context.Context, group *ent.Group) (*ent.Group, error)
		// Delete deletes a group.
		Delete(ctx context.Context, id int) error
	}
	ListGroupParameters struct {
		*PaginationArgs
	}
	ListGroupResult struct {
		*PaginationResults
		Groups []*ent.Group
	}
)

func NewGroupClient(client *ent.Client, dbType conf.DBType, cache cache.Driver) GroupClient {
	return &groupClient{client: client, maxSQlParam: sqlParamLimit(dbType), cache: cache}
}

type groupClient struct {
	client      *ent.Client
	cache       cache.Driver
	maxSQlParam int
}

func (c *groupClient) SetClient(newClient *ent.Client) TxOperator {
	return &groupClient{client: newClient, maxSQlParam: c.maxSQlParam, cache: c.cache}
}

func (c *groupClient) GetClient() *ent.Client {
	return c.client
}

func (c *groupClient) CountUsers(ctx context.Context, id int) (int, error) {
	// Count users whose primary group is id OR who hold a membership in it.
	primary, err := c.client.User.Query().Where(user.GroupUsers(id)).Count(ctx)
	if err != nil {
		return 0, err
	}
	memberIDs, err := c.client.GroupMembership.Query().
		Where(groupmembership.GroupID(id)).
		Select(groupmembership.FieldUserID).
		Ints(ctx)
	if err != nil {
		return 0, err
	}
	memberCount := 0
	if len(memberIDs) > 0 {
		memberCount, err = c.client.User.Query().
			Where(user.IDIn(memberIDs...), user.GroupUsersNEQ(id)).
			Count(ctx)
		if err != nil {
			return 0, err
		}
	}
	return primary + memberCount, nil
}

func (c *groupClient) AnonymousGroup(ctx context.Context) (*ent.Group, error) {
	return withGroupEagerLoading(ctx, c.client.Group.Query().Where(group.ID(AnonymousGroupID))).First(ctx)
}

func (c *groupClient) ListAll(ctx context.Context) ([]*ent.Group, error) {
	return withGroupEagerLoading(ctx, c.client.Group.Query()).All(ctx)
}

func (c *groupClient) Upsert(ctx context.Context, group *ent.Group) (*ent.Group, error) {
	allowedIDs := lo.Map(group.Edges.AllowedPolicies, func(p *ent.StoragePolicy, _ int) int {
		return p.ID
	})

	if group.ID == 0 {
		stm := c.client.Group.Create().
			SetName(group.Name).
			SetMaxStorage(group.MaxStorage).
			SetSpeedLimit(group.SpeedLimit).
			SetPermissions(group.Permissions).
			SetSettings(group.Settings)

		if group.Edges.StoragePolicies != nil && group.Edges.StoragePolicies.ID > 0 {
			stm.SetStoragePolicyID(group.Edges.StoragePolicies.ID)
		}
		if len(allowedIDs) > 0 {
			stm.AddAllowedPolicyIDs(allowedIDs...)
		}

		return stm.Save(ctx)
	}

	stm := c.client.Group.UpdateOne(group).
		SetName(group.Name).
		SetMaxStorage(group.MaxStorage).
		SetSpeedLimit(group.SpeedLimit).
		SetPermissions(group.Permissions).
		SetSettings(group.Settings).
		ClearStoragePolicies().
		ClearAllowedPolicies()

	if group.Edges.StoragePolicies != nil && group.Edges.StoragePolicies.ID > 0 {
		stm.SetStoragePolicyID(group.Edges.StoragePolicies.ID)
	}
	if len(allowedIDs) > 0 {
		stm.AddAllowedPolicyIDs(allowedIDs...)
	}

	res, err := stm.Save(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *groupClient) Delete(ctx context.Context, id int) error {
	if err := c.client.Group.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

func (c *groupClient) ListGroups(ctx context.Context, args *ListGroupParameters) (*ListGroupResult, error) {
	query := withGroupEagerLoading(ctx, c.client.Group.Query())
	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 10)
	queryWithoutOrder := query.Clone()
	query.Order(getGroupOrderOption(args)...)

	// Count total items
	total, err := queryWithoutOrder.Clone().
		Count(ctx)
	if err != nil {
		return nil, err
	}

	groups, err := query.Clone().
		Limit(pageSize).
		Offset(args.Page * pageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return &ListGroupResult{
		Groups: groups,
		PaginationResults: &PaginationResults{
			TotalItems: total,
			Page:       args.Page,
			PageSize:   pageSize,
		},
	}, nil
}

func (c *groupClient) GetByID(ctx context.Context, id int) (*ent.Group, error) {
	return withGroupEagerLoading(ctx, c.client.Group.Query().Where(group.ID(id))).First(ctx)
}

func getGroupOrderOption(args *ListGroupParameters) []group.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case group.FieldName:
		return []group.OrderOption{group.ByName(orderTerm), group.ByID(orderTerm)}
	case group.FieldMaxStorage:
		return []group.OrderOption{group.ByMaxStorage(orderTerm), group.ByID(orderTerm)}
	default:
		return []group.OrderOption{group.ByID(orderTerm)}
	}
}

func withGroupEagerLoading(ctx context.Context, q *ent.GroupQuery) *ent.GroupQuery {
	if _, ok := ctx.Value(LoadGroupPolicy{}).(bool); ok {
		q.WithStoragePolicies(func(spq *ent.StoragePolicyQuery) {
			withStoragePolicyEagerLoading(ctx, spq)
		})
	}
	if _, ok := ctx.Value(LoadGroupAllowedPolicies{}).(bool); ok {
		q.WithAllowedPolicies()
	}
	return q
}

// GroupsOf returns the user's primary group plus all unexpired membership
// groups, deduped by ID. Memberships are eager-loaded under LoadUserGroup.
func GroupsOf(u *ent.User) []*ent.Group {
	res := []*ent.Group{}
	if u.Edges.Group != nil {
		res = append(res, u.Edges.Group)
	}
	now := time.Now()
	for _, m := range u.Edges.Memberships {
		g := m.Edges.Group
		if g == nil || (m.Expires != nil && m.Expires.Before(now)) {
			continue
		}
		if !lo.ContainsBy(res, func(x *ent.Group) bool { return x.ID == g.ID }) {
			res = append(res, g)
		}
	}
	return res
}

// GroupIDsOf returns the IDs of GroupsOf — used for group-subject ACL checks.
func GroupIDsOf(u *ent.User) []int {
	return lo.Map(GroupsOf(u), func(g *ent.Group, _ int) int { return g.ID })
}

// EffectiveGroup merges the user's primary group with every unexpired
// membership group into a synthetic group carrying union semantics:
// permission bits are OR'ed, quotas and limits take the most permissive
// value (0 = unlimited), and policy lists are unioned. The merged group has
// ID 0 and no usable edges beyond AllowedPolicies — functions that re-query
// by group ID must take GroupsOf instead. With no memberships it returns the
// primary group unchanged.
func EffectiveGroup(u *ent.User) *ent.Group {
	groups := GroupsOf(u)
	switch len(groups) {
	case 0:
		return nil
	case 1:
		return groups[0]
	}

	// Seed from the first (primary) group — an empty accumulator's zero
	// values would read as "unlimited" for caps where 0 means no limit.
	merged := &ent.Group{
		Name: strings.Join(lo.Map(groups, func(g *ent.Group, _ int) string {
			return g.Name
		}), " + "),
		MaxStorage:      groups[0].MaxStorage,
		SpeedLimit:      groups[0].SpeedLimit,
		StoragePolicyID: groups[0].StoragePolicyID,
	}
	merged.Edges.StoragePolicies = groups[0].Edges.StoragePolicies
	settings := cloneGroupSetting(groups[0].Settings)
	perm := boolset.BooleanSet{}
	allowed := map[int]*ent.StoragePolicy{}
	for _, p := range groups[0].Edges.AllowedPolicies {
		allowed[p.ID] = p
	}
	for _, g := range groups {
		if g.Permissions == nil {
			continue
		}
		for i, b := range *g.Permissions {
			if len(perm) <= i {
				perm = append(perm, make([]byte, i+1-len(perm))...)
			}
			perm[i] |= b
		}
	}
	for _, g := range groups[1:] {
		merged.MaxStorage = mostPermissiveInt64(merged.MaxStorage, g.MaxStorage)
		merged.SpeedLimit = mostPermissive(merged.SpeedLimit, g.SpeedLimit)
		for _, p := range g.Edges.AllowedPolicies {
			allowed[p.ID] = p
		}
		mergeGroupSettings(settings, g.Settings)
	}
	merged.Permissions = &perm
	merged.Settings = settings
	merged.Edges.AllowedPolicies = lo.Values(allowed)
	return merged
}

func cloneGroupSetting(src *types.GroupSetting) *types.GroupSetting {
	if src == nil {
		return &types.GroupSetting{}
	}
	cpy := *src
	cpy.LoginIPWhitelist = append([]string(nil), src.LoginIPWhitelist...)
	cpy.DefaultPinned = append([]int(nil), src.DefaultPinned...)
	cpy.AllowedNodes = append([]int(nil), src.AllowedNodes...)
	if src.RemoteDownloadOptions != nil {
		cpy.RemoteDownloadOptions = lo.Assign(map[string]interface{}{}, src.RemoteDownloadOptions)
	}
	return &cpy
}

// mostPermissive picks the most permissive of two quota/limit values where
// 0 means unlimited: any 0 wins, otherwise the larger value.
func mostPermissive(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return max(a, b)
}

func mostPermissiveInt64(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	return max(a, b)
}

// mergeGroupSettings folds src into dst under most-permissive semantics.
func mergeGroupSettings(dst, src *types.GroupSetting) {
	if src == nil {
		return
	}
	dst.CompressSize = mostPermissiveInt64(dst.CompressSize, src.CompressSize)
	dst.DecompressSize = mostPermissiveInt64(dst.DecompressSize, src.DecompressSize)
	// SourceBatchSize 0 disables the feature, so 0 is least permissive.
	dst.SourceBatchSize = max(dst.SourceBatchSize, src.SourceBatchSize)
	dst.Aria2BatchSize = mostPermissive(dst.Aria2BatchSize, src.Aria2BatchSize)
	dst.Aria2TaskLimit = mostPermissive(dst.Aria2TaskLimit, src.Aria2TaskLimit)
	dst.Aria2MaxFileSize = mostPermissiveInt64(dst.Aria2MaxFileSize, src.Aria2MaxFileSize)
	// MaxWalkedFiles is clamped to >=1 at use, so 0 is least permissive.
	dst.MaxWalkedFiles = max(dst.MaxWalkedFiles, src.MaxWalkedFiles)
	// TrashRetention is a duration, not a quota — 0 collects trash
	// immediately, so most-permissive is the larger value.
	dst.TrashRetention = max(dst.TrashRetention, src.TrashRetention)
	dst.RedirectedSource = dst.RedirectedSource || src.RedirectedSource
	dst.AllowSelectNode = dst.AllowSelectNode || src.AllowSelectNode
	dst.WeightedPolicies = dst.WeightedPolicies || src.WeightedPolicies
	if len(src.RemoteDownloadOptions) > 0 {
		if dst.RemoteDownloadOptions == nil {
			dst.RemoteDownloadOptions = map[string]interface{}{}
		}
		for k, v := range src.RemoteDownloadOptions {
			if _, ok := dst.RemoteDownloadOptions[k]; !ok {
				dst.RemoteDownloadOptions[k] = v
			}
		}
	}
	// Allowlists union: an IP/node/pinned share granted by any membership is
	// granted. The login IP whitelist is a restriction, not a grant — an
	// empty list means "allow all", so any unrestricted group wins.
	if len(dst.LoginIPWhitelist) > 0 && len(src.LoginIPWhitelist) > 0 {
		dst.LoginIPWhitelist = lo.Union(dst.LoginIPWhitelist, src.LoginIPWhitelist)
	} else {
		dst.LoginIPWhitelist = nil
	}
	dst.DefaultPinned = lo.Union(dst.DefaultPinned, src.DefaultPinned)
	dst.AllowedNodes = lo.Union(dst.AllowedNodes, src.AllowedNodes)
}
