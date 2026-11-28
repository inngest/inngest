package pgdialect

import (
	"database/sql"
	"encoding/json"
	"net"
	"reflect"
	"strings"

	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

const (
	// Date / Time
	pgTypeTimestamp       = "TIMESTAMP"                // Timestamp
	pgTypeTimestampWithTz = "TIMESTAMP WITH TIME ZONE" // Timestamp with a time zone
	pgTypeTimestampTz     = "TIMESTAMPTZ"              // Timestamp with a time zone (alias)
	pgTypeDate            = "DATE"                     // Date
	pgTypeTime            = "TIME"                     // Time without a time zone
	pgTypeTimeTz          = "TIME WITH TIME ZONE"      // Time with a time zone
	pgTypeInterval        = "INTERVAL"                 // Time interval

	// Network Addresses
	pgTypeInet    = "INET"    // IPv4 or IPv6 hosts and networks
	pgTypeCidr    = "CIDR"    // IPv4 or IPv6 networks
	pgTypeMacaddr = "MACADDR" // MAC addresses

	// Serial Types
	pgTypeSmallSerial = "SMALLSERIAL" // 2 byte autoincrementing integer
	pgTypeSerial      = "SERIAL"      // 4 byte autoincrementing integer
	pgTypeBigSerial   = "BIGSERIAL"   // 8 byte autoincrementing integer

	// Character Types
	pgTypeChar             = "CHAR"              // fixed length string (blank padded)
	pgTypeCharacter        = "CHARACTER"         // alias for CHAR
	pgTypeText             = "TEXT"              // variable length string without limit
	pgTypeVarchar          = "VARCHAR"           // variable length string with optional limit
	pgTypeCharacterVarying = "CHARACTER VARYING" // alias for VARCHAR

	// Binary Data Types
	pgTypeBytea = "BYTEA" // binary string
)

var (
	ipType             = reflect.TypeFor[net.IP]()
	ipNetType          = reflect.TypeFor[net.IPNet]()
	jsonRawMessageType = reflect.TypeFor[json.RawMessage]()
	nullStringType     = reflect.TypeFor[sql.NullString]()
)

func (d *Dialect) DefaultVarcharLen() int {
	return 0
}

func (d *Dialect) DefaultSchema() string {
	return "public"
}

func fieldSQLType(field *schema.Field) string {
	if field.UserSQLType != "" {
		return field.UserSQLType
	}

	if v, ok := field.Tag.Option("composite"); ok {
		return v
	}
	if field.Tag.HasOption("hstore") {
		return sqltype.HSTORE
	}

	if field.Tag.HasOption("array") {
		switch field.IndirectType.Kind() {
		case reflect.Slice, reflect.Array:
			sqlType := sqlType(field.IndirectType.Elem())
			return sqlType + "[]"
		}
	}

	if field.DiscoveredSQLType == sqltype.Blob {
		return pgTypeBytea
	}

	return sqlType(field.IndirectType)
}

func sqlType(typ reflect.Type) string {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ {
	case nullStringType: // typ.Kind() == reflect.Struct, test for exact match
		return sqltype.VarChar
	case ipType:
		return pgTypeInet
	case ipNetType:
		return pgTypeCidr
	case jsonRawMessageType:
		return sqltype.JSONB
	}

	sqlType := schema.DiscoverSQLType(typ)
	switch sqlType {
	case sqltype.Timestamp:
		sqlType = pgTypeTimestampTz
	}

	switch typ.Kind() {
	case reflect.Map, reflect.Struct: // except typ == nullStringType, see above
		if sqlType == sqltype.VarChar {
			return sqltype.JSONB
		}
		return sqlType
	case reflect.Array, reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return pgTypeBytea
		}
		return sqltype.JSONB
	}

	return sqlType
}

var (
	char        = newAliases(pgTypeChar, pgTypeCharacter)
	varchar     = newAliases(pgTypeVarchar, pgTypeCharacterVarying)
	timestampTz = newAliases(sqltype.Timestamp, pgTypeTimestampTz, pgTypeTimestampWithTz)
	bigint      = newAliases(sqltype.BigInt, pgTypeBigSerial)
	integer     = newAliases(sqltype.Integer, pgTypeSerial)
	smallint    = newAliases(sqltype.SmallInt, pgTypeSmallSerial)
)

func (d *Dialect) CompareType(col1, col2 sqlschema.Column) bool {
	typ1, typ2 := strings.ToUpper(col1.GetSQLType()), strings.ToUpper(col2.GetSQLType())

	if typ1 == typ2 {
		return checkVarcharLen(col1, col2, d.DefaultVarcharLen())
	}

	switch {
	case char.IsAlias(typ1) && char.IsAlias(typ2):
		return checkVarcharLen(col1, col2, d.DefaultVarcharLen())
	case varchar.IsAlias(typ1) && varchar.IsAlias(typ2):
		return checkVarcharLen(col1, col2, d.DefaultVarcharLen())
	case timestampTz.IsAlias(typ1) && timestampTz.IsAlias(typ2):
		return true
	case bigint.IsAlias(typ1) && bigint.IsAlias(typ2),
		integer.IsAlias(typ1) && integer.IsAlias(typ2),
		smallint.IsAlias(typ1) && smallint.IsAlias(typ2):
		return true
	}
	return false
}

// checkVarcharLen returns true if columns have the same VarcharLen, or,
// if one specifies no VarcharLen and the other one has the default lenght for pgdialect.
// We assume that the types are otherwise equivalent and that any non-character column
// would have VarcharLen == 0;
func checkVarcharLen(col1, col2 sqlschema.Column, defaultLen int) bool {
	vl1, vl2 := col1.GetVarcharLen(), col2.GetVarcharLen()

	if vl1 == vl2 {
		return true
	}

	if (vl1 == 0 && vl2 == defaultLen) || (vl1 == defaultLen && vl2 == 0) {
		return true
	}
	return false
}

// typeAlias defines aliases for common data types. It is a lightweight string set implementation.
type typeAlias map[string]struct{}

// IsAlias checks if typ1 and typ2 are aliases of the same data type.
func (t typeAlias) IsAlias(typ string) bool {
	_, ok := t[typ]
	return ok
}

// newAliases creates a set of aliases.
func newAliases(aliases ...string) typeAlias {
	types := make(typeAlias)
	for _, a := range aliases {
		types[a] = struct{}{}
	}
	return types
}
