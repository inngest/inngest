package schema

import (
	"cmp"
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/jinzhu/inflection"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/tagparser"
)

const (
	beforeAppendModelHookFlag internal.Flag = 1 << iota
	beforeScanRowHookFlag
	afterScanRowHookFlag
)

var (
	baseModelType      = reflect.TypeFor[BaseModel]()
	tableNameInflector = inflection.Plural
)

type BaseModel struct{}

// SetTableNameInflector overrides the default func that pluralizes
// model name to get table name, e.g. my_article becomes my_articles.
func SetTableNameInflector(fn func(string) string) {
	tableNameInflector = fn
}

// Table represents a SQL table created from Go struct.
type Table struct {
	dialect Dialect

	Type      reflect.Type
	ZeroValue reflect.Value // reflect.Struct
	ZeroIface any           // struct pointer

	TypeName  string
	ModelName string

	Schema            string
	Name              string
	SQLName           Safe
	SQLNameForSelects Safe
	Alias             string
	SQLAlias          Safe

	allFields  []*Field // all fields including scanonly
	Fields     []*Field // PKs + DataFields
	PKs        []*Field
	DataFields []*Field
	relFields  []*Field

	FieldMap  map[string]*Field
	StructMap map[string]*structField

	IsM2MTable bool // If true, this table is the "junction table" of an m2m relation.
	Relations  map[string]*Relation
	Unique     map[string][]*Field

	SoftDeleteField       *Field
	UpdateSoftDeleteField func(fv reflect.Value, tm time.Time) error

	flags internal.Flag
}

type structField struct {
	Index []int
	Table *Table
}

func (table *Table) init(dialect Dialect, typ reflect.Type) {
	table.dialect = dialect
	table.Type = typ
	table.ZeroValue = reflect.New(table.Type).Elem()
	table.ZeroIface = reflect.New(table.Type).Interface()
	table.TypeName = internal.ToExported(table.Type.Name())
	table.ModelName = internal.Underscore(table.Type.Name())
	tableName := tableNameInflector(table.ModelName)
	table.setName(tableName)
	table.Alias = table.ModelName
	table.SQLAlias = table.quoteIdent(table.ModelName)
	table.Schema = dialect.DefaultSchema()

	table.Fields = make([]*Field, 0, typ.NumField())
	table.FieldMap = make(map[string]*Field, typ.NumField())
	table.processFields(typ)

	hooks := []struct {
		typ  reflect.Type
		flag internal.Flag
	}{
		{beforeAppendModelHookType, beforeAppendModelHookFlag},

		{beforeScanRowHookType, beforeScanRowHookFlag},
		{afterScanRowHookType, afterScanRowHookFlag},
	}

	typ = reflect.PointerTo(table.Type)
	for _, hook := range hooks {
		if typ.Implements(hook.typ) {
			table.flags = table.flags.Set(hook.flag)
		}
	}
}

