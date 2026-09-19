package user

import (
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/email"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

const (
	emailChangeSessionKey = "email_change_"
	emailChangeTTL        = 24 * 3600
)

type (
	// RequestEmailChangeService starts the email-change flow: validates the new
	// address and the current password, stores the pending address in KV, and
	// sends a signed confirmation link to the new mailbox.
	RequestEmailChangeService struct {
		NewEmail string `json:"new_email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	RequestEmailChangeParamCtx struct{}
)

func (s *RequestEmailChangeService) Request(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	userClient := dep.UserClient()

	if err := auth.CheckScope(c, types.ScopeUserSecurityInfoWrite); err != nil {
		return err
	}

	if err := inventory.CheckPassword(u, s.Password); err != nil {
		return serializer.NewError(serializer.CodeIncorrectPassword, "Incorrect password", err)
	}

	if err := CheckEmailAllowed(dep.SettingProvider().EmailFilter(c), s.NewEmail); err != nil {
		return err
	}

	if _, err := userClient.GetByEmail(c, s.NewEmail); err == nil {
		return serializer.NewError(serializer.CodeEmailExisted, "Email already registered", nil)
	}

	// The pending address lives in KV — the signed link carries only the user
	// id, so the target address can't be tampered with.
	kv := dep.KV()
	if err := kv.Set(fmt.Sprintf("%s%d", emailChangeSessionKey, u.ID), s.NewEmail, emailChangeTTL); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to store email change session", err)
	}

	base := dep.SettingProvider().SiteURL(c)
	userID := hashid.EncodeUserID(dep.HashIDEncoder(), u.ID)
	ttl := time.Now().Add(emailChangeTTL * time.Second)
	activateURL, err := auth.SignURI(c, dep.GeneralAuth(), routes.MasterUserActivateEmailAPIUrl(base, userID).String(), &ttl)
	if err != nil {
		return serializer.NewError(serializer.CodeEncryptError, "Failed to sign the activation link", err)
	}

	credential := activateURL.Query().Get("sign")
	finalURL := routes.MasterUserActivateEmailUrl(base)
	queries := finalURL.Query()
	queries.Add("id", userID)
	queries.Add("sign", credential)
	finalURL.RawQuery = queries.Encode()

	title, body, err := email.NewActivationEmail(c, dep.SettingProvider(), u, finalURL.String())
	if err != nil {
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send confirmation email", err)
	}

	if err := dep.EmailClient(c).Send(c, s.NewEmail, title, body); err != nil {
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send confirmation email", err)
	}

	return nil
}

// ActivateEmailChange applies a pending email change once the signed link
// from the confirmation mail is opened. The target address comes from KV so a
// valid signature alone can't pick an arbitrary address.
func ActivateEmailChange(c *gin.Context) error {
	uid := hashid.FromContext(c)
	dep := dependency.FromContext(c)
	userClient := dep.UserClient()
	kv := dep.KV()

	pending, ok := kv.Get(fmt.Sprintf("%s%d", emailChangeSessionKey, uid))
	if !ok {
		return serializer.NewError(serializer.CodeParamErr, "No pending email change", nil)
	}
	newEmail, ok := pending.(string)
	if !ok {
		return serializer.NewError(serializer.CodeInternalSetting, "Invalid pending email change", nil)
	}

	target, err := userClient.GetByID(c, uid)
	if err != nil {
		return serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
	}

	if _, err := userClient.GetByEmail(c, newEmail); err == nil {
		return serializer.NewError(serializer.CodeEmailExisted, "Email already registered", nil)
	}

	target.Email = newEmail
	if _, err := userClient.Upsert(c, target, "", ""); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to update email", err)
	}

	kv.Delete("", fmt.Sprintf("%s%d", emailChangeSessionKey, uid))
	return nil
}
