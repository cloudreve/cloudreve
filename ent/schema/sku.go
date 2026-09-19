package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Sku holds the schema definition for purchasable products: a storage
// capacity pack (amount = bytes) or a membership upgrade (amount = target
// group id). duration is seconds; 0 means the grant never expires. points
// is the credit price; NULL means the product cannot be bought with
// points. price is the display cash price in the smallest currency unit —
// cash payment processors are intentionally out of scope.
type Sku struct {
	ent.Schema
}

// Fields of the Sku.
func (Sku) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Enum("type").
			Values("storage", "group"),
		field.Int64("amount"),
		field.Int64("duration").
			Optional(),
		field.Int64("price").
			Optional(),
		field.Int64("points").
			Optional().
			Nillable(),
		field.String("label").
			Optional(),
		field.String("des").
			Optional(),
		field.Bool("enabled").
			Default(true),
		field.Int("weight").
			Default(0),
	}
}

// Edges of the Sku.
func (Sku) Edges() []ent.Edge {
	return nil
}

func (Sku) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
