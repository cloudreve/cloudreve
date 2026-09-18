package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// StoragePolicy holds the schema definition for the storage policy entity.
type StoragePolicy struct {
	ent.Schema
}

func (StoragePolicy) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("type"),
		// Suspended policies keep serving reads but reject new uploads.
		field.Enum("status").
			Values("active", "suspended").
			Default("active"),
		field.String("server").
			Optional(),
		field.String("bucket_name").
			Optional(),
		field.Bool("is_private").
			Optional(),
		field.Text("access_key").
			Optional(),
		field.Text("secret_key").
			Optional(),
		field.Int64("max_size").
			Optional(),
		field.String("dir_name_rule").
			Optional(),
		field.String("file_name_rule").
			Optional(),
		field.JSON("settings", &types.PolicySetting{}).
			Default(&types.PolicySetting{}).
			Optional(),
		field.Int("node_id").Optional(),
	}
}

func (StoragePolicy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}

func (StoragePolicy) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("groups", Group.Type),
		edge.To("files", File.Type),
		edge.To("entities", Entity.Type),
		edge.From("node", Node.Type).
			Ref("storage_policy").
			Field("node_id").
			Unique(),
	}
}
