package duckdb

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Quack message types, mirroring duckdb_quack::MessageType for protocol
// version 3. APPEND_REQUEST from the older v1.5-variegata protocol is gone,
// replaced by SEND_DATA_REQUEST/RESPONSE (see quack_senddata.go).
// DISCONNECT_MESSAGE(11), ACKNOWLEDGEMENT(15), and HEARTBEAT_REQUEST(16)
// exist server-side but aren't implemented here. CANCEL_REQUEST(12) is (see
// encodeQuackCancelRequest).
const (
	quackMsgConnectionRequest  byte = 1
	quackMsgConnectionResponse byte = 2
	quackMsgPrepareRequest     byte = 3
	quackMsgPrepareResponse    byte = 4
	quackMsgFetchRequest       byte = 7
	quackMsgFetchResponse      byte = 8
	quackMsgSendDataRequest    byte = 9
	quackMsgSuccessResponse    byte = 10
	quackMsgCancelRequest      byte = 12
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
// field2 is a mandatory hugeint query_uuid (omitting it crashes the server
// with a bodyless HTTP 500). queryID is generated by the caller, not here —
// it's the same value a later FetchRequest's uuid (already threaded through
// from PrepareResponse's own echoed-back query_uuid, field5) and a
// CancelRequest's query_uuid (see encodeQuackCancelRequest) both key off of,
// so the caller needs to hold onto it before this ever gets sent.
func encodeQuackPrepareRequest(connectionID, sql string, queryID quackHugeint) []byte {
	return encodeQuackMessage(quackMsgPrepareRequest, connectionID, func(w *quackWriter) {
		w.writeStringDefault(1, sql)
		w.writeHugeint(2, queryID)
	})
}

// encodeQuackCancelRequest builds a CancelRequest: field1 is the hugeint
// query_uuid of the statement to interrupt — the same value passed to the
// PrepareRequest that started it. The server accepts the zero hugeint
// {0,0} as a wildcard meaning "cancel whatever is running on this
// connection", but this client always sends the real value: matching by
// the exact id means a cancel that loses a race against the query's own
// natural completion (or against a next query already started on the same
// connection) is safely rejected by the server instead of ever being able
// to interrupt the wrong query.
func encodeQuackCancelRequest(connectionID string, queryID quackHugeint) []byte {
	return encodeQuackMessage(quackMsgCancelRequest, connectionID, func(w *quackWriter) {
		w.writeHugeint(1, queryID)
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

// encodeQuackFetchRequest builds a FetchRequest asking the server for
// batchIndex of an already-prepared result (uuid, from the PrepareResponse
// or a prior FetchResponse's result_uuid). field2 batch_index is written
// with the plain (never-omitted) writeUint64, not writeUint64Default:
// confirmed empirically that the server requires it explicitly present —
// "FETCH_REQUEST is missing its batch index" — even when the value is 0
// (the first fetch), which writeUint64Default would otherwise silently
// drop as "the default, so omit it".
func encodeQuackFetchRequest(connectionID string, uuid quackHugeint, batchIndex uint64) []byte {
	return encodeQuackMessage(quackMsgFetchRequest, connectionID, func(w *quackWriter) {
		w.writeHugeint(1, uuid)
		w.writeUint64(2, batchIndex)
	})
}

// decodeQuackFetchResponseBody reads a FetchResponse body and returns rows
// decoded from every DataChunk it carries, keyed against names (the
// PrepareResponse's own result_names — a FetchResponse repeats the same
// columns, so it carries no names of its own). Confirmed against the real
// quack extension's disassembly (no source is vendored in this repo):
// unlike PrepareResponse, whose inline chunks are embedded as a field4 list
// of DataChunkWrapper objects, FetchResponseMessage carries no chunk data
// as a structured field at all — the message body is just three optional
// int fields (field1 chunk_count, field2 total_batches, field3
// batch_index, all omitted-when-zero like any optional field on this
// wire), and the rows are chunkCount bare DataChunk objects appended
// back-to-back immediately after the message's own terminator — the same
// concatenated-blob convention SEND_DATA_REQUEST uses on the request side
// (see quack_senddata.go's doc comment), not PrepareResponse's
// DataChunkWrapper-wrapped, field-300-nested inline chunks.
//
// chunkCount is also the caller's "more to fetch?" signal: the wire format
// has no explicit "no more rows" flag, so the server signals it by
// eventually returning a response with chunk_count 0 (see
// quackSession.query's fetch loop). batchIndex (field3) is the batch this
// response just delivered — the caller's next FetchRequest must echo
// batchIndex+1 back (encodeQuackFetchRequest's doc comment has the full
// story on why the request side needs this at all, including why 0 is
// never a usable value for it).
func decodeQuackFetchResponseBody(r *quackReader, names []string) (rows []map[string]any, chunkCount int, batchIndex uint64, err error) {
	var chunkCountU64 uint64
	if ok, terr := r.tryBeginProperty(1); terr != nil {
		return nil, 0, 0, terr
	} else if ok {
		chunkCountU64, terr = r.readUInt64()
		if terr != nil {
			return nil, 0, 0, terr
		}
	}

	if ok, terr := r.tryBeginProperty(2); terr != nil {
		return nil, 0, 0, terr
	} else if ok {
		if _, terr := r.readUInt64(); terr != nil { // total_batches: unused here
			return nil, 0, 0, terr
		}
	}

	if ok, terr := r.tryBeginProperty(3); terr != nil {
		return nil, 0, 0, terr
	} else if ok {
		batchIndex, err = r.readUInt64()
		if err != nil {
			return nil, 0, 0, err
		}
	}

	if err := r.endObject(); err != nil {
		return nil, 0, 0, err
	}

	for i := uint64(0); i < chunkCountU64; i++ {
		chunk, terr := decodeQuackDataChunk(r)
		if terr != nil {
			return nil, 0, 0, terr
		}
		chunkRows, terr := chunk.namedRows(names)
		if terr != nil {
			return nil, 0, 0, terr
		}
		rows = append(rows, chunkRows...)
	}

	return rows, int(chunkCountU64), batchIndex, nil
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
	quackLogicalTypeUnion        byte = 107
	quackLogicalTypeArray        byte = 108
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
	// arraySize is set only when id == quackLogicalTypeArray: the fixed
	// element count every row's array holds (ArrayTypeInfo.size,
	// extra_type_info.hpp), read from field201 following ARRAY's child type
	// at field200.
	arraySize uint32
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
				case quackLogicalTypeStruct, quackLogicalTypeUnion:
					// UNION's type_info is, byte-for-byte, a STRUCT's: its
					// LogicalType is built from a StructTypeInfo with a
					// hidden field 0 (name "", type UTINYINT) holding each
					// row's member tag, followed by the user-visible member
					// fields (types.cpp: LogicalType::UNION prepends that
					// tag field before constructing its StructTypeInfo) —
					// confirmed empirically: parsing UNION's type_info via
					// the single-child default branch below (as if it were
					// LIST/MAP-shaped) desyncs the wire and fails much
					// later, downstream, with an unrelated field-id error.
					fields, ferr := decodeQuackStructFieldTypes(r)
					if ferr != nil {
						return quackLogicalType{}, ferr
					}
					lt.structFields = fields
				case quackLogicalTypeArray:
					child, cerr := decodeQuackLogicalType(r)
					if cerr != nil {
						return quackLogicalType{}, cerr
					}
					lt.child = &child
					// field201 (size) is ARRAY's ArrayTypeInfo.size
					// (extra_type_info.hpp), a plain ULEB128 following the
					// child type at field200 — confirmed empirically: without
					// this, endObject below fails on an unexpected field 201.
					if err := r.beginProperty(201); err != nil {
						return quackLogicalType{}, err
					}
					size, serr := r.readUnsignedLeb128()
					if serr != nil {
						return quackLogicalType{}, serr
					}
					lt.arraySize = uint32(size)
				case quackLogicalTypeDecimal:
					width, werr := r.readByte()
					if werr != nil {
						return quackLogicalType{}, werr
					}
					lt.decimalWidth = width
					// field201 (scale) is default-omit like every other
					// scalar field on the wire — a DECIMAL(_,0) never
					// writes it, so try rather than require it, leaving
					// decimalScale at its zero value when absent.
					if ok, serr := r.tryBeginProperty(201); serr != nil {
						return quackLogicalType{}, serr
					} else if ok {
						scale, serr := r.readByte()
						if serr != nil {
							return quackLogicalType{}, serr
						}
						lt.decimalScale = scale
					}
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
	case quackLogicalTypeUnion:
		parts := make([]string, 0, len(lt.structFields))
		for _, f := range lt.structFields[1:] { // structFields[0] is the hidden tag field
			parts = append(parts, f.name+" "+f.typ.typeName())
		}
		return "UNION(" + strings.Join(parts, ", ") + ")"
	case quackLogicalTypeArray:
		if lt.child == nil {
			return "UNKNOWN[]"
		}
		return fmt.Sprintf("%s[%d]", lt.child.typeName(), lt.arraySize)
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
	// decimalWidth/decimalScale are set only when typeID ==
	// quackLogicalTypeDecimal — see quackLogicalType's fields of the same
	// name. decimalWidth picks fixedData's physical byte width per row (see
	// decimalPhysicalWidth); decimalScale places the decimal point.
	decimalWidth byte
	decimalScale byte
	// enumValues is set only when typeID == quackLogicalTypeEnum: the
	// dictionary values() looks a row's decoded index up in.
	enumValues []string
	rowCount   int
	// precomputed, when non-nil, is returned by values() as-is instead of
	// decoding fixedData/varchar/list/structVals. Used for the non-flat
	// vector encodings (CONSTANT_VECTOR, DICTIONARY_VECTOR, SEQUENCE_VECTOR)
	// whose physical layout doesn't fit quackColumn's flat-vector shape —
	// see decodeQuackConstantVector, decodeQuackDictionaryVector,
	// decodeQuackSequenceVector, all called from decodeQuackVector.
	precomputed []any
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
	if c.precomputed != nil {
		return c.precomputed, nil
	}
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
	case quackLogicalTypeTinyInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int8(c.fixedData[i]))
		}
	case quackLogicalTypeUTinyInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(c.fixedData[i])
		}
	case quackLogicalTypeSmallInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int16(le16(c.fixedData[i*2:])))
		}
	case quackLogicalTypeUSmallInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(le16(c.fixedData[i*2:]))
		}
	case quackLogicalTypeInteger:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(int32(le32(c.fixedData[i*4:])))
		}
	case quackLogicalTypeUInteger:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(le32(c.fixedData[i*4:]))
		}
	case quackLogicalTypeBigInt:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = int64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeUBigInt:
		// UBIGINT's range (up to 2^64-1) overflows int64, so it's decoded as
		// a decimal-digit string instead — the same choice DuckDB's own JSON
		// output makes (confirmed against a real binary: `SELECT
		// 18000000000000000000::UBIGINT` renders as the JSON string
		// "18000000000000000000", not a JSON number) since a value that
		// large would silently overflow/wrap as int64 or lose precision as
		// float64.
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = strconv.FormatUint(le64(c.fixedData[i*8:]), 10)
		}
	case quackLogicalTypeFloat:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = float64(math.Float32frombits(le32(c.fixedData[i*4:])))
		}
	case quackLogicalTypeDouble:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = le64ToFloat64(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeHugeint, quackLogicalTypeUHugeint:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = quackHugeintBigInt(c.fixedData[i*16:i*16+16], c.typeID == quackLogicalTypeHugeint).String()
		}
	case quackLogicalTypeDecimal:
		width := quackDecimalPhysicalWidth(c.decimalWidth)
		for i := range out {
			if c.isNull(i) {
				continue
			}
			var v *big.Int
			switch width {
			case 2:
				v = big.NewInt(int64(int16(le16(c.fixedData[i*2:]))))
			case 4:
				v = big.NewInt(int64(int32(le32(c.fixedData[i*4:]))))
			case 8:
				v = big.NewInt(int64(le64(c.fixedData[i*8:])))
			default: // 16: DECIMAL(19..38, _)
				v = quackHugeintBigInt(c.fixedData[i*16:i*16+16], true)
			}
			out[i] = quackDecimalToString(v, c.decimalScale)
		}
	case quackLogicalTypeInterval:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			row := c.fixedData[i*16 : i*16+16]
			months := int32(le32(row[0:4]))
			days := int32(le32(row[4:8]))
			micros := int64(le64(row[8:16]))
			out[i] = quackIntervalToString(months, days, micros)
		}
	case quackLogicalTypeEnum:
		idxWidth := quackEnumIndexWidth(len(c.enumValues))
		for i := range out {
			if c.isNull(i) {
				continue
			}
			var idx uint32
			switch idxWidth {
			case 1:
				idx = uint32(c.fixedData[i])
			case 2:
				idx = uint32(le16(c.fixedData[i*2:]))
			default:
				idx = le32(c.fixedData[i*4:])
			}
			if int(idx) >= len(c.enumValues) {
				return nil, fmt.Errorf("duckdb: quack column decode: ENUM index %d out of range for a %d-value dictionary", idx, len(c.enumValues))
			}
			out[i] = c.enumValues[idx]
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
	case quackLogicalTypeBlob:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = append([]byte(nil), c.varchar[i]...)
		}
	case quackLogicalTypeBit:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = quackBitToString(c.varchar[i])
		}
	case quackLogicalTypeTimestamp, quackLogicalTypeTimestampMs, quackLogicalTypeTimestampSec, quackLogicalTypeTimestampNs, quackLogicalTypeTimestampTZ:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			raw := int64(le64(c.fixedData[i*8:]))
			out[i] = quackTimestampToTime(c.typeID, raw)
		}
	case quackLogicalTypeDate:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			days := int32(le32(c.fixedData[i*4:]))
			out[i] = quackDateToTime(days)
		}
	case quackLogicalTypeTime:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			micros := int64(le64(c.fixedData[i*8:]))
			out[i] = quackTimeToTime(micros)
		}
	case quackLogicalTypeTimeTZ:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = quackTimeTZToTime(le64(c.fixedData[i*8:]))
		}
	case quackLogicalTypeUUID:
		for i := range out {
			if c.isNull(i) {
				continue
			}
			out[i] = quackUUIDToString(c.fixedData[i*16 : i*16+16])
		}
	case quackLogicalTypeList, quackLogicalTypeArray:
		copy(out, c.list)
	case quackLogicalTypeStruct, quackLogicalTypeUnion:
		copy(out, c.structVals)
	default:
		return nil, fmt.Errorf("duckdb: quack column decode: unsupported LogicalTypeId %d", c.typeID)
	}
	return out, nil
}

