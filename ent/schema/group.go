package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
)

// Group holds the schema definition for the Group entity.
type Group struct {
	ent.Schema
}

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Int64("max_storage").
			Optional(),
		field.Int("speed_limit").
			Optional(),
		field.Bytes("permissions").GoType(&boolset.BooleanSet{}),
		field.JSON("settings", &types.GroupSetting{}).
			Default(&types.GroupSetting{}).
			Optional(),
		field.Int("storage_policy_id").Optional(),
	}
}

func (Group) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type),
		edge.From("storage_policies", StoragePolicy.Type).
			Ref("groups").
			Field("storage_policy_id").
			Unique(),
		// storage_policies_allowed is the set of storage policies a member of this
		// group may upload to. storage_policy_id above remains the default policy
		// used when the client does not pick one. Many-to-many.
		edge.To("storage_policies_allowed", StoragePolicy.Type),
	}
}
