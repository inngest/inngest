package rueidis

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/rueian/rueidis/internal/util"
)

const messageStructSize = int(unsafe.Sizeof(RedisMessage{}))

// Nil represents a Redis Nil message
var Nil = &RedisError{typ: '_'}

// IsRedisNil is a handy method to check if error is redis nil response.
// All redis nil response returns as an error.
func IsRedisNil(err error) bool {
	return err == Nil
}

// RedisError is an error response or a nil message from redis instance
type RedisError RedisMessage

func (r *RedisError) Error() string {
	if r.IsNil() {
		return "redis nil message"
	}
	return r.string
}

// IsNil checks if it is a redis nil message.
func (r *RedisError) IsNil() bool {
	return r.typ == '_'
}

// IsMoved checks if it is a redis MOVED message and returns moved address.
func (r *RedisError) IsMoved() (addr string, ok bool) {
	if ok = strings.HasPrefix(r.string, "MOVED"); ok {
		addr = strings.Split(r.string, " ")[2]
	}
	return
}

// IsAsk checks if it is a redis ASK message and returns ask address.
func (r *RedisError) IsAsk() (addr string, ok bool) {
	if ok = strings.HasPrefix(r.string, "ASK"); ok {
		addr = strings.Split(r.string, " ")[2]
	}
	return
}

// IsTryAgain checks if it is a redis TRYAGAIN message and returns ask address.
func (r *RedisError) IsTryAgain() bool {
	return strings.HasPrefix(r.string, "TRYAGAIN")
}

// IsClusterDown checks if it is a redis CLUSTERDOWN message and returns ask address.
func (r *RedisError) IsClusterDown() bool {
	return strings.HasPrefix(r.string, "CLUSTERDOWN")
}

// IsNoScript checks if it is a redis NOSCRIPT message.
func (r *RedisError) IsNoScript() bool {
	return strings.HasPrefix(r.string, "NOSCRIPT")
}

func newResult(val RedisMessage, err error) RedisResult {
	return RedisResult{val: val, err: err}
}

func newErrResult(err error) RedisResult {
	return RedisResult{err: err}
}

// RedisResult is the return struct from Client.Do or Client.DoCache
// it contains either a redis response or an underlying error (ex. network timeout).
type RedisResult struct {
	err error
	val RedisMessage
}

// RedisError can be used to check if the redis response is an error message.
func (r RedisResult) RedisError() *RedisError {
	if err := r.val.Error(); err != nil {
		return err.(*RedisError)
	}
	return nil
}

// NonRedisError can be used to check if there is an underlying error (ex. network timeout).
func (r RedisResult) NonRedisError() error {
	return r.err
}

// Error returns either underlying error or redis error or nil
func (r RedisResult) Error() (err error) {
	if r.err != nil {
		err = r.err
	} else {
		err = r.val.Error()
	}
	return
}

// ToMessage retrieves the RedisMessage
func (r RedisResult) ToMessage() (v RedisMessage, err error) {
	if r.err != nil {
		err = r.err
	} else {
		err = r.val.Error()
	}
	return r.val, err
}

// ToInt64 delegates to RedisMessage.ToInt64
func (r RedisResult) ToInt64() (v int64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToInt64()
	}
	return
}

// ToBool delegates to RedisMessage.ToBool
func (r RedisResult) ToBool() (v bool, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToBool()
	}
	return
}

// ToFloat64 delegates to RedisMessage.ToFloat64
func (r RedisResult) ToFloat64() (v float64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToFloat64()
	}
	return
}

// ToString delegates to RedisMessage.ToString
func (r RedisResult) ToString() (v string, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToString()
	}
	return
}

// AsReader delegates to RedisMessage.AsReader
func (r RedisResult) AsReader() (v io.Reader, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsReader()
	}
	return
}

// AsBytes delegates to RedisMessage.AsBytes
func (r RedisResult) AsBytes() (v []byte, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsBytes()
	}
	return
}

// DecodeJSON delegates to RedisMessage.DecodeJSON
func (r RedisResult) DecodeJSON(v any) (err error) {
	if r.err != nil {
		err = r.err
	} else {
		err = r.val.DecodeJSON(v)
	}
	return
}

// AsInt64 delegates to RedisMessage.AsInt64
func (r RedisResult) AsInt64() (v int64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsInt64()
	}
	return
}

