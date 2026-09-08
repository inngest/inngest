package duckdb

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// Quack message types, mirroring duckdb_quack::MessageType for protocol
// version 3. APPEND_REQUEST from the older v1.5-variegata protocol is gone,
// replaced by SEND_DATA_REQUEST/RESPONSE (see quack_senddata.go).
// DISCONNECT_MESSAGE(11), CANCEL_REQUEST(12), ACKNOWLEDGEMENT(15), and
// HEARTBEAT_REQUEST(16) exist server-side but aren't implemented here.
const (
	quackMsgConnectionRequest  byte = 1
	quackMsgConnectionResponse byte = 2
	quackMsgPrepareRequest     byte = 3
	quackMsgPrepareResponse    byte = 4
	quackMsgFetchRequest       byte = 7
	quackMsgFetchResponse      byte = 8
	quackMsgSendDataRequest    byte = 9
	quackMsgSuccessResponse    byte = 10
	quackMsgSendDataResponse   byte = 14
	quackMsgErrorResponse      byte = 100
)

// quackHugeint is DuckDB's hugeint_t: a signed-LEB128 upper half followed by
// an unsigned-LEB128 lower half. This client only ever treats it as an
// opaque correlation token, never as a number.
type quackHugeint struct {
	hi int64
	lo uint64
}

// randomQuackHugeint returns a random 128-bit value for use as an opaque
// client-generated correlation token (see encodeQuackPrepareRequest).
func randomQuackHugeint() quackHugeint {
	var b [16]byte
	_, _ = rand.Read(b[:]) // never errors on any supported platform
	return quackHugeint{
		hi: int64(binary.BigEndian.Uint64(b[:8])),
		lo: binary.BigEndian.Uint64(b[8:]),
	}
}

// Every quack message is two top-level objects back to back: a MessageHeader
// (field1 type, field2 connection_id default-omit, field3 client_query_id —
// an optional_idx whose "not set" sentinel is uint64 max) followed by the
// message-specific body object.
const quackOptionalIdxInvalid = ^uint64(0)

type quackMessageHeader struct {
	Type         byte
	ConnectionID string
}

func encodeQuackMessage(msgType byte, connectionID string, writeBody func(w *quackWriter)) []byte {
	w := &quackWriter{}
	w.beginObject()
	w.writeByte(1, msgType)
	w.writeStringDefault(2, connectionID)
	w.writeUint64(3, quackOptionalIdxInvalid)
	w.endObject()
	w.beginObject()
	writeBody(w)
	w.endObject()
	return w.bytes()
}

// decodeQuackMessageHeader reads the header object and leaves r positioned at
// the start of the body object.
func decodeQuackMessageHeader(r *quackReader) (quackMessageHeader, error) {
	r.beginObject()
	if err := r.beginProperty(1); err != nil {
		return quackMessageHeader{}, err
	}
	msgType, err := r.readByte()
	if err != nil {
		return quackMessageHeader{}, err
	}

	var connID string
	present, err := r.tryBeginProperty(2)
	if err != nil {
		return quackMessageHeader{}, err
	}
	if present {
		connID, err = r.readString()
		if err != nil {
			return quackMessageHeader{}, err
		}
	}

	if err := r.beginProperty(3); err != nil {
		return quackMessageHeader{}, err
	}
	if _, err := r.readUInt64(); err != nil { // client_query_id, unused by this client
		return quackMessageHeader{}, err
	}
	if err := r.endObject(); err != nil {
		return quackMessageHeader{}, err
	}

	r.beginObject() // positions r at the start of the body object
	return quackMessageHeader{Type: msgType, ConnectionID: connID}, nil
}

// ---------- ConnectionRequest / ConnectionResponse ----------

type quackConnectionRequest struct {
	AuthString               string
	ClientDuckDBVersion      string
	ClientPlatform           string
	MinSupportedQuackVersion uint64
	MaxSupportedQuackVersion uint64
	// HeartbeatTimeoutSeconds is required as of protocol version 3 (the
	// server rejects an out-of-range value; version 1 defaulted an absent
	// field to "no timeout"). Field 6 (client_id) sits between
	// MaxSupportedQuackVersion and this field but is never sent — see
	// quackHeartbeatTimeoutSeconds for the value this client uses.
	HeartbeatTimeoutSeconds uint64
}

func (m quackConnectionRequest) encode() []byte {
	return encodeQuackMessage(quackMsgConnectionRequest, "", func(w *quackWriter) {
		w.writeStringDefault(1, m.AuthString)
		w.writeStringDefault(2, m.ClientDuckDBVersion)
		w.writeStringDefault(3, m.ClientPlatform)
		w.writeUint64Default(4, m.MinSupportedQuackVersion)
		w.writeUint64Default(5, m.MaxSupportedQuackVersion)
		// field 6 (client_id) intentionally omitted.
		w.writeUint64Default(7, m.HeartbeatTimeoutSeconds)
	})
}

type quackConnectionResponse struct {
	ServerDuckDBVersion string
	ServerPlatform      string
	QuackVersion        uint64
	// HeartbeatTimeoutSeconds echoes back the (possibly clamped) heartbeat
	// lease the server granted. Decoded but otherwise unused: this client
	// never renews the lease and always requests the server's maximum (see
	// quackHeartbeatTimeoutSeconds).
	HeartbeatTimeoutSeconds uint64
}

