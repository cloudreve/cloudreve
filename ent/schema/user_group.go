package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type UserGroup struct {
	ent.Schema
}

func (UserGroup) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("group_id"),
		field.Bool("is_primary").
			Default(false),
		field.Time("expires_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
	}
}

func (UserGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Required().
			Unique().
			Field("user_id"),
		edge.To("group", Group.Type).
			Required().
			Unique().
			Field("group_id"),
	}
}
