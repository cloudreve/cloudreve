package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SharePurchase records a points purchase of a paid share. One row per
// (share, buyer); the ticket acts as a bearer credential to restore access
// after session loss.
type SharePurchase struct {
	ent.Schema
}

// Fields of the SharePurchase.
func (SharePurchase) Fields() []ent.Field {
	return []ent.Field{
		field.Int("share_id"),
		field.Int("buyer_id"),
		field.Int("points"),
		field.String("ticket").
			Unique().
			NotEmpty(),
	}
}

// Edges of the SharePurchase.
func (SharePurchase) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("share", Share.Type).
			Ref("purchases").
			Field("share_id").
			Unique().
			Required(),
		edge.From("buyer", User.Type).
			Ref("share_purchases").
			Field("buyer_id").
			Unique().
			Required(),
	}
}

func (SharePurchase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("share_id", "buyer_id").
			Unique(),
	}
}

func (SharePurchase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
