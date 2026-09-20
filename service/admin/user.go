package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/samber/lo"
)

// AddUserService 用户添加服务
type AddUserService struct {
	//User     model.User `json:"User" binding:"required"`
	Password string `json:"password"`
}

// UserService 用户ID服务
type UserService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

// UserBatchService 用户批量操作服务
type UserBatchService struct {
	ID []uint `json:"id" binding:"min=1"`
}

const (
	userStatusCondition = "user_status"
	userGroupCondition  = "user_group"
	userNickCondition   = "user_nick"
	userEmailCondition  = "user_email"
	// userIDsCondition filters by a comma-separated list of numeric user IDs.
	userIDsCondition = "user_ids"
)

func (service *AdminListService) Users(c *gin.Context) (*ListUserResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()
	userClient := dep.UserClient()

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	ctx = context.WithValue(ctx, inventory.LoadUserPasskey{}, true)

	var (
		err     error
		groupID int
		ids     []int
	)
	if service.Conditions[userGroupCondition] != "" {
		groupID, err = strconv.Atoi(service.Conditions[userGroupCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid group ID", err)
		}
	}

	if service.Conditions[userIDsCondition] != "" {
		for _, part := range strings.Split(service.Conditions[userIDsCondition], ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil {
				return nil, serializer.NewError(serializer.CodeParamErr, "Invalid user ID list", err)
			}
			ids = append(ids, id)
		}
	}

	res, err := userClient.ListUsers(ctx, &inventory.ListUserParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     service.Page - 1,
			PageSize: service.PageSize,
			OrderBy:  service.OrderBy,
			Order:    inventory.OrderDirection(service.OrderDirection),
		},
		Status:  user.Status(service.Conditions[userStatusCondition]),
		GroupID: groupID,
		Nick:    service.Conditions[userNickCondition],
		Email:   service.Conditions[userEmailCondition],
		IDs:     ids,
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list users", err)
	}

	return &ListUserResponse{
		Pagination: res.PaginationResults,
		Users: lo.Map(res.Users, func(user *ent.User, _ int) GetUserResponse {
			return GetUserResponse{
				User:         user,
				HashID:       hashid.EncodeUserID(hasher, user.ID),
				TwoFAEnabled: user.TwoFactorSecret != "",
			}
		}),
	}, nil
}

type (
	SingleUserService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleUserParamCtx struct{}
)

func (service *SingleUserService) Get(c *gin.Context) (*GetUserResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()
	userClient := dep.UserClient()

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	ctx = context.WithValue(ctx, inventory.LoadUserPasskey{}, true)

	user, err := userClient.GetByID(ctx, service.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user", err)
	}

	m := manager.NewFileManager(dep, user)
	capacity, err := m.Capacity(ctx)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to get user capacity", err)
	}

	return &GetUserResponse{
		User:         user,
		HashID:       hashid.EncodeUserID(hasher, user.ID),
		TwoFAEnabled: user.TwoFactorSecret != "",
		Capacity:     capacity,
	}, nil
}

func (service *SingleUserService) CalibrateStorage(c *gin.Context) (*GetUserResponse, error) {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	_, err := userClient.CalculateStorage(ctx, service.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to calculate storage", err)
	}

	subService := &SingleUserService{ID: service.ID}
	return subService.Get(c)
}

type (
	UpsertUserService struct {
		User     *ent.User `json:"user" binding:"required"`
		Password string    `json:"password" binding:"omitempty,min=6,max=128"`
		TwoFA    string    `json:"two_fa"`
		// Memberships, when present, replaces the user's additional group
		// set (primary group untouched). nil leaves memberships alone.
		Memberships *[]int `json:"memberships"`
	}
	UpsertUserParamCtx struct{}
)

type adminUserEmailValidation struct {
	Email string `binding:"required,email"`
}

// groupIsAdminCapable reports whether a permission set grants full or
// delegated admin access.
func groupIsAdminCapable(permissions *boolset.BooleanSet) bool {
	if permissions == nil {
		return false
	}
	for _, p := range types.AdminPermissionBits() {
		if permissions.Enabled(int(p)) {
			return true
		}
	}
	return false
}

// actorIsFullAdmin reports whether the acting user belongs to a group with the
// full admin permission.
func actorIsFullAdmin(c *gin.Context) bool {
	actor := inventory.UserFromContext(c)
	return inventory.EffectiveGroup(actor) != nil && inventory.EffectiveGroup(actor).Permissions != nil &&
		inventory.EffectiveGroup(actor).Permissions.Enabled(int(types.GroupPermissionIsAdmin))
}

