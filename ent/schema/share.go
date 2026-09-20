package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// Share holds the schema definition for the Share entity.
type Share struct {
	ent.Schema
}

// Fields of the Share.
func (Share) Fields() []ent.Field {
	return []ent.Field{
		field.String("password").
			Optional(),
		field.Int("views").
			Default(0),
		field.Int("downloads").
			Default(0),
		field.Time("expires").
			Nillable().
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
		field.Int("remain_downloads").
			Nillable().
			Optional(),
		// Points price visitors must pay before downloading. 0 = free share.
		field.Int("price_points").
			Default(0).
			NonNegative(),
		// Listed in the public share directory. Never settable on
		// password-protected shares; enforced at the service layer.
		field.Bool("listed_publicly").
			Default(false),
		field.JSON("props", &types.ShareProps{}).Optional(),
	}
}

// Edges of the Share.
func (Share) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("shares").Unique(),
		edge.From("file", File.Type).
			Ref("shares").Unique(),
		// All files covered by a multi-file share, anchor included. Empty
		// for legacy single-file shares.
		edge.To("files", File.Type),
		edge.To("purchases", SharePurchase.Type),
	}
}

func (Share) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