func (t *Table) processFields(typ reflect.Type) {
	type embeddedField struct {
		prefix     string
		index      []int
		unexported bool
		subtable   *Table
		subfield   *Field
	}

	names := make(map[string]struct{})
	embedded := make([]embeddedField, 0, 10)
	ebdStructs := make(map[string]*structField, 0)

	for i, n := 0, typ.NumField(); i < n; i++ {
		sf := typ.Field(i)
		unexported := sf.PkgPath != ""

		tagstr := sf.Tag.Get("bun")
		if tagstr == "-" {
			names[sf.Name] = struct{}{}
			continue
		}
		tag := tagparser.Parse(tagstr)

		if unexported && !sf.Anonymous { // unexported
			continue
		}

		if sf.Anonymous {
			if sf.Name == "BaseModel" && sf.Type == baseModelType {
				t.processBaseModelField(sf)
				continue
			}

			sfType := sf.Type
			if sfType.Kind() == reflect.Ptr {
				sfType = sfType.Elem()
			}

			if sfType.Kind() != reflect.Struct { // ignore unexported non-struct types
				continue
			}

			subtable := t.dialect.Tables().InProgress(sfType)

			for _, subfield := range subtable.allFields {
				embedded = append(embedded, embeddedField{
					index:      sf.Index,
					unexported: unexported,
					subtable:   subtable,
					subfield:   subfield,
				})
			}
			if len(subtable.StructMap) > 0 {
				for k, v := range subtable.StructMap {
					// NOTE: conflict Struct name
					if _, ok := ebdStructs[k]; !ok {
						ebdStructs[k] = &structField{
							Index: makeIndex(sf.Index, v.Index),
							Table: v.Table,
						}
					}
				}
			}

			if tagstr != "" {
				tag := tagparser.Parse(tagstr)
				if tag.HasOption("inherit") || tag.HasOption("extend") {
					t.Name = subtable.Name
					t.TypeName = subtable.TypeName
					t.SQLName = subtable.SQLName
					t.SQLNameForSelects = subtable.SQLNameForSelects
					t.Alias = subtable.Alias
					t.SQLAlias = subtable.SQLAlias
					t.ModelName = subtable.ModelName
				}
			}

			continue
		}

		if prefix, ok := tag.Option("embed"); ok {
			fieldType := indirectType(sf.Type)
			if fieldType.Kind() != reflect.Struct {
				panic(fmt.Errorf("bun: embed %s.%s: got %s, wanted reflect.Struct",
					t.TypeName, sf.Name, fieldType.Kind()))
			}

			subtable := t.dialect.Tables().InProgress(fieldType)
			for _, subfield := range subtable.allFields {
				embedded = append(embedded, embeddedField{
					prefix:     prefix,
					index:      sf.Index,
					unexported: unexported,
					subtable:   subtable,
					subfield:   subfield,
				})
			}
			if len(subtable.StructMap) > 0 {
				for k, v := range subtable.StructMap {
					// NOTE: conflict Struct name
					k = prefix + k
					if _, ok := ebdStructs[k]; !ok {
						ebdStructs[k] = &structField{
							Index: makeIndex(sf.Index, v.Index),
							Table: subtable,
						}
					}
				}
			}
			continue
		}

		field := t.newField(sf, tag)
		t.addField(field)
		names[field.Name] = struct{}{}

		if field.IndirectType.Kind() == reflect.Struct {
			if t.StructMap == nil {
				t.StructMap = make(map[string]*structField)
			}
			t.StructMap[field.Name] = &structField{
				Index: field.Index,
				Table: t.dialect.Tables().InProgress(field.IndirectType),
			}
		}
	}

	// Only unambiguous embedded fields must be serialized.
	ambiguousNames := make(map[string]int)
	ambiguousTags := make(map[string]int)

	// Embedded types can never override a field that was already present at
	// the top-level.
	for name := range names {
		ambiguousNames[name]++
		ambiguousTags[name]++
	}

	for _, f := range embedded {
		ambiguousNames[f.prefix+f.subfield.Name]++
		if !f.subfield.Tag.IsZero() {
			ambiguousTags[f.prefix+f.subfield.Name]++
		}
	}

	for _, embfield := range embedded {
		if ambiguousNames[embfield.prefix+embfield.subfield.Name] > 1 &&
			!(!embfield.subfield.Tag.IsZero() && ambiguousTags[embfield.prefix+embfield.subfield.Name] == 1) {
			continue // ambiguous embedded field
		}

		subfield := embfield.subfield.Clone()

		subfield.Index = makeIndex(embfield.index, subfield.Index)
		if embfield.prefix != "" {
			subfield.Name = embfield.prefix + subfield.Name
			subfield.SQLName = t.quoteIdent(subfield.Name)
		}
		t.addField(subfield)
		if v, ok := subfield.Tag.Options["unique"]; ok {
			t.addUnique(subfield, embfield.prefix, v)
		}
	}

	if len(ebdStructs) > 0 && t.StructMap == nil {
		t.StructMap = make(map[string]*structField)
	}
	for name, sfield := range ebdStructs {
		if _, ok := t.StructMap[name]; !ok {
			t.StructMap[name] = sfield
		}
	}

	if len(embedded) > 0 {
		// https://github.com/uptrace/bun/issues/1095
		// < v1.2, all fields follow the order corresponding to the struct
		// >= v1.2, < v1.2.8, fields of nested structs have been moved to the end.
		// >= v1.2.8, The default behavior remains the same as initially,
		sortFieldsByStruct(t.allFields)
		sortFieldsByStruct(t.Fields)
		sortFieldsByStruct(t.PKs)
		sortFieldsByStruct(t.DataFields)
	}
}

