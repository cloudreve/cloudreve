package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GroupMembership holds the schema definition for additional group
// memberships beyond a user's primary group (users.group_users). A NULL
// expires means the membership never expires; expired rows are filtered at
// read time and swept by the grant_expire cron.
type GroupMembership struct {
	ent.Schema
}

// Fields of the GroupMembership.
func (GroupMembership) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("group_id"),
		field.Time("expires").
			Optional().
			Nillable(),
	}
}

// Edges of the GroupMembership.
func (GroupMembership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("memberships").
			Field("user_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("memberships").
			Field("group_id").
			Unique().
			Required(),
	}
}

// Indexes of the GroupMembership.
func (GroupMembership) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "group_id").Unique(),
		index.Fields("expires"),
	}
}

func (GroupMembership) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