// AsUint64 delegates to RedisMessage.AsUint64
func (r RedisResult) AsUint64() (v uint64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsUint64()
	}
	return
}

// AsBool delegates to RedisMessage.AsBool
func (r RedisResult) AsBool() (v bool, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsBool()
	}
	return
}

// AsFloat64 delegates to RedisMessage.AsFloat64
func (r RedisResult) AsFloat64() (v float64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsFloat64()
	}
	return
}

// ToArray delegates to RedisMessage.ToArray
func (r RedisResult) ToArray() (v []RedisMessage, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToArray()
	}
	return
}

// AsStrSlice delegates to RedisMessage.AsStrSlice
func (r RedisResult) AsStrSlice() (v []string, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsStrSlice()
	}
	return
}

// AsIntSlice delegates to RedisMessage.AsIntSlice
func (r RedisResult) AsIntSlice() (v []int64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsIntSlice()
	}
	return
}

// AsFloatSlice delegates to RedisMessage.AsFloatSlice
func (r RedisResult) AsFloatSlice() (v []float64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsFloatSlice()
	}
	return
}

// AsXRangeEntry delegates to RedisMessage.AsXRangeEntry
func (r RedisResult) AsXRangeEntry() (v XRangeEntry, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsXRangeEntry()
	}
	return
}

// AsXRange delegates to RedisMessage.AsXRange
func (r RedisResult) AsXRange() (v []XRangeEntry, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsXRange()
	}
	return
}

// AsZScore delegates to RedisMessage.AsZScore
func (r RedisResult) AsZScore() (v ZScore, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsZScore()
	}
	return
}

// AsZScores delegates to RedisMessage.AsZScores
func (r RedisResult) AsZScores() (v []ZScore, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsZScores()
	}
	return
}

// AsXRead delegates to RedisMessage.AsXRead
func (r RedisResult) AsXRead() (v map[string][]XRangeEntry, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsXRead()
	}
	return
}

func (r RedisResult) AsLMPop() (v KeyValues, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsLMPop()
	}
	return
}

func (r RedisResult) AsZMPop() (v KeyZScores, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsZMPop()
	}
	return
}

func (r RedisResult) AsFtSearch() (total int64, docs []FtSearchDoc, err error) {
	if r.err != nil {
		err = r.err
	} else {
		total, docs, err = r.val.AsFtSearch()
	}
	return
}

// AsMap delegates to RedisMessage.AsMap
func (r RedisResult) AsMap() (v map[string]RedisMessage, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsMap()
	}
	return
}

// AsStrMap delegates to RedisMessage.AsStrMap
func (r RedisResult) AsStrMap() (v map[string]string, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsStrMap()
	}
	return
}

// AsIntMap delegates to RedisMessage.AsIntMap
func (r RedisResult) AsIntMap() (v map[string]int64, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsIntMap()
	}
	return
}

// AsScanEntry delegates to RedisMessage.AsScanEntry.
func (r RedisResult) AsScanEntry() (v ScanEntry, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.AsScanEntry()
	}
	return
}

// ToMap delegates to RedisMessage.ToMap
func (r RedisResult) ToMap() (v map[string]RedisMessage, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToMap()
	}
	return
}

// ToAny delegates to RedisMessage.ToAny
func (r RedisResult) ToAny() (v any, err error) {
	if r.err != nil {
		err = r.err
	} else {
		v, err = r.val.ToAny()
	}
	return
}

// IsCacheHit delegates to RedisMessage.IsCacheHit
func (r RedisResult) IsCacheHit() bool {
	return r.val.IsCacheHit()
}

// CacheTTL delegates to RedisMessage.CacheTTL
func (r RedisResult) CacheTTL() int64 {
	return r.val.CacheTTL()
}

// RedisMessage is a redis response message, it may be a nil response
type RedisMessage struct {
	attrs   *RedisMessage
	string  string
	values  []RedisMessage
	integer int64
	typ     byte
	ttl     [7]byte
}

// IsNil check if message is a redis nil response
func (m *RedisMessage) IsNil() bool {
	return m.typ == '_'
}

// IsInt64 check if message is a redis RESP3 int response
func (m *RedisMessage) IsInt64() bool {
	return m.typ == ':'
}

// IsFloat64 check if message is a redis RESP3 double response
func (m *RedisMessage) IsFloat64() bool {
	return m.typ == ','
}