// quackTimestampToTime also covers TimestampTZ: DuckDB stores TIMESTAMP WITH
// TIME ZONE as the same 8-byte microseconds-since-epoch UTC instant as plain
// TIMESTAMP (timestamp.hpp) — the timezone only affects display formatting,
// never the physical encoding — so it falls into the same default branch.
func quackTimestampToTime(typeID byte, raw int64) time.Time {
	switch typeID {
	case quackLogicalTypeTimestampSec:
		return time.Unix(raw, 0).UTC()
	case quackLogicalTypeTimestampMs:
		return time.UnixMilli(raw).UTC()
	case quackLogicalTypeTimestampNs:
		return time.Unix(0, raw).UTC()
	default: // microseconds (Timestamp, TimestampTZ)
		return time.UnixMicro(raw).UTC()
	}
}

// quackDateToTime decodes a DATE column's physical representation — a
// signed count of days relative to the Unix epoch (1970-01-01), matching
// DuckDB's date_t — into a UTC midnight time.Time, the same shape
// convertTimeValue produces for a "DATE" column over the jsonlines
// transport (layout "2006-01-02", no time-of-day component).
func quackDateToTime(days int32) time.Time {
	return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(days))
}

// quackTimeToTime decodes a TIME column's physical representation — signed
// microseconds since midnight, matching DuckDB's dtime_t — into a time.Time
// on the zero date (year 0, January 1, UTC): the same result Go's
// time.Parse produces for jsonlines' TIME layout ("15:04:05.999999999"),
// whose text carries no date component.
func quackTimeToTime(micros int64) time.Time {
	return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(micros) * time.Microsecond)
}

