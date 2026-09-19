package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/gofrs/uuid"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.String("type"),
		field.Enum("status").
			Values("queued", "processing", "suspending", "error", "canceled", "completed").
			Default("queued"),
		field.JSON("public_state", &types.TaskPublicState{}),
		field.Text("private_state").Optional(),
		field.UUID("correlation_id", uuid.Must(uuid.NewV4())).
			Optional().
			Immutable(),
		field.Int("user_tasks").Optional(),
		// creator_ip records the client IP of the request that created the task,
		// when the creation happened on an HTTP request context.
		field.String("creator_ip").Optional().MaxLen(45),
		// hidden is set when the owner deletes the task record; admin views keep it.
		field.Bool("hidden").Default(false),
	}
}

// Edges of the Task.
func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("tasks").
			Field("user_tasks").
			Unique(),
	}
}

func (Task) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