func decodeQuackConnectionResponseBody(r *quackReader) (quackConnectionResponse, error) {
	var resp quackConnectionResponse
	if ok, err := r.tryBeginProperty(1); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readString()
		if err != nil {
			return resp, err
		}
		resp.ServerDuckDBVersion = v
	}
	if ok, err := r.tryBeginProperty(2); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readString()
		if err != nil {
			return resp, err
		}
		resp.ServerPlatform = v
	}
	if ok, err := r.tryBeginProperty(3); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readUInt64()
		if err != nil {
			return resp, err
		}
		resp.QuackVersion = v
	}
	if ok, err := r.tryBeginProperty(4); err != nil {
		return resp, err
	} else if ok {
		v, err := r.readUInt64()
		if err != nil {
			return resp, err
		}
		resp.HeartbeatTimeoutSeconds = v
	}
	if err := r.endObject(); err != nil {
		return resp, err
	}
	return resp, nil
}

// ---------- ErrorResponse ----------

func decodeQuackErrorResponseBody(r *quackReader) (string, error) {
	var message string
	if ok, err := r.tryBeginProperty(1); err != nil {
		return "", err
	} else if ok {
		v, err := r.readString()
		if err != nil {
			return "", err
		}
		message = v
	}
	if err := r.endObject(); err != nil {
		return "", err
	}
	return message, nil
}

// ---------- PrepareRequest / PrepareResponse ----------

// encodeQuackPrepareRequest builds a PrepareRequest: field1 is the SQL text,
// field2 is a mandatory hugeint query token this client has no other use for
// (omitting it crashes the server with a bodyless HTTP 500). A fresh token
// is generated per call.
func encodeQuackPrepareRequest(connectionID, sql string) []byte {
	return encodeQuackMessage(quackMsgPrepareRequest, connectionID, func(w *quackWriter) {
		w.writeStringDefault(1, sql)
		w.writeHugeint(2, randomQuackHugeint())
	})
}

// decodeQuackPrepareResponseBody reads a PrepareResponse body and returns
// rows keyed by result column name, built from every inline DataChunk the
// server returned, alongside names (field2) and types (field1), both in the
// query's own left-to-right order — the only place that order survives once
// namedRows folds a chunk's columns into a map. Both are populated even for
// a zero-row result, unlike rows.go's jsonlines session (see
// quackSession.query). needsMoreFetch signals more rows than fit in this
// response; resultUUID (field5) is the token a FetchRequest passes back to
// pull the rest.
func decodeQuackPrepareResponseBody(r *quackReader) (names []string, types []quackLogicalType, rows []map[string]any, needsMoreFetch bool, resultUUID quackHugeint, err error) {
	if ok, terr := r.tryBeginProperty(1); terr != nil {
		return nil, nil, nil, false, quackHugeint{}, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, nil, nil, false, quackHugeint{}, terr
		}
		types = make([]quackLogicalType, n)
		for i := uint64(0); i < n; i++ {
			types[i], terr = decodeQuackLogicalType(r)
			if terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
		}
	}

	if ok, terr := r.tryBeginProperty(2); terr != nil {
		return nil, nil, nil, false, quackHugeint{}, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, nil, nil, false, quackHugeint{}, terr
		}
		names = make([]string, n)
		for i := range names {
			names[i], terr = r.readString()
			if terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
		}
	}

	if ok, terr := r.tryBeginProperty(3); terr != nil {
		return nil, nil, nil, false, quackHugeint{}, terr
	} else if ok {
		needsMoreFetch, terr = r.readBool()
		if terr != nil {
			return nil, nil, nil, false, quackHugeint{}, terr
		}
	}

	if ok, terr := r.tryBeginProperty(4); terr != nil {
		return nil, nil, nil, false, quackHugeint{}, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, nil, nil, false, quackHugeint{}, terr
		}
		for i := uint64(0); i < n; i++ {
			present, terr := r.beginNullable()
			if terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
			if !present {
				continue
			}
			// DataChunkWrapper is itself an object wrapping field 300, whose
			// value is the DataChunk object decodeQuackDataChunk reads
			// (including that inner object's own terminator) — so an outer
			// endObject is still needed here for the wrapper's terminator.
			r.beginObject()
			if terr := r.beginProperty(300); terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
			chunk, terr := decodeQuackDataChunk(r)
			if terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
			if terr := r.endObject(); terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
			chunkRows, terr := chunk.namedRows(names)
			if terr != nil {
				return nil, nil, nil, false, quackHugeint{}, terr
			}
			rows = append(rows, chunkRows...)
		}
	}

	// field5 result_uuid: hugeint (signed-leb128 upper, unsigned-leb128
	// lower) — the token a follow-up FetchRequest must echo back.
	if err := r.beginProperty(5); err != nil {
		return nil, nil, nil, false, quackHugeint{}, err
	}
	hi, err := r.readSignedLeb128()
	if err != nil {
		return nil, nil, nil, false, quackHugeint{}, err
	}
	lo, err := r.readUnsignedLeb128()
	if err != nil {
		return nil, nil, nil, false, quackHugeint{}, err
	}
	resultUUID = quackHugeint{hi: hi, lo: lo}

	if err := r.endObject(); err != nil {
		return nil, nil, nil, false, quackHugeint{}, err
	}
	return names, types, rows, needsMoreFetch, resultUUID, nil
}

// ---------- FetchRequest / FetchResponse ----------