// guardAdminGroupChange rejects delegated admins that try to move users into
// or out of admin-capable groups. newGroupID == 0 keeps the current group.
func guardAdminGroupChange(c *gin.Context, dep dependency.Dep, existing *ent.User, newGroupID int) error {
	if actorIsFullAdmin(c) {
		return nil
	}

	if existing != nil && inventory.EffectiveGroup(existing) != nil && groupIsAdminCapable(inventory.EffectiveGroup(existing).Permissions) {
		return serializer.NewError(serializer.CodeNoPermissionErr, "Cannot modify an administrator", nil)
	}

	targetGroupID := newGroupID
	if targetGroupID == 0 && existing != nil {
		targetGroupID = existing.GroupUsers
	}
	if targetGroupID == 0 {
		return nil
	}

	target, err := dep.GroupClient().GetByID(c, targetGroupID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to get group", err)
	}
	if groupIsAdminCapable(target.Permissions) {
		return serializer.NewError(serializer.CodeNoPermissionErr, "Cannot assign an administrator group", nil)
	}

	return nil
}

// guardAdminMembershipChange applies the same delegated-admin restriction to
// additional group memberships.
func guardAdminMembershipChange(c *gin.Context, dep dependency.Dep, membershipIDs []int) error {
	if len(membershipIDs) == 0 || actorIsFullAdmin(c) {
		return nil
	}
	for _, id := range membershipIDs {
		g, err := dep.GroupClient().GetByID(c, id)
		if err != nil {
			return serializer.NewError(serializer.CodeParamErr, "Invalid membership group", err)
		}
		if groupIsAdminCapable(g.Permissions) {
			return serializer.NewError(serializer.CodeNoPermissionErr, "Cannot assign an administrator group", nil)
		}
	}
	return nil
}

func (s *UpsertUserService) validateEmail() error {
	if s.User == nil {
		return serializer.NewError(serializer.CodeParamErr, "Email format error", nil)
	}
	if err := binding.Validator.ValidateStruct(adminUserEmailValidation{Email: s.User.Email}); err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Email format error", err)
	}
	return nil
}

func (s *UpsertUserService) Update(c *gin.Context) (*GetUserResponse, error) {
	if err := s.validateEmail(); err != nil {
		return nil, err
	}

	dep := dependency.FromContext(c)
	userClient := dep.UserClient()

	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	existing, err := userClient.GetByID(ctx, s.User.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user", err)
	}

	if s.User.ID == 1 && inventory.EffectiveGroup(existing).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		if s.User.GroupUsers != existing.GroupUsers {
			return nil, serializer.NewError(serializer.CodeInvalidActionOnDefaultUser, "Cannot change default user's group", nil)
		}

		if s.User.Status != user.StatusActive {
			return nil, serializer.NewError(serializer.CodeInvalidActionOnDefaultUser, "Cannot change default user's status", nil)
		}

	}

	if err := guardAdminGroupChange(c, dep, existing, s.User.GroupUsers); err != nil {
		return nil, err
	}
	if s.Memberships != nil {
		if err := guardAdminMembershipChange(c, dep, *s.Memberships); err != nil {
			return nil, err
		}
	}

	newUser, err := userClient.Upsert(ctx, s.User, s.Password, s.TwoFA)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update user", err)
	}

	if s.Memberships != nil {
		if err := userClient.SetMemberships(ctx, newUser.ID, *s.Memberships); err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to update group memberships", err)
		}
	}

	subject := activity.Extra(map[string]any{"user_id": newUser.ID})
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventUserChanged, subject)
	if existing.GroupUsers != newUser.GroupUsers {
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventGroupChanged,
			subject, activity.Extra(map[string]any{"from": existing.GroupUsers, "to": newUser.GroupUsers}))
	}
	if existing.Storage != newUser.Storage {
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventStorageAdded,
			subject, activity.Extra(map[string]any{"from": existing.Storage, "to": newUser.Storage}))
	}

	service := &SingleUserService{ID: newUser.ID}
	return service.Get(c)
}

func (s *UpsertUserService) Create(c *gin.Context) (*GetUserResponse, error) {
	if err := s.validateEmail(); err != nil {
		return nil, err
	}

	if s.Password == "" {
		return nil, serializer.NewError(serializer.CodeParamErr, "Password is required", nil)
	}

	dep := dependency.FromContext(c)
	userClient := dep.UserClient()

	if s.User.ID != 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID must be 0", nil)
	}

	if err := guardAdminGroupChange(c, dep, nil, s.User.GroupUsers); err != nil {
		return nil, err
	}
	if s.Memberships != nil {
		if err := guardAdminMembershipChange(c, dep, *s.Memberships); err != nil {
			return nil, err
		}
	}

	user, err := userClient.Upsert(c, s.User, s.Password, s.TwoFA)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create user", err)
	}

	if s.Memberships != nil && len(*s.Memberships) > 0 {
		if err := userClient.SetMemberships(c, user.ID, *s.Memberships); err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to set group memberships", err)
		}
	}

	service := &SingleUserService{ID: user.ID}
	return service.Get(c)

}