// quackTimeTZOffsetBits/quackTimeTZMaxOffset mirror DuckDB's dtime_tz_t
// layout (duckdb/common/types/datetime.hpp): a TIME WITH TIME ZONE value is
// one uint64 packing microseconds-since-midnight in the high 40 bits and a
// biased, sign-reversed UTC offset in the low 24 bits.
const (
	quackTimeTZOffsetBits = 24
	quackTimeTZMaxOffset  = 16*60*60 - 1 // ±15:59:59, DuckDB's MAX_OFFSET
)

// quackTimeTZToTime decodes a TIME WITH TIME ZONE column's packed bits (see
// quackTimeTZOffsetBits) into a time.Time on the zero date, in a
// time.FixedZone carrying the value's own UTC offset. jsonlines has no
// converted shape to match here (rows.go passes TIME WITH TIME ZONE through
// as a raw string), so this is quack's own best-effort decode rather than a
// parity requirement.
func quackTimeTZToTime(bits uint64) time.Time {
	offsetMask := uint64(1)<<quackTimeTZOffsetBits - 1
	micros := int64(bits >> quackTimeTZOffsetBits)
	offsetSeconds := quackTimeTZMaxOffset - int(bits&offsetMask)
	loc := time.FixedZone(quackUTCOffsetName(offsetSeconds), offsetSeconds)
	return time.Date(0, 1, 1, 0, 0, 0, 0, loc).Add(time.Duration(micros) * time.Microsecond)
}