// encodeQuackFetchRequest builds a FetchRequest asking the server for the
// next batch of an already-prepared result (uuid, from the PrepareResponse
// or a prior FetchResponse's result_uuid).
func encodeQuackFetchRequest(connectionID string, uuid quackHugeint) []byte {
	return encodeQuackMessage(quackMsgFetchRequest, connectionID, func(w *quackWriter) {
		w.writeHugeint(1, uuid)
	})
}

// decodeQuackFetchResponseBody reads a FetchResponse body and returns rows
// decoded from every inline DataChunk it carries, keyed against names (the
// PrepareResponse's own result_names — a FetchResponse repeats the same
// columns, so it carries no names of its own). chunkCount is the number of
// DataChunks this response actually contained: the wire format has no
// explicit "no more rows" flag, so the server signals it by eventually
// returning a response with zero chunks (see quackSession.exec's fetch loop).
func decodeQuackFetchResponseBody(r *quackReader, names []string) (rows []map[string]any, chunkCount int, err error) {
	if ok, terr := r.tryBeginProperty(1); terr != nil {
		return nil, 0, terr
	} else if ok {
		n, terr := r.beginList()
		if terr != nil {
			return nil, 0, terr
		}
		for i := uint64(0); i < n; i++ {
			present, terr := r.beginNullable()
			if terr != nil {
				return nil, 0, terr
			}
			if !present {
				continue
			}
			r.beginObject()
			if terr := r.beginProperty(300); terr != nil {
				return nil, 0, terr
			}
			chunk, terr := decodeQuackDataChunk(r)
			if terr != nil {
				return nil, 0, terr
			}
			if terr := r.endObject(); terr != nil {
				return nil, 0, terr
			}
			chunkRows, terr := chunk.namedRows(names)
			if terr != nil {
				return nil, 0, terr
			}
			rows = append(rows, chunkRows...)
			chunkCount++
		}
	}

	// field2 batch_index is always present on the wire but unused here: this
	// client doesn't retry a specific batch.
	if err := r.beginProperty(2); err != nil {
		return nil, 0, err
	}
	if _, err := r.readUInt64(); err != nil {
		return nil, 0, err
	}

	if err := r.endObject(); err != nil {
		return nil, 0, err
	}
	return rows, chunkCount, nil
}

// ---------- LogicalType ----------

// quackLogicalTypeInteger and friends are DuckDB's LogicalTypeId values.
// This is the *schema*-decoding vocabulary typeName() maps from — broader
// than what decodeQuackVector actually decodes *values* for, since a
// column's declared type can be reported correctly even for a type this
// client can't yet fetch rows of.
const (
	// quackLogicalTypeSQLNull is the type an untyped NULL literal with no
	// other value to infer from gets, e.g. a STRUCT field whose only literal
	// is NULL. It decodes via decodeFlatVector's ordinary fixed-data path
	// like any scalar type (every row's validity bit is false by
	// construction); only values() needs a case for it, to report those
	// NULLs instead of erroring.
	quackLogicalTypeSQLNull byte = 1

	quackLogicalTypeBoolean      byte = 10
	quackLogicalTypeTinyInt      byte = 11
	quackLogicalTypeSmallInt     byte = 12
	quackLogicalTypeInteger      byte = 13
	quackLogicalTypeBigInt       byte = 14
	quackLogicalTypeDate         byte = 15
	quackLogicalTypeTime         byte = 16
	quackLogicalTypeTimestampSec byte = 17
	quackLogicalTypeTimestampMs  byte = 18
	quackLogicalTypeTimestamp    byte = 19
	quackLogicalTypeTimestampNs  byte = 20
	quackLogicalTypeDecimal      byte = 21
	quackLogicalTypeFloat        byte = 22
	quackLogicalTypeDouble       byte = 23
	quackLogicalTypeVarchar      byte = 25
	quackLogicalTypeBlob         byte = 26
	quackLogicalTypeInterval     byte = 27
	quackLogicalTypeUTinyInt     byte = 28
	quackLogicalTypeUSmallInt    byte = 29
	quackLogicalTypeUInteger     byte = 30
	quackLogicalTypeUBigInt      byte = 31
	quackLogicalTypeTimestampTZ  byte = 32
	quackLogicalTypeTimeTZ       byte = 34
	quackLogicalTypeBit          byte = 36
	quackLogicalTypeUHugeint     byte = 49
	quackLogicalTypeHugeint      byte = 50
	quackLogicalTypeUUID         byte = 54
	quackLogicalTypeStruct       byte = 100
	quackLogicalTypeList         byte = 101
	quackLogicalTypeMap          byte = 102
	quackLogicalTypeEnum         byte = 104
)

// quackAliasJSON is the LogicalType alias DuckDB's JSON type carries: it's
// physically a VARCHAR, distinguished only by this alias in its
// ExtraTypeInfo (type_info = {100: extraTypeInfoKind, 101: "JSON"}).
const quackAliasJSON = "JSON"

// quackStructField is one named child of a STRUCT LogicalType: field id 0
// (name) and field id 1 (nested LogicalType) of a type_info field200 entry.
type quackStructField struct {
	name string
	typ  quackLogicalType
}