func sortFieldsByStruct(fields []*Field) {
	slices.SortFunc(fields, func(left, right *Field) int {
		for k := 0; k < len(left.Index) && k < len(right.Index); k++ {
			if res := cmp.Compare(left.Index[k], right.Index[k]); res != 0 {
				return res
			}
		}
		// NOTE: should not reach
		return 0
	})
}

func (t *Table) addUnique(field *Field, prefix string, tagOptions []string) {
	var names []string
	if len(tagOptions) == 1 {
		// Split the value by comma, this will allow multiple names to be specified.
		// We can use this to create multiple named unique constraints where a single column
		// might be included in multiple constraints.
		names = strings.Split(tagOptions[0], ",")
	} else {
		names = tagOptions
	}

	for _, uname := range names {
		if t.Unique == nil {
			t.Unique = make(map[string][]*Field)
		}
		if uname != "" && prefix != "" {
			uname = prefix + uname
		}
		t.Unique[uname] = append(t.Unique[uname], field)
	}
}

func (t *Table) setName(name string) {
	t.Name = name
	t.SQLName = t.quoteIdent(name)
	t.SQLNameForSelects = t.quoteIdent(name)
	if t.SQLAlias == "" {
		t.Alias = name
		t.SQLAlias = t.quoteIdent(name)
	}
}

func (t *Table) String() string {
	return "model=" + t.TypeName
}

func (t *Table) CheckPKs() error {
	if len(t.PKs) == 0 {
		return fmt.Errorf("bun: %s does not have primary keys", t)
	}
	return nil
}

func (t *Table) addField(field *Field) {
	t.allFields = append(t.allFields, field)

	if field.Tag.HasOption("rel") || field.Tag.HasOption("m2m") {
		t.relFields = append(t.relFields, field)
		return
	}

	if field.Tag.HasOption("join") {
		internal.Warn.Printf(
			`%s.%s "join" option must come together with "rel" option`,
			t.TypeName, field.GoName,
		)
	}

	t.FieldMap[field.Name] = field
	if altName, ok := field.Tag.Option("alt"); ok {
		t.FieldMap[altName] = field
	}

	if field.Tag.HasOption("scanonly") {
		return
	}

	if _, ok := field.Tag.Options["soft_delete"]; ok {
		t.SoftDeleteField = field
		t.UpdateSoftDeleteField = softDeleteFieldUpdater(field)
	}

	t.Fields = append(t.Fields, field)
	if field.IsPK {
		t.PKs = append(t.PKs, field)
	} else {
		t.DataFields = append(t.DataFields, field)
	}
}

func (t *Table) LookupField(name string) *Field {
	if field, ok := t.FieldMap[name]; ok {
		return field
	}

	table := t
	var index []int
	for {
		structName, columnName, ok := strings.Cut(name, "__")
		if !ok {
			field, ok := table.FieldMap[name]
			if !ok {
				return nil
			}
			return field.WithIndex(index)
		}
		name = columnName

		strct := table.StructMap[structName]
		if strct == nil {
			return nil
		}
		table = strct.Table
		index = append(index, strct.Index...)
	}
}

func (t *Table) HasField(name string) bool {
	_, ok := t.FieldMap[name]
	return ok
}

func (t *Table) Field(name string) (*Field, error) {
	field, ok := t.FieldMap[name]
	if !ok {
		return nil, fmt.Errorf("bun: %s does not have column=%s", t, name)
	}
	return field, nil
}

func (t *Table) fieldByGoName(name string) *Field {
	for _, f := range t.allFields {
		if f.GoName == name {
			return f
		}
	}
	return nil
}

