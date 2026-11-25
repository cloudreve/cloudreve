package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

// RegisterParameterCtx define key fore UserRegisterService
type RegisterParameterCtx struct{}

// UserRegisterService 管理用户注册的服务
type UserRegisterService struct {
	UserName string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required,min=6,max=128"`
	Language string `form:"language" json:"language"`
}

// Register 新用户注册
func (service *UserRegisterService) Register(c *gin.Context) serializer.Response {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	isEmailRequired := settings.EmailActivationEnabled(c)
	args := &inventory.NewUserArgs{
		Email:         strings.ToLower(service.UserName),
		PlainPassword: service.Password,
		Status:        user.StatusActive,
		GroupID:       settings.DefaultGroup(c),
		Language:      service.Language,
	}
	if isEmailRequired {
		args.Status = user.StatusInactive
	}

	userClient := dep.UserClient()
	uc, tx, _, err := inventory.WithTx(c, userClient)
	if err != nil {
		return serializer.DBErr(c, "Failed to start transaction", err)
	}

	expectedUser, err := uc.Create(c, args)
	isReRegistration := errors.Is(err, inventory.ErrInactiveUserExisted)
	if isReRegistration {
		expectedUser, err = uc.UpdateInactivePassword(c, expectedUser, service.Password)
		isEmailRequired = true
	}
	if expectedUser != nil {
		util.WithValue(c, inventory.UserCtx{}, expectedUser)
	}

	if err != nil {
		_ = inventory.Rollback(tx)
		if errors.Is(err, inventory.ErrUserEmailExisted) {
			return serializer.ErrWithDetails(c, serializer.CodeEmailExisted, "Email already in use", err)
		}

		if isReRegistration && ent.IsNotFound(err) {
			return serializer.ErrWithDetails(c, serializer.CodeConflict, "Registration has changed, please try again", nil)
		}

		return serializer.DBErr(c, "Failed to insert user row", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return serializer.DBErr(c, "Failed to commit user row", err)
	}

	if isEmailRequired {
		if err := sendActivationEmail(c, dep, expectedUser); err != nil {
			return serializer.ErrWithDetails(c, serializer.CodeNotSet, "", err)
		}
		return serializer.Response{Code: serializer.CodeNotFullySuccess}
	}

	return serializer.Response{Data: BuildUser(expectedUser, dep.HashIDEncoder())}
}

func sendActivationEmail(ctx context.Context, dep dependency.Dep, newUser *ent.User) error {
	base := dep.SettingProvider().SiteURL(ctx)
	userID := hashid.EncodeUserID(dep.HashIDEncoder(), newUser.ID)
	ttl := time.Now().Add(time.Duration(24) * time.Hour)
	credential := dep.GeneralAuth().Sign(activationSignContent(newUser), ttl.Unix())
	if credential == "" {
		return serializer.NewError(serializer.CodeEncryptError, "Failed to sign the activation link", nil)
	}

	// 生成对用户访问的激活地址
	finalURL := routes.MasterUserActivateUrl(base)
	queries := finalURL.Query()
	queries.Add("id", userID)
	queries.Add("sign", credential)
	finalURL.RawQuery = queries.Encode()

	// 返送激活邮件
	title, body, err := email.NewActivationEmail(ctx, dep.SettingProvider(), newUser, finalURL.String())
	if err != nil {
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send activation email", err)
	}

	if err := dep.EmailClient(ctx).Send(ctx, newUser.Email, title, body); err != nil {
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send activation email", err)
	}

	return nil
}

func activationSignContent(u *ent.User) string {
	// The salted password digest is the registration revision and never leaves the server.
	return fmt.Sprintf("user-activation-v2:%d:%q:%q", u.ID, u.Email, u.Password)
}

// ActivateUser 激活用户
func ActivateUser(c *gin.Context) serializer.Response {
	const invalidLinkMessage = "Activation link is invalid or expired, please register again to request a new link"

	uid := hashid.FromContext(c)
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()

	// 查找待激活用户
	inactiveUser, err := userClient.GetByID(c, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return serializer.ErrWithDetails(c, serializer.CodeInvalidSign, invalidLinkMessage, nil)
		}
		return serializer.DBErr(c, "Failed to get user", err)
	}

	if err := dep.GeneralAuth().Check(activationSignContent(inactiveUser), c.Query("sign")); err != nil {
		return serializer.Err(c, serializer.NewError(serializer.CodeInvalidSign, invalidLinkMessage, err))
	}

	// 检查状态
	if inactiveUser.Status != user.StatusInactive {
		return serializer.ErrWithDetails(c, serializer.CodeUserCannotActivate, "This user cannot be activated", nil)
	}

	// 激活用户
	activeUser, err := userClient.Activate(c, inactiveUser)
	if err != nil {
		if ent.IsNotFound(err) {
			return serializer.ErrWithDetails(c, serializer.CodeInvalidSign, invalidLinkMessage, nil)
		}
		return serializer.DBErr(c, "Failed to update user", err)
	}

	util.WithValue(c, inventory.UserCtx{}, activeUser)
	return serializer.Response{Data: BuildUser(activeUser, dep.HashIDEncoder())}
}