// IsString check if message is a redis string response
func (m *RedisMessage) IsString() bool {
	return m.typ == '$' || m.typ == '+'
}

// IsBool check if message is a redis RESP3 bool response
func (m *RedisMessage) IsBool() bool {
	return m.typ == '#'
}

// IsArray check if message is a redis array response
func (m *RedisMessage) IsArray() bool {
	return m.typ == '*' || m.typ == '~'
}

// IsMap check if message is a redis RESP3 map response
func (m *RedisMessage) IsMap() bool {
	return m.typ == '%'
}

// Error check if message is a redis error response, including nil response
func (m *RedisMessage) Error() error {
	if m.typ == '_' {
		return Nil
	}
	if m.typ == '-' || m.typ == '!' {
		// kvrocks: https://github.com/rueian/rueidis/issues/152#issuecomment-1333923750
		mm := *m
		mm.string = strings.TrimPrefix(m.string, "ERR ")
		return (*RedisError)(&mm)
	}
	return nil
}

// ToString check if message is a redis string response, and return it
func (m *RedisMessage) ToString() (val string, err error) {
	if m.IsString() {
		return m.string, nil
	}
	if m.IsInt64() || m.values != nil {
		typ := m.typ
		panic(fmt.Sprintf("redis message type %c is not a string", typ))
	}
	return m.string, m.Error()
}

// AsReader check if message is a redis string response and wrap it with the strings.NewReader
func (m *RedisMessage) AsReader() (reader io.Reader, err error) {
	str, err := m.ToString()
	if err != nil {
		return nil, err
	}
	return strings.NewReader(str), nil
}

// AsBytes check if message is a redis string response and return it as an immutable []byte
func (m *RedisMessage) AsBytes() (bs []byte, err error) {
	str, err := m.ToString()
	if err != nil {
		return nil, err
	}
	return unsafe.Slice(unsafe.StringData(str), len(str)), nil
}

// DecodeJSON check if message is a redis string response and treat it as json, then unmarshal it into provided value
func (m *RedisMessage) DecodeJSON(v any) (err error) {
	str, err := m.ToString()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(str))
	return decoder.Decode(v)
}