func (t *Table) processBaseModelField(f reflect.StructField) {
	tag := tagparser.Parse(f.Tag.Get("bun"))

	if isKnownTableOption(tag.Name) {
		internal.Warn.Printf(
			"%s.%s tag name %q is also an option name, is it a mistake? Try table:%s.",
			t.TypeName, f.Name, tag.Name, tag.Name,
		)
	}

	for name := range tag.Options {
		if !isKnownTableOption(name) {
			internal.Warn.Printf("%s.%s has unknown tag option: %q", t.TypeName, f.Name, name)
		}
	}

	if tag.Name != "" {
		schema, _ := t.schemaFromTagName(tag.Name)
		t.Schema = schema

		// Eventually, we should only assign the "table" portion as the table name,
		// which will also require a change in how the table name is appended to queries.
		// Until that is done, set table name to tag.Name.
		t.setName(tag.Name)
	}

	if s, ok := tag.Option("table"); ok {
		schema, _ := t.schemaFromTagName(s)
		t.Schema = schema
		t.setName(s)
	}

	if s, ok := tag.Option("select"); ok {
		t.SQLNameForSelects = t.quoteTableName(s)
	}

	if s, ok := tag.Option("alias"); ok {
		t.Alias = s
		t.SQLAlias = t.quoteIdent(s)
	}
}

// schemaFromTagName splits the bun.BaseModel tag name into schema and table name
// in case it is specified in the "schema"."table" format.
// Assume default schema if one isn't explicitly specified.
func (t *Table) schemaFromTagName(name string) (string, string) {
	schema, table := t.dialect.DefaultSchema(), name
	if schemaTable := strings.Split(name, "."); len(schemaTable) == 2 {
		schema, table = schemaTable[0], schemaTable[1]
	}
	return schema, table
}

// nolint
func (t *Table) newField(sf reflect.StructField, tag tagparser.Tag) *Field {
	sqlName := internal.Underscore(sf.Name)
	if tag.Name != "" && tag.Name != sqlName {
		if isKnownFieldOption(tag.Name) {
			internal.Warn.Printf(
				"%s.%s tag name %q is also an option name, is it a mistake? Try column:%s.",
				t.TypeName, sf.Name, tag.Name, tag.Name,
			)
		}
		sqlName = tag.Name
	}

	if s, ok := tag.Option("column"); ok {
		sqlName = s
	}

	for name := range tag.Options {
		if !isKnownFieldOption(name) {
			internal.Warn.Printf("%s.%s has unknown tag option: %q", t.TypeName, sf.Name, name)
		}
	}

	field := &Field{
		Table:       t,
		StructField: sf,
		IsPtr:       sf.Type.Kind() == reflect.Ptr,

		Tag:          tag,
		IndirectType: indirectType(sf.Type),
		Index:        sf.Index,

		Name:    sqlName,
		GoName:  sf.Name,
		SQLName: t.quoteIdent(sqlName),
	}

	field.NotNull = tag.HasOption("notnull")
	field.NullZero = tag.HasOption("nullzero")
	if tag.HasOption("pk") {
		field.IsPK = true
		field.NotNull = true
	}
	if tag.HasOption("autoincrement") {
		field.AutoIncrement = true
		field.NotNull = true
		field.NullZero = true
	}
	if tag.HasOption("identity") {
		field.Identity = true
	}

	if v, ok := tag.Options["unique"]; ok {
		t.addUnique(field, "", v)
	}
	if s, ok := tag.Option("default"); ok {
		field.SQLDefault = s
	}
	if s, ok := field.Tag.Option("type"); ok {
		field.UserSQLType = s
	}
	field.DiscoveredSQLType = DiscoverSQLType(field.IndirectType)
	field.Append = FieldAppender(t.dialect, field)
	field.Scan = FieldScanner(t.dialect, field)
	field.IsZero = zeroChecker(field.StructField.Type)

	return field
}

//---------------------------------------------------------------------------------------

func (t *Table) initRelations() {
	for _, field := range t.relFields {
		t.processRelation(field)
	}
	t.relFields = nil
}

func (t *Table) processRelation(field *Field) {
	if rel, ok := field.Tag.Option("rel"); ok {
		t.initRelation(field, rel)
		return
	}
	if field.Tag.HasOption("m2m") {
		t.addRelation(t.m2mRelation(field))
		return
	}
	panic("not reached")
}

func (t *Table) initRelation(field *Field, rel string) {
	switch rel {
	case "belongs-to":
		t.addRelation(t.belongsToRelation(field))
	case "has-one":
		t.addRelation(t.hasOneRelation(field))
	case "has-many":
		t.addRelation(t.hasManyRelation(field))
	default:
		panic(fmt.Errorf("bun: unknown relation=%s on field=%s", rel, field.GoName))
	}
}

