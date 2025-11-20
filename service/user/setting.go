package user

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/thumb"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

const (
	twoFaEnableSessionKey = "2fa_init_"
)

// Init2FA 初始化二步验证
func Init2FA(c *gin.Context) (string, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Cloudreve",
		AccountName: user.Email,
	})
	if err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to generate TOTP secret", err)
	}

	if err := dep.KV().Set(fmt.Sprintf("%s%d", twoFaEnableSessionKey, user.ID), key.Secret(), 600); err != nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Failed to store TOTP session", err)
	}

	return key.Secret(), nil
}

type (
	// AvatarService Service to get avatar
	GetAvatarService struct {
		NoCache bool `form:"nocache"`
	}
	GetAvatarServiceParamsCtx struct{}
)

const (
	GravatarAvatar = "gravatar"
	FileAvatar     = "file"
)

// Get 获取用户头像
func (service *GetAvatarService) Get(c *gin.Context) error {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()
	// 查找目标用户
	uid := hashid.FromContext(c)
	userClient := dep.UserClient()
	user, err := userClient.GetByID(c, uid)

	if err != nil {
		return serializer.NewError(serializer.CodeUserNotFound, "", err)
	}

	if !service.NoCache {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", settings.PublicResourceMaxAge(c)))
	}

	// 未设定头像时，返回404错误
	if user.Avatar == "" {
		c.Status(404)
		return nil
	}

	avatarSettings := settings.Avatar(c)

	// Gravatar 头像重定向
	if user.Avatar == GravatarAvatar {
		gravatarRoot, err := url.Parse(avatarSettings.Gravatar)
		if err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to parse Gravatar server", err)
		}
		email_lowered := strings.ToLower(user.Email)
		has := md5.Sum([]byte(email_lowered))
		avatar, _ := url.Parse(fmt.Sprintf("/avatar/%x?d=mm&s=200", has))

		c.Redirect(http.StatusFound, gravatarRoot.ResolveReference(avatar).String())
		return nil
	}

	// 本地文件头像
	if user.Avatar == FileAvatar {
		avatarRoot := util.DataPath(avatarSettings.Path)

		avatar, err := os.Open(filepath.Join(avatarRoot, fmt.Sprintf("avatar_%d.png", user.ID)))
		if err != nil {
			dep.Logger().Warning("Failed to open avatar file", err)
			c.Status(404)
		}
		defer avatar.Close()

		http.ServeContent(c.Writer, c.Request, "avatar.png", user.UpdatedAt, avatar)
		return nil
	}

	c.Status(404)
	return nil
}

// Settings 获取用户设定
func GetUserSettings(c *gin.Context) (*UserSettings, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	userClient := dep.UserClient()
	passkeys, err := userClient.ListPasskeys(c, u.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user passkey", err)
	}

	return BuildUserSettings(u, passkeys, dep.UAParser()), nil

	// 用户组有效期

	//return serializer.Response{
	//	Data: map[string]interface{}{
	//		"uid":           user.ID,
	//		"qq":            user.OpenID != "",
	//		"homepage":      !user.OptionsSerialized.ProfileOff,
	//		"two_factor":    user.TwoFactor != "",
	//		"prefer_theme":  user.OptionsSerialized.PreferredTheme,
	//		"themes":        model.GetSettingByName("themes"),
	//		"group_expires": groupExpires,
	//		"authn":         serializer.BuildWebAuthnList(user.WebAuthnCredentials()),
	//	},
	//}
}

func UpdateUserAvatar(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	settings := dep.SettingProvider()

	avatarSettings := settings.AvatarProcess(c)
	if c.Request.ContentLength == -1 || c.Request.ContentLength > avatarSettings.MaxFileSize {
		request.BlackHole(c.Request.Body)
		return serializer.NewError(serializer.CodeFileTooLarge, "", nil)
	}

	if c.Request.ContentLength == 0 {
		// Use Gravatar for empty body
		if _, err := dep.UserClient().UpdateAvatar(c, u, GravatarAvatar); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user avatar", err)
		}

		return nil
	}

	return updateAvatarFile(c, u, c.GetHeader("Content-Type"), c.Request.Body, avatarSettings)
}

