package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// InvitationCode holds the schema definition for invitation codes that can
// gate public registration.
type InvitationCode struct {
	ent.Schema
}

func (InvitationCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			MaxLen(64).
			Unique(),
		// group_id assigns new users to this group; 0 means the default group.
		field.Int("group_id").
			Default(0),
		// max_uses limits how many accounts the code can create; 0 = unlimited.
		field.Int("max_uses").
			Default(1),
		field.Int("used_count").
			Default(0),
		field.Time("expires_at").
			Optional().
			Nillable(),
	}
}

func (InvitationCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