func (t *Table) addRelation(rel *Relation) {
	if t.Relations == nil {
		t.Relations = make(map[string]*Relation)
	}
	_, ok := t.Relations[rel.Field.GoName]
	if ok {
		panic(fmt.Errorf("%s already has %s", t, rel))
	}
	t.Relations[rel.Field.GoName] = rel
}

func (t *Table) belongsToRelation(field *Field) *Relation {
	joinTable := t.dialect.Tables().InProgress(field.IndirectType)
	if err := joinTable.CheckPKs(); err != nil {
		panic(err)
	}

	rel := &Relation{
		Type:      BelongsToRelation,
		Field:     field,
		JoinTable: joinTable,
	}

	if field.Tag.HasOption("join_on") {
		rel.Condition = field.Tag.Options["join_on"]
	}

	if t.dialect.Features().Has(feature.FKDefaultOnAction) {
		rel.OnUpdate = "ON UPDATE NO ACTION"
		rel.OnDelete = "ON DELETE NO ACTION"
	}
	if onUpdate, ok := field.Tag.Options["on_update"]; ok {
		if len(onUpdate) > 1 {
			panic(fmt.Errorf("bun: %s belongs-to %s: on_update option must be a single field", t.TypeName, field.GoName))
		}

		rule := strings.ToUpper(onUpdate[0])
		if !isKnownFKRule(rule) {
			internal.Warn.Printf("bun: %s belongs-to %s: unknown on_update rule %s", t.TypeName, field.GoName, rule)
		}

		s := fmt.Sprintf("ON UPDATE %s", rule)
		rel.OnUpdate = s
	}

	if onDelete, ok := field.Tag.Options["on_delete"]; ok {
		if len(onDelete) > 1 {
			panic(fmt.Errorf("bun: %s belongs-to %s: on_delete option must be a single field", t.TypeName, field.GoName))
		}

		rule := strings.ToUpper(onDelete[0])
		if !isKnownFKRule(rule) {
			internal.Warn.Printf("bun: %s belongs-to %s: unknown on_delete rule %s", t.TypeName, field.GoName, rule)
		}
		s := fmt.Sprintf("ON DELETE %s", rule)
		rel.OnDelete = s
	}

	if join, ok := field.Tag.Options["join"]; ok {
		baseColumns, joinColumns := parseRelationJoin(join)
		for i, baseColumn := range baseColumns {
			joinColumn := joinColumns[i]

			if f := t.FieldMap[baseColumn]; f != nil {
				rel.BasePKs = append(rel.BasePKs, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s belongs-to %s: %s must have column %s",
					t.TypeName, field.GoName, t.TypeName, baseColumn,
				))
			}

			if f := joinTable.FieldMap[joinColumn]; f != nil {
				rel.JoinPKs = append(rel.JoinPKs, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s belongs-to %s: %s must have column %s",
					t.TypeName, field.GoName, joinTable.TypeName, joinColumn,
				))
			}
		}
		return rel
	}

	rel.JoinPKs = joinTable.PKs
	fkPrefix := internal.Underscore(field.GoName) + "_"
	for _, joinPK := range joinTable.PKs {
		fkName := fkPrefix + joinPK.Name
		if fk := t.FieldMap[fkName]; fk != nil {
			rel.BasePKs = append(rel.BasePKs, fk)
			continue
		}

		if fk := t.FieldMap[joinPK.Name]; fk != nil {
			rel.BasePKs = append(rel.BasePKs, fk)
			continue
		}

		panic(fmt.Errorf(
			"bun: %s belongs-to %s: %s must have column %s "+
				"(to override, use join:base_column=join_column tag on %s field)",
			t.TypeName, field.GoName, t.TypeName, fkName, field.GoName,
		))
	}
	return rel
}

