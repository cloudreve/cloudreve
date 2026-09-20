package user

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/thumb"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/samber/lo"
)

const (
	twoFaEnableSessionKey = "2fa_init_"
	// backupCodeCount is the number of one-time recovery codes issued per batch.
	backupCodeCount = 10
	// backupCodeAlphabet excludes ambiguous glyphs (0/o, 1/l/i).
	backupCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"
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
	// Backup2FAService regenerates one-time 2FA recovery codes.
	Backup2FAService struct {
		TwoFACode string `json:"two_fa_code" binding:"required"`
	}
	Backup2FAParameterCtx struct{}
)

// Process generates a fresh batch of recovery codes for the current user.
// Requires 2FA to be enabled and a valid current TOTP code, the same trust
// bar as disabling 2FA. Plaintext codes are returned once and only their
// digests are persisted.
func (service *Backup2FAService) Process(c *gin.Context) ([]string, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	if err := auth.CheckScope(c, types.ScopeUserSecurityInfoWrite); err != nil {
		return nil, err
	}

	if u.TwoFactorSecret == "" {
		return nil, serializer.NewError(serializer.CodeFeatureNotEnabled, "2FA is not enabled", nil)
	}

	if !totp.Validate(service.TwoFACode, u.TwoFactorSecret) {
		return nil, serializer.NewError(serializer.Code2FACodeErr, "Incorrect 2FA code", nil)
	}

	codes := make([]string, 0, backupCodeCount)
	digests := make([]string, 0, backupCodeCount)
	for i := 0; i < backupCodeCount; i++ {
		code, err := generateBackupCode()
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to generate recovery codes", err)
		}
		digest, err := inventory.DigestPassword(code)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to hash recovery codes", err)
		}
		codes = append(codes, code[:4]+"-"+code[4:])
		digests = append(digests, digest)
	}

	if _, err := dep.UserClient().UpdateTwoFABackupCodes(c, u, digests); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to store recovery codes", err)
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventEnable2FA,
		activity.Extra(map[string]any{"recovery_codes": true}))

	return codes, nil
}

// generateBackupCode returns an 8-char code drawn from an unambiguous
// alphabet (~41 bits of entropy per code).
func generateBackupCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := make([]byte, 8)
	for i, b := range buf {
		code[i] = backupCodeAlphabet[int(b)%len(backupCodeAlphabet)]
	}
	return string(code), nil
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

	ctx := context.WithValue(c, inventory.LoadOAuthGrantClient{}, true)
	grants, err := dep.OAuthClientClient().GetGrantsByUserID(ctx, u.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user OAuth grants", err)
	}

	bindings, err := dep.SsoBindingClient().ListByUser(c, u.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user linked accounts", err)
	}

	res := BuildUserSettings(u, passkeys, dep.UAParser(), grants, bindings)
	if u.Settings.PreferredPolicy > 0 {
		res.PreferredPolicy = hashid.EncodePolicyID(dep.HashIDEncoder(), u.Settings.PreferredPolicy)
	}
	if res.VaultEnabled {
		_, res.VaultUnlocked = dep.KV().Get(dbfs.VaultUnlockCachePrefix + strconv.Itoa(u.ID))
	}
	return res, nil

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
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventChangeAvatar)

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
	activity.Record(ctx, dep.SettingProvider(), dep.ActivityClient(), types.EventChangeAvatar)

	return nil
}

// AnnouncementService serves the current site announcement to logged-in
// users, filtered by their dismissal record.
type AnnouncementService struct{}

type AnnouncementParamCtx struct{}

// Get returns the announcement content, or "" when unset or dismissed.
func (s *AnnouncementService) Get(c *gin.Context) (*AnnouncementResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	current := dep.SettingProvider().Localized(c, "announcement", u.Settings.Language)
	if current == "" || current == u.Settings.DismissedAnnouncement {
		return &AnnouncementResponse{}, nil
	}

	return &AnnouncementResponse{Content: current}, nil
}

type AnnouncementResponse struct {
	Content string `json:"content,omitempty"`
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
		// ShareDefaultPrivate accepts "true", "false" or "" (clear the
		// override and inherit the site default).
		ShareDefaultPrivate *string `json:"share_default_private" binding:"omitempty"`
		// PreferredViewers replaces the whole extension → viewer-id map.
		PreferredViewers *map[string]string `json:"preferred_viewers" binding:"omitempty"`
		// TrashRetention overrides group trash retention, in seconds.
		// 0 clears the override. Capped at ~10 years.
		TrashRetention *int `json:"trash_retention" binding:"omitempty,min=0,max=315360000"`
		// PreferredPolicy selects the user's default storage policy from the
		// group's allowed set, hashid-encoded. "" clears the preference.
		PreferredPolicy *string `json:"preferred_policy" binding:"omitempty"`
		// DismissAnnouncement records the current site announcement as seen
		// so the modal does not show again until the content changes.
		DismissAnnouncement *bool `json:"dismiss_announcement" binding:"omitempty"`
	}
	PatchUserSettingParamsCtx struct{}
)