// quackUTCOffsetName formats a UTC offset in seconds as "UTC+HH:MM" (or
// "UTC-HH:MM"), used only as time.FixedZone's cosmetic zone name.
func quackUTCOffsetName(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, offsetSeconds/3600, (offsetSeconds%3600)/60)
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

// quackHugeintBigInt decodes a 16-byte little-endian hugeint_t/uhugeint_t
// (hugeint.hpp: `lower uint64` at bytes[0:8], `upper` at bytes[8:16]) into a
// big.Int, sign-extending from upper's top bit when signed is true
// (hugeint_t's upper is int64; uhugeint_t's is uint64, so signed is false).
func quackHugeintBigInt(b []byte, signed bool) *big.Int {
	lower := le64(b[0:8])
	upper := le64(b[8:16])
	v := new(big.Int).Lsh(new(big.Int).SetUint64(upper), 64)
	v.Or(v, new(big.Int).SetUint64(lower))
	if signed && int64(upper) < 0 {
		v.Sub(v, new(big.Int).Lsh(big.NewInt(1), 128))
	}
	return v
}

// quackDecimalPhysicalWidth maps a DECIMAL's declared width to the byte
// width of its physical storage, per DuckDB's own LogicalType::
// GetInternalType (types.cpp) and Decimal::MAX_WIDTH_INT16/32/64/128
// (decimal.hpp): width<=4 fits int16, <=9 fits int32, <=18 fits int64,
// <=38 (DECIMAL's max) needs the full 16-byte hugeint.
func quackDecimalPhysicalWidth(width byte) int {
	switch {
	case width <= 4:
		return 2
	case width <= 9:
		return 4
	case width <= 18:
		return 8
	default:
		return 16
	}
}