func (t *Table) hasOneRelation(field *Field) *Relation {
	if err := t.CheckPKs(); err != nil {
		panic(err)
	}

	joinTable := t.dialect.Tables().InProgress(field.IndirectType)
	rel := &Relation{
		Type:      HasOneRelation,
		Field:     field,
		JoinTable: joinTable,
	}

	if field.Tag.HasOption("join_on") {
		rel.Condition = field.Tag.Options["join_on"]
	}

	if join, ok := field.Tag.Options["join"]; ok {
		baseColumns, joinColumns := parseRelationJoin(join)
		for i, baseColumn := range baseColumns {
			if f := t.FieldMap[baseColumn]; f != nil {
				rel.BasePKs = append(rel.BasePKs, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-one %s: %s must have column %s",
					field.GoName, t.TypeName, t.TypeName, baseColumn,
				))
			}

			joinColumn := joinColumns[i]
			if f := joinTable.FieldMap[joinColumn]; f != nil {
				rel.JoinPKs = append(rel.JoinPKs, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-one %s: %s must have column %s",
					field.GoName, t.TypeName, joinTable.TypeName, joinColumn,
				))
			}
		}
		return rel
	}

	rel.BasePKs = t.PKs
	fkPrefix := internal.Underscore(t.ModelName) + "_"
	for _, pk := range t.PKs {
		fkName := fkPrefix + pk.Name
		if f := joinTable.FieldMap[fkName]; f != nil {
			rel.JoinPKs = append(rel.JoinPKs, f)
			continue
		}

		if f := joinTable.FieldMap[pk.Name]; f != nil {
			rel.JoinPKs = append(rel.JoinPKs, f)
			continue
		}

		panic(fmt.Errorf(
			"bun: %s has-one %s: %s must have column %s "+
				"(to override, use join:base_column=join_column tag on %s field)",
			field.GoName, t.TypeName, joinTable.TypeName, fkName, field.GoName,
		))
	}
	return rel
}

func (t *Table) hasManyRelation(field *Field) *Relation {
	if err := t.CheckPKs(); err != nil {
		panic(err)
	}
	if field.IndirectType.Kind() != reflect.Slice {
		panic(fmt.Errorf(
			"bun: %s.%s has-many relation requires slice, got %q",
			t.TypeName, field.GoName, field.IndirectType.Kind(),
		))
	}

	joinTable := t.dialect.Tables().InProgress(indirectType(field.IndirectType.Elem()))
	polymorphicValue, isPolymorphic := field.Tag.Option("polymorphic")
	rel := &Relation{
		Type:      HasManyRelation,
		Field:     field,
		JoinTable: joinTable,
	}

	if field.Tag.HasOption("join_on") {
		rel.Condition = field.Tag.Options["join_on"]
	}

	var polymorphicColumn string

	if join, ok := field.Tag.Options["join"]; ok {
		baseColumns, joinColumns := parseRelationJoin(join)
		for i, baseColumn := range baseColumns {
			joinColumn := joinColumns[i]

			if isPolymorphic && baseColumn == "type" {
				polymorphicColumn = joinColumn
				continue
			}

			if f := t.FieldMap[baseColumn]; f != nil {
				rel.BasePKs = append(rel.BasePKs, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-many %s: %s must have column %s",
					t.TypeName, field.GoName, t.TypeName, baseColumn,
				))
			}

			if f := joinTable.FieldMap[joinColumn]; f != nil {
				rel.JoinPKs = append(rel.JoinPKs, f)
			} else {
				panic(fmt.Errorf(
					"bun: %s has-many %s: %s must have column %s",
					t.TypeName, field.GoName, joinTable.TypeName, joinColumn,
				))
			}
		}
	} else {
		rel.BasePKs = t.PKs
		fkPrefix := internal.Underscore(t.ModelName) + "_"
		if isPolymorphic {
			polymorphicColumn = fkPrefix + "type"
		}

		for _, pk := range t.PKs {
			joinColumn := fkPrefix + pk.Name
			if fk := joinTable.FieldMap[joinColumn]; fk != nil {
				rel.JoinPKs = append(rel.JoinPKs, fk)
				continue
			}

			if fk := joinTable.FieldMap[pk.Name]; fk != nil {
				rel.JoinPKs = append(rel.JoinPKs, fk)
				continue
			}

			panic(fmt.Errorf(
				"bun: %s has-many %s: %s must have column %s "+
					"(to override, use join:base_column=join_column tag on the field %s)",
				t.TypeName, field.GoName, joinTable.TypeName, joinColumn, field.GoName,
			))
		}
	}

	if isPolymorphic {
		rel.PolymorphicField = joinTable.FieldMap[polymorphicColumn]
		if rel.PolymorphicField == nil {
			panic(fmt.Errorf(
				"bun: %s has-many %s: %s must have polymorphic column %s",
				t.TypeName, field.GoName, joinTable.TypeName, polymorphicColumn,
			))
		}

		if polymorphicValue == "" {
			polymorphicValue = t.ModelName
		}
		rel.PolymorphicValue = polymorphicValue
	}

	return rel
}

