package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
)

// AclEntry holds the schema definition for per-file access control entries.
// Each row grants a capability bitmask (read/create/update/delete) on a file
// or directory to a subject: a specific user, a user group, anonymous
// visitors, or every other authenticated user.
type AclEntry struct {
	ent.Schema
}

// Fields of the AclEntry.
func (AclEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int("file_id"),
		field.Enum("subject_type").
			Values("user", "group", "anonymous", "everyone"),
		field.Int("subject_id").
			Optional(),
		field.Bytes("permissions").GoType(&boolset.BooleanSet{}),
	}
}

// Edges of the AclEntry.
func (AclEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("file", File.Type).
			Ref("acl_entries").
			Field("file_id").
			Unique().
			Required(),
	}
}

// Indexes of the AclEntry.
func (AclEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("file_id", "subject_type", "subject_id").
			Unique(),
	}
}

func (AclEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