var (
	preferredViewerExtPattern = regexp.MustCompile(`^[a-z0-9]{1,20}$`)
	preferredViewerIDPattern  = regexp.MustCompile(`^[\w.\-:]{1,64}$`)
)

const preferredViewersMaxEntries = 200

// validatePreferredViewers bounds the extension → viewer-id map to sane
// shapes so it can't be abused as arbitrary JSON storage.
func validatePreferredViewers(m map[string]string) error {
	if len(m) > preferredViewersMaxEntries {
		return serializer.NewError(serializer.CodeParamErr, "Too many preferred viewers", nil)
	}
	for ext, viewerID := range m {
		if !preferredViewerExtPattern.MatchString(ext) || !preferredViewerIDPattern.MatchString(viewerID) {
			return serializer.NewError(serializer.CodeParamErr, "Invalid preferred viewer entry", nil)
		}
	}
	return nil
}

func (s *PatchUserSetting) Patch(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	userClient := dep.UserClient()
	saveSetting := false

	if s.Nick != nil {
		if _, err := userClient.UpdateNickname(c, u, *s.Nick); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user nick", err)
		}
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventChangeNick)
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
		level := types.ShareLinksInProfileLevel(*s.ShareLinksInProfile)
		switch level {
		case types.ProfilePublicShareOnly, types.ProfileAllShare, types.ProfileHideShare, types.ProfileSharePublic:
		default:
			return serializer.NewError(serializer.CodeParamErr, "Invalid share links visibility", nil)
		}
		u.Settings.ShareLinksInProfile = level
		saveSetting = true
	}

	if s.ShareDefaultPrivate != nil {
		if *s.ShareDefaultPrivate != "" && *s.ShareDefaultPrivate != "true" && *s.ShareDefaultPrivate != "false" {
			return serializer.NewError(serializer.CodeParamErr, "Invalid share privacy default", nil)
		}
		u.Settings.ShareDefaultPrivate = nil
		if *s.ShareDefaultPrivate != "" {
			v := *s.ShareDefaultPrivate == "true"
			u.Settings.ShareDefaultPrivate = &v
		}
		saveSetting = true
	}

	if s.PreferredViewers != nil {
		if err := validatePreferredViewers(*s.PreferredViewers); err != nil {
			return err
		}
		u.Settings.PreferredViewers = *s.PreferredViewers
		saveSetting = true
	}

	if s.TrashRetention != nil {
		u.Settings.TrashRetention = *s.TrashRetention
		saveSetting = true
	}

	if s.PreferredPolicy != nil {
		if *s.PreferredPolicy == "" {
			u.Settings.PreferredPolicy = 0
		} else {
			pid, err := dep.HashIDEncoder().Decode(*s.PreferredPolicy, hashid.PolicyID)
			if err != nil {
				return serializer.NewError(serializer.CodeParamErr, "Invalid storage policy", err)
			}
			allowed, err := dep.StoragePolicyClient().ListByGroups(c, inventory.GroupsOf(u))
			if err != nil {
				return serializer.NewError(serializer.CodeDBError, "Failed to list storage policies", err)
			}
			if !lo.ContainsBy(allowed, func(p *ent.StoragePolicy) bool { return p.ID == pid }) {
				return serializer.NewError(serializer.CodeNoPermissionErr, "Storage policy is not available for your group", nil)
			}
			u.Settings.PreferredPolicy = pid
		}
		saveSetting = true
	}

	if s.DismissAnnouncement != nil && *s.DismissAnnouncement {
		u.Settings.DismissedAnnouncement = dep.SettingProvider().Announcement(c)
		saveSetting = true
	}

	if s.CurrentPassword != nil && s.NewPassword != nil {
		if err := auth.CheckScope(c, types.ScopeUserSecurityInfoWrite); err != nil {
			return err
		}

		if err := inventory.CheckPassword(u, *s.CurrentPassword); err != nil {
			return serializer.NewError(serializer.CodeIncorrectPassword, "Incorrect password", err)
		}

		if _, err := userClient.UpdatePassword(c, u, *s.NewPassword); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user password", err)
		}
		activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventChangePassword)
	}

	if s.TwoFAEnabled != nil {
		if err := auth.CheckScope(c, types.ScopeUserSecurityInfoWrite); err != nil {
			return err
		}

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
			activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventEnable2FA)

		} else {
			if !totp.Validate(*s.TwoFACode, u.TwoFactorSecret) {
				return serializer.NewError(serializer.Code2FACodeErr, "Incorrect 2FA code", nil)
			}

			if _, err := userClient.UpdateTwoFASecret(c, u, ""); err != nil {
				return serializer.NewError(serializer.CodeDBError, "Failed to update user 2FA", err)
			}
			activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventDisable2FA)

		}
	}

	if saveSetting {
		if err := userClient.SaveSettings(c, u); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to update user settings", err)
		}
	}

	return nil
}