func (t *Table) m2mRelation(field *Field) *Relation {
	if field.IndirectType.Kind() != reflect.Slice {
		panic(fmt.Errorf(
			"bun: %s.%s m2m relation requires slice, got %q",
			t.TypeName, field.GoName, field.IndirectType.Kind(),
		))
	}
	joinTable := t.dialect.Tables().InProgress(indirectType(field.IndirectType.Elem()))

	if err := t.CheckPKs(); err != nil {
		panic(err)
	}
	if err := joinTable.CheckPKs(); err != nil {
		panic(err)
	}

	m2mTableName, ok := field.Tag.Option("m2m")
	if !ok {
		panic(fmt.Errorf("bun: %s must have m2m tag option", field.GoName))
	}

	m2mTable := t.dialect.Tables().ByName(m2mTableName)
	if m2mTable == nil {
		panic(fmt.Errorf(
			"bun: can't find m2m %s table (use db.RegisterModel)",
			m2mTableName,
		))
	}

	rel := &Relation{
		Type:      ManyToManyRelation,
		Field:     field,
		JoinTable: joinTable,
		M2MTable:  m2mTable,
	}
	m2mTable.markM2M()

	if field.Tag.HasOption("join_on") {
		rel.Condition = field.Tag.Options["join_on"]
	}

	var leftColumn, rightColumn string

	if join, ok := field.Tag.Options["join"]; ok {
		left, right := parseRelationJoin(join)
		leftColumn = left[0]
		rightColumn = right[0]
	} else {
		leftColumn = t.TypeName
		rightColumn = joinTable.TypeName
	}

	leftField := m2mTable.fieldByGoName(leftColumn)
	if leftField == nil {
		panic(fmt.Errorf(
			"bun: %s many-to-many %s: %s must have field %s "+
				"(to override, use tag join:LeftField=RightField on field %s.%s",
			t.TypeName, field.GoName, m2mTable.TypeName, leftColumn, t.TypeName, field.GoName,
		))
	}

	rightField := m2mTable.fieldByGoName(rightColumn)
	if rightField == nil {
		panic(fmt.Errorf(
			"bun: %s many-to-many %s: %s must have field %s "+
				"(to override, use tag join:LeftField=RightField on field %s.%s",
			t.TypeName, field.GoName, m2mTable.TypeName, rightColumn, t.TypeName, field.GoName,
		))
	}

	leftRel := m2mTable.belongsToRelation(leftField)
	rel.BasePKs = leftRel.JoinPKs
	rel.M2MBasePKs = leftRel.BasePKs

	rightRel := m2mTable.belongsToRelation(rightField)
	rel.JoinPKs = rightRel.JoinPKs
	rel.M2MJoinPKs = rightRel.BasePKs

	return rel
}

func (t *Table) markM2M() {
	t.IsM2MTable = true
}

//------------------------------------------------------------------------------

func (t *Table) Dialect() Dialect { return t.dialect }

func (t *Table) HasBeforeAppendModelHook() bool { return t.flags.Has(beforeAppendModelHookFlag) }

func (t *Table) HasBeforeScanRowHook() bool { return t.flags.Has(beforeScanRowHookFlag) }
func (t *Table) HasAfterScanRowHook() bool  { return t.flags.Has(afterScanRowHookFlag) }

//------------------------------------------------------------------------------

func (t *Table) AppendNamedArg(
	gen QueryGen, b []byte, name string, strct reflect.Value,
) ([]byte, bool) {
	if field, ok := t.FieldMap[name]; ok {
		return field.AppendValue(gen, b, strct), true
	}
	return b, false
}

