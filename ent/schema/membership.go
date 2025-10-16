package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Membership holds the schema definition for the Membership entity.
type Membership struct {
	ent.Schema
}

func (Membership) Annotations() []schema.Annotation {
	return []schema.Annotation{
		field.ID("group_id", "user_id"),
	}
}

func (Membership) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("group_id"),
		field.Bool("is_primary").Default(false).Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("expires_at").Nillable().Optional(),
	}
}

func (Membership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Unique().
			Required().
			Field("user_id"),
		edge.To("group", Group.Type).
			Unique().
			Required().
			Field("group_id"),
	}
}
