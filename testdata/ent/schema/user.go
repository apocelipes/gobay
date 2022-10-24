package schema

import (
	"github.com/apocelipes/ent"
	"github.com/apocelipes/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("nickname").Default("jeff"),
		field.String("username").Unique(),
	}
}