func (t *Table) quoteTableName(s string) Safe {
	// Don't quote if table name contains placeholder (?) or parentheses.
	if strings.IndexByte(s, '?') >= 0 ||
		strings.IndexByte(s, '(') >= 0 ||
		strings.IndexByte(s, ')') >= 0 {
		return Safe(s)
	}
	return t.quoteIdent(s)
}

func (t *Table) quoteIdent(s string) Safe {
	return Safe(NewQueryGen(t.dialect).AppendIdent(nil, s))
}

func isKnownTableOption(name string) bool {
	switch name {
	case "table", "alias", "select":
		return true
	}
	return false
}

func isKnownFieldOption(name string) bool {
	switch name {
	case "column",
		"alt",
		"type",
		"array",
		"hstore",
		"composite",
		"multirange",
		"json_use_number",
		"msgpack",
		"notnull",
		"nullzero",
		"default",
		"unique",
		"soft_delete",
		"scanonly",
		"skipupdate",

		"pk",
		"autoincrement",
		"rel",
		"join",
		"join_on",
		"on_update",
		"on_delete",
		"m2m",
		"polymorphic",
		"identity":
		return true
	}
	return false
}

func isKnownFKRule(name string) bool {
	switch name {
	case "CASCADE",
		"RESTRICT",
		"SET NULL",
		"SET DEFAULT":
		return true
	}
	return false
}

func parseRelationJoin(join []string) ([]string, []string) {
	var ss []string
	if len(join) == 1 {
		ss = strings.Split(join[0], ",")
	} else {
		ss = join
	}

	baseColumns := make([]string, len(ss))
	joinColumns := make([]string, len(ss))
	for i, s := range ss {
		ss := strings.Split(strings.TrimSpace(s), "=")
		if len(ss) != 2 {
			panic(fmt.Errorf("can't parse relation join: %q", join))
		}
		baseColumns[i] = ss[0]
		joinColumns[i] = ss[1]
	}
	return baseColumns, joinColumns
}

//------------------------------------------------------------------------------

func softDeleteFieldUpdater(field *Field) func(fv reflect.Value, tm time.Time) error {
	typ := field.StructField.Type

	switch typ {
	case timeType:
		return func(fv reflect.Value, tm time.Time) error {
			ptr := fv.Addr().Interface().(*time.Time)
			*ptr = tm
			return nil
		}
	case nullTimeType:
		return func(fv reflect.Value, tm time.Time) error {
			ptr := fv.Addr().Interface().(*sql.NullTime)
			*ptr = sql.NullTime{Time: tm}
			return nil
		}
	case nullIntType:
		return func(fv reflect.Value, tm time.Time) error {
			ptr := fv.Addr().Interface().(*sql.NullInt64)
			*ptr = sql.NullInt64{Int64: tm.UnixNano()}
			return nil
		}
	}

	switch field.IndirectType.Kind() {
	case reflect.Int64:
		return func(fv reflect.Value, tm time.Time) error {
			ptr := fv.Addr().Interface().(*int64)
			*ptr = tm.UnixNano()
			return nil
		}
	case reflect.Ptr:
		typ = typ.Elem()
	default:
		return softDeleteFieldUpdaterFallback(field)
	}

	switch typ { //nolint:gocritic
	case timeType:
		return func(fv reflect.Value, tm time.Time) error {
			fv.Set(reflect.ValueOf(&tm))
			return nil
		}
	}

	switch typ.Kind() { //nolint:gocritic
	case reflect.Int64:
		return func(fv reflect.Value, tm time.Time) error {
			utime := tm.UnixNano()
			fv.Set(reflect.ValueOf(&utime))
			return nil
		}
	}

	return softDeleteFieldUpdaterFallback(field)
}

func softDeleteFieldUpdaterFallback(field *Field) func(fv reflect.Value, tm time.Time) error {
	return func(fv reflect.Value, tm time.Time) error {
		return field.ScanWithCheck(fv, tm)
	}
}

func makeIndex(a, b []int) []int {
	dest := make([]int, 0, len(a)+len(b))
	dest = append(dest, a...)
	dest = append(dest, b...)
	return dest
}