// quackDecimalToString formats a DECIMAL column's raw fixed-point integer
// as DuckDB's own Decimal::ToString would: exactly scale digits after the
// decimal point (no trailing-zero trimming, unlike timestamps/intervals),
// no decimal point at all when scale is 0, and a "-" sign only when the
// underlying integer is actually negative (confirmed against a real
// binary: 0::DECIMAL(10,3) is "0.000", never "-0.000").
func quackDecimalToString(v *big.Int, scale byte) string {
	neg := v.Sign() < 0
	digits := new(big.Int).Abs(v).String()
	if scale == 0 {
		if neg {
			return "-" + digits
		}
		return digits
	}
	for len(digits) <= int(scale) {
		digits = "0" + digits
	}
	s := digits[:len(digits)-int(scale)] + "." + digits[len(digits)-int(scale):]
	if neg {
		return "-" + s
	}
	return s
}

// quackIntervalToString decodes an INTERVAL column's 16-byte physical
// interval_t{months int32; days int32; micros int64} (interval.hpp) into
// the same text DuckDB's own Interval::ToString produces. Each component is
// independently signed and omitted when zero — falling back to "00:00:00"
// only when all three are zero — matching output confirmed against a real
// duckdb binary (`-jsonlines`): "1 year 2 months 3 days 04:05:06.789",
// "-1 year -2 days -03:00:00", "1 day -03:00:00", "0 seconds" -> "00:00:00".
func quackIntervalToString(months, days int32, micros int64) string {
	var parts []string
	if months != 0 {
		years, remMonths := months/12, months%12
		if years != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", years, quackIntervalUnit("year", years)))
		}
		if remMonths != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", remMonths, quackIntervalUnit("month", remMonths)))
		}
	}
	if days != 0 {
		parts = append(parts, fmt.Sprintf("%d %s", days, quackIntervalUnit("day", days)))
	}
	if micros != 0 || len(parts) == 0 {
		parts = append(parts, quackIntervalTimeToString(micros))
	}
	return strings.Join(parts, " ")
}

// quackIntervalUnit pluralizes an interval component's unit name: DuckDB
// only keeps it singular for exactly ±1.
func quackIntervalUnit(unit string, n int32) string {
	if n == 1 || n == -1 {
		return unit
	}
	return unit + "s"
}

// quackIntervalTimeToString formats INTERVAL's micros component as
// "[-]HH:MM:SS[.frac]" — hours are total elapsed hours, not wrapped to
// 24 (DuckDB prints "25:00:00" for a 25-hour interval), and a fractional
// part is trimmed of trailing zeros (never a fixed 6 digits) but included
// only when non-zero, matching TIMESTAMP's fractional-seconds convention.
func quackIntervalTimeToString(micros int64) string {
	sign := ""
	if micros < 0 {
		sign = "-"
		micros = -micros
	}
	totalSeconds := micros / 1_000_000
	fracMicros := micros % 1_000_000
	hours, minutes, seconds := totalSeconds/3600, (totalSeconds%3600)/60, totalSeconds%60
	s := fmt.Sprintf("%s%02d:%02d:%02d", sign, hours, minutes, seconds)
	if fracMicros != 0 {
		s += "." + strings.TrimRight(fmt.Sprintf("%06d", fracMicros), "0")
	}
	return s
}

