package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CreditTxn holds the schema definition for the credit ledger. Each row is a
// signed movement on a user's credit balance; the balance itself is
// denormalized onto user.credits and kept consistent inside the same
// transaction.
type CreditTxn struct {
	ent.Schema
}

// Fields of the CreditTxn.
func (CreditTxn) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int64("amount"),
		field.Enum("type").
			Values("purchase", "gift", "share_income", "adjust"),
		field.String("ref").
			Optional(),
		field.String("des").
			Optional(),
	}
}

// Edges of the CreditTxn.
func (CreditTxn) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("credit_txns").
			Field("user_id").
			Unique().
			Required(),
	}
}

// Indexes of the CreditTxn.
func (CreditTxn) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
	}
}

func (CreditTxn) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