func updateAvatarFile(ctx context.Context, u *ent.User, contentType string, file io.Reader, avatarSettings *setting.AvatarProcess) error {
	dep := dependency.FromContext(ctx)
	// Detect ext from content type
	ext := "png"
	switch contentType {
	case "image/jpeg", "image/jpg":
		ext = "jpg"
	case "image/gif":
		ext = "gif"
	}
	avatar, err := thumb.NewThumbFromFile(file, ext)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Invalid image", err)
	}

	// Resize and save avatar
	avatar.CreateAvatar(avatarSettings.MaxWidth)
	avatarRoot := util.DataPath(avatarSettings.Path)
	f, err := util.CreatNestedFile(filepath.Join(avatarRoot, fmt.Sprintf("avatar_%d.png", u.ID)))
	if err != nil {
		return serializer.NewError(serializer.CodeIOFailed, "Failed to create avatar file", err)
	}

	defer f.Close()
	if err := avatar.Save(f, &setting.ThumbEncode{
		Quality: 100,
		Format:  "png",
	}); err != nil {
		return serializer.NewError(serializer.CodeIOFailed, "Failed to save avatar file", err)
	}

	if _, err := dep.UserClient().UpdateAvatar(ctx, u, FileAvatar); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to update user avatar", err)
	}

	return nil
}

type (
	PatchUserSetting struct {
		Nick                    *string   `json:"nick" binding:"omitempty,min=1,max=255"`
		Language                *string   `json:"language" binding:"omitempty,min=1,max=255"`
		PreferredTheme          *string   `json:"preferred_theme" binding:"omitempty,hexcolor|rgb|rgba|hsl"`
		VersionRetentionEnabled *bool     `json:"version_retention_enabled" binding:"omitempty"`
		VersionRetentionExt     *[]string `json:"version_retention_ext" binding:"omitempty"`
		VersionRetentionMax     *int      `json:"version_retention_max" binding:"omitempty,min=0"`
		CurrentPassword         *string   `json:"current_password" binding:"omitempty,min=4,max=128"`
		NewPassword             *string   `json:"new_password" binding:"omitempty,min=6,max=128"`
		TwoFAEnabled            *bool     `json:"two_fa_enabled" binding:"omitempty"`
		TwoFACode               *string   `json:"two_fa_code" binding:"omitempty"`
		DisableViewSync         *bool     `json:"disable_view_sync" binding:"omitempty"`
		ShareLinksInProfile     *string   `json:"share_links_in_profile" binding:"omitempty"`
	}
	PatchUserSettingParamsCtx struct{}
)

func (s *PatchUserSetting) Patch(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	userClient := dep.UserClient()
	saveSetting := false

	if s.Nick != nil {
		if _, err := userClient.UpdateNickname(c, u, *s.Nick); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user nick", err)
		}
	}

	if s.Language != nil {
		u.Settings.Language = *s.Language
		saveSetting = true
	}

	if s.PreferredTheme != nil {
		u.Settings.PreferredTheme = *s.PreferredTheme
		saveSetting = true
	}

	if s.VersionRetentionEnabled != nil {
		u.Settings.VersionRetention = *s.VersionRetentionEnabled
		saveSetting = true
	}

	if s.VersionRetentionExt != nil {
		u.Settings.VersionRetentionExt = *s.VersionRetentionExt
		saveSetting = true
	}

	if s.VersionRetentionMax != nil {
		u.Settings.VersionRetentionMax = *s.VersionRetentionMax
		saveSetting = true
	}

	if s.DisableViewSync != nil {
		u.Settings.DisableViewSync = *s.DisableViewSync
		saveSetting = true
	}

	if s.ShareLinksInProfile != nil {
		u.Settings.ShareLinksInProfile = types.ShareLinksInProfileLevel(*s.ShareLinksInProfile)
		saveSetting = true
	}

	if s.CurrentPassword != nil && s.NewPassword != nil {
		if err := inventory.CheckPassword(u, *s.CurrentPassword); err != nil {
			return serializer.NewError(serializer.CodeIncorrectPassword, "Incorrect password", err)
		}

		if _, err := userClient.UpdatePassword(c, u, *s.NewPassword); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user password", err)
		}
	}

	if s.TwoFAEnabled != nil {
		if *s.TwoFAEnabled {
			kv := dep.KV()
			secret, ok := kv.Get(fmt.Sprintf("%s%d", twoFaEnableSessionKey, u.ID))
			if !ok {
				return serializer.NewError(serializer.CodeInternalSetting, "You have not initiated 2FA session", nil)
			}

			if !totp.Validate(*s.TwoFACode, secret.(string)) {
				return serializer.NewError(serializer.Code2FACodeErr, "Incorrect 2FA code", nil)
			}

			if _, err := userClient.UpdateTwoFASecret(c, u, secret.(string)); err != nil {
				return serializer.NewError(serializer.CodeDBError, "Failed to update user 2FA", err)
			}

		} else {
			if !totp.Validate(*s.TwoFACode, u.TwoFactorSecret) {
				return serializer.NewError(serializer.Code2FACodeErr, "Incorrect 2FA code", nil)
			}

			if _, err := userClient.UpdateTwoFASecret(c, u, ""); err != nil {
				return serializer.NewError(serializer.CodeDBError, "Failed to update user 2FA", err)
			}

		}
	}

	if saveSetting {
		if err := userClient.SaveSettings(c, u); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user settings", err)
		}
	}

	return nil
}
