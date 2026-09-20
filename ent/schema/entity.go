package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/gofrs/uuid"
)

// Entity holds the schema definition for the Entity.
type Entity struct {
	ent.Schema
}

// Fields of the Entity.
func (Entity) Fields() []ent.Field {
	return []ent.Field{
		field.Int("type"),
		field.Text("source"),
		field.Int64("size"),
		field.Int("reference_count").Default(1),
		// Content hash asserted by the uploader (sha256 hex). Used for
		// duplicate detection / instant upload; not verified server-side
		// because remote-policy blobs never pass through this server.
		field.String("hash").
			Optional().
			MaxLen(64),
		field.Int("storage_policy_entities"),
		field.Int("created_by").Optional(),
		field.UUID("upload_session_id", uuid.Must(uuid.NewV4())).
			Optional().
			Nillable(),
		field.JSON("props", &types.EntityProps{}).
			Optional().
			StorageKey("recycle_options"),
	}
}

// Edges of the Entity.
func (Entity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("file", File.Type).
			Ref("entities"),
		edge.From("user", User.Type).
			Field("created_by").
			Unique().
			Ref("entities"),
		edge.From("storage_policy", StoragePolicy.Type).
			Ref("entities").
			Field("storage_policy_entities").
			Unique().
			Required(),
	}
}

func (Entity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}

func (Entity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("hash", "size"),
	}
}
