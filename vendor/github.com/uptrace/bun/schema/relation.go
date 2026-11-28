package schema

import (
	"fmt"
)

// Relation type constants define the possible types of table relationships.
const (
	InvalidRelation = iota
	HasOneRelation
	BelongsToRelation
	HasManyRelation
	ManyToManyRelation
)

// Relation describes how two tables are related (has-one, belongs-to, has-many, or many-to-many).
type Relation struct {
	Type  int
	Field *Field // Has the bun tag defining this relation.

	// Base and Join can be explained with this query:
	//
	// SELECT * FROM base_table JOIN join_table
	JoinTable *Table
	BasePKs   []*Field
	JoinPKs   []*Field
	OnUpdate  string
	OnDelete  string
	Condition []string

	PolymorphicField *Field
	PolymorphicValue string

	M2MTable   *Table
	M2MBasePKs []*Field
	M2MJoinPKs []*Field
}

// References returns true if the table which defines this Relation
// needs to declare a foreign key constraint, as is the case
// for 'has-one' and 'belongs-to' relations. For other relations,
// the constraint is created either in the referencing table (1:N, 'has-many' relations)
// or the junction table (N:N, 'm2m' relations).
//
// Usage of `rel:` tag does not always imply creation of foreign keys (when WithForeignKeys() is not set)
// and can be used exclusively for joining tables at query time. For example:
//
//	type User struct {
//		ID int64			`bun:",pk"`
//		Profile *Profile	`bun:",rel:has-one,join:id=user_id"`
//	}
//
// Creating a FK users.id -> profiles.user_id would be confusing and incorrect,
// so for such cases References() returns false. One notable exception to this rule
// is when a Relation is defined in a junction table, in which case it is perfectly
// fine for its primary keys to reference other tables. Consider:
//
//	// UsersToGroups maps users to groups they follow.
//	type UsersToGroups struct {
//		UserID string	`bun:"user_id,pk"`		// Needs FK to users.id
//		GroupID string	`bun:"group_id,pk"`		// Needs FK to groups.id
//
//		User	*User	`bun:"rel:belongs-to,join:user_id=id"`
//		Group	*Group	`bun:"rel:belongs-to,join:group_id=id"`
//	}
//
// Here BooksToReaders has a composite primary key, composed of other primary keys.
func (r *Relation) References() bool {
	allPK := true
	nonePK := true
	for _, f := range r.BasePKs {
		allPK = allPK && f.IsPK
		nonePK = nonePK && !f.IsPK
	}

	// Erring on the side of caution, only create foreign keys
	// if the referencing columns are part of a composite PK
	// in the junction table of the m2m relationship.
	effectsM2M := r.Field.Table.IsM2MTable && allPK

	return (r.Type == HasOneRelation || r.Type == BelongsToRelation) && (effectsM2M || nonePK)
}

func (r *Relation) String() string {
	return fmt.Sprintf("relation=%s", r.Field.GoName)
}
