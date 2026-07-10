package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// MediaProcessTask holds the schema definition for the MediaProcessTask entity.
//
// It is a deferred media post-processing work item (APP-101): one row per blob
// pending compression. The table is shared between image and video work via the
// media_type discriminator (video is a follow-up ticket).
//
// The references to entity/file/owner are stored as plain int columns (no ent
// FK edges) on purpose: entities are hard-deleted during recycle
// (inventory/file.go), so a hard FK constraint would break that path. The task
// resolves them through the existing inventory clients and self-heals (skips)
// when the referenced blob is gone.
type MediaProcessTask struct {
	ent.Schema
}

// Fields of the MediaProcessTask.
func (MediaProcessTask) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("media_type").
			Values("image", "video").
			Default("image"),
		field.Enum("status").
			Values("pending", "processing", "done", "failed", "skipped").
			Default("pending"),
		// entity_id is the physical blob to process (required).
		field.Int("entity_id"),
		// file_id is the logical file, for navigation/write-back (optional: the
		// file may be gone by processing time).
		field.Int("file_id").Optional(),
		// owner_id is the user that owns the blob, for quota accounting (required).
		field.Int("owner_id"),
		field.Int("attempts").Default(0),
		field.Text("error").Optional(),
		field.Int64("result_size").Optional(),
		field.JSON("props", &types.MediaProcessTaskProps{}).Optional(),
	}
}

// Edges of the MediaProcessTask.
func (MediaProcessTask) Edges() []ent.Edge {
	return nil
}

// Indexes of the MediaProcessTask.
func (MediaProcessTask) Indexes() []ent.Index {
	return []ent.Index{
		// Batch pickup: ListPending filters by (status, media_type).
		index.Fields("status", "media_type"),
		// Idempotency guard: "is there an active row for this entity?".
		index.Fields("entity_id", "status"),
	}
}

func (MediaProcessTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
