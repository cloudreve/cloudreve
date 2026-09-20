package admin

import (
	"context"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// AddGroupService 用户组添加服务
type AddGroupService struct {
	//Group model.Group `json:"group" binding:"required"`
}

// GroupService 用户组ID服务
type GroupService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

// Get 获取用户组详情
func (service *GroupService) Get() serializer.Response {
	//group, err := model.GetGroupByID(service.ID)
	//if err != nil {
	//}
	//
	//return serializer.Response{Data: group}

	return serializer.Response{}
}

// Delete 删除用户组
func (service *GroupService) Delete() serializer.Response {
	//// 查找用户组
	//group, err := model.GetGroupByID(service.ID)
	//if err != nil {
	//}
	//
	//// 是否为系统用户组
	//if group.ID <= 3 {
	//}
	//
	//// 检查是否有用户使用
	//total := 0
	//row := model.DB.Model(&model.User{}).Where("group_id = ?", service.ID).
	//	Select("count(id)").Row()
	//row.Scan(&total)
	//if total > 0 {
	//}
	//
	//model.DB.Delete(&group)

	return serializer.Response{}
}

func (service *SingleGroupService) Delete(c *gin.Context) error {
	if service.ID <= 3 {
		return serializer.NewError(serializer.CodeInvalidActionOnSystemGroup, "", nil)
	}

	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	// Any user still under this group?
	users, err := groupClient.CountUsers(c, int(service.ID))
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to count users", err)
	}

	if users > 0 {
		return serializer.NewError(serializer.CodeGroupUsedByUser, strconv.Itoa(users), nil)
	}

	err = groupClient.Delete(c, service.ID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete group", err)
	}

	return nil
}

func (s *AdminListService) List(c *gin.Context) (*ListGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	ctx := context.WithValue(c, inventory.LoadGroupPolicy{}, true)
	res, err := groupClient.ListGroups(ctx, &inventory.ListGroupParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list groups", err)
	}

	return &ListGroupResponse{
		Pagination: res.PaginationResults,
		Groups:     res.Groups,
	}, nil
}

type (
	SingleGroupService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleGroupParamCtx struct{}
)

const (
	countUserQuery = "countUser"
)

func (s *SingleGroupService) Get(c *gin.Context) (*GetGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	ctx := context.WithValue(c, inventory.LoadGroupPolicy{}, true)
	ctx = context.WithValue(ctx, inventory.LoadGroupAllowedPolicies{}, true)
	group, err := groupClient.GetByID(ctx, s.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get group", err)
	}

	res := &GetGroupResponse{Group: group}

	if c.Query(countUserQuery) != "" {
		totalUsers, err := groupClient.CountUsers(ctx, int(s.ID))
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to count users", err)
		}
		res.TotalUsers = totalUsers
	}

	return res, nil
}

type (
	UpsertGroupService struct {
		Group *ent.Group `json:"group" binding:"required"`
	}
	UpsertGroupParamCtx struct{}
)

func (s *UpsertGroupService) Update(c *gin.Context) (*GetGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	if s.Group.ID == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID is required", nil)
	}

	// Initial admin group have to be admin
	if s.Group.ID == 1 && !s.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Initial admin group have to be admin", nil)
	}

	// Delegated admins cannot modify admin-capable permission bits — the
	// target group's existing admin bits are preserved.
	if err := s.maskAdminPermissions(c, groupClient); err != nil {
		return nil, err
	}

	group, err := groupClient.Upsert(c, s.Group)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update group", err)
	}

	service := &SingleGroupService{ID: group.ID}
	return service.Get(c)
}

func (s *UpsertGroupService) Create(c *gin.Context) (*GetGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	if s.Group.ID > 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID must be 0", nil)
	}

	// Delegated admins cannot create groups carrying admin-capable bits.
	if err := s.maskAdminPermissions(c, groupClient); err != nil {
		return nil, err
	}

	group, err := groupClient.Upsert(c, s.Group)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create group", err)
	}

	service := &SingleGroupService{ID: group.ID}
	return service.Get(c)
}

// maskAdminPermissions strips admin-capable permission bits from the payload
// when the acting user is not a full administrator, preventing delegated
// admins from escalating privileges. For updates, the target group's existing
// admin bits are preserved.
func (s *UpsertGroupService) maskAdminPermissions(c *gin.Context, groupClient inventory.GroupClient) error {
	actor := inventory.UserFromContext(c)
	if inventory.EffectiveGroup(actor) != nil && inventory.EffectiveGroup(actor).Permissions != nil &&
		inventory.EffectiveGroup(actor).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil
	}

	if s.Group.Permissions == nil {
		s.Group.Permissions = &boolset.BooleanSet{}
	}
	if s.Group.ID > 0 {
		existing, err := groupClient.GetByID(c, s.Group.ID)
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to get group", err)
		}
		for _, p := range types.AdminPermissionBits() {
			boolset.Set(int(p), existing.Permissions != nil && existing.Permissions.Enabled(int(p)), s.Group.Permissions)
		}
	} else {
		for _, p := range types.AdminPermissionBits() {
			boolset.Set(int(p), false, s.Group.Permissions)
		}
	}

	return nil
}
