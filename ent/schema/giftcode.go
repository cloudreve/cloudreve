package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// GiftCode holds the schema definition for admin-generated redemption codes.
// A code grants credits, a storage bonus, a direct-link traffic pack, or a
// group upgrade when redeemed.
// Redemption is claimed atomically by updating used_by_id from NULL.
type GiftCode struct {
	ent.Schema
}

// Fields of the GiftCode.
func (GiftCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Unique(),
		field.Enum("type").
			Values("points", "storage", "group", "traffic"),
		field.Int64("amount"),
		field.Int64("duration").
			Optional(),
		field.Int("used_by_id").
			Optional(),
		field.Time("used_at").
			Optional().
			Nillable(),
		field.String("des").
			Optional(),
	}
}

// Edges of the GiftCode.
func (GiftCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("redeemer", User.Type).
			Ref("redeemed_codes").
			Field("used_by_id").
			Unique(),
	}
}

func (GiftCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