// quackLogicalType is a decoded LogicalType: the base id, its alias if any,
// and whatever extra shape its id implies: LIST/MAP's single unnamed child
// type (child, at type_info's field200 — MAP's child is itself a
// STRUCT{key,value}, not a LIST), STRUCT's named child fields (structFields,
// a counted list at that same field200), or DECIMAL's width/scale or ENUM's
// dictionary values (decimalWidth/decimalScale, enumValues — plain
// scalars/lists at fields 200/201 despite sharing those field numbers with
// the other shapes).
type quackLogicalType struct {
	id           byte
	alias        string
	child        *quackLogicalType
	structFields []quackStructField
	// decimalWidth/decimalScale are set only when id == quackLogicalTypeDecimal.
	decimalWidth byte
	decimalScale byte
	// enumValues is set only when id == quackLogicalTypeEnum: the enum's
	// dictionary values in declaration order.
	enumValues []string
}

// decodeQuackLogicalType reads a LogicalType object (field100 id, optional
// field101 type_info). type_info, when present, is ExtraTypeInfo's own
// object — {100: extraTypeInfoKind byte, 101: alias string, 200/201: a
// shape that depends on id: LIST/MAP get a single nested child LogicalType
// at 200 (no count prefix); STRUCT gets a counted list of named child
// entries at 200 (field id 0 = name, field id 1 = nested LogicalType, each
// entry self-terminating); DECIMAL gets two plain ULEB128 scalars, width at
// 200 and scale at 201; ENUM gets a plain ULEB128 scalar at 200 (the
// dictionary's internal physical-size marker, unused here) and a counted
// list of raw length-prefixed strings at 201 (the dictionary values)}.
// Anything else errors, surfaced as an unexpected field id where the
// terminator was expected.
func decodeQuackLogicalType(r *quackReader) (quackLogicalType, error) {
	r.beginObject()
	if err := r.beginProperty(100); err != nil {
		return quackLogicalType{}, err
	}
	id, err := r.readByte()
	if err != nil {
		return quackLogicalType{}, err
	}
	lt := quackLogicalType{id: id}
	if ok, err := r.tryBeginProperty(101); err != nil {
		return quackLogicalType{}, err
	} else if ok {
		present, err := r.beginNullable()
		if err != nil {
			return quackLogicalType{}, err
		}
		if present {
			r.beginObject()
			if err := r.beginProperty(100); err != nil {
				return quackLogicalType{}, fmt.Errorf("duckdb: quack LogicalType id %d has unsupported extended type_info: %w", id, err)
			}
			if _, err := r.readByte(); err != nil { // extraTypeInfoKind; only used to stay positioned on the wire
				return quackLogicalType{}, err
			}
			if ok, err := r.tryBeginProperty(101); err != nil {
				return quackLogicalType{}, err
			} else if ok {
				lt.alias, err = r.readString()
				if err != nil {
					return quackLogicalType{}, err
				}
			}
			if ok, err := r.tryBeginProperty(200); err != nil {
				return quackLogicalType{}, err
			} else if ok {
				switch id {
				case quackLogicalTypeStruct:
					fields, ferr := decodeQuackStructFieldTypes(r)
					if ferr != nil {
						return quackLogicalType{}, ferr
					}
					lt.structFields = fields
				case quackLogicalTypeDecimal:
					width, werr := r.readByte()
					if werr != nil {
						return quackLogicalType{}, werr
					}
					lt.decimalWidth = width
					if err := r.beginProperty(201); err != nil {
						return quackLogicalType{}, err
					}
					scale, serr := r.readByte()
					if serr != nil {
						return quackLogicalType{}, serr
					}
					lt.decimalScale = scale
				case quackLogicalTypeEnum:
					// field200 is the dictionary's internal physical-size
					// marker, unused beyond staying positioned on the wire.
					if _, err := r.readByte(); err != nil {
						return quackLogicalType{}, err
					}
					if err := r.beginProperty(201); err != nil {
						return quackLogicalType{}, err
					}
					n, nerr := r.beginList()
					if nerr != nil {
						return quackLogicalType{}, nerr
					}
					values := make([]string, n)
					for i := range values {
						values[i], err = r.readString()
						if err != nil {
							return quackLogicalType{}, err
						}
					}
					lt.enumValues = values
				default:
					child, cerr := decodeQuackLogicalType(r)
					if cerr != nil {
						return quackLogicalType{}, cerr
					}
					lt.child = &child
				}
			}
			if err := r.endObject(); err != nil {
				return quackLogicalType{}, fmt.Errorf("duckdb: quack LogicalType id %d has extended type_info this client does not support: %w", id, err)
			}
		}
	}
	if err := r.endObject(); err != nil {
		return quackLogicalType{}, err
	}
	return lt, nil
}

