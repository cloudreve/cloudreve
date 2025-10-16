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
		field.String("nick").
			MaxLen(100),
		field.String("password").
			Optional().
			Sensitive(),
		field.Enum("status").
			Values("active", "inactive", "manual_banned", "sys_banned").
			Default("active"),
		field.Int64("storage").
			Default(0),
		field.String("two_factor_secret").
			Sensitive().
			Optional(),
		field.String("avatar").
			Optional(),
		field.JSON("settings", &types.UserSetting{}).
			Default(&types.UserSetting{}).
			Optional(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).
			Ref("users").
			Through("membership", Membership.Type),
		edge.To("files", File.Type),
		edge.To("dav_accounts", DavAccount.Type),
		edge.To("shares", Share.Type),
		edge.To("passkey", Passkey.Type),
		edge.To("tasks", Task.Type),
		edge.To("entities", Entity.Type),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
