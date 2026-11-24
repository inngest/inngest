package libduckdb

import "fmt"

// State matches duckdb_state in duckdb.h.
type State int32

const (
	StateSuccess State = 0
	StateError   State = 1
)

// Type matches enum DUCKDB_TYPE in duckdb.h (DuckDB v2.0.0).
type Type int32

const (
	TypeInvalid       Type = 0
	TypeBoolean       Type = 1
	TypeTinyInt       Type = 2
	TypeSmallInt      Type = 3
	TypeInteger       Type = 4
	TypeBigInt        Type = 5
	TypeUTinyInt      Type = 6
	TypeUSmallInt     Type = 7
	TypeUInteger      Type = 8
	TypeUBigInt       Type = 9
	TypeFloat         Type = 10
	TypeDouble        Type = 11
	TypeTimestamp     Type = 12
	TypeDate          Type = 13
	TypeTime          Type = 14
	TypeInterval      Type = 15
	TypeHugeInt       Type = 16
	TypeVarchar       Type = 17
	TypeBlob          Type = 18
	TypeDecimal       Type = 19
	TypeTimestampS    Type = 20
	TypeTimestampMS   Type = 21
	TypeTimestampNS   Type = 22
	TypeEnum          Type = 23
	TypeList          Type = 24
	TypeStruct        Type = 25
	TypeMap           Type = 26
	TypeUUID          Type = 27
	TypeUnion         Type = 28
	TypeBit           Type = 29
	TypeTimeTZ        Type = 30
	TypeTimestampTZ   Type = 31
	TypeUHugeInt      Type = 32
	TypeArray         Type = 33
	TypeAny           Type = 34
	TypeBignum        Type = 35
	TypeSQLNull       Type = 36
	TypeStringLiteral Type = 37
	TypeIntLiteral    Type = 38
	TypeTimeNS        Type = 39
	TypeGeometry      Type = 40
	TypeVariant       Type = 41
	TypeTimestampTZNS Type = 42
)

func (t Type) String() string {
	switch t {
	case TypeBoolean:
		return "BOOLEAN"
	case TypeTinyInt:
		return "TINYINT"
	case TypeSmallInt:
		return "SMALLINT"
	case TypeInteger:
		return "INTEGER"
	case TypeBigInt:
		return "BIGINT"
	case TypeUTinyInt:
		return "UTINYINT"
	case TypeUSmallInt:
		return "USMALLINT"
	case TypeUInteger:
		return "UINTEGER"
	case TypeUBigInt:
		return "UBIGINT"
	case TypeFloat:
		return "FLOAT"
	case TypeDouble:
		return "DOUBLE"
	case TypeTimestamp:
		return "TIMESTAMP"
	case TypeDate:
		return "DATE"
	case TypeTime:
		return "TIME"
	case TypeInterval:
		return "INTERVAL"
	case TypeHugeInt:
		return "HUGEINT"
	case TypeVarchar:
		return "VARCHAR"
	case TypeBlob:
		return "BLOB"
	case TypeDecimal:
		return "DECIMAL"
	case TypeTimestampS:
		return "TIMESTAMP_S"
	case TypeTimestampMS:
		return "TIMESTAMP_MS"
	case TypeTimestampNS:
		return "TIMESTAMP_NS"
	case TypeEnum:
		return "ENUM"
	case TypeList:
		return "LIST"
	case TypeStruct:
		return "STRUCT"
	case TypeMap:
		return "MAP"
	case TypeUUID:
		return "UUID"
	case TypeUnion:
		return "UNION"
	case TypeBit:
		return "BIT"
	case TypeTimeTZ:
		return "TIME WITH TIME ZONE"
	case TypeTimestampTZ:
		return "TIMESTAMP WITH TIME ZONE"
	case TypeUHugeInt:
		return "UHUGEINT"
	case TypeArray:
		return "ARRAY"
	case TypeBignum:
		return "BIGNUM"
	case TypeSQLNull:
		return "SQLNULL"
	case TypeTimeNS:
		return "TIME_NS"
	case TypeGeometry:
		return "GEOMETRY"
	case TypeVariant:
		return "VARIANT"
	case TypeTimestampTZNS:
		return "TIMESTAMPTZ_NS"
	default:
		return fmt.Sprintf("DUCKDB_TYPE(%d)", int32(t))
	}
}

// ColumnInfo describes a single column in a DuckDB query result.
type ColumnInfo struct {
	Name     string
	Type     Type
	TypeName string
}

// cResult matches struct duckdb_result in duckdb.h (6 x 8 = 48 bytes on 64-bit).
//
//nolint:unused // C ABI layout padding fields required by duckdb_query/duckdb_fetch_chunk
type cResult struct {
	deprecatedColumnCount uint64
	deprecatedRowCount    uint64
	deprecatedRowsChanged uint64
	deprecatedColumns     uintptr
	deprecatedErrorMsg    uintptr
	internalData          uintptr
}

// cStringT matches the 16-byte duckdb_string_t layout on 64-bit systems:
//   - If length <= 12: inline bytes occupy data[0:length] (offset 4..15).
//   - If length > 12:  prefix occupies data[0:4], and a 64-bit char* pointer
//     occupies data[4:12] (offset 8..15).
type cStringT struct {
	length uint32
	data   [12]byte
}

// cHugeInt matches struct duckdb_hugeint in duckdb.h.
type cHugeInt struct {
	lower uint64
	upper int64
}

// cUHugeInt matches struct duckdb_uhugeint in duckdb.h.
type cUHugeInt struct {
	lower uint64
	upper uint64
}

// cListEntry matches struct duckdb_list_entry in duckdb.h.
type cListEntry struct {
	offset uint64
	length uint64
}

// cInterval matches struct duckdb_interval in duckdb.h.
type cInterval struct {
	months int32
	days   int32
	micros int64
}