// typeName renders lt as the same type-name string DuckDB's own DESCRIBE
// would print, so a quack-sourced schema feeds DuckDBToColumnType
// (pkg/duckdb/insights/columntype.go) identically to a DESCRIBE-sourced one.
// An unrecognized id renders as "UNKNOWN(<id>)" rather than guessing.
func (lt quackLogicalType) typeName() string {
	switch lt.id {
	case quackLogicalTypeSQLNull:
		return `"NULL"`
	case quackLogicalTypeBoolean:
		return "BOOLEAN"
	case quackLogicalTypeTinyInt:
		return "TINYINT"
	case quackLogicalTypeSmallInt:
		return "SMALLINT"
	case quackLogicalTypeInteger:
		return "INTEGER"
	case quackLogicalTypeBigInt:
		return "BIGINT"
	case quackLogicalTypeHugeint:
		return "HUGEINT"
	case quackLogicalTypeUTinyInt:
		return "UTINYINT"
	case quackLogicalTypeUSmallInt:
		return "USMALLINT"
	case quackLogicalTypeUInteger:
		return "UINTEGER"
	case quackLogicalTypeUBigInt:
		return "UBIGINT"
	case quackLogicalTypeUHugeint:
		return "UHUGEINT"
	case quackLogicalTypeFloat:
		return "FLOAT"
	case quackLogicalTypeDouble:
		return "DOUBLE"
	case quackLogicalTypeDecimal:
		return fmt.Sprintf("DECIMAL(%d,%d)", lt.decimalWidth, lt.decimalScale)
	case quackLogicalTypeDate:
		return "DATE"
	case quackLogicalTypeTime:
		return "TIME"
	case quackLogicalTypeTimeTZ:
		return "TIME WITH TIME ZONE"
	case quackLogicalTypeTimestampSec, quackLogicalTypeTimestampMs, quackLogicalTypeTimestamp, quackLogicalTypeTimestampNs:
		return "TIMESTAMP"
	case quackLogicalTypeTimestampTZ:
		return "TIMESTAMP WITH TIME ZONE"
	case quackLogicalTypeInterval:
		return "INTERVAL"
	case quackLogicalTypeVarchar:
		if lt.alias == quackAliasJSON {
			return "JSON"
		}
		return "VARCHAR"
	case quackLogicalTypeBlob:
		return "BLOB"
	case quackLogicalTypeBit:
		return "BIT"
	case quackLogicalTypeUUID:
		return "UUID"
	case quackLogicalTypeEnum:
		values := make([]string, len(lt.enumValues))
		for i, v := range lt.enumValues {
			values[i] = "'" + v + "'"
		}
		return "ENUM(" + strings.Join(values, ", ") + ")"
	case quackLogicalTypeList:
		if lt.child == nil {
			return "UNKNOWN[]"
		}
		return lt.child.typeName() + "[]"
	case quackLogicalTypeStruct:
		parts := make([]string, len(lt.structFields))
		for i, f := range lt.structFields {
			parts[i] = f.name + " " + f.typ.typeName()
		}
		return "STRUCT(" + strings.Join(parts, ", ") + ")"
	case quackLogicalTypeMap:
		if lt.child == nil || len(lt.child.structFields) != 2 {
			return "MAP"
		}
		return "MAP(" + lt.child.structFields[0].typ.typeName() + ", " + lt.child.structFields[1].typ.typeName() + ")"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", lt.id)
	}
}

// decodeQuackStructFieldTypes reads a STRUCT LogicalType's field200 value: a
// raw ULEB128 count followed by that many child entries, each an object with
// field id 0 = name (string) and field id 1 = type (a nested LogicalType,
// itself fully self-terminating) followed by the entry's own terminator —
// see decodeQuackLogicalType's doc comment.
func decodeQuackStructFieldTypes(r *quackReader) ([]quackStructField, error) {
	n, err := r.beginList()
	if err != nil {
		return nil, err
	}
	fields := make([]quackStructField, n)
	for i := range fields {
		r.beginObject()
		if err := r.beginProperty(0); err != nil {
			return nil, err
		}
		name, err := r.readString()
		if err != nil {
			return nil, err
		}
		if err := r.beginProperty(1); err != nil {
			return nil, err
		}
		typ, err := decodeQuackLogicalType(r)
		if err != nil {
			return nil, err
		}
		if err := r.endObject(); err != nil {
			return nil, err
		}
		fields[i] = quackStructField{name: name, typ: typ}
	}
	return fields, nil
}

// ---------- DataChunk / Vector ----------

type quackColumn struct {
	typeID byte
	// alias is the LogicalType's ExtraTypeInfo alias, if any ("" for a plain
	// unaliased type) — e.g. "JSON" for DuckDB's JSON type, which is
	// otherwise a physically ordinary VARCHAR (see quackAliasJSON).
	alias string
	// validity is nil when the column has no NULLs (DuckDB omits the mask
	// entirely in that case — see decodeFlatVector); otherwise one entry per
	// row, true = valid, false = SQL NULL.
	validity []bool
	// Exactly one of these is populated, chosen by typeID's physical shape.
	// DuckDB still writes a real (unspecified/garbage) entry at a NULL row's
	// position in both, so indexing is always safe — values() just discards
	// whatever's there and substitutes nil when validity says so.
	fixedData []byte // constant-size types, column-major, no per-row length
	varchar   [][]byte
	// list holds one entry per row for a LIST column: nil for a SQL NULL
	// row, otherwise a []any of the row's decoded child values (empty, not
	// nil, for a zero-length list). Populated directly by
	// decodeQuackListVector rather than derived in values(), since building
	// it requires the list_entry_t offsets/lengths that aren't otherwise
	// kept on quackColumn.
	list []any
	// structVals holds one entry per row for a STRUCT column: nil for a SQL
	// NULL row, otherwise a map[string]any keyed by field name. Populated
	// directly by decodeQuackStructVector, same rationale as list.
	structVals []any
	rowCount   int
}

// isNull reports whether row i is a SQL NULL.
func (c quackColumn) isNull(i int) bool {
	return c.validity != nil && !c.validity[i]
}