// quackEnumIndexWidth maps an ENUM's dictionary size to its dictionary
// index's physical byte width, per DuckDB's EnumTypeInfo::DictType
// (extra_type_info.cpp): a dictionary small enough to fit in uint8 uses a
// 1-byte index, uint16 range uses 2 bytes, otherwise 4 (uint32; DuckDB
// rejects a dictionary larger than that outright).
func quackEnumIndexWidth(dictSize int) int {
	switch {
	case dictSize <= 0xFF:
		return 1
	case dictSize <= 0xFFFF:
		return 2
	default:
		return 4
	}
}

// quackBitToString decodes a BIT column's physical bitstring_t bytes (BIT
// shares VARCHAR's variable-length wire shape — bit.hpp: "using
// bitstring_t = duckdb::string_t;") into the same '0'/'1' text DuckDB's own
// Bit::ToString (bit.cpp) renders: byte 0 is a padding count (0-7) for the
// partial first data byte; byte 1's low (8-padding) bits come first (MSB to
// LSB), then every remaining byte's 8 bits, MSB to LSB.
func quackBitToString(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	padding := int(b[0])
	var sb strings.Builder
	sb.Grow((8 - padding) + 8*(len(b)-2))
	for bitIdx := padding; bitIdx < 8; bitIdx++ {
		sb.WriteByte(quackBitChar(b[1], 7-bitIdx))
	}
	for byteIdx := 2; byteIdx < len(b); byteIdx++ {
		for bitIdx := 0; bitIdx < 8; bitIdx++ {
			sb.WriteByte(quackBitChar(b[byteIdx], 7-bitIdx))
		}
	}
	return sb.String()
}

func quackBitChar(b byte, bitPos int) byte {
	if b&(1<<uint(bitPos)) != 0 {
		return '1'
	}
	return '0'
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
func decodeQuackVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	vectorType := byte(0)
	if ok, err := r.tryBeginProperty(90); err != nil {
		return quackColumn{}, err
	} else if ok {
		vectorType, err = r.readByte()
		if err != nil {
			return quackColumn{}, err
		}
	}
	// DuckDB's VectorType enum ordinals, confirmed empirically against the
	// wire (a plain ULEB128, not the human-readable string
	// EnumUtil::FromString<VectorType> uses elsewhere in the engine for an
	// unrelated serializer — that string path was a red herring found while
	// reverse-engineering this field): 0=FLAT_VECTOR (handled below via
	// decodeFlatVector and friends), 1=FSST_VECTOR (compressed strings, not
	// yet seen/handled), 2=CONSTANT_VECTOR, 3=DICTIONARY_VECTOR,
	// 4=SEQUENCE_VECTOR. Note this is NOT 0/1/2/3 as the field's own name
	// might suggest — DuckDB inserted FSST_VECTOR as ordinal 1 after
	// FLAT_VECTOR in a later release, shifting the rest.
	switch vectorType {
	case 0:
		return decodeQuackVectorBody(r, lt, count)
	case 2:
		return decodeQuackConstantVector(r, lt, count)
	case 3:
		return decodeQuackDictionaryVector(r, lt, count)
	case 4:
		return decodeQuackSequenceVector(r, count)
	default:
		return quackColumn{}, fmt.Errorf("duckdb: quack vector type %d is not supported by this client", vectorType)
	}
}

// decodeQuackVectorBody decodes a vector's per-type-shape fields once its
// optional leading vector_type field (90) has already been consumed by the
// caller (decodeQuackVector, or decodeQuackConstantVector/
// decodeQuackDictionaryVector reusing it for their nested value vector — see
// those functions' doc comments for why no further vector_type field, nor
// object bracketing, follows for that nested vector on the wire).
func decodeQuackVectorBody(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	// MAP's wire (and in-memory, per LogicalType::GetInternalType) shape is
	// byte-for-byte a LIST of STRUCT{key,value} — decodeQuackLogicalType
	// already parsed MAP's child as that STRUCT type, so no separate decode
	// path is needed; values() never distinguishes the two afterwards
	// either, since decodeQuackListVector's result always reports typeID
	// quackLogicalTypeList regardless of which one produced it.
	if lt.id == quackLogicalTypeList || lt.id == quackLogicalTypeMap {
		return decodeQuackListVector(r, lt, count)
	}
	if lt.id == quackLogicalTypeStruct {
		return decodeQuackStructVector(r, lt, count)
	}
	if lt.id == quackLogicalTypeUnion {
		return decodeQuackUnionVector(r, lt, count)
	}
	if lt.id == quackLogicalTypeArray {
		return decodeQuackArrayVector(r, lt, count)
	}
	return decodeFlatVector(r, lt, count)
}

