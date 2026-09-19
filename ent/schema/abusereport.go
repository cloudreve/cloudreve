package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AbuseReport holds one user-submitted abuse report against a share or a
// user. Target references are plain IDs (no edges) so reports survive
// deletion of the reported share or user.
type AbuseReport struct {
	ent.Schema
}

// Fields of the AbuseReport.
func (AbuseReport) Fields() []ent.Field {
	return []ent.Field{
		field.Int("reporter_id").
			Optional(),
		field.String("reporter_email").
			Optional(),
		field.String("target_type"),
		field.Int("target_id"),
		field.Int("reason"),
		field.String("description").
			Optional(),
		field.String("status").
			Default("open"),
		field.String("admin_note").
			Optional(),
	}
}

// Indexes of the AbuseReport.
func (AbuseReport) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("target_type", "target_id"),
	}
}

func (AbuseReport) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