// values decodes the column's raw bytes into canonical driver.Value-shaped Go
// types: nil (for a NULL row), bool, int64, float64, string, time.Time, or —
// for a VARCHAR aliased as JSON — the value's own json.Unmarshal result,
// matching the shape the stdio/jsonlines transport produces for the same
// column type.
func (c quackColumn) values() ([]any, error) {
	out := make([]any, c.rowCount)
	switch c.typeID {
	case quackLogicalTypeSQLNull:
		// Every row is NULL by construction; out is already all-nil.
	case quackLogicalTypeBoolean:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = c.fixedData[i] != 0
		}
	case quackLogicalTypeSmallInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int16(le16(c.fixedData[i*2:])))
		}
	case quackLogicalTypeInteger:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int32(le32(c.fixedData[i*4:])))
		}
	case quackLogicalTypeBigInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeDouble:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = le64ToFloat64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeVarchar:
		if c.alias == quackAliasJSON {
			for i := range out {
				if c.isNull(i) {
					continue
				}
				var v any
				if err := json.Unmarshal(c.varchar[i], &v); err != nil {
					return nil, fmt.Errorf("duckdb: quack column decode: unmarshaling JSON column value: %w", err)
				}
				out[i] = v
			}
			return out, nil
		}
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = string(c.varchar[i])
		}
	case quackLogicalTypeTimestamp, quackLogicalTypeTimestampMs, quackLogicalTypeTimestampSec, quackLogicalTypeTimestampNs:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			raw := int64(le64(c.fixedData[i*8:]))
			out[i] = quackTimestampToTime(c.typeID, raw)
		}
	case quackLogicalTypeUUID:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = quackUUIDToString(c.fixedData[i*16 : i*16+16])
		}
	case quackLogicalTypeList:
		copy(out, c.list)
	case quackLogicalTypeStruct:
		copy(out, c.structVals)
	default:
		return nil, fmt.Errorf("duckdb: quack column decode: unsupported LogicalTypeId %d", c.typeID)
	}
	return out, nil
}

func quackTimestampToTime(typeID byte, raw int64) time.Time {
	switch typeID {
	case quackLogicalTypeTimestampSec:
		return time.Unix(raw, 0).UTC()
	case quackLogicalTypeTimestampMs:
		return time.UnixMilli(raw).UTC()
	case quackLogicalTypeTimestampNs:
		return time.Unix(0, raw).UTC()
	default: // microseconds (Timestamp)
		return time.UnixMicro(raw).UTC()
	}
}

// quackUUIDToString decodes a UUID column's 16-byte wire representation into
// standard "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" text. DuckDB stores a UUID
// internally as a signed hugeint with the sign bit of the first byte
// flipped, then serializes that little-endian — so on the wire it's the
// UUID's bytes reversed, with the *last* wire byte's top bit flipped instead.
func quackUUIDToString(b []byte) string {
	var u [16]byte
	for i := range u {
		u[i] = b[15-i]
	}
	u[0] ^= 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

func le16(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }
func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
func le64(b []byte) uint64 {
	var v uint64
	for i := 0; i < 8; i++ {
		v |= uint64(b[i]) << (8 * i)
	}
	return v
}
func le64ToFloat64(bits uint64) float64 {
	return math.Float64frombits(bits)
}

type quackDataChunk struct {
	rowCount int
	columns  []quackColumn
}

// namedRows zips this chunk's columns against names (from the enclosing
// PrepareResponse's result_names) into one map[string]any per row, matching
// the shape process.exec has always returned (see sqlExecer).
func (c quackDataChunk) namedRows(names []string) ([]map[string]any, error) {
	if len(names) != len(c.columns) {
		return nil, fmt.Errorf("duckdb: quack result has %d names but %d columns", len(names), len(c.columns))
	}
	colValues := make([][]any, len(c.columns))
	for i, col := range c.columns {
		vs, err := col.values()
		if err != nil {
			return nil, err
		}
		colValues[i] = vs
	}
	rows := make([]map[string]any, c.rowCount)
	for r := 0; r < c.rowCount; r++ {
		row := make(map[string]any, len(names))
		for i, name := range names {
			row[name] = colValues[i][r]
		}
		rows[r] = row
	}
	return rows, nil
}

// decodeQuackDataChunk reads a DataChunk object: field100 rows, field101
// types (list<LogicalType>), field102 columns (list of Vector objects).
func decodeQuackDataChunk(r *quackReader) (quackDataChunk, error) {
	r.beginObject()
	if err := r.beginProperty(100); err != nil {
		return quackDataChunk{}, err
	}
	rowCount, err := r.readUInt32()
	if err != nil {
		return quackDataChunk{}, err
	}

	if err := r.beginProperty(101); err != nil {
		return quackDataChunk{}, err
	}
	typesCount, err := r.beginList()
	if err != nil {
		return quackDataChunk{}, err
	}
	types := make([]quackLogicalType, typesCount)
	for i := range types {
		types[i], err = decodeQuackLogicalType(r)
		if err != nil {
			return quackDataChunk{}, err
		}
	}

	var columns []quackColumn
	if ok, terr := r.tryBeginProperty(102); terr != nil {
		return quackDataChunk{}, terr
	} else if ok {
		colCount, terr := r.beginList()
		if terr != nil {
			return quackDataChunk{}, terr
		}
		if colCount != typesCount {
			return quackDataChunk{}, fmt.Errorf("duckdb: quack DataChunk has %d types but %d columns", typesCount, colCount)
		}
		columns = make([]quackColumn, colCount)
		for i := range columns {
			// Matches DataChunk::Deserialize: the object begin/end wrapping
			// a column's Vector is done by the DataChunk's own columns
			// loop, not by Vector::Deserialize itself.
			r.beginObject()
			col, terr := decodeQuackVector(r, types[i], int(rowCount))
			if terr != nil {
				return quackDataChunk{}, terr
			}
			if terr := r.endObject(); terr != nil {
				return quackDataChunk{}, terr
			}
			columns[i] = col
		}
	}

	if err := r.endObject(); err != nil {
		return quackDataChunk{}, err
	}
	return quackDataChunk{rowCount: int(rowCount), columns: columns}, nil
}

// decodeQuackVector reads one column's Vector object. Only VectorType.Flat
// (the wire default, field90 omitted) is supported; Constant/Sequence/
// Dictionary vectors are unimplemented.
func decodeQuackVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if ok, err := r.tryBeginProperty(90); err != nil {
		return quackColumn{}, err
	} else if ok {
		vectorType, err := r.readByte()
		if err != nil {
			return quackColumn{}, err
		}
		if vectorType != 0 { // 0 == VectorType.Flat
			return quackColumn{}, fmt.Errorf("duckdb: quack vector type %d is not supported by this client (flat only)", vectorType)
		}
	}
	if lt.id == quackLogicalTypeList {
		return decodeQuackListVector(r, lt, count)
	}
	if lt.id == quackLogicalTypeStruct {
		return decodeQuackStructVector(r, lt, count)
	}
	return decodeFlatVector(r, lt.id, lt.alias, count)
}

