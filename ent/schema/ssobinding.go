package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SsoBinding holds the schema definition for external identity bindings.
// Each row links a local user to a subject at an external sign-in provider
// (e.g. a QQ Connect openid). One binding per (provider, subject) and one
// binding per (user, provider).
type SsoBinding struct {
	ent.Schema
}

// Fields of the SsoBinding.
func (SsoBinding) Fields() []ent.Field {
	return []ent.Field{
		field.String("provider").
			MaxLen(32).
			NotEmpty(),
		field.String("subject").
			MaxLen(128).
			NotEmpty(),
		field.Int("user_id"),
	}
}

// Edges of the SsoBinding.
func (SsoBinding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("sso_bindings").
			Field("user_id").
			Unique().
			Required(),
	}
}

// Indexes of the SsoBinding.
func (SsoBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "subject").Unique(),
		index.Fields("user_id", "provider").Unique(),
	}
}

func (SsoBinding) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