// decodeQuackConstantVector reads a CONSTANT_VECTOR. Its single logical value
// follows immediately, in the SAME field-id stream as this vector's own
// object (no nested beginObject/endObject bracketing, and no repeated
// vector_type field) — confirmed empirically: wrapping the recursive decode
// in its own beginObject/endObject (matching the convention
// decodeQuackListVector's child vector, field106, uses) produced "expected
// end-of-object terminator but found field id 0x0005", i.e. there is no
// second terminator to consume there. So this just decodes one row's worth
// of the vector's ordinary per-type fields (100/101/102, or a nested
// struct/list/etc.) directly, then broadcasts that single row across count.
func decodeQuackConstantVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	inner, err := decodeQuackVectorBody(r, lt, 1)
	if err != nil {
		return quackColumn{}, err
	}
	innerVals, err := inner.values()
	if err != nil {
		return quackColumn{}, err
	}
	out := make([]any, count)
	for i := range out {
		out[i] = innerVals[0]
	}
	return quackColumn{rowCount: count, precomputed: out}, nil
}

// decodeQuackDictionaryVector reads a DICTIONARY_VECTOR:
//
//   - field91 "sel_vector": a raw blob of count*4 bytes — one little-endian
//     int32 selection index per row, into the dictionary vector below.
//   - field92 "dict_count": a ULEB128 row count for the dictionary vector.
//   - a nested Vector object (beginObject/decodeQuackVector/endObject, same
//     convention as the CONSTANT_VECTOR and LIST child cases) holding the
//     dictionary's dict_count unique values.
//
// Materializes as out[row] = dictionary_values[sel_vector[row]]. Confirmed
// via disassembly (field names/order) plus the sel_vector blob's size always
// matching count*4 bytes in practice.
func decodeQuackDictionaryVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if err := r.beginProperty(91); err != nil {
		return quackColumn{}, err
	}
	selBytes, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}
	if len(selBytes) != count*4 {
		return quackColumn{}, fmt.Errorf("duckdb: quack dictionary sel_vector has %d bytes, expected %d (count=%d)", len(selBytes), count*4, count)
	}

	if err := r.beginProperty(92); err != nil {
		return quackColumn{}, err
	}
	dictCount, err := r.readUnsignedLeb128()
	if err != nil {
		return quackColumn{}, err
	}

	// No object bracketing around this nested vector — see
	// decodeQuackConstantVector's doc comment for why.
	dict, err := decodeQuackVector(r, lt, int(dictCount))
	if err != nil {
		return quackColumn{}, err
	}
	dictVals, err := dict.values()
	if err != nil {
		return quackColumn{}, err
	}

	out := make([]any, count)
	for i := range out {
		idx := int32(le32(selBytes[i*4:]))
		if idx < 0 || int(idx) >= len(dictVals) {
			return quackColumn{}, fmt.Errorf("duckdb: quack dictionary sel_vector[%d]=%d out of range [0,%d)", i, idx, len(dictVals))
		}
		out[i] = dictVals[idx]
	}
	return quackColumn{rowCount: count, precomputed: out}, nil
}