// decodeQuackListVector reads a LIST Vector object:
//
//   - field100: hasValidity (whether a given row's whole list is itself SQL
//     NULL), same convention as decodeFlatVector.
//   - field101: validity mask, present only when hasValidity is true.
//   - field104: the flattened child vector's total element count
//     (DuckDB's ListVector::GetListSize()), a plain ULEB128.
//   - field105: a list of list_entry_t objects, each {100: offset ULEB128,
//     101: length ULEB128}. A NULL row's entry is still present, with an
//     unspecified offset/length — validity governs it, not the entry.
//   - field106: the flattened child vector itself, wrapped in its own
//     object, decoded recursively via decodeQuackVector so a LIST of LIST
//     also works.
func decodeQuackListVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if lt.child == nil {
		return quackColumn{}, fmt.Errorf("duckdb: quack LIST LogicalType is missing its child type")
	}

	if err := r.beginProperty(100); err != nil {
		return quackColumn{}, err
	}
	hasValidity, err := r.readBool()
	if err != nil {
		return quackColumn{}, err
	}
	var validity []bool
	if hasValidity {
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		maskBytes, err := r.readData()
		if err != nil {
			return quackColumn{}, err
		}
		validity = decodeQuackValidityMask(maskBytes, count)
	}

	if err := r.beginProperty(104); err != nil {
		return quackColumn{}, err
	}
	childCount, err := r.readUnsignedLeb128()
	if err != nil {
		return quackColumn{}, err
	}

	if err := r.beginProperty(105); err != nil {
		return quackColumn{}, err
	}
	entryCount, err := r.beginList()
	if err != nil {
		return quackColumn{}, err
	}
	if int(entryCount) != count {
		return quackColumn{}, fmt.Errorf("duckdb: quack LIST vector has %d list_entry_t entries, expected %d", entryCount, count)
	}
	type listEntry struct{ offset, length uint64 }
	entries := make([]listEntry, count)
	for i := range entries {
		r.beginObject()
		if err := r.beginProperty(100); err != nil {
			return quackColumn{}, err
		}
		entries[i].offset, err = r.readUnsignedLeb128()
		if err != nil {
			return quackColumn{}, err
		}
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		entries[i].length, err = r.readUnsignedLeb128()
		if err != nil {
			return quackColumn{}, err
		}
		if err := r.endObject(); err != nil {
			return quackColumn{}, err
		}
	}

	if err := r.beginProperty(106); err != nil {
		return quackColumn{}, err
	}
	r.beginObject()
	child, err := decodeQuackVector(r, *lt.child, int(childCount))
	if err != nil {
		return quackColumn{}, err
	}
	if err := r.endObject(); err != nil {
		return quackColumn{}, err
	}

	childValues, err := child.values()
	if err != nil {
		return quackColumn{}, err
	}
	lists := make([]any, count)
	for i, e := range entries {
		if validity != nil && !validity[i] {
			continue
		}
		lists[i] = append([]any{}, childValues[e.offset:e.offset+e.length]...)
	}
	return quackColumn{typeID: quackLogicalTypeList, list: lists, rowCount: count}, nil
}

