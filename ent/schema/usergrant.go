package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserGrant holds the schema definition for expiring per-user benefits:
// a storage bonus (amount = bytes added to group capacity) or a group
// upgrade (amount = target group id, prev_group_id = revert target).
// A NULL expires_at means the grant never expires. Expired rows are swept
// by the grant_expire cron; capacity math only counts unexpired storage
// grants, so no row deletion is required for correctness.
type UserGrant struct {
	ent.Schema
}

// Fields of the UserGrant.
func (UserGrant) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Enum("type").
			Values("storage", "group"),
		field.Int64("amount"),
		field.Int("prev_group_id").
			Optional(),
		field.Time("expires_at").
			Optional().
			Nillable(),
	}
}

// Edges of the UserGrant.
func (UserGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("grants").
			Field("user_id").
			Unique().
			Required(),
	}
}

// Indexes of the UserGrant.
func (UserGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "type"),
		index.Fields("expires_at"),
	}
}

func (UserGrant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