type (
	BatchUserService struct {
		IDs []int `json:"ids" binding:"min=1"`
	}
	BatchUserParamCtx struct{}
)

func (s *BatchUserService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()
	fileClient := dep.FileClient()

	current := inventory.UserFromContext(c)
	fullAdmin := actorIsFullAdmin(c)
	groupCtx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	ae := serializer.NewAggregateError()
	for _, id := range s.IDs {
		if current.ID == id || id == 1 {
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeInvalidActionOnDefaultUser, "Cannot delete current user", nil))
			continue
		}

		// Delegated admins cannot delete members of admin-capable groups.
		if !fullAdmin {
			target, err := userClient.GetByID(groupCtx, id)
			if err != nil {
				ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to get user", err))
				continue
			}
			if inventory.EffectiveGroup(target) != nil && groupIsAdminCapable(inventory.EffectiveGroup(target).Permissions) {
				ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeNoPermissionErr, "Cannot delete an administrator", nil))
				continue
			}
		}

		fc, tx, ctx, err := inventory.WithTx(c, fileClient)
		if err != nil {
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err))
			continue
		}

		uc, _, ctx, err := inventory.WithTx(ctx, userClient)
		if err != nil {
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err))
			continue
		}

		if err := fc.DeleteByUser(ctx, id); err != nil {
			_ = inventory.Rollback(tx)
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to delete user files", err))
			continue
		}

		if err := uc.Delete(ctx, id); err != nil {
			_ = inventory.Rollback(tx)
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to delete user", err))
			continue
		}

		if err := inventory.Commit(tx); err != nil {
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to commit transaction", err))
			continue
		}
	}

	return ae.Aggregate()
}

type (
	// BatchUserUpdateService applies a status and/or group change to a set of
	// users. At least one of the two fields must be set.
	BatchUserUpdateService struct {
		IDs     []int  `json:"ids" binding:"min=1"`
		Status  string `json:"status" binding:"omitempty,oneof=active inactive manual_banned"`
		GroupID int    `json:"group_id" binding:"omitempty,min=1"`
	}
	BatchUserUpdateParamCtx struct{}
)

func (s *BatchUserUpdateService) Update(c *gin.Context) error {
	if s.Status == "" && s.GroupID == 0 {
		return serializer.NewError(serializer.CodeParamErr, "Nothing to update", nil)
	}

	dep := dependency.FromContext(c)
	userClient := dep.UserClient()
	current := inventory.UserFromContext(c)
	fullAdmin := actorIsFullAdmin(c)

	// Delegated admins cannot move users into an admin-capable group.
	if !fullAdmin && s.GroupID > 0 {
		target, err := dep.GroupClient().GetByID(c, s.GroupID)
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to get group", err)
		}
		if groupIsAdminCapable(target.Permissions) {
			return serializer.NewError(serializer.CodeNoPermissionErr, "Cannot assign an administrator group", nil)
		}
	}

	// The caller and the reserved initial admin cannot be modified in bulk.
	ae := serializer.NewAggregateError()
	groupCtx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	ids := lo.Filter(s.IDs, func(id int, _ int) bool {
		if id == current.ID || id == 1 {
			ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeInvalidActionOnDefaultUser, "Cannot modify this user in bulk", nil))
			return false
		}
		// Delegated admins cannot modify members of admin-capable groups.
		if !fullAdmin {
			target, err := userClient.GetByID(groupCtx, id)
			if err != nil {
				ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeDBError, "Failed to get user", err))
				return false
			}
			if inventory.EffectiveGroup(target) != nil && groupIsAdminCapable(inventory.EffectiveGroup(target).Permissions) {
				ae.Add(strconv.Itoa(id), serializer.NewError(serializer.CodeNoPermissionErr, "Cannot modify an administrator", nil))
				return false
			}
		}
		return true
	})

	var status *user.Status
	if s.Status != "" {
		st := user.Status(s.Status)
		status = &st
	}

	if _, err := userClient.BatchUpdate(c, ids, status, s.GroupID); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to update users", err)
	}

	return ae.Aggregate()
}
