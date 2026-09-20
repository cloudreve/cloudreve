package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			MaxLen(100).
			Unique(),
		// phone is the mobile number used for SMS sign-in and password
		// reset; NULL when the user never bound one.
		field.String("phone").
			MaxLen(20).
			Optional().
			Nillable().
			Unique(),
		field.String("nick").
			MaxLen(100),
		field.String("password").
			Optional().
			Sensitive(),
		field.Enum("status").
			Values("active", "inactive", "manual_banned", "sys_banned").
			Default("active"),
		// ban_expires lifts a banned status after this time; nil = permanent.
		field.Time("ban_expires").
			Optional().
			Nillable(),
		// ban_reason is shown to the user when a banned login is rejected.
		field.Text("ban_reason").
			Optional(),
		field.Time("last_login").
			Optional().
			Nillable().
			Comment("Time of the last successful sign-in"),
		field.Int64("storage").
			Default(0),
		// credits is the user's point balance; movements are recorded in
		// credit_txns and applied under the users-row lock.
		field.Int64("credits").
			Default(0),
		field.String("two_factor_secret").
			Sensitive().
			Optional(),
		// vault_password gates the user's private space; same salt:digest
		// format as the account password but independent from it.
		field.String("vault_password").
			Sensitive().
			Optional(),
		// vault_folder is the file ID of the user's private-space root
		// folder. 0 means the feature is not set up.
		field.Int("vault_folder").
			Optional().
			Default(0),
		// two_factor_backup_codes holds salt:sha256 digests of one-time
		// recovery codes, same store format as account passwords.
		field.JSON("two_factor_backup_codes", []string{}).
			Sensitive().
			Optional(),
		field.String("avatar").
			Optional(),
		field.JSON("settings", &types.UserSetting{}).
			Default(&types.UserSetting{}).
			Optional(),
		field.Int("group_users"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).
			Ref("users").
			Field("group_users").
			Unique().
			Required(),
		edge.To("files", File.Type),
		edge.To("dav_accounts", DavAccount.Type),
		edge.To("shares", Share.Type),
		edge.To("passkey", Passkey.Type),
		edge.To("tasks", Task.Type),
		edge.To("fsevents", FsEvent.Type),
		edge.To("entities", Entity.Type),
		edge.To("oauth_grants", OAuthGrant.Type),
		edge.To("credit_txns", CreditTxn.Type),
		edge.To("redeemed_codes", GiftCode.Type),
		edge.To("grants", UserGrant.Type),
		edge.To("share_purchases", SharePurchase.Type),
		edge.To("sso_bindings", SsoBinding.Type),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
