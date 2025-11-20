package user

import (
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/samber/lo"
	"github.com/ua-parser/uap-go/uaparser"
)

type PreparePasskeyLoginResponse struct {
	Options   *protocol.CredentialAssertion `json:"options"`
	SessionID string                        `json:"session_id"`
}

type UserSettings struct {
	VersionRetentionEnabled bool      `json:"version_retention_enabled"`
	VersionRetentionExt     []string  `json:"version_retention_ext,omitempty"`
	VersionRetentionMax     int       `json:"version_retention_max,omitempty"`
	Paswordless             bool      `json:"passwordless"`
	TwoFAEnabled            bool      `json:"two_fa_enabled"`
	Passkeys                []Passkey `json:"passkeys,omitempty"`
	DisableViewSync         bool      `json:"disable_view_sync"`
	ShareLinksInProfile     string    `json:"share_links_in_profile"`
}

func BuildUserSettings(u *ent.User, passkeys []*ent.Passkey, parser *uaparser.Parser) *UserSettings {
	return &UserSettings{
		VersionRetentionEnabled: u.Settings.VersionRetention,
		VersionRetentionExt:     u.Settings.VersionRetentionExt,
		VersionRetentionMax:     u.Settings.VersionRetentionMax,
		TwoFAEnabled:            u.TwoFactorSecret != "",
		Paswordless:             u.Password == "",
		Passkeys: lo.Map(passkeys, func(item *ent.Passkey, index int) Passkey {
			return BuildPasskey(item)
		}),
		DisableViewSync:     u.Settings.DisableViewSync,
		ShareLinksInProfile: string(u.Settings.ShareLinksInProfile),
	}
}

type Passkey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func BuildPasskey(passkey *ent.Passkey) Passkey {
	return Passkey{
		ID:        passkey.CredentialID,
		Name:      passkey.Name,
		UsedAt:    passkey.UsedAt,
		CreatedAt: passkey.CreatedAt,
	}
}

// Node option for handling workflows.
type Node struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Type         string              `json:"type"`
	Capabilities *boolset.BooleanSet `json:"capabilities"`
}

// BuildNodes serialize a list of nodes.
func BuildNodes(nodes []*ent.Node, idEncoder hashid.Encoder) []*Node {
	res := make([]*Node, 0, len(nodes))
	for _, v := range nodes {
		res = append(res, BuildNode(v, idEncoder))
	}

	return res
}

// BuildNode serialize a node.
func BuildNode(node *ent.Node, idEncoder hashid.Encoder) *Node {
	return &Node{
		ID:           hashid.EncodeNodeID(idEncoder, node.ID),
		Name:         node.Name,
		Type:         string(node.Type),
		Capabilities: node.Capabilities,
	}
}

// BuiltinLoginResponse response for a successful login for builtin auth provider.
type BuiltinLoginResponse struct {
	User  User       `json:"user"`
	Token auth.Token `json:"token"`
}

// User 用户序列化器
type User struct {
	ID                  string                         `json:"id"`
	Email               string                         `json:"email,omitempty"`
	Nickname            string                         `json:"nickname"`
	Status              user.Status                    `json:"status,omitempty"`
	Avatar              string                         `json:"avatar,omitempty"`
	CreatedAt           time.Time                      `json:"created_at"`
	PreferredTheme      string                         `json:"preferred_theme,omitempty"`
	Anonymous           bool                           `json:"anonymous,omitempty"`
	Group               *Group                         `json:"group,omitempty"`
	Pined               []types.PinedFile              `json:"pined,omitempty"`
	Language            string                         `json:"language,omitempty"`
	DisableViewSync     bool                           `json:"disable_view_sync,omitempty"`
	ShareLinksInProfile types.ShareLinksInProfileLevel `json:"share_links_in_profile,omitempty"`
}

type Group struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Permission          *boolset.BooleanSet `json:"permission,omitempty"`
	DirectLinkBatchSize int                 `json:"direct_link_batch_size,omitempty"`
	TrashRetention      int                 `json:"trash_retention,omitempty"`
}

type storage struct {
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Total uint64 `json:"total"`
}

// WebAuthnCredentials 外部验证器凭证
type WebAuthnCredentials struct {
	ID          []byte `json:"id"`
	FingerPrint string `json:"fingerprint"`
}

type PrepareLoginResponse struct {
	WebAuthnEnabled bool `json:"webauthn_enabled"`
	PasswordEnabled bool `json:"password_enabled"`
}

// BuildWebAuthnList 构建设置页面凭证列表
func BuildWebAuthnList(credentials []webauthn.Credential) []WebAuthnCredentials {
	res := make([]WebAuthnCredentials, 0, len(credentials))
	for _, v := range credentials {
		credential := WebAuthnCredentials{
			ID:          v.ID,
			FingerPrint: fmt.Sprintf("% X", v.Authenticator.AAGUID),
		}
		res = append(res, credential)
	}

	return res
}

// BuildUser 序列化用户
func BuildUser(user *ent.User, idEncoder hashid.Encoder) User {
	return User{
		ID:                  hashid.EncodeUserID(idEncoder, user.ID),
		Email:               user.Email,
		Nickname:            user.Nick,
		Status:              user.Status,
		Avatar:              user.Avatar,
		CreatedAt:           user.CreatedAt,
		PreferredTheme:      user.Settings.PreferredTheme,
		Anonymous:           user.ID == 0,
		Group:               BuildGroup(user.Edges.Group, idEncoder),
		Pined:               user.Settings.Pined,
		Language:            user.Settings.Language,
		DisableViewSync:     user.Settings.DisableViewSync,
		ShareLinksInProfile: user.Settings.ShareLinksInProfile,
	}
}

func BuildGroup(group *ent.Group, idEncoder hashid.Encoder) *Group {
	if group == nil {
		return nil
	}
	return &Group{
		ID:                  hashid.EncodeGroupID(idEncoder, group.ID),
		Name:                group.Name,
		Permission:          group.Permissions,
		DirectLinkBatchSize: group.Settings.SourceBatchSize,
		TrashRetention:      group.Settings.TrashRetention,
	}
}

const sensitiveTag = "redacted"

const (
	RedactLevelAnonymous = iota
	RedactLevelUser
)

// BuildUserRedacted Serialize a user without sensitive information.
func BuildUserRedacted(u *ent.User, level int, idEncoder hashid.Encoder) User {
	userRaw := BuildUser(u, idEncoder)

	user := User{
		ID:                  userRaw.ID,
		Nickname:            userRaw.Nickname,
		Avatar:              userRaw.Avatar,
		CreatedAt:           userRaw.CreatedAt,
		ShareLinksInProfile: userRaw.ShareLinksInProfile,
	}

	if userRaw.Group != nil {
		user.Group = RedactedGroup(userRaw.Group)
	}

	if level == RedactLevelUser {
		user.Email = userRaw.Email
	}

	return user
}

// BuildGroupRedacted Serialize a group without sensitive information.
func RedactedGroup(g *Group) *Group {
	if g == nil {
		return nil
	}

	return &Group{
		ID:   g.ID,
		Name: g.Name,
	}
}