// AsInt64 check if message is a redis string response, and parse it as int64
func (m *RedisMessage) AsInt64() (val int64, err error) {
	if m.IsInt64() {
		return m.integer, nil
	}
	v, err := m.ToString()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

// AsUint64 check if message is a redis string response, and parse it as uint64
func (m *RedisMessage) AsUint64() (val uint64, err error) {
	if m.IsInt64() {
		return uint64(m.integer), nil
	}
	v, err := m.ToString()
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(v, 10, 64)
}

// AsBool checks if message is non-nil redis response, and parses it as bool
func (m *RedisMessage) AsBool() (val bool, err error) {
	if err = m.Error(); err != nil {
		return
	}
	switch m.typ {
	case '$', '+':
		val = m.string == "OK"
		return
	case ':':
		val = m.integer != 0
		return
	case '#':
		val = m.integer == 1
		return
	default:
		typ := m.typ
		panic(fmt.Sprintf("redis message type %c is not a int, string or bool", typ))
	}
}

// AsFloat64 check if message is a redis string response, and parse it as float64
func (m *RedisMessage) AsFloat64() (val float64, err error) {
	if m.IsFloat64() {
		return util.ToFloat64(m.string)
	}
	v, err := m.ToString()
	if err != nil {
		return 0, err
	}
	return util.ToFloat64(v)
}

// ToInt64 check if message is a redis RESP3 int response, and return it
func (m *RedisMessage) ToInt64() (val int64, err error) {
	if m.IsInt64() {
		return m.integer, nil
	}
	if err = m.Error(); err != nil {
		return 0, err
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a RESP3 int64", typ))
}

// ToBool check if message is a redis RESP3 bool response, and return it
func (m *RedisMessage) ToBool() (val bool, err error) {
	if m.IsBool() {
		return m.integer == 1, nil
	}
	if err = m.Error(); err != nil {
		return false, err
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a RESP3 bool", typ))
}

// ToFloat64 check if message is a redis RESP3 double response, and return it
func (m *RedisMessage) ToFloat64() (val float64, err error) {
	if m.IsFloat64() {
		return util.ToFloat64(m.string)
	}
	if err = m.Error(); err != nil {
		return 0, err
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a RESP3 float64", typ))
}

// ToArray check if message is a redis array/set response, and return it
func (m *RedisMessage) ToArray() ([]RedisMessage, error) {
	if m.IsArray() {
		return m.values, nil
	}
	if err := m.Error(); err != nil {
		return nil, err
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a array", typ))
}

// AsStrSlice check if message is a redis array/set response, and convert to []string.
// redis nil element and other non string element will be present as zero.
func (m *RedisMessage) AsStrSlice() ([]string, error) {
	values, err := m.ToArray()
	if err != nil {
		return nil, err
	}
	s := make([]string, 0, len(values))
	for _, v := range values {
		s = append(s, v.string)
	}
	return s, nil
}

// AsIntSlice check if message is a redis array/set response, and convert to []int64.
// redis nil element and other non integer element will be present as zero.
func (m *RedisMessage) AsIntSlice() ([]int64, error) {
	values, err := m.ToArray()
	if err != nil {
		return nil, err
	}
	s := make([]int64, 0, len(values))
	for _, v := range values {
		s = append(s, v.integer)
	}
	return s, nil
}

// AsFloatSlice check if message is a redis array/set response, and convert to []float64.
// redis nil element and other non float element will be present as zero.
func (m *RedisMessage) AsFloatSlice() ([]float64, error) {
	values, err := m.ToArray()
	if err != nil {
		return nil, err
	}
	s := make([]float64, 0, len(values))
	for _, v := range values {
		if len(v.string) != 0 {
			i, err := util.ToFloat64(v.string)
			if err != nil {
				return nil, err
			}
			s = append(s, i)
		} else {
			s = append(s, float64(v.integer))
		}
	}
	return s, nil
}

// XRangeEntry is the element type of both XRANGE and XREVRANGE command response array
type XRangeEntry struct {
	FieldValues map[string]string
	ID          string
}

// AsXRangeEntry check if message is a redis array/set response of length 2, and convert to XRangeEntry
func (m *RedisMessage) AsXRangeEntry() (XRangeEntry, error) {
	values, err := m.ToArray()
	if err != nil {
		return XRangeEntry{}, err
	}
	if len(values) != 2 {
		return XRangeEntry{}, fmt.Errorf("got %d, wanted 2", len(values))
	}
	id, err := values[0].ToString()
	if err != nil {
		return XRangeEntry{}, err
	}
	fieldValues, err := values[1].AsStrMap()
	if err != nil {
		if IsRedisNil(err) {
			return XRangeEntry{ID: id, FieldValues: nil}, nil
		}
		return XRangeEntry{}, err
	}
	return XRangeEntry{
		ID:          id,
		FieldValues: fieldValues,
	}, nil
}

// AsXRange check if message is a redis array/set response, and convert to []XRangeEntry
func (m *RedisMessage) AsXRange() ([]XRangeEntry, error) {
	values, err := m.ToArray()
	if err != nil {
		return nil, err
	}
	msgs := make([]XRangeEntry, 0, len(values))
	for _, v := range values {
		msg, err := v.AsXRangeEntry()
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// AsXRead converts XREAD/XREADGRUOP response to map[string][]XRangeEntry
func (m *RedisMessage) AsXRead() (ret map[string][]XRangeEntry, err error) {
	if err = m.Error(); err != nil {
		return nil, err
	}
	if m.IsMap() {
		ret = make(map[string][]XRangeEntry, len(m.values)/2)
		for i := 0; i < len(m.values); i += 2 {
			if ret[m.values[i].string], err = m.values[i+1].AsXRange(); err != nil {
				return nil, err
			}
		}
		return ret, nil
	}
	if m.IsArray() {
		ret = make(map[string][]XRangeEntry, len(m.values))
		for _, v := range m.values {
			if !v.IsArray() || len(v.values) != 2 {
				return nil, fmt.Errorf("got %d, wanted 2", len(v.values))
			}
			if ret[v.values[0].string], err = v.values[1].AsXRange(); err != nil {
				return nil, err
			}
		}
		return ret, nil
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a map/array/set or its length is not even", typ))
}

// ZScore is the element type of ZRANGE WITHSCORES, ZDIFF WITHSCORES and ZPOPMAX command response
type ZScore struct {
	Member string
	Score  float64
}

func toZScore(values []RedisMessage) (s ZScore, err error) {
	if len(values) == 2 {
		if s.Member, err = values[0].ToString(); err == nil {
			s.Score, err = values[1].AsFloat64()
		}
		return s, err
	}
	panic(fmt.Sprintf("redis message is not a map/array/set or its length is not 2"))
}

// AsZScore converts ZPOPMAX and ZPOPMIN command with count 1 response to a single ZScore
func (m *RedisMessage) AsZScore() (s ZScore, err error) {
	arr, err := m.ToArray()
	if err != nil {
		return s, err
	}
	return toZScore(arr)
}

// AsZScores converts ZRANGE WITHSCROES, ZDIFF WITHSCROES and ZPOPMAX/ZPOPMIN command with count > 1 responses to []ZScore
func (m *RedisMessage) AsZScores() ([]ZScore, error) {
	arr, err := m.ToArray()
	if err != nil {
		return nil, err
	}
	if len(arr) > 0 && arr[0].IsArray() {
		scores := make([]ZScore, len(arr))
		for i, v := range arr {
			if scores[i], err = toZScore(v.values); err != nil {
				return nil, err
			}
		}
		return scores, nil
	}
	scores := make([]ZScore, len(arr)/2)
	for i := 0; i < len(scores); i++ {
		j := i * 2
		if scores[i], err = toZScore(arr[j : j+2]); err != nil {
			return nil, err
		}
	}
	return scores, nil
}

// ScanEntry is the element type of both SCAN, SSCAN, HSCAN and ZSCAN command response.
type ScanEntry struct {
	Cursor   uint64
	Elements []string
}

// AsScanEntry check if message is a redis array/set response of length 2, and convert to ScanEntry.
func (m *RedisMessage) AsScanEntry() (e ScanEntry, err error) {
	msgs, err := m.ToArray()
	if err != nil {
		return ScanEntry{}, err
	}
	if len(msgs) >= 2 {
		if e.Cursor, err = msgs[0].AsUint64(); err == nil {
			e.Elements, err = msgs[1].AsStrSlice()
		}
		return e, err
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a scan response or its length is not at least 2", typ))
}

// AsMap check if message is a redis array/set response, and convert to map[string]RedisMessage
func (m *RedisMessage) AsMap() (map[string]RedisMessage, error) {
	if err := m.Error(); err != nil {
		return nil, err
	}
	if (m.IsMap() || m.IsArray()) && len(m.values)%2 == 0 {
		return toMap(m.values), nil
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a map/array/set or its length is not even", typ))
}

// AsStrMap check if message is a redis map/array/set response, and convert to map[string]string.
// redis nil element and other non string element will be present as zero.
func (m *RedisMessage) AsStrMap() (map[string]string, error) {
	if err := m.Error(); err != nil {
		return nil, err
	}
	if (m.IsMap() || m.IsArray()) && len(m.values)%2 == 0 {
		r := make(map[string]string, len(m.values)/2)
		for i := 0; i < len(m.values); i += 2 {
			k := m.values[i]
			v := m.values[i+1]
			r[k.string] = v.string
		}
		return r, nil
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a map/array/set or its length is not even", typ))
}

// AsIntMap check if message is a redis map/array/set response, and convert to map[string]int64.
// redis nil element and other non integer element will be present as zero.
func (m *RedisMessage) AsIntMap() (map[string]int64, error) {
	if err := m.Error(); err != nil {
		return nil, err
	}
	if (m.IsMap() || m.IsArray()) && len(m.values)%2 == 0 {
		var err error
		r := make(map[string]int64, len(m.values)/2)
		for i := 0; i < len(m.values); i += 2 {
			k := m.values[i]
			v := m.values[i+1]
			if k.typ == '$' || k.typ == '+' {
				if len(v.string) != 0 {
					if r[k.string], err = strconv.ParseInt(v.string, 0, 64); err != nil {
						return nil, err
					}
				} else if v.typ == ':' || v.typ == '_' {
					r[k.string] = v.integer
				}
			}
		}
		return r, nil
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a map/array/set or its length is not even", typ))
}

type KeyValues struct {
	Key    string
	Values []string
}

func (m *RedisMessage) AsLMPop() (kvs KeyValues, err error) {
	if err = m.Error(); err != nil {
		return KeyValues{}, err
	}
	if len(m.values) >= 2 {
		kvs.Key = m.values[0].string
		kvs.Values, err = m.values[1].AsStrSlice()
		return
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a LMPOP response", typ))
}

type KeyZScores struct {
	Key    string
	Values []ZScore
}

func (m *RedisMessage) AsZMPop() (kvs KeyZScores, err error) {
	if err = m.Error(); err != nil {
		return KeyZScores{}, err
	}
	if len(m.values) >= 2 {
		kvs.Key = m.values[0].string
		kvs.Values, err = m.values[1].AsZScores()
		return
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a ZMPOP response", typ))
}

type FtSearchDoc struct {
	Doc map[string]string
	Key string
}

func (m *RedisMessage) AsFtSearch() (total int64, docs []FtSearchDoc, err error) {
	if err = m.Error(); err != nil {
		return 0, nil, err
	}
	if len(m.values) > 0 {
		total = m.values[0].integer
		if len(m.values) > 2 && m.values[2].string == "" {
			docs = make([]FtSearchDoc, 0, (len(m.values)-1)/2)
			for i := 1; i < len(m.values); i += 2 {
				doc, _ := m.values[i+1].AsStrMap()
				docs = append(docs, FtSearchDoc{Doc: doc, Key: m.values[i].string})
			}
		} else {
			docs = make([]FtSearchDoc, 0, len(m.values)-1)
			for i := 1; i < len(m.values); i++ {
				docs = append(docs, FtSearchDoc{Doc: nil, Key: m.values[i].string})
			}
		}
		return
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a FT.SEARCH response", typ))
}

// ToMap check if message is a redis RESP3 map response, and return it
func (m *RedisMessage) ToMap() (map[string]RedisMessage, error) {
	if m.IsMap() {
		return toMap(m.values), nil
	}
	if err := m.Error(); err != nil {
		return nil, err
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a RESP3 map", typ))
}

// ToAny turns message into go any value
func (m *RedisMessage) ToAny() (any, error) {
	if err := m.Error(); err != nil {
		return nil, err
	}
	switch m.typ {
	case ',':
		return util.ToFloat64(m.string)
	case '$', '+', '=', '(':
		return m.string, nil
	case '#':
		return m.integer == 1, nil
	case ':':
		return m.integer, nil
	case '%':
		vs := make(map[string]any, len(m.values)/2)
		for i := 0; i < len(m.values); i += 2 {
			if v, err := m.values[i+1].ToAny(); err != nil && !IsRedisNil(err) {
				vs[m.values[i].string] = err
			} else {
				vs[m.values[i].string] = v
			}
		}
		return vs, nil
	case '~', '*':
		vs := make([]any, len(m.values))
		for i := 0; i < len(m.values); i++ {
			if v, err := m.values[i].ToAny(); err != nil && !IsRedisNil(err) {
				vs[i] = err
			} else {
				vs[i] = v
			}
		}
		return vs, nil
	}
	typ := m.typ
	panic(fmt.Sprintf("redis message type %c is not a supported in ToAny", typ))
}

// IsCacheHit check if message is from client side cache
func (m *RedisMessage) IsCacheHit() bool {
	return m.attrs == cacheMark
}

// CacheTTL returns the remaining TTL in seconds of client side cache
func (m *RedisMessage) CacheTTL() int64 {
	unix := int64(m.ttl[0]) | int64(m.ttl[1])<<8 | int64(m.ttl[2])<<16 | int64(m.ttl[3])<<24 |
		int64(m.ttl[4])<<32 | int64(m.ttl[5])<<40 | int64(m.ttl[6])<<48
	if unix > 0 {
		return unix - time.Now().Unix()
	}
	return 0
}

func (m *RedisMessage) setTTL(pttl int64) {
	m.ttl[0] = byte(pttl)
	m.ttl[1] = byte(pttl >> 8)
	m.ttl[2] = byte(pttl >> 16)
	m.ttl[3] = byte(pttl >> 24)
	m.ttl[4] = byte(pttl >> 32)
	m.ttl[5] = byte(pttl >> 40)
	m.ttl[6] = byte(pttl >> 48)
}

func toMap(values []RedisMessage) map[string]RedisMessage {
	r := make(map[string]RedisMessage, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		if values[i].typ == '$' || values[i].typ == '+' {
			r[values[i].string] = values[i+1]
			continue
		}
		typ := values[i].typ
		panic(fmt.Sprintf("redis message type %c as map key is not supported", typ))
	}
	return r
}

func (m *RedisMessage) approximateSize() (s int) {
	s += messageStructSize
	s += len(m.string)
	for _, v := range m.values {
		s += v.approximateSize()
	}
	return
}
