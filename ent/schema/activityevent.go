package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ActivityEvent holds one audit-log entry describing an action performed
// on the site. Rows are append-only; subject references are stored as plain
// IDs (no edges) so events survive hard-deletion of the referenced file,
// share, or user.
type ActivityEvent struct {
	ent.Schema
}

// Fields of the ActivityEvent.
func (ActivityEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int("type"),
		field.Int("actor_id").
			Optional(),
		field.String("ip").
			Optional(),
		field.String("cid").
			Optional(),
		field.Int("file_id").
			Optional(),
		field.Int("share_id").
			Optional(),
		field.JSON("extra", map[string]any{}).
			Optional(),
	}
}

// Indexes of the ActivityEvent.
func (ActivityEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("type", "created_at"),
		index.Fields("file_id", "created_at"),
		index.Fields("actor_id", "created_at"),
		index.Fields("share_id", "created_at"),
	}
}

func (ActivityEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