// decodeQuackStructVector reads a STRUCT Vector object:
//
//   - field100: hasValidity (whether a given row's whole struct is itself
//     SQL NULL), same convention as decodeFlatVector/decodeQuackListVector.
//   - field101: validity mask, present only when hasValidity is true.
//   - field103: a raw ULEB128 count (expected to equal len(lt.structFields))
//     followed by that many child Vector objects, decoded recursively via
//     decodeQuackVector, in the LogicalType's own field order (children are
//     not name-tagged on the wire). Unlike LIST, each child vector carries
//     the same row count as the parent — a STRUCT's children are
//     positionally aligned with their parent's rows, no flattening.
func decodeQuackStructVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if lt.structFields == nil {
		return quackColumn{}, fmt.Errorf("duckdb: quack STRUCT LogicalType is missing its child fields")
	}

	if err := r.beginProperty(100); err != nil {
		return quackColumn{}, err
	}
	hasValidity, err := r.readBool()
	if err != nil {
		return quackColumn{}, err
	}
	var validity []bool
	if hasValidity {
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		maskBytes, err := r.readData()
		if err != nil {
			return quackColumn{}, err
		}
		validity = decodeQuackValidityMask(maskBytes, count)
	}

	if err := r.beginProperty(103); err != nil {
		return quackColumn{}, err
	}
	childCount, err := r.beginList()
	if err != nil {
		return quackColumn{}, err
	}
	if int(childCount) != len(lt.structFields) {
		return quackColumn{}, fmt.Errorf("duckdb: quack STRUCT vector has %d children, expected %d", childCount, len(lt.structFields))
	}

	childValues := make([][]any, childCount)
	for i := range childValues {
		r.beginObject()
		child, err := decodeQuackVector(r, lt.structFields[i].typ, count)
		if err != nil {
			return quackColumn{}, err
		}
		if err := r.endObject(); err != nil {
			return quackColumn{}, err
		}
		childValues[i], err = child.values()
		if err != nil {
			return quackColumn{}, err
		}
	}

	structs := make([]any, count)
	for i := range structs {
		if validity != nil && !validity[i] {
			continue
		}
		m := make(map[string]any, len(lt.structFields))
		for j, f := range lt.structFields {
			m[f.name] = childValues[j][i]
		}
		structs[i] = m
	}
	return quackColumn{typeID: quackLogicalTypeStruct, structVals: structs, rowCount: count}, nil
}

func decodeFlatVector(r *quackReader, typeID byte, alias string, count int) (quackColumn, error) {
	if err := r.beginProperty(100); err != nil {
		return quackColumn{}, err
	}
	hasValidity, err := r.readBool()
	if err != nil {
		return quackColumn{}, err
	}
	// field101 (validity mask), present only when hasValidity is true, is a
	// packed-bit mask covering count rows, one bit per row, LSB-first within
	// each byte, 1=valid/0=null. field102 (the column's data) is always
	// present regardless, at the same width/shape as an all-valid column —
	// DuckDB writes an unspecified entry for a null row rather than omitting
	// it, so values() alone is responsible for substituting nil.
	var validity []bool
	if hasValidity {
		if err := r.beginProperty(101); err != nil {
			return quackColumn{}, err
		}
		maskBytes, err := r.readData()
		if err != nil {
			return quackColumn{}, err
		}
		validity = decodeQuackValidityMask(maskBytes, count)
	}

	if typeID == quackLogicalTypeVarchar {
		return decodeQuackVarcharFlatVector(r, typeID, alias, validity, count)
	}

	if err := r.beginProperty(102); err != nil {
		return quackColumn{}, err
	}
	data, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}
	return quackColumn{typeID: typeID, alias: alias, validity: validity, fixedData: data, rowCount: count}, nil
}

// decodeQuackVarcharFlatVector reads a VARCHAR flat vector's data. Unlike
// the old per-row length-prefixed list, protocol version 3 amortizes the
// length prefix across the whole vector via three fields:
//
//   - field107: a ULEB128 scalar equal to field109's total byte length,
//     restated; this client only reads past it.
//   - field108: a raw data blob of exactly 4*count bytes: one little-endian
//     uint32 byte-length per row, in row order. A NULL row still gets a real
//     zero-length entry — validity (field101) is the only NULL signal.
//   - field109: every row's string bytes concatenated back-to-back with no
//     separator — each row's slice is recovered by walking field108's
//     lengths and advancing a running offset.
func decodeQuackVarcharFlatVector(r *quackReader, typeID byte, alias string, validity []bool, count int) (quackColumn, error) {
	if err := r.beginProperty(107); err != nil {
		return quackColumn{}, err
	}
	if _, err := r.readUnsignedLeb128(); err != nil {
		return quackColumn{}, err
	}

	if err := r.beginProperty(108); err != nil {
		return quackColumn{}, err
	}
	lengthsRaw, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}
	if len(lengthsRaw) != count*4 {
		return quackColumn{}, fmt.Errorf("duckdb: quack VARCHAR vector has a %d-byte lengths array, expected %d for %d rows", len(lengthsRaw), count*4, count)
	}

	if err := r.beginProperty(109); err != nil {
		return quackColumn{}, err
	}
	allData, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}

	values := make([][]byte, count)
	offset := 0
	for i := range values {
		n := int(binary.LittleEndian.Uint32(lengthsRaw[i*4 : i*4+4]))
		if offset+n > len(allData) {
			return quackColumn{}, fmt.Errorf("duckdb: quack VARCHAR vector row %d length %d exceeds remaining data (%d bytes left of %d total)", i, n, len(allData)-offset, len(allData))
		}
		values[i] = allData[offset : offset+n]
		offset += n
	}
	return quackColumn{typeID: typeID, alias: alias, validity: validity, varchar: values, rowCount: count}, nil
}

// decodeQuackValidityMask unpacks a DuckDB ValidityMask's packed-bit wire
// representation into one bool per row (true = valid, false = SQL NULL).
// mask may be shorter than ceil(count/8) bytes — DuckDB only grows a
// validity mask's backing storage to cover bits it has actually cleared, so
// any row past the mask's end is implicitly valid.
func decodeQuackValidityMask(mask []byte, count int) []bool {
	valid := make([]bool, count)
	for i := range valid {
		byteIdx, bitIdx := i/8, uint(i%8)
		if byteIdx >= len(mask) {
			valid[i] = true
			continue
		}
		valid[i] = mask[byteIdx]&(1<<bitIdx) != 0
	}
	return valid
}