// decodeQuackSequenceVector reads a SEQUENCE_VECTOR: field91 "seq_start"
// (signed LEB128 int64) and field92 "seq_increment" (signed LEB128 int64).
// Materializes as out[i] = seq_start + i*seq_increment for i in [0,count) —
// no validity mask, since a sequence vector can never contain a NULL.
// Confirmed via disassembly (field names/order/types).
func decodeQuackSequenceVector(r *quackReader, count int) (quackColumn, error) {
	if err := r.beginProperty(91); err != nil {
		return quackColumn{}, err
	}
	start, err := r.readSignedLeb128()
	if err != nil {
		return quackColumn{}, err
	}
	if err := r.beginProperty(92); err != nil {
		return quackColumn{}, err
	}
	increment, err := r.readSignedLeb128()
	if err != nil {
		return quackColumn{}, err
	}
	out := make([]any, count)
	for i := range out {
		out[i] = start + int64(i)*increment
	}
	return quackColumn{rowCount: count, precomputed: out}, nil
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

// decodeQuackUnionVector reads a UNION Vector object. UNION's physical
// representation (LogicalType::GetInternalType, types.cpp) is
// PhysicalType::STRUCT — the exact same wire shape decodeQuackStructVector
// reads above — but with a hidden field 0 (name "", type UTINYINT) holding
// each row's member tag (an index into the remaining fields) rather than a
// value of its own. Rather than exposing every member (mostly nil, since
// only the tagged one is ever valid) plus the raw tag, this decodes into a
// single-entry map{selected member's name: its value} — matching DuckDB's
// own JSON rendering of a UNION value, confirmed against a real binary:
// `SELECT union_value(a := 1)` renders as {"a": 1}, not the full struct.
func decodeQuackUnionVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if len(lt.structFields) < 2 {
		return quackColumn{}, fmt.Errorf("duckdb: quack UNION LogicalType is missing its tag/member fields")
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
		return quackColumn{}, fmt.Errorf("duckdb: quack UNION vector has %d children, expected %d", childCount, len(lt.structFields))
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

	unions := make([]any, count)
	for i := range unions {
		if validity != nil && !validity[i] {
			continue
		}
		// childValues[0] is the tag field (UTINYINT), already decoded by
		// values()'s UTinyInt case into an int64.
		tag, ok := childValues[0][i].(int64)
		if !ok {
			return quackColumn{}, fmt.Errorf("duckdb: quack UNION tag decoded as %T, expected int64", childValues[0][i])
		}
		memberIdx := int(tag) + 1 // +1 to skip the hidden tag field itself
		if memberIdx < 1 || memberIdx >= len(lt.structFields) {
			return quackColumn{}, fmt.Errorf("duckdb: quack UNION tag %d out of range for %d members", tag, len(lt.structFields)-1)
		}
		unions[i] = map[string]any{lt.structFields[memberIdx].name: childValues[memberIdx][i]}
	}
	return quackColumn{typeID: quackLogicalTypeUnion, structVals: unions, rowCount: count}, nil
}

// decodeQuackArrayVector reads an ARRAY Vector object: a fixed-size list,
// physically PhysicalType::ARRAY (types.cpp) rather than LIST. Its wire
// shape mirrors decodeQuackListVector's — field _n_ is the flattened child
// vector's total element count, field _n+1_ wraps that one child vector,
// decoded recursively — but shifted to field103/104 (LIST uses 104/106) and
// missing LIST's field105 list_entry_t offsets/lengths table entirely,
// since every row holds exactly lt.arraySize elements at a static stride:
// row i's elements are simply the flattened child vector's slice
// [i*arraySize : (i+1)*arraySize], no per-row offset needed. Confirmed
// empirically by hex-dumping a live `SELECT [1,2,3]::INTEGER[3]` response.
func decodeQuackArrayVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	if lt.child == nil {
		return quackColumn{}, fmt.Errorf("duckdb: quack ARRAY LogicalType is missing its child type")
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

	// field103 merely restates lt.arraySize (confirmed empirically: it does
	// NOT vary with count, even for a nested ARRAY-of-ARRAY where the true
	// flattened total per decodeQuackListVector's analogous field104 would
	// be count*arraySize) — this client only reads past it, same as
	// VARCHAR's field107 restated total length (decodeQuackVarcharFlatVector).
	if err := r.beginProperty(103); err != nil {
		return quackColumn{}, err
	}
	restatedSize, err := r.readUnsignedLeb128()
	if err != nil {
		return quackColumn{}, err
	}
	size := int(lt.arraySize)
	if int(restatedSize) != size {
		return quackColumn{}, fmt.Errorf("duckdb: quack ARRAY vector restates size %d, expected %d", restatedSize, size)
	}

	if err := r.beginProperty(104); err != nil {
		return quackColumn{}, err
	}
	r.beginObject()
	child, err := decodeQuackVector(r, *lt.child, count*size)
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
	arrays := make([]any, count)
	for i := range arrays {
		if validity != nil && !validity[i] {
			continue
		}
		arrays[i] = append([]any{}, childValues[i*size:(i+1)*size]...)
	}
	return quackColumn{typeID: quackLogicalTypeArray, list: arrays, rowCount: count}, nil
}

func decodeFlatVector(r *quackReader, lt quackLogicalType, count int) (quackColumn, error) {
	typeID, alias := lt.id, lt.alias
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

	// VARCHAR, BLOB, and BIT all share the same physical representation —
	// PhysicalType::VARCHAR, i.e. a variable-length string_t/bitstring_t
	// (bit.hpp: "using bitstring_t = duckdb::string_t;") — so all three use
	// the same var-length wire shape; values() tells them apart afterwards
	// by typeID alone.
	if typeID == quackLogicalTypeVarchar || typeID == quackLogicalTypeBlob || typeID == quackLogicalTypeBit {
		return decodeQuackVarcharFlatVector(r, typeID, alias, validity, count)
	}

	if err := r.beginProperty(102); err != nil {
		return quackColumn{}, err
	}
	data, err := r.readData()
	if err != nil {
		return quackColumn{}, err
	}
	return quackColumn{
		typeID:       typeID,
		alias:        alias,
		validity:     validity,
		fixedData:    data,
		decimalWidth: lt.decimalWidth,
		decimalScale: lt.decimalScale,
		enumValues:   lt.enumValues,
		rowCount:     count,
	}, nil
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
