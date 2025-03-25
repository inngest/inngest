// Copyright (c) 2013 The github.com/go-redis/redis Authors.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package rueidiscompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/internal/util"
)

type Cmder interface {
	SetErr(error)
	Err() error
	from(result rueidis.RedisResult)
}

type baseCmd[T any] struct {
	err    error
	val    T
	rawVal any
}

func (cmd *baseCmd[T]) SetVal(val T) {
	cmd.val = val
}

func (cmd *baseCmd[T]) Val() T {
	return cmd.val
}

func (cmd *baseCmd[T]) SetRawVal(rawVal any) {
	cmd.rawVal = rawVal
}

func (cmd *baseCmd[T]) RawVal() any {
	return cmd.rawVal
}

func (cmd *baseCmd[T]) SetErr(err error) {
	cmd.err = err
}

func (cmd *baseCmd[T]) Err() error {
	return cmd.err
}

func (cmd *baseCmd[T]) Result() (T, error) {
	return cmd.Val(), cmd.Err()
}

func (cmd *baseCmd[T]) RawResult() (any, error) {
	return cmd.RawVal(), cmd.Err()
}

type Cmd struct {
	baseCmd[any]
}

func (cmd *Cmd) from(res rueidis.RedisResult) {
	val, err := res.ToAny()
	if err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(val)
}

func newCmd(res rueidis.RedisResult) *Cmd {
	cmd := &Cmd{}
	cmd.from(res)
	return cmd
}

func (cmd *Cmd) Text() (string, error) {
	if cmd.err != nil {
		return "", cmd.err
	}
	return toString(cmd.val)
}

func toString(val any) (string, error) {
	switch val := val.(type) {
	case string:
		return val, nil
	default:
		err := fmt.Errorf("redis: unexpected type=%T for String", val)
		return "", err
	}
}

func (cmd *Cmd) Int() (int, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	switch val := cmd.val.(type) {
	case int64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	default:
		err := fmt.Errorf("redis: unexpected type=%T for Int", val)
		return 0, err
	}
}

func (cmd *Cmd) Int64() (int64, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return toInt64(cmd.val)
}

func toInt64(val any) (int64, error) {
	switch val := val.(type) {
	case int64:
		return val, nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		err := fmt.Errorf("redis: unexpected type=%T for Int64", val)
		return 0, err
	}
}

func (cmd *Cmd) Uint64() (uint64, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return toUint64(cmd.val)
}

func toUint64(val any) (uint64, error) {
	switch val := val.(type) {
	case int64:
		return uint64(val), nil
	case string:
		return strconv.ParseUint(val, 10, 64)
	default:
		err := fmt.Errorf("redis: unexpected type=%T for Uint64", val)
		return 0, err
	}
}

func (cmd *Cmd) Float32() (float32, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return toFloat32(cmd.val)
}

func toFloat32(val any) (float32, error) {
	switch val := val.(type) {
	case int64:
		return float32(val), nil
	case string:
		f, err := util.ToFloat32(val)
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		err := fmt.Errorf("redis: unexpected type=%T for Float32", val)
		return 0, err
	}
}

func (cmd *Cmd) Float64() (float64, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return toFloat64(cmd.val)
}

func toFloat64(val any) (float64, error) {
	switch val := val.(type) {
	case int64:
		return float64(val), nil
	case string:
		return util.ToFloat64(val)
	default:
		err := fmt.Errorf("redis: unexpected type=%T for Float64", val)
		return 0, err
	}
}

func (cmd *Cmd) Bool() (bool, error) {
	if cmd.err != nil {
		return false, cmd.err
	}
	return toBool(cmd.val)
}

func toBool(val any) (bool, error) {
	switch val := val.(type) {
	case int64:
		return val != 0, nil
	case string:
		return strconv.ParseBool(val)
	default:
		err := fmt.Errorf("redis: unexpected type=%T for Bool", val)
		return false, err
	}
}

func (cmd *Cmd) Slice() ([]any, error) {
	if cmd.err != nil {
		return nil, cmd.err
	}
	switch val := cmd.val.(type) {
	case []any:
		return val, nil
	default:
		return nil, fmt.Errorf("redis: unexpected type=%T for Slice", val)
	}
}

func (cmd *Cmd) StringSlice() ([]string, error) {
	slice, err := cmd.Slice()
	if err != nil {
		return nil, err
	}

	ss := make([]string, len(slice))
	for i, iface := range slice {
		val, err := toString(iface)
		if err != nil {
			return nil, err
		}
		ss[i] = val
	}
	return ss, nil
}

func (cmd *Cmd) Int64Slice() ([]int64, error) {
	slice, err := cmd.Slice()
	if err != nil {
		return nil, err
	}

	nums := make([]int64, len(slice))
	for i, iface := range slice {
		val, err := toInt64(iface)
		if err != nil {
			return nil, err
		}
		nums[i] = val
	}
	return nums, nil
}

func (cmd *Cmd) Uint64Slice() ([]uint64, error) {
	slice, err := cmd.Slice()
	if err != nil {
		return nil, err
	}

	nums := make([]uint64, len(slice))
	for i, iface := range slice {
		val, err := toUint64(iface)
		if err != nil {
			return nil, err
		}
		nums[i] = val
	}
	return nums, nil
}

func (cmd *Cmd) Float32Slice() ([]float32, error) {
	slice, err := cmd.Slice()
	if err != nil {
		return nil, err
	}

	floats := make([]float32, len(slice))
	for i, iface := range slice {
		val, err := toFloat32(iface)
		if err != nil {
			return nil, err
		}
		floats[i] = val
	}
	return floats, nil
}

func (cmd *Cmd) Float64Slice() ([]float64, error) {
	slice, err := cmd.Slice()
	if err != nil {
		return nil, err
	}

	floats := make([]float64, len(slice))
	for i, iface := range slice {
		val, err := toFloat64(iface)
		if err != nil {
			return nil, err
		}
		floats[i] = val
	}
	return floats, nil
}

func (cmd *Cmd) BoolSlice() ([]bool, error) {
	slice, err := cmd.Slice()
	if err != nil {
		return nil, err
	}

	bools := make([]bool, len(slice))
	for i, iface := range slice {
		val, err := toBool(iface)
		if err != nil {
			return nil, err
		}
		bools[i] = val
	}
	return bools, nil
}

type StringCmd struct {
	baseCmd[string]
}

func (cmd *StringCmd) from(res rueidis.RedisResult) {
	val, err := res.ToString()
	cmd.SetErr(err)
	cmd.SetVal(val)
}

func newStringCmd(res rueidis.RedisResult) *StringCmd {
	cmd := &StringCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *StringCmd) Bytes() ([]byte, error) {
	return []byte(cmd.val), cmd.err
}

func (cmd *StringCmd) Bool() (bool, error) {
	return cmd.val != "", cmd.err
}

func (cmd *StringCmd) Int() (int, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return strconv.Atoi(cmd.Val())
}

func (cmd *StringCmd) Int64() (int64, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return strconv.ParseInt(cmd.Val(), 10, 64)
}

func (cmd *StringCmd) Uint64() (uint64, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return strconv.ParseUint(cmd.Val(), 10, 64)
}

func (cmd *StringCmd) Float32() (float32, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	v, err := util.ToFloat32(cmd.Val())
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (cmd *StringCmd) Float64() (float64, error) {
	if cmd.err != nil {
		return 0, cmd.err
	}
	return util.ToFloat64(cmd.Val())
}

func (cmd *StringCmd) Time() (time.Time, error) {
	if cmd.err != nil {
		return time.Time{}, cmd.err
	}
	return time.Parse(time.RFC3339Nano, cmd.Val())
}

func (cmd *StringCmd) String() string {
	return cmd.val
}

type BoolCmd struct {
	baseCmd[bool]
}

func (cmd *BoolCmd) from(res rueidis.RedisResult) {
	val, err := res.AsBool()
	if rueidis.IsRedisNil(err) {
		val = false
		err = nil
	}
	cmd.SetVal(val)
	cmd.SetErr(err)
}

func newBoolCmd(res rueidis.RedisResult) *BoolCmd {
	cmd := &BoolCmd{}
	cmd.from(res)
	return cmd
}

type IntCmd struct {
	baseCmd[int64]
}

func (cmd *IntCmd) from(res rueidis.RedisResult) {
	val, err := res.AsInt64()
	cmd.SetErr(err)
	cmd.SetVal(val)
}

func newIntCmd(res rueidis.RedisResult) *IntCmd {
	cmd := &IntCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *IntCmd) Uint64() (uint64, error) {
	return uint64(cmd.val), cmd.err
}

type DurationCmd struct {
	baseCmd[time.Duration]
	precision time.Duration
}

func (cmd *DurationCmd) from(res rueidis.RedisResult) {
	val, err := res.AsInt64()
	cmd.SetErr(err)
	if val > 0 {
		cmd.SetVal(time.Duration(val) * cmd.precision)
		return
	}
	cmd.SetVal(time.Duration(val))
}

func newDurationCmd(res rueidis.RedisResult, precision time.Duration) *DurationCmd {
	cmd := &DurationCmd{precision: precision}
	cmd.from(res)
	return cmd
}

type StatusCmd = StringCmd

func newStatusCmd(res rueidis.RedisResult) *StatusCmd {
	cmd := &StatusCmd{}
	val, err := res.ToString()
	cmd.SetErr(err)
	cmd.SetVal(val)
	return cmd
}

type SliceCmd struct {
	baseCmd[[]any]
	keys []string
	json bool
}

func (cmd *SliceCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	vals := make([]any, len(arr))
	if cmd.json {
		for i, v := range arr {
			// for JSON.OBJKEYS
			if v.IsNil() {
				continue
			}
			// convert to any which underlying type is []any
			arr, err := v.ToAny()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			vals[i] = arr
		}
		cmd.SetVal(vals)
		return
	}
	for i, v := range arr {
		// keep the old behavior the same as before (don't handle error while parsing v as string)
		if s, err := v.ToString(); err == nil {
			vals[i] = s
		}
	}
	cmd.SetVal(vals)
}

// newSliceCmd returns SliceCmd according to input arguments, if the caller is JSONObjKeys,
// set isJSONObjKeys to true.
func newSliceCmd(res rueidis.RedisResult, isJSONObjKeys bool, keys ...string) *SliceCmd {
	cmd := &SliceCmd{keys: keys, json: isJSONObjKeys}
	cmd.from(res)
	return cmd
}

// Scan scans the results from the map into a destination struct. The map keys
// are matched in the Redis struct fields by the `redis:"field"` tag.
// NOTE: result from JSON.ObjKeys should not call this.
func (cmd *SliceCmd) Scan(dst any) error {
	if cmd.err != nil {
		return cmd.err
	}
	return Scan(dst, cmd.keys, cmd.val)
}

type StringSliceCmd struct {
	baseCmd[[]string]
}

func (cmd *StringSliceCmd) from(res rueidis.RedisResult) {
	val, err := res.AsStrSlice()
	cmd.SetVal(val)
	cmd.SetErr(err)
}

func newStringSliceCmd(res rueidis.RedisResult) *StringSliceCmd {
	cmd := &StringSliceCmd{}
	cmd.from(res)
	return cmd
}

type IntSliceCmd struct {
	err error
	val []int64
}

func (cmd *IntSliceCmd) from(res rueidis.RedisResult) {
	cmd.val, cmd.err = res.AsIntSlice()

}

func newIntSliceCmd(res rueidis.RedisResult) *IntSliceCmd {
	cmd := &IntSliceCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *IntSliceCmd) SetVal(val []int64) {
	cmd.val = val
}

func (cmd *IntSliceCmd) SetErr(err error) {
	cmd.err = err
}

func (cmd *IntSliceCmd) Val() []int64 {
	return cmd.val
}

func (cmd *IntSliceCmd) Err() error {
	return cmd.err
}

func (cmd *IntSliceCmd) Result() ([]int64, error) {
	return cmd.val, cmd.err
}

type BoolSliceCmd struct {
	baseCmd[[]bool]
}

func (cmd *BoolSliceCmd) from(res rueidis.RedisResult) {
	ints, err := res.AsIntSlice()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]bool, 0, len(ints))
	for _, i := range ints {
		val = append(val, i == 1)
	}
	cmd.SetVal(val)
}

func newBoolSliceCmd(res rueidis.RedisResult) *BoolSliceCmd {
	cmd := &BoolSliceCmd{}
	cmd.from(res)
	return cmd
}

type FloatSliceCmd struct {
	baseCmd[[]float64]
}

func (cmd *FloatSliceCmd) from(res rueidis.RedisResult) {
	val, err := res.AsFloatSlice()
	cmd.SetErr(err)
	cmd.SetVal(val)
}

func newFloatSliceCmd(res rueidis.RedisResult) *FloatSliceCmd {
	cmd := &FloatSliceCmd{}
	cmd.from(res)
	return cmd
}

type ZSliceCmd struct {
	baseCmd[[]Z]
	single bool
}

func (cmd *ZSliceCmd) from(res rueidis.RedisResult) {
	if cmd.single {
		s, err := res.AsZScore()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		cmd.SetVal([]Z{{Member: s.Member, Score: s.Score}})
	} else {
		scores, err := res.AsZScores()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		val := make([]Z, 0, len(scores))
		for _, s := range scores {
			val = append(val, Z{Member: s.Member, Score: s.Score})
		}
		cmd.SetVal(val)
	}
}

func newZSliceCmd(res rueidis.RedisResult) *ZSliceCmd {
	cmd := &ZSliceCmd{}
	cmd.from(res)
	return cmd
}

func newZSliceSingleCmd(res rueidis.RedisResult) *ZSliceCmd {
	cmd := &ZSliceCmd{single: true}
	cmd.from(res)
	return cmd
}

type FloatCmd struct {
	baseCmd[float64]
}

func (cmd *FloatCmd) from(res rueidis.RedisResult) {
	val, err := res.AsFloat64()
	cmd.SetErr(err)
	cmd.SetVal(val)
}

func newFloatCmd(res rueidis.RedisResult) *FloatCmd {
	cmd := &FloatCmd{}
	cmd.from(res)
	return cmd
}

type ScanCmd struct {
	err    error
	keys   []string
	cursor uint64
}

func (cmd *ScanCmd) from(res rueidis.RedisResult) {
	e, err := res.AsScanEntry()
	if err != nil {
		cmd.err = err
		return
	}
	cmd.cursor, cmd.keys = e.Cursor, e.Elements
}

func newScanCmd(res rueidis.RedisResult) *ScanCmd {
	cmd := &ScanCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *ScanCmd) SetVal(keys []string, cursor uint64) {
	cmd.keys = keys
	cmd.cursor = cursor
}

func (cmd *ScanCmd) Val() (keys []string, cursor uint64) {
	return cmd.keys, cmd.cursor
}

func (cmd *ScanCmd) SetErr(err error) {
	cmd.err = err
}

func (cmd *ScanCmd) Err() error {
	return cmd.err
}

func (cmd *ScanCmd) Result() (keys []string, cursor uint64, err error) {
	return cmd.keys, cmd.cursor, cmd.err
}

type KeyValue struct {
	Key   string
	Value string
}

type KeyValueSliceCmd struct {
	baseCmd[[]KeyValue]
}

func (cmd *KeyValueSliceCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	for _, a := range arr {
		kv, _ := a.AsStrSlice()
		for i := 0; i < len(kv); i += 2 {
			cmd.val = append(cmd.val, KeyValue{Key: kv[i], Value: kv[i+1]})
		}
	}
	cmd.SetErr(err)
}

func newKeyValueSliceCmd(res rueidis.RedisResult) *KeyValueSliceCmd {
	cmd := &KeyValueSliceCmd{}
	cmd.from(res)
	return cmd
}

type KeyValuesCmd struct {
	err error
	val rueidis.KeyValues
}

func (cmd *KeyValuesCmd) from(res rueidis.RedisResult) {
	cmd.val, cmd.err = res.AsLMPop()
}

func newKeyValuesCmd(res rueidis.RedisResult) *KeyValuesCmd {
	cmd := &KeyValuesCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *KeyValuesCmd) SetVal(key string, val []string) {
	cmd.val.Key = key
	cmd.val.Values = val
}

func (cmd *KeyValuesCmd) SetErr(err error) {
	cmd.err = err
}

func (cmd *KeyValuesCmd) Val() (string, []string) {
	return cmd.val.Key, cmd.val.Values
}

func (cmd *KeyValuesCmd) Err() error {
	return cmd.err
}

func (cmd *KeyValuesCmd) Result() (string, []string, error) {
	return cmd.val.Key, cmd.val.Values, cmd.err
}

type KeyFlags struct {
	Key   string
	Flags []string
}

type KeyFlagsCmd struct {
	baseCmd[[]KeyFlags]
}

func (cmd *KeyFlagsCmd) from(res rueidis.RedisResult) {
	if cmd.err = res.Error(); cmd.err == nil {
		kfs, _ := res.ToArray()
		cmd.val = make([]KeyFlags, len(kfs))
		for i := 0; i < len(kfs); i++ {
			if kf, _ := kfs[i].ToArray(); len(kf) >= 2 {
				cmd.val[i].Key, _ = kf[0].ToString()
				cmd.val[i].Flags, _ = kf[1].AsStrSlice()
			}
		}
	}
}

func newKeyFlagsCmd(res rueidis.RedisResult) *KeyFlagsCmd {
	cmd := &KeyFlagsCmd{}
	cmd.from(res)
	return cmd
}

type ZSliceWithKeyCmd struct {
	err error
	key string
	val []Z
}

func (cmd *ZSliceWithKeyCmd) from(res rueidis.RedisResult) {
	v, err := res.AsZMPop()
	if err != nil {
		cmd.err = err
		return
	}
	val := make([]Z, 0, len(v.Values))
	for _, s := range v.Values {
		val = append(val, Z{Member: s.Member, Score: s.Score})
	}
	cmd.key, cmd.val = v.Key, val
}

func newZSliceWithKeyCmd(res rueidis.RedisResult) *ZSliceWithKeyCmd {
	cmd := &ZSliceWithKeyCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *ZSliceWithKeyCmd) SetVal(key string, val []Z) {
	cmd.key = key
	cmd.val = val
}

func (cmd *ZSliceWithKeyCmd) SetErr(err error) {
	cmd.err = err
}

func (cmd *ZSliceWithKeyCmd) Val() (string, []Z) {
	return cmd.key, cmd.val
}

func (cmd *ZSliceWithKeyCmd) Err() error {
	return cmd.err
}

func (cmd *ZSliceWithKeyCmd) Result() (string, []Z, error) {
	return cmd.key, cmd.val, cmd.err
}

type StringStringMapCmd struct {
	baseCmd[map[string]string]
}

func (cmd *StringStringMapCmd) from(res rueidis.RedisResult) {
	val, err := res.AsStrMap()
	cmd.SetErr(err)
	cmd.SetVal(val)
}

func newStringStringMapCmd(res rueidis.RedisResult) *StringStringMapCmd {
	cmd := &StringStringMapCmd{}
	cmd.from(res)
	return cmd
}

// Scan scans the results from the map into a destination struct. The map keys
// are matched in the Redis struct fields by the `redis:"field"` tag.
func (cmd *StringStringMapCmd) Scan(dest interface{}) error {
	if cmd.Err() != nil {
		return cmd.Err()
	}

	strct, err := Struct(dest)
	if err != nil {
		return err
	}

	for k, v := range cmd.val {
		if err := strct.Scan(k, v); err != nil {
			return err
		}
	}

	return nil
}

type StringIntMapCmd struct {
	baseCmd[map[string]int64]
}

func (cmd *StringIntMapCmd) from(res rueidis.RedisResult) {
	val, err := res.AsIntMap()
	cmd.SetErr(err)
	cmd.SetVal(val)
}

func newStringIntMapCmd(res rueidis.RedisResult) *StringIntMapCmd {
	cmd := &StringIntMapCmd{}
	cmd.from(res)
	return cmd
}

type StringStructMapCmd struct {
	baseCmd[map[string]struct{}]
}

func (cmd *StringStructMapCmd) from(res rueidis.RedisResult) {
	strSlice, err := res.AsStrSlice()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make(map[string]struct{}, len(strSlice))
	for _, v := range strSlice {
		val[v] = struct{}{}
	}
	cmd.SetVal(val)
}

func newStringStructMapCmd(res rueidis.RedisResult) *StringStructMapCmd {
	cmd := &StringStructMapCmd{}
	cmd.from(res)
	return cmd
}

type XMessageSliceCmd struct {
	baseCmd[[]XMessage]
}

func (cmd *XMessageSliceCmd) from(res rueidis.RedisResult) {
	val, err := res.AsXRange()
	cmd.SetErr(err)
	cmd.val = make([]XMessage, len(val))
	for i, r := range val {
		cmd.val[i] = newXMessage(r)
	}
}

func newXMessageSliceCmd(res rueidis.RedisResult) *XMessageSliceCmd {
	cmd := &XMessageSliceCmd{}
	cmd.from(res)
	return cmd
}

func newXMessage(r rueidis.XRangeEntry) XMessage {
	if r.FieldValues == nil {
		return XMessage{ID: r.ID, Values: nil}
	}
	m := XMessage{ID: r.ID, Values: make(map[string]any, len(r.FieldValues))}
	for k, v := range r.FieldValues {
		m.Values[k] = v
	}
	return m
}

type XStream struct {
	Stream   string
	Messages []XMessage
}

type XStreamSliceCmd struct {
	baseCmd[[]XStream]
}

func (cmd *XStreamSliceCmd) from(res rueidis.RedisResult) {
	streams, err := res.AsXRead()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]XStream, 0, len(streams))
	for name, messages := range streams {
		msgs := make([]XMessage, 0, len(messages))
		for _, r := range messages {
			msgs = append(msgs, newXMessage(r))
		}
		val = append(val, XStream{Stream: name, Messages: msgs})
	}
	cmd.SetVal(val)
}

func newXStreamSliceCmd(res rueidis.RedisResult) *XStreamSliceCmd {
	cmd := &XStreamSliceCmd{}
	cmd.from(res)
	return cmd
}

type XPending struct {
	Consumers map[string]int64
	Lower     string
	Higher    string
	Count     int64
}

type XPendingCmd struct {
	baseCmd[XPending]
}

func (cmd *XPendingCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if len(arr) < 4 {
		cmd.SetErr(fmt.Errorf("got %d, wanted 4", len(arr)))
		return
	}
	count, err := arr[0].AsInt64()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	lower, err := arr[1].ToString()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	higher, err := arr[2].ToString()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := XPending{
		Count:  count,
		Lower:  lower,
		Higher: higher,
	}
	consumerArr, err := arr[3].ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	for _, v := range consumerArr {
		consumer, err := v.ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		if len(consumer) < 2 {
			cmd.SetErr(fmt.Errorf("got %d, wanted 2", len(arr)))
			return
		}
		consumerName, err := consumer[0].ToString()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		consumerPending, err := consumer[1].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		if val.Consumers == nil {
			val.Consumers = make(map[string]int64)
		}
		val.Consumers[consumerName] = consumerPending
	}
	cmd.SetVal(val)
}

func newXPendingCmd(res rueidis.RedisResult) *XPendingCmd {
	cmd := &XPendingCmd{}
	cmd.from(res)
	return cmd
}

type XPendingExt struct {
	ID         string
	Consumer   string
	Idle       time.Duration
	RetryCount int64
}

type XPendingExtCmd struct {
	baseCmd[[]XPendingExt]
}

func (cmd *XPendingExtCmd) from(res rueidis.RedisResult) {
	arrs, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]XPendingExt, 0, len(arrs))
	for _, v := range arrs {
		arr, err := v.ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		if len(arr) < 4 {
			cmd.SetErr(fmt.Errorf("got %d, wanted 4", len(arr)))
			return
		}
		id, err := arr[0].ToString()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		consumer, err := arr[1].ToString()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		idle, err := arr[2].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		retryCount, err := arr[3].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		val = append(val, XPendingExt{
			ID:         id,
			Consumer:   consumer,
			Idle:       time.Duration(idle) * time.Millisecond,
			RetryCount: retryCount,
		})
	}
	cmd.SetVal(val)
}

func newXPendingExtCmd(res rueidis.RedisResult) *XPendingExtCmd {
	cmd := &XPendingExtCmd{}
	cmd.from(res)
	return cmd
}

type XAutoClaimCmd struct {
	err   error
	start string
	val   []XMessage
}

func (cmd *XAutoClaimCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if len(arr) < 2 {
		cmd.SetErr(fmt.Errorf("got %d, wanted 2", len(arr)))
		return
	}
	start, err := arr[0].ToString()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	ranges, err := arr[1].AsXRange()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]XMessage, 0, len(ranges))
	for _, r := range ranges {
		val = append(val, newXMessage(r))
	}
	cmd.val, cmd.start = val, start
}

func newXAutoClaimCmd(res rueidis.RedisResult) *XAutoClaimCmd {
	cmd := &XAutoClaimCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *XAutoClaimCmd) SetVal(val []XMessage, start string) {
	cmd.val = val
	cmd.start = start
}

func (cmd *XAutoClaimCmd) SetErr(err error) {
	cmd.err = err
}

func (cmd *XAutoClaimCmd) Val() (messages []XMessage, start string) {
	return cmd.val, cmd.start
}

func (cmd *XAutoClaimCmd) Err() error {
	return cmd.err
}

func (cmd *XAutoClaimCmd) Result() (messages []XMessage, start string, err error) {
	return cmd.val, cmd.start, cmd.err
}

type XAutoClaimJustIDCmd struct {
	err   error
	start string
	val   []string
}

func (cmd *XAutoClaimJustIDCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if len(arr) < 2 {
		cmd.SetErr(fmt.Errorf("got %d, wanted 2", len(arr)))
		return
	}
	start, err := arr[0].ToString()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val, err := arr[1].AsStrSlice()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.val, cmd.start = val, start
}

func newXAutoClaimJustIDCmd(res rueidis.RedisResult) *XAutoClaimJustIDCmd {
	cmd := &XAutoClaimJustIDCmd{}
	cmd.from(res)
	return cmd

}

func (cmd *XAutoClaimJustIDCmd) SetVal(val []string, start string) {
	cmd.val = val
	cmd.start = start
}

func (cmd *XAutoClaimJustIDCmd) SetErr(err error) {
	cmd.err = err
}

func (cmd *XAutoClaimJustIDCmd) Val() (ids []string, start string) {
	return cmd.val, cmd.start
}

func (cmd *XAutoClaimJustIDCmd) Err() error {
	return cmd.err
}

func (cmd *XAutoClaimJustIDCmd) Result() (ids []string, start string, err error) {
	return cmd.val, cmd.start, cmd.err
}

type XInfoGroup struct {
	Name            string
	LastDeliveredID string
	Consumers       int64
	Pending         int64
	EntriesRead     int64
	Lag             int64
}

type XInfoGroupsCmd struct {
	baseCmd[[]XInfoGroup]
}

func (cmd *XInfoGroupsCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	groupInfos := make([]XInfoGroup, 0, len(arr))
	for _, v := range arr {
		info, err := v.AsMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		var group XInfoGroup
		if attr, ok := info["name"]; ok {
			group.Name, _ = attr.ToString()
		}
		if attr, ok := info["consumers"]; ok {
			group.Consumers, _ = attr.AsInt64()
		}
		if attr, ok := info["pending"]; ok {
			group.Pending, _ = attr.AsInt64()
		}
		if attr, ok := info["entries-read"]; ok {
			group.EntriesRead, _ = attr.AsInt64()
		}
		if attr, ok := info["lag"]; ok {
			group.Lag, _ = attr.AsInt64()
		}
		if attr, ok := info["last-delivered-id"]; ok {
			group.LastDeliveredID, _ = attr.ToString()
		}
		groupInfos = append(groupInfos, group)
	}
	cmd.SetVal(groupInfos)
}

func newXInfoGroupsCmd(res rueidis.RedisResult) *XInfoGroupsCmd {
	cmd := &XInfoGroupsCmd{}
	cmd.from(res)
	return cmd
}

type XInfoStream struct {
	FirstEntry           XMessage
	LastEntry            XMessage
	LastGeneratedID      string
	MaxDeletedEntryID    string
	RecordedFirstEntryID string
	Length               int64
	RadixTreeKeys        int64
	RadixTreeNodes       int64
	Groups               int64
	EntriesAdded         int64
}
type XInfoStreamCmd struct {
	baseCmd[XInfoStream]
}

func (cmd *XInfoStreamCmd) from(res rueidis.RedisResult) {
	kv, err := res.AsMap()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	var val XInfoStream
	if v, ok := kv["length"]; ok {
		val.Length, _ = v.AsInt64()
	}
	if v, ok := kv["radix-tree-keys"]; ok {
		val.RadixTreeKeys, _ = v.AsInt64()
	}
	if v, ok := kv["radix-tree-nodes"]; ok {
		val.RadixTreeNodes, _ = v.AsInt64()
	}
	if v, ok := kv["groups"]; ok {
		val.Groups, _ = v.AsInt64()
	}
	if v, ok := kv["last-generated-id"]; ok {
		val.LastGeneratedID, _ = v.ToString()
	}
	if v, ok := kv["max-deleted-entry-id"]; ok {
		val.MaxDeletedEntryID, _ = v.ToString()
	}
	if v, ok := kv["recorded-first-entry-id"]; ok {
		val.RecordedFirstEntryID, _ = v.ToString()
	}
	if v, ok := kv["entries-added"]; ok {
		val.EntriesAdded, _ = v.AsInt64()
	}
	if v, ok := kv["first-entry"]; ok {
		if r, err := v.AsXRangeEntry(); err == nil {
			val.FirstEntry = newXMessage(r)
		}
	}
	if v, ok := kv["last-entry"]; ok {
		if r, err := v.AsXRangeEntry(); err == nil {
			val.LastEntry = newXMessage(r)
		}
	}
	cmd.SetVal(val)
}

func newXInfoStreamCmd(res rueidis.RedisResult) *XInfoStreamCmd {
	cmd := &XInfoStreamCmd{}
	cmd.from(res)
	return cmd
}

type XInfoStreamConsumerPending struct {
	DeliveryTime  time.Time
	ID            string
	DeliveryCount int64
}

type XInfoStreamGroupPending struct {
	DeliveryTime  time.Time
	ID            string
	Consumer      string
	DeliveryCount int64
}

type XInfoStreamConsumer struct {
	SeenTime time.Time
	Name     string
	Pending  []XInfoStreamConsumerPending
	PelCount int64
}

type XInfoStreamGroup struct {
	Name            string
	LastDeliveredID string
	Pending         []XInfoStreamGroupPending
	Consumers       []XInfoStreamConsumer
	EntriesRead     int64
	Lag             int64
	PelCount        int64
}

type XInfoStreamFull struct {
	LastGeneratedID      string
	MaxDeletedEntryID    string
	RecordedFirstEntryID string
	Entries              []XMessage
	Groups               []XInfoStreamGroup
	Length               int64
	RadixTreeKeys        int64
	RadixTreeNodes       int64
	EntriesAdded         int64
}

type XInfoStreamFullCmd struct {
	baseCmd[XInfoStreamFull]
}

func (cmd *XInfoStreamFullCmd) from(res rueidis.RedisResult) {
	kv, err := res.AsMap()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	var val XInfoStreamFull
	if v, ok := kv["length"]; ok {
		val.Length, _ = v.AsInt64()
	}
	if v, ok := kv["radix-tree-keys"]; ok {
		val.RadixTreeKeys, _ = v.AsInt64()
	}
	if v, ok := kv["radix-tree-nodes"]; ok {
		val.RadixTreeNodes, _ = v.AsInt64()
	}
	if v, ok := kv["last-generated-id"]; ok {
		val.LastGeneratedID, _ = v.ToString()
	}
	if v, ok := kv["entries-added"]; ok {
		val.EntriesAdded, _ = v.AsInt64()
	}
	if v, ok := kv["max-deleted-entry-id"]; ok {
		val.MaxDeletedEntryID, _ = v.ToString()
	}
	if v, ok := kv["recorded-first-entry-id"]; ok {
		val.RecordedFirstEntryID, _ = v.ToString()
	}
	if v, ok := kv["groups"]; ok {
		val.Groups, err = readStreamGroups(v)
		if err != nil {
			cmd.SetErr(err)
			return
		}
	}
	if v, ok := kv["entries"]; ok {
		ranges, err := v.AsXRange()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		val.Entries = make([]XMessage, 0, len(ranges))
		for _, r := range ranges {
			val.Entries = append(val.Entries, newXMessage(r))
		}
	}
	cmd.SetVal(val)
}

func newXInfoStreamFullCmd(res rueidis.RedisResult) *XInfoStreamFullCmd {
	cmd := &XInfoStreamFullCmd{}
	cmd.from(res)
	return cmd
}

func readStreamGroups(res rueidis.RedisMessage) ([]XInfoStreamGroup, error) {
	arr, err := res.ToArray()
	if err != nil {
		return nil, err
	}
	groups := make([]XInfoStreamGroup, 0, len(arr))
	for _, v := range arr {
		info, err := v.AsMap()
		if err != nil {
			return nil, err
		}
		var group XInfoStreamGroup
		if attr, ok := info["name"]; ok {
			group.Name, _ = attr.ToString()
		}
		if attr, ok := info["last-delivered-id"]; ok {
			group.LastDeliveredID, _ = attr.ToString()
		}
		if attr, ok := info["entries-read"]; ok {
			group.EntriesRead, _ = attr.AsInt64()
		}
		if attr, ok := info["lag"]; ok {
			group.Lag, _ = attr.AsInt64()
		}
		if attr, ok := info["pel-count"]; ok {
			group.PelCount, _ = attr.AsInt64()
		}
		if attr, ok := info["pending"]; ok {
			group.Pending, err = readXInfoStreamGroupPending(attr)
			if err != nil {
				return nil, err
			}
		}
		if attr, ok := info["consumers"]; ok {
			group.Consumers, err = readXInfoStreamConsumers(attr)
			if err != nil {
				return nil, err
			}
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func readXInfoStreamGroupPending(res rueidis.RedisMessage) ([]XInfoStreamGroupPending, error) {
	arr, err := res.ToArray()
	if err != nil {
		return nil, err
	}
	pending := make([]XInfoStreamGroupPending, 0, len(arr))
	for _, v := range arr {
		info, err := v.ToArray()
		if err != nil {
			return nil, err
		}
		if len(info) < 4 {
			return nil, fmt.Errorf("got %d, wanted 4", len(arr))
		}
		var p XInfoStreamGroupPending
		p.ID, err = info[0].ToString()
		if err != nil {
			return nil, err
		}
		p.Consumer, err = info[1].ToString()
		if err != nil {
			return nil, err
		}
		delivery, err := info[2].AsInt64()
		if err != nil {
			return nil, err
		}
		p.DeliveryTime = time.Unix(delivery/1000, delivery%1000*int64(time.Millisecond))
		p.DeliveryCount, err = info[3].AsInt64()
		if err != nil {
			return nil, err
		}
		pending = append(pending, p)
	}
	return pending, nil
}

func readXInfoStreamConsumers(res rueidis.RedisMessage) ([]XInfoStreamConsumer, error) {
	arr, err := res.ToArray()
	if err != nil {
		return nil, err
	}
	consumer := make([]XInfoStreamConsumer, 0, len(arr))
	for _, v := range arr {
		info, err := v.AsMap()
		if err != nil {
			return nil, err
		}
		var c XInfoStreamConsumer
		if attr, ok := info["name"]; ok {
			c.Name, _ = attr.ToString()
		}
		if attr, ok := info["seen-time"]; ok {
			seen, _ := attr.AsInt64()
			c.SeenTime = time.Unix(seen/1000, seen%1000*int64(time.Millisecond))
		}
		if attr, ok := info["pel-count"]; ok {
			c.PelCount, _ = attr.AsInt64()
		}
		if attr, ok := info["pending"]; ok {
			pending, err := attr.ToArray()
			if err != nil {
				return nil, err
			}
			c.Pending = make([]XInfoStreamConsumerPending, 0, len(pending))
			for _, v := range pending {
				pendingInfo, err := v.ToArray()
				if err != nil {
					return nil, err
				}
				if len(pendingInfo) < 3 {
					return nil, fmt.Errorf("got %d, wanted 3", len(arr))
				}
				var p XInfoStreamConsumerPending
				p.ID, err = pendingInfo[0].ToString()
				if err != nil {
					return nil, err
				}
				delivery, err := pendingInfo[1].AsInt64()
				if err != nil {
					return nil, err
				}
				p.DeliveryTime = time.Unix(delivery/1000, delivery%1000*int64(time.Millisecond))
				p.DeliveryCount, err = pendingInfo[2].AsInt64()
				if err != nil {
					return nil, err
				}
				c.Pending = append(c.Pending, p)
			}
		}
		consumer = append(consumer, c)
	}
	return consumer, nil
}

type XInfoConsumer struct {
	Name    string
	Pending int64
	Idle    time.Duration
}
type XInfoConsumersCmd struct {
	baseCmd[[]XInfoConsumer]
}

func (cmd *XInfoConsumersCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]XInfoConsumer, 0, len(arr))
	for _, v := range arr {
		info, err := v.AsMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		var consumer XInfoConsumer
		if attr, ok := info["name"]; ok {
			consumer.Name, _ = attr.ToString()
		}
		if attr, ok := info["pending"]; ok {
			consumer.Pending, _ = attr.AsInt64()
		}
		if attr, ok := info["idle"]; ok {
			idle, _ := attr.AsInt64()
			consumer.Idle = time.Duration(idle) * time.Millisecond
		}
		val = append(val, consumer)
	}
	cmd.SetVal(val)
}

func newXInfoConsumersCmd(res rueidis.RedisResult) *XInfoConsumersCmd {
	cmd := &XInfoConsumersCmd{}
	cmd.from(res)
	return cmd
}

// Z represents sorted set member.
type Z struct {
	Member string
	Score  float64
}

// ZWithKey represents sorted set member including the name of the key where it was popped.
type ZWithKey struct {
	Z
	Key string
}

// ZStore is used as an arg to ZInter/ZInterStore and ZUnion/ZUnionStore.
type ZStore struct {
	Aggregate string
	Keys      []string
	Weights   []int64
}

type ZWithKeyCmd struct {
	baseCmd[ZWithKey]
}

func (cmd *ZWithKeyCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if len(arr) < 3 {
		cmd.SetErr(fmt.Errorf("got %d, wanted 3", len(arr)))
		return
	}
	val := ZWithKey{}
	val.Key, err = arr[0].ToString()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val.Member, err = arr[1].ToString()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val.Score, err = arr[2].AsFloat64()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetVal(val)
}

func newZWithKeyCmd(res rueidis.RedisResult) *ZWithKeyCmd {
	cmd := &ZWithKeyCmd{}
	cmd.from(res)
	return cmd
}

type RankScore struct {
	Rank  int64
	Score float64
}

type RankWithScoreCmd struct {
	baseCmd[RankScore]
}

func (cmd *RankWithScoreCmd) from(res rueidis.RedisResult) {
	if cmd.err = res.Error(); cmd.err == nil {
		vs, _ := res.ToArray()
		if len(vs) >= 2 {
			cmd.val.Rank, _ = vs[0].AsInt64()
			cmd.val.Score, _ = vs[1].AsFloat64()
		}
	}
}

func newRankWithScoreCmd(res rueidis.RedisResult) *RankWithScoreCmd {
	cmd := &RankWithScoreCmd{}
	cmd.from(res)
	return cmd
}

type TimeCmd struct {
	baseCmd[time.Time]
}

func (cmd *TimeCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if len(arr) < 2 {
		cmd.SetErr(fmt.Errorf("got %d, wanted 2", len(arr)))
		return
	}
	sec, err := arr[0].AsInt64()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	microSec, err := arr[1].AsInt64()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetVal(time.Unix(sec, microSec*1000))
}

func newTimeCmd(res rueidis.RedisResult) *TimeCmd {
	cmd := &TimeCmd{}
	cmd.from(res)
	return cmd
}

type ClusterNode struct {
	ID   string
	Addr string
}

type ClusterSlot struct {
	Nodes []ClusterNode
	Start int64
	End   int64
}

type ClusterSlotsCmd struct {
	baseCmd[[]ClusterSlot]
}

func (cmd *ClusterSlotsCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]ClusterSlot, 0, len(arr))
	for _, v := range arr {
		slot, err := v.ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		if len(slot) < 2 {
			cmd.SetErr(fmt.Errorf("got %d, excpected atleast 2", len(slot)))
			return
		}
		start, err := slot[0].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		end, err := slot[1].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		nodes := make([]ClusterNode, len(slot)-2)
		for i, j := 2, 0; i < len(slot); i, j = i+1, j+1 {
			node, err := slot[i].ToArray()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			if len(node) < 2 {
				cmd.SetErr(fmt.Errorf("got %d, expected 2 or 3", len(node)))
				return
			}
			ip, err := node[0].ToString()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			port, err := node[1].AsInt64()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			nodes[j].Addr = net.JoinHostPort(ip, strconv.FormatInt(port, 10))
			if len(node) > 2 {
				id, err := node[2].ToString()
				if err != nil {
					cmd.SetErr(err)
					return
				}
				nodes[j].ID = id
			}
		}
		val = append(val, ClusterSlot{
			Start: start,
			End:   end,
			Nodes: nodes,
		})
	}
	cmd.SetVal(val)
}

func newClusterSlotsCmd(res rueidis.RedisResult) *ClusterSlotsCmd {
	cmd := &ClusterSlotsCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *ClusterShardsCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]ClusterShard, 0, len(arr))
	for _, v := range arr {
		dict, err := v.ToMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		shard := ClusterShard{}
		{
			slots := dict["slots"]
			arr, _ := slots.ToArray()
			for i := 0; i+1 < len(arr); i += 2 {
				start, _ := arr[i].AsInt64()
				end, _ := arr[i+1].AsInt64()
				shard.Slots = append(shard.Slots, SlotRange{Start: start, End: end})
			}
		}
		{
			nodes, ok := dict["nodes"]
			if !ok {
				cmd.SetErr(errors.New("nodes not found"))
				return
			}
			arr, err := nodes.ToArray()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			shard.Nodes = make([]Node, len(arr))
			for i := 0; i < len(arr); i++ {
				nodeMap, err := arr[i].ToMap()
				if err != nil {
					cmd.SetErr(err)
					return
				}
				for k, v := range nodeMap {
					switch k {
					case "id":
						shard.Nodes[i].ID, _ = v.ToString()
					case "endpoint":
						shard.Nodes[i].Endpoint, _ = v.ToString()
					case "ip":
						shard.Nodes[i].IP, _ = v.ToString()
					case "hostname":
						shard.Nodes[i].Hostname, _ = v.ToString()
					case "port":
						shard.Nodes[i].Port, _ = v.ToInt64()
					case "tls-port":
						shard.Nodes[i].TLSPort, _ = v.ToInt64()
					case "role":
						shard.Nodes[i].Role, _ = v.ToString()
					case "replication-offset":
						shard.Nodes[i].ReplicationOffset, _ = v.ToInt64()
					case "health":
						shard.Nodes[i].Health, _ = v.ToString()
					}
				}
			}
		}
		val = append(val, shard)
	}
	cmd.SetVal(val)
}

func newClusterShardsCmd(res rueidis.RedisResult) *ClusterShardsCmd {
	cmd := &ClusterShardsCmd{}
	cmd.from(res)
	return cmd
}

type SlotRange struct {
	Start int64
	End   int64
}
type Node struct {
	ID                string
	Endpoint          string
	IP                string
	Hostname          string
	Role              string
	Health            string
	Port              int64
	TLSPort           int64
	ReplicationOffset int64
}
type ClusterShard struct {
	Slots []SlotRange
	Nodes []Node
}

type ClusterShardsCmd struct {
	baseCmd[[]ClusterShard]
}

type GeoPos struct {
	Longitude, Latitude float64
}

type GeoPosCmd struct {
	baseCmd[[]*GeoPos]
}

func (cmd *GeoPosCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]*GeoPos, 0, len(arr))
	for _, v := range arr {
		loc, err := v.ToArray()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				val = append(val, nil)
				continue
			}
			cmd.SetErr(err)
			return
		}
		if len(loc) != 2 {
			cmd.SetErr(fmt.Errorf("got %d, expected 2", len(loc)))
			return
		}
		long, err := loc[0].AsFloat64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		lat, err := loc[1].AsFloat64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		val = append(val, &GeoPos{
			Longitude: long,
			Latitude:  lat,
		})
	}
	cmd.SetVal(val)
}

func newGeoPosCmd(res rueidis.RedisResult) *GeoPosCmd {
	cmd := &GeoPosCmd{}
	cmd.from(res)
	return cmd
}

type GeoLocationCmd struct {
	baseCmd[[]rueidis.GeoLocation]
}

func (cmd *GeoLocationCmd) from(res rueidis.RedisResult) {
	cmd.val, cmd.err = res.AsGeosearch()
}

func newGeoLocationCmd(res rueidis.RedisResult) *GeoLocationCmd {
	cmd := &GeoLocationCmd{}
	cmd.from(res)
	return cmd
}

type CommandInfo struct {
	Name        string
	Flags       []string
	ACLFlags    []string
	Arity       int64
	FirstKeyPos int64
	LastKeyPos  int64
	StepCount   int64
	ReadOnly    bool
}

type CommandsInfoCmd struct {
	baseCmd[map[string]CommandInfo]
}

func (cmd *CommandsInfoCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make(map[string]CommandInfo, len(arr))
	for _, v := range arr {
		info, err := v.ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		if len(info) < 6 {
			cmd.SetErr(fmt.Errorf("got %d, wanted at least 6", len(info)))
			return
		}
		var _cmd CommandInfo
		_cmd.Name, err = info[0].ToString()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		_cmd.Arity, err = info[1].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		_cmd.Flags, err = info[2].AsStrSlice()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				_cmd.Flags = []string{}
			} else {
				cmd.SetErr(err)
				return
			}
		}
		_cmd.FirstKeyPos, err = info[3].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		_cmd.LastKeyPos, err = info[4].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		_cmd.StepCount, err = info[5].AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		for _, flag := range _cmd.Flags {
			if flag == "readonly" {
				_cmd.ReadOnly = true
				break
			}
		}
		if len(info) == 6 {
			val[_cmd.Name] = _cmd
			continue
		}
		_cmd.ACLFlags, err = info[6].AsStrSlice()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				_cmd.ACLFlags = []string{}
			} else {
				cmd.SetErr(err)
				return
			}
		}
		val[_cmd.Name] = _cmd
	}
	cmd.SetVal(val)
}

func newCommandsInfoCmd(res rueidis.RedisResult) *CommandsInfoCmd {
	cmd := &CommandsInfoCmd{}
	cmd.from(res)
	return cmd
}

type HExpireArgs struct {
	NX bool
	XX bool
	GT bool
	LT bool
}

type Sort struct {
	By     string
	Order  string
	Get    []string
	Offset int64
	Count  int64
	Alpha  bool
}

// SetArgs provides arguments for the SetArgs function.
type SetArgs struct {
	ExpireAt time.Time
	Mode     string
	TTL      time.Duration
	Get      bool
	KeepTTL  bool
}

type BitCount struct {
	Start, End int64
	Unit       string // Stores BIT or BYTE
}

//type BitPos struct {
//	BitCount
//	Byte bool
//}

type BitFieldArg struct {
	Encoding string
	Offset   int64
}

type BitField struct {
	Get       *BitFieldArg
	Set       *BitFieldArg
	IncrBy    *BitFieldArg
	Overflow  string
	Increment int64
}

type LPosArgs struct {
	Rank, MaxLen int64
}

// Note: MaxLen/MaxLenApprox and MinID are in conflict, only one of them can be used.
type XAddArgs struct {
	Values     any
	Stream     string
	MinID      string
	ID         string
	MaxLen     int64
	Limit      int64
	NoMkStream bool
	Approx     bool
}

type XReadArgs struct {
	Streams []string // list of streams
	Count   int64
	Block   time.Duration
}

type XReadGroupArgs struct {
	Group    string
	Consumer string
	Streams  []string // list of streams
	Count    int64
	Block    time.Duration
	NoAck    bool
}

type XPendingExtArgs struct {
	Stream   string
	Group    string
	Start    string
	End      string
	Consumer string
	Idle     time.Duration
	Count    int64
}

type XClaimArgs struct {
	Stream   string
	Group    string
	Consumer string
	Messages []string
	MinIdle  time.Duration
}

type XAutoClaimArgs struct {
	Stream   string
	Group    string
	Start    string
	Consumer string
	MinIdle  time.Duration
	Count    int64
}

type XMessage struct {
	Values map[string]any
	ID     string
}

// Note: The GT, LT and NX options are mutually exclusive.
type ZAddArgs struct {
	Members []Z
	NX      bool
	XX      bool
	LT      bool
	GT      bool
	Ch      bool
}

// ZRangeArgs is all the options of the ZRange command.
// In version> 6.2.0, you can replace the(cmd):
//
//	ZREVRANGE,
//	ZRANGEBYSCORE,
//	ZREVRANGEBYSCORE,
//	ZRANGEBYLEX,
//	ZREVRANGEBYLEX.
//
// Please pay attention to your redis-server version.
//
// Rev, ByScore, ByLex and Offset+Count options require redis-server 6.2.0 and higher.
type ZRangeArgs struct {
	Start   any
	Stop    any
	Key     string
	Offset  int64
	Count   int64
	ByScore bool
	ByLex   bool
	Rev     bool
}

type ZRangeBy struct {
	Min, Max      string
	Offset, Count int64
}

type GeoLocation = rueidis.GeoLocation

// GeoRadiusQuery is used with GeoRadius to query geospatial index.
type GeoRadiusQuery struct {
	Unit        string
	Sort        string
	Store       string
	StoreDist   string
	Radius      float64
	Count       int64
	WithCoord   bool
	WithDist    bool
	WithGeoHash bool
}

// GeoSearchQuery is used for GEOSearch/GEOSearchStore command query.
type GeoSearchQuery struct {
	Member     string
	RadiusUnit string
	BoxUnit    string
	Sort       string
	Longitude  float64
	Latitude   float64
	Radius     float64
	BoxWidth   float64
	BoxHeight  float64
	Count      int64
	CountAny   bool
}

type GeoSearchLocationQuery struct {
	GeoSearchQuery

	WithCoord bool
	WithDist  bool
	WithHash  bool
}

type GeoSearchStoreQuery struct {
	GeoSearchQuery

	// When using the StoreDist option, the command stores the items in a
	// sorted set populated with their distance from the center of the circle or box,
	// as a floating-point number, in the same unit specified for that shape.
	StoreDist bool
}

func (q *GeoRadiusQuery) args() []string {
	args := make([]string, 0, 2)
	args = append(args, strconv.FormatFloat(q.Radius, 'f', -1, 64))
	if q.Unit != "" {
		args = append(args, q.Unit)
	} else {
		args = append(args, "km")
	}
	if q.WithCoord {
		args = append(args, "WITHCOORD")
	}
	if q.WithDist {
		args = append(args, "WITHDIST")
	}
	if q.WithGeoHash {
		args = append(args, "WITHHASH")
	}
	if q.Count > 0 {
		args = append(args, "COUNT", strconv.FormatInt(q.Count, 10))
	}
	if q.Sort != "" {
		args = append(args, q.Sort)
	}
	if q.Store != "" {
		args = append(args, "STORE")
		args = append(args, q.Store)
	}
	if q.StoreDist != "" {
		args = append(args, "STOREDIST")
		args = append(args, q.StoreDist)
	}
	return args
}

func (q *GeoSearchQuery) args() []string {
	args := make([]string, 0, 2)
	if q.Member != "" {
		args = append(args, "FROMMEMBER", q.Member)
	} else {
		args = append(args, "FROMLONLAT", strconv.FormatFloat(q.Longitude, 'f', -1, 64), strconv.FormatFloat(q.Latitude, 'f', -1, 64))
	}
	if q.Radius > 0 {
		if q.RadiusUnit == "" {
			q.RadiusUnit = "KM"
		}
		args = append(args, "BYRADIUS", strconv.FormatFloat(q.Radius, 'f', -1, 64), q.RadiusUnit)
	} else {
		if q.BoxUnit == "" {
			q.BoxUnit = "KM"
		}
		args = append(args, "BYBOX", strconv.FormatFloat(q.BoxWidth, 'f', -1, 64), strconv.FormatFloat(q.BoxHeight, 'f', -1, 64), q.BoxUnit)
	}
	if q.Sort != "" {
		args = append(args, q.Sort)
	}
	if q.Count > 0 {
		args = append(args, "COUNT", strconv.FormatInt(q.Count, 10))
		if q.CountAny {
			args = append(args, "ANY")
		}
	}
	return args
}

func (q *GeoSearchLocationQuery) args() []string {
	args := q.GeoSearchQuery.args()
	if q.WithCoord {
		args = append(args, "WITHCOORD")
	}
	if q.WithDist {
		args = append(args, "WITHDIST")
	}
	if q.WithHash {
		args = append(args, "WITHHASH")
	}
	return args
}

type Function struct {
	Name        string
	Description string
	Flags       []string
}

type Library struct {
	Name      string
	Engine    string
	Code      string
	Functions []Function
}

type FunctionListQuery struct {
	LibraryNamePattern string
	WithCode           bool
}

type FunctionListCmd struct {
	baseCmd[[]Library]
}

func (cmd *FunctionListCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	val := make([]Library, len(arr))
	for i := 0; i < len(arr); i++ {
		kv, _ := arr[i].AsMap()
		for k, v := range kv {
			switch k {
			case "library_name":
				val[i].Name, _ = v.ToString()
			case "engine":
				val[i].Engine, _ = v.ToString()
			case "library_code":
				val[i].Code, _ = v.ToString()
			case "functions":
				fns, _ := v.ToArray()
				val[i].Functions = make([]Function, len(fns))
				for j := 0; j < len(fns); j++ {
					fkv, _ := fns[j].AsMap()
					for k, v := range fkv {
						switch k {
						case "name":
							val[i].Functions[j].Name, _ = v.ToString()
						case "description":
							val[i].Functions[j].Description, _ = v.ToString()
						case "flags":
							val[i].Functions[j].Flags, _ = v.AsStrSlice()
						}
					}
				}
			}
		}
	}
	cmd.SetVal(val)
}

func newFunctionListCmd(res rueidis.RedisResult) *FunctionListCmd {
	cmd := &FunctionListCmd{}
	cmd.from(res)
	return cmd
}

func (cmd *FunctionListCmd) First() (*Library, error) {
	if cmd.err != nil {
		return nil, cmd.err
	}
	if len(cmd.val) > 0 {
		return &cmd.val[0], nil
	}
	return nil, rueidis.Nil
}

func usePrecise(dur time.Duration) bool {
	return dur < time.Second || dur%time.Second != 0
}

func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		// too small, truncate too 1ms
		return 1
	}
	return int64(dur / time.Millisecond)
}

func formatSec(dur time.Duration) int64 {
	if dur > 0 && dur < time.Second {
		// too small, truncate too 1s
		return 1
	}
	return int64(dur / time.Second)
}

// https://github.com/redis/go-redis/blob/f994ff1cd96299a5c8029ae3403af7b17ef06e8a/gears_commands.go#L21C1-L35C2
type TFunctionLoadOptions struct {
	Config  string
	Replace bool
}

type TFunctionListOptions struct {
	Library  string
	Verbose  int
	Withcode bool
}

type TFCallOptions struct {
	Keys      []string
	Arguments []string
}

type MapStringInterfaceSliceCmd struct {
	baseCmd[[]map[string]any]
}

func (cmd *MapStringInterfaceSliceCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.val = make([]map[string]any, 0, len(arr))
	for _, ele := range arr {
		m, err := ele.AsMap()
		eleMap := make(map[string]any, len(m))
		if err != nil {
			cmd.SetErr(err)
			return
		}
		for k, v := range m {
			var val any
			if !v.IsNil() {
				var err error
				val, err = v.ToAny()
				if err != nil {
					cmd.SetErr(err)
					return
				}
			}
			eleMap[k] = val
		}
		cmd.val = append(cmd.val, eleMap)
	}
}

func newMapStringInterfaceSliceCmd(res rueidis.RedisResult) *MapStringInterfaceSliceCmd {
	cmd := &MapStringInterfaceSliceCmd{}
	cmd.from(res)
	return cmd
}

type BFInsertOptions struct {
	Capacity   int64
	Error      float64
	Expansion  int64
	NonScaling bool
	NoCreate   bool
}

type BFReserveOptions struct {
	Capacity   int64
	Error      float64
	Expansion  int64
	NonScaling bool
}

type CFReserveOptions struct {
	Capacity      int64
	BucketSize    int64
	MaxIterations int64
	Expansion     int64
}

type CFInsertOptions struct {
	Capacity int64
	NoCreate bool
}

type BFInfo struct {
	Capacity      int64 `redis:"Capacity"`
	Size          int64 `redis:"Size"`
	Filters       int64 `redis:"Number of filters"`
	ItemsInserted int64 `redis:"Number of items inserted"`
	ExpansionRate int64 `redis:"Expansion rate"`
}

type BFInfoCmd struct {
	baseCmd[BFInfo]
}

func (cmd *BFInfoCmd) from(res rueidis.RedisResult) {
	info := BFInfo{}
	if err := res.Error(); err != nil {
		cmd.err = err
		return
	}
	m, err := res.AsIntMap()
	if err != nil {
		cmd.err = err
		return
	}
	keys := make([]string, 0, len(m))
	values := make([]any, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, strconv.FormatInt(v, 10))
	}
	if err := Scan(&info, keys, values); err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(info)
}

func newBFInfoCmd(res rueidis.RedisResult) *BFInfoCmd {
	cmd := &BFInfoCmd{}
	cmd.from(res)
	return cmd
}

type ScanDump struct {
	Data string
	Iter int64
}

type ScanDumpCmd struct {
	baseCmd[ScanDump]
}

func (cmd *ScanDumpCmd) from(res rueidis.RedisResult) {
	scanDump := ScanDump{}
	if err := res.Error(); err != nil {
		cmd.err = err
		return
	}
	arr, err := res.ToArray()
	if err != nil {
		cmd.err = err
		return
	}
	if len(arr) != 2 {
		panic(fmt.Sprintf("wrong length of redis message, got %v, want %v", len(arr), 2))
	}
	iter, err := arr[0].AsInt64()
	if err != nil {
		cmd.err = err
		return
	}
	data, err := arr[1].ToString()
	if err != nil {
		cmd.err = err
		return
	}
	scanDump.Iter = iter
	scanDump.Data = data
	cmd.SetVal(scanDump)
}

func newScanDumpCmd(res rueidis.RedisResult) *ScanDumpCmd {
	cmd := &ScanDumpCmd{}
	cmd.from(res)
	return cmd
}

type CFInfo struct {
	Size             int64 `redis:"Size"`
	NumBuckets       int64 `redis:"Number of buckets"`
	NumFilters       int64 `redis:"Number of filters"`
	NumItemsInserted int64 `redis:"Number of items inserted"`
	NumItemsDeleted  int64 `redis:"Number of items deleted"`
	BucketSize       int64 `redis:"Bucket size"`
	ExpansionRate    int64 `redis:"Expansion rate"`
	MaxIteration     int64 `redis:"Max iterations"`
}

type CFInfoCmd struct {
	baseCmd[CFInfo]
}

func (cmd *CFInfoCmd) from(res rueidis.RedisResult) {
	info := CFInfo{}
	m, err := res.AsMap()
	if err != nil {
		cmd.err = err
		return
	}
	keys := make([]string, 0, len(m))
	values := make([]any, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		val, err := v.AsInt64()
		if err != nil {
			cmd.err = err
			return
		}
		values = append(values, strconv.FormatInt(val, 10))
	}
	if err := Scan(&info, keys, values); err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(info)
}

func newCFInfoCmd(res rueidis.RedisResult) *CFInfoCmd {
	cmd := &CFInfoCmd{}
	cmd.from(res)
	return cmd
}

type CMSInfo struct {
	Width int64 `redis:"width"`
	Depth int64 `redis:"depth"`
	Count int64 `redis:"count"`
}

type CMSInfoCmd struct {
	baseCmd[CMSInfo]
}

func (cmd *CMSInfoCmd) from(res rueidis.RedisResult) {
	info := CMSInfo{}
	m, err := res.AsIntMap()
	if err != nil {
		cmd.err = err
		return
	}
	keys := make([]string, 0, len(m))
	values := make([]any, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, strconv.FormatInt(v, 10))
	}
	if err := Scan(&info, keys, values); err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(info)
}

func newCMSInfoCmd(res rueidis.RedisResult) *CMSInfoCmd {
	cmd := &CMSInfoCmd{}
	cmd.from(res)
	return cmd
}

type TopKInfo struct {
	K     int64   `redis:"k"`
	Width int64   `redis:"width"`
	Depth int64   `redis:"depth"`
	Decay float64 `redis:"decay"`
}

type TopKInfoCmd struct {
	baseCmd[TopKInfo]
}

func (cmd *TopKInfoCmd) from(res rueidis.RedisResult) {
	info := TopKInfo{}
	m, err := res.ToMap()
	if err != nil {
		cmd.err = err
		return
	}
	keys := make([]string, 0, len(m))
	values := make([]any, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		switch k {
		case "k", "width", "depth":
			intVal, err := v.AsInt64()
			if err != nil {
				cmd.err = err
				return
			}
			values = append(values, strconv.FormatInt(intVal, 10))
		case "decay":
			decay, err := v.AsFloat64()
			if err != nil {
				cmd.err = err
				return
			}
			// args of strconv.FormatFloat is copied from cmds.TopkReserveParamsDepth.Decay
			values = append(values, strconv.FormatFloat(decay, 'f', -1, 64))
		default:
			panic("unexpected key")
		}
	}
	if err := Scan(&info, keys, values); err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(info)
}

func newTopKInfoCmd(res rueidis.RedisResult) *TopKInfoCmd {
	cmd := &TopKInfoCmd{}
	cmd.from(res)
	return cmd
}

type MapStringIntCmd struct {
	baseCmd[map[string]int64]
}

func (cmd *MapStringIntCmd) from(res rueidis.RedisResult) {
	m, err := res.AsIntMap()
	if err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(m)
}

func newMapStringIntCmd(res rueidis.RedisResult) *MapStringIntCmd {
	cmd := &MapStringIntCmd{}
	cmd.from(res)
	return cmd
}

// Ref: https://redis.io/commands/tdigest.info/
type TDigestInfo struct {
	Compression       int64 `redis:"Compression"`
	Capacity          int64 `redis:"Capacity"`
	MergedNodes       int64 `redis:"Merged nodes"`
	UnmergedNodes     int64 `redis:"UnmergedNodes"`
	MergedWeight      int64 `redis:"MergedWeight"`
	UnmergedWeight    int64 `redis:"Unmerged weight"`
	Observations      int64 `redis:"Observations"`
	TotalCompressions int64 `redis:"Total compressions"`
	MemoryUsage       int64 `redis:"Memory usage"`
}

type TDigestInfoCmd struct {
	baseCmd[TDigestInfo]
}

func (cmd *TDigestInfoCmd) from(res rueidis.RedisResult) {
	info := TDigestInfo{}
	m, err := res.AsIntMap()
	if err != nil {
		cmd.err = err
		return
	}
	keys := make([]string, 0, len(m))
	values := make([]any, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, strconv.FormatInt(v, 10))
	}
	if err := Scan(&info, keys, values); err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(info)
}

func newTDigestInfoCmd(res rueidis.RedisResult) *TDigestInfoCmd {
	cmd := &TDigestInfoCmd{}
	cmd.from(res)
	return cmd
}

type TDigestMergeOptions struct {
	Compression int64
	Override    bool
}

type TSOptions struct {
	Retention       int
	ChunkSize       int
	Encoding        string
	DuplicatePolicy string
	Labels          map[string]string
}
type TSIncrDecrOptions struct {
	Timestamp    int64
	Retention    int
	ChunkSize    int
	Uncompressed bool
	Labels       map[string]string
}

type TSAlterOptions struct {
	Retention       int
	ChunkSize       int
	DuplicatePolicy string
	Labels          map[string]string
}

type TSCreateRuleOptions struct {
	AlignTimestamp int64
}

type TSGetOptions struct {
	Latest bool
}

type TSInfoOptions struct {
	Debug bool
}
type Aggregator int

const (
	Invalid = Aggregator(iota)
	Avg
	Sum
	Min
	Max
	Range
	Count
	First
	Last
	StdP
	StdS
	VarP
	VarS
	Twa
)

func (a Aggregator) String() string {
	switch a {
	case Invalid:
		return ""
	case Avg:
		return "AVG"
	case Sum:
		return "SUM"
	case Min:
		return "MIN"
	case Max:
		return "MAX"
	case Range:
		return "RANGE"
	case Count:
		return "COUNT"
	case First:
		return "FIRST"
	case Last:
		return "LAST"
	case StdP:
		return "STD.P"
	case StdS:
		return "STD.S"
	case VarP:
		return "VAR.P"
	case VarS:
		return "VAR.S"
	case Twa:
		return "TWA"
	default:
		return ""
	}
}

type TSTimestampValue struct {
	Timestamp int64
	Value     float64
}
type TSTimestampValueCmd struct {
	baseCmd[TSTimestampValue]
}

func (cmd *TSTimestampValueCmd) from(res rueidis.RedisResult) {
	val := TSTimestampValue{}
	if err := res.Error(); err != nil {
		cmd.err = err
		return
	}
	arr, err := res.ToArray()
	if err != nil {
		cmd.err = err
		return
	}
	if len(arr) != 2 {
		panic(fmt.Sprintf("wrong len of array reply, should be 2, got %v", len(arr)))
	}
	val.Timestamp, err = arr[0].AsInt64()
	if err != nil {
		cmd.err = err
		return
	}
	val.Value, err = arr[1].AsFloat64()
	if err != nil {
		cmd.err = err
		return
	}
	cmd.SetVal(val)
}

func newTSTimestampValueCmd(res rueidis.RedisResult) *TSTimestampValueCmd {
	cmd := &TSTimestampValueCmd{}
	cmd.from(res)
	return cmd
}

type MapStringInterfaceCmd struct {
	baseCmd[map[string]any]
}

func (cmd *MapStringInterfaceCmd) from(res rueidis.RedisResult) {
	m, err := res.AsMap()
	if err != nil {
		cmd.err = err
		return
	}
	strIntMap := make(map[string]any, len(m))
	for k, ele := range m {
		var v any
		var err error
		if !ele.IsNil() {
			v, err = ele.ToAny()
			if err != nil {
				cmd.err = err
				return
			}
		}
		strIntMap[k] = v
	}
	cmd.SetVal(strIntMap)
}

func newMapStringInterfaceCmd(res rueidis.RedisResult) *MapStringInterfaceCmd {
	cmd := &MapStringInterfaceCmd{}
	cmd.from(res)
	return cmd
}

type TSTimestampValueSliceCmd struct {
	baseCmd[[]TSTimestampValue]
}

func (cmd *TSTimestampValueSliceCmd) from(res rueidis.RedisResult) {
	msgSlice, err := res.ToArray()
	if err != nil {
		cmd.err = err
		return
	}
	tsValSlice := make([]TSTimestampValue, 0, len(msgSlice))
	for i := 0; i < len(msgSlice); i++ {
		msgArray, err := msgSlice[i].ToArray()
		if err != nil {
			cmd.err = err
			return
		}
		tstmp, err := msgArray[0].AsInt64()
		if err != nil {
			cmd.err = err
			return
		}
		val, err := msgArray[1].AsFloat64()
		if err != nil {
			cmd.err = err
			return
		}
		tsValSlice = append(tsValSlice, TSTimestampValue{Timestamp: tstmp, Value: val})
	}
	cmd.SetVal(tsValSlice)
}

func newTSTimestampValueSliceCmd(res rueidis.RedisResult) *TSTimestampValueSliceCmd {
	cmd := &TSTimestampValueSliceCmd{}
	cmd.from(res)
	return cmd
}

type MapStringSliceInterfaceCmd struct {
	baseCmd[map[string][]any]
}

func (cmd *MapStringSliceInterfaceCmd) from(res rueidis.RedisResult) {
	m, err := res.ToMap()
	if err != nil {
		cmd.err = err
		return
	}
	mapStrSliceInt := make(map[string][]any, len(m))
	for k, entry := range m {
		vals, err := entry.ToArray()
		if err != nil {
			cmd.err = err
			return
		}
		anySlice := make([]any, 0, len(vals))
		for _, v := range vals {
			var err error
			ele, err := v.ToAny()
			if err != nil {
				cmd.err = err
				return
			}
			anySlice = append(anySlice, ele)
		}
		mapStrSliceInt[k] = anySlice
	}
	cmd.SetVal(mapStrSliceInt)
}

func newMapStringSliceInterfaceCmd(res rueidis.RedisResult) *MapStringSliceInterfaceCmd {
	cmd := &MapStringSliceInterfaceCmd{}
	cmd.from(res)
	return cmd
}

type TSRangeOptions struct {
	Latest          bool
	FilterByTS      []int
	FilterByValue   []int
	Count           int
	Align           interface{}
	Aggregator      Aggregator
	BucketDuration  int
	BucketTimestamp interface{}
	Empty           bool
}

type TSRevRangeOptions struct {
	Latest          bool
	FilterByTS      []int
	FilterByValue   []int
	Count           int
	Align           interface{}
	Aggregator      Aggregator
	BucketDuration  int
	BucketTimestamp interface{}
	Empty           bool
}

type TSMRangeOptions struct {
	Latest          bool
	FilterByTS      []int
	FilterByValue   []int
	WithLabels      bool
	SelectedLabels  []interface{}
	Count           int
	Align           interface{}
	Aggregator      Aggregator
	BucketDuration  int
	BucketTimestamp interface{}
	Empty           bool
	GroupByLabel    interface{}
	Reducer         interface{}
}

type TSMRevRangeOptions struct {
	Latest          bool
	FilterByTS      []int
	FilterByValue   []int
	WithLabels      bool
	SelectedLabels  []interface{}
	Count           int
	Align           interface{}
	Aggregator      Aggregator
	BucketDuration  int
	BucketTimestamp interface{}
	Empty           bool
	GroupByLabel    interface{}
	Reducer         interface{}
}

type TSMGetOptions struct {
	Latest         bool
	WithLabels     bool
	SelectedLabels []interface{}
}

type JSONSetArgs struct {
	Key   string
	Path  string
	Value interface{}
}

type JSONArrIndexArgs struct {
	Start int
	Stop  *int
}

type JSONArrTrimArgs struct {
	Start int
	Stop  *int
}

type JSONCmd struct {
	baseCmd[string]
	expanded []any // expanded will be used at JSONCmd.Expanded
	typ      jsonCmdTyp
}

type jsonCmdTyp int

const (
	TYP_STRING jsonCmdTyp = iota
	TYP_ARRAY
)

// https://github.com/redis/go-redis/blob/v9.3.0/json.go#L86
func (cmd *JSONCmd) Val() string {
	return cmd.val
}

// https://github.com/redis/go-redis/blob/v9.3.0/json.go#L105
func (cmd *JSONCmd) Expanded() (any, error) {
	if cmd.typ == TYP_STRING {
		return cmd.Val(), nil
	}
	// TYP_ARRAY
	return cmd.expanded, nil
}

func (cmd *JSONCmd) Result() (string, error) {
	return cmd.Val(), nil
}

func (cmd *JSONCmd) from(res rueidis.RedisResult) {
	msg, err := res.ToMessage()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	switch {
	// JSON.GET
	case msg.IsString():
		cmd.typ = TYP_STRING
		str, err := res.ToString()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		cmd.SetVal(str)
	// JSON.NUMINCRBY
	case msg.IsArray():
		// we set marshaled string to cmd.val
		// which will be used at cmd.Val()
		// and also stored parsed result to cmd.expanded,
		// which will be used at cmd.Expanded()
		cmd.typ = TYP_ARRAY
		arr, err := res.ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		expanded := make([]any, len(arr))
		for i, e := range arr {
			anyE, err := e.ToAny()
			if err != nil {
				if err == rueidis.Nil {
					continue
				}
				cmd.SetErr(err)
				return
			}
			expanded[i] = anyE
		}
		cmd.expanded = expanded
		val, err := json.Marshal(cmd.expanded)
		if err != nil {
			cmd.SetErr(err)
			return
		}
		cmd.SetVal(string(val))
	default:
		panic("invalid type, expect array or string")
	}
}

func newJSONCmd(res rueidis.RedisResult) *JSONCmd {
	cmd := &JSONCmd{}
	cmd.from(res)
	return cmd
}

type JSONGetArgs struct {
	Indent  string
	Newline string
	Space   string
}

type IntPointerSliceCmd struct {
	baseCmd[[]*int64]
}

func (cmd *IntPointerSliceCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	intPtrSlice := make([]*int64, len(arr))
	for i, e := range arr {
		if e.IsNil() {
			continue
		}
		length, err := e.ToInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		intPtrSlice[i] = &length
	}
	cmd.SetVal(intPtrSlice)
}

// newIntPointerSliceCmd initialises an IntPointerSliceCmd
func newIntPointerSliceCmd(res rueidis.RedisResult) *IntPointerSliceCmd {
	cmd := &IntPointerSliceCmd{}
	cmd.from(res)
	return cmd
}

type JSONSliceCmd struct {
	baseCmd[[]any]
}

func (cmd *JSONSliceCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	anySlice := make([]any, len(arr))
	for i, e := range arr {
		if e.IsNil() {
			continue
		}
		anyE, err := e.ToAny()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		anySlice[i] = anyE
	}
	cmd.SetVal(anySlice)
}

func newJSONSliceCmd(res rueidis.RedisResult) *JSONSliceCmd {
	cmd := &JSONSliceCmd{}
	cmd.from(res)
	return cmd
}

type MapMapStringInterfaceCmd struct {
	baseCmd[map[string]any]
}

func (cmd *MapMapStringInterfaceCmd) from(res rueidis.RedisResult) {
	arr, err := res.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	data := make(map[string]any, len(arr)/2)
	for i := 0; i < len(arr); i++ {
		arr1, err := arr[i].ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		for _i := 0; _i < len(arr1); _i += 2 {
			key, err := arr1[_i].ToString()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			if !arr1[_i+1].IsNil() {
				value, err := arr1[_i+1].ToAny()
				if err != nil {
					cmd.SetErr(err)
					return
				}
				data[key] = value
			} else {
				data[key] = nil
			}
		}
	}
	cmd.SetVal(data)
}

func newMapMapStringInterfaceCmd(res rueidis.RedisResult) *MapMapStringInterfaceCmd {
	cmd := &MapMapStringInterfaceCmd{}
	cmd.from(res)
	return cmd
}

type FTAggregateResult struct {
	Total int
	Rows  []AggregateRow
}

type AggregateRow struct {
	Fields map[string]any
}

// Each AggregateReducer have different args.
// Please follow https://redis.io/docs/interact/search-and-query/search/aggregations/#supported-groupby-reducers for more information.
type FTAggregateReducer struct {
	Reducer SearchAggregator
	Args    []interface{}
	As      string
}

type FTAggregateGroupBy struct {
	Fields []interface{}
	Reduce []FTAggregateReducer
}

type FTAggregateSortBy struct {
	FieldName string
	Asc       bool
	Desc      bool
}

type FTAggregateApply struct {
	Field string
	As    string
}

type FTAggregateLoad struct {
	Field string
	As    string
}

type FTAggregateWithCursor struct {
	Count   int
	MaxIdle int
}

type FTAggregateOptions struct {
	Verbatim          bool
	LoadAll           bool
	Load              []FTAggregateLoad
	Timeout           int
	GroupBy           []FTAggregateGroupBy
	SortBy            []FTAggregateSortBy
	SortByMax         int
	Apply             []FTAggregateApply
	LimitOffset       int
	Limit             int
	Filter            string
	WithCursor        bool
	WithCursorOptions *FTAggregateWithCursor
	Params            map[string]interface{}
	DialectVersion    int
}

type AggregateCmd struct {
	baseCmd[*FTAggregateResult]
}

func (cmd *AggregateCmd) from(res rueidis.RedisResult) {
	if err := res.Error(); err != nil {
		cmd.SetErr(err)
		return
	}
	anyRes, err := res.ToAny()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetRawVal(anyRes)
	msg, err := res.ToMessage()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if !(msg.IsMap() || msg.IsArray()) {
		panic("res should be either map(RESP3) or array(RESP2)")
	}
	if msg.IsMap() {
		total, docs, err := msg.AsFtAggregate()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		aggResult := &FTAggregateResult{Total: int(total)}
		for _, doc := range docs {
			anyMap := make(map[string]any, len(doc))
			for k, v := range doc {
				anyMap[k] = v
			}
			aggResult.Rows = append(aggResult.Rows, AggregateRow{anyMap})
		}
		cmd.SetVal(aggResult)
		return
	}
	// is RESP2 array
	rows, err := msg.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	anyArr := make([]any, 0, len(rows))
	for _, e := range rows {
		anyE, err := e.ToAny()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		anyArr = append(anyArr, anyE)
	}
	result, err := processAggregateResult(anyArr)
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetVal(result)
}

// Ref: https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L584
func processAggregateResult(data []interface{}) (*FTAggregateResult, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data returned")
	}

	total, ok := data[0].(int64)
	if !ok {
		return nil, fmt.Errorf("invalid total format")
	}

	rows := make([]AggregateRow, 0, len(data)-1)
	for _, row := range data[1:] {
		fields, ok := row.([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid row format")
		}

		rowMap := make(map[string]interface{})
		for i := 0; i < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				return nil, fmt.Errorf("invalid field key format")
			}
			value := fields[i+1]
			rowMap[key] = value
		}
		rows = append(rows, AggregateRow{Fields: rowMap})
	}

	result := &FTAggregateResult{
		Total: int(total),
		Rows:  rows,
	}
	return result, nil
}

func newAggregateCmd(res rueidis.RedisResult) *AggregateCmd {
	cmd := &AggregateCmd{}
	cmd.from(res)
	return cmd
}

type FTCreateOptions struct {
	OnHash          bool
	OnJSON          bool
	Prefix          []any
	Filter          string
	DefaultLanguage string
	LanguageField   string
	Score           float64
	ScoreField      string
	PayloadField    string
	MaxTextFields   int
	NoOffsets       bool
	Temporary       int
	NoHL            bool
	NoFields        bool
	NoFreqs         bool
	StopWords       []any
	SkipInitialScan bool
}

type SearchAggregator int

const (
	SearchInvalid = SearchAggregator(iota)
	SearchAvg
	SearchSum
	SearchMin
	SearchMax
	SearchCount
	SearchCountDistinct
	SearchCountDistinctish
	SearchStdDev
	SearchQuantile
	SearchToList
	SearchFirstValue
	SearchRandomSample
)

func (a SearchAggregator) String() string {
	switch a {
	case SearchInvalid:
		return ""
	case SearchAvg:
		return "AVG"
	case SearchSum:
		return "SUM"
	case SearchMin:
		return "MIN"
	case SearchMax:
		return "MAX"
	case SearchCount:
		return "COUNT"
	case SearchCountDistinct:
		return "COUNT_DISTINCT"
	case SearchCountDistinctish:
		return "COUNT_DISTINCTISH"
	case SearchStdDev:
		return "STDDEV"
	case SearchQuantile:
		return "QUANTILE"
	case SearchToList:
		return "TOLIST"
	case SearchFirstValue:
		return "FIRST_VALUE"
	case SearchRandomSample:
		return "RANDOM_SAMPLE"
	default:
		return ""
	}
}

type SearchFieldType int

const (
	SearchFieldTypeInvalid = SearchFieldType(iota)
	SearchFieldTypeNumeric
	SearchFieldTypeTag
	SearchFieldTypeText
	SearchFieldTypeGeo
	SearchFieldTypeVector
	SearchFieldTypeGeoShape
)

func (t SearchFieldType) String() string {
	switch t {
	case SearchFieldTypeInvalid:
		return ""
	case SearchFieldTypeNumeric:
		return "NUMERIC"
	case SearchFieldTypeTag:
		return "TAG"
	case SearchFieldTypeText:
		return "TEXT"
	case SearchFieldTypeGeo:
		return "GEO"
	case SearchFieldTypeVector:
		return "VECTOR"
	case SearchFieldTypeGeoShape:
		return "GEOSHAPE"
	default:
		return "TEXT"
	}
}

type FieldSchema struct {
	FieldName         string
	As                string
	FieldType         SearchFieldType
	Sortable          bool
	UNF               bool
	NoStem            bool
	NoIndex           bool
	PhoneticMatcher   string
	Weight            float64
	Separator         string
	CaseSensitive     bool
	WithSuffixtrie    bool
	VectorArgs        *FTVectorArgs
	GeoShapeFieldType string
	IndexEmpty        bool
	IndexMissing      bool
}

type FTVectorArgs struct {
	FlatOptions *FTFlatOptions
	HNSWOptions *FTHNSWOptions
}

type FTFlatOptions struct {
	Type            string
	Dim             int
	DistanceMetric  string
	InitialCapacity int
	BlockSize       int
}

type FTHNSWOptions struct {
	Type                   string
	Dim                    int
	DistanceMetric         string
	InitialCapacity        int
	MaxEdgesPerNode        int
	MaxAllowedEdgesPerNode int
	EFRunTime              int
	Epsilon                float64
}

type SpellCheckTerms struct {
	Include    bool
	Exclude    bool
	Dictionary string
}

type FTSearchFilter struct {
	FieldName any
	Min       any
	Max       any
}

type FTSearchGeoFilter struct {
	FieldName string
	Longitude float64
	Latitude  float64
	Radius    float64
	Unit      string
}

type FTSearchReturn struct {
	FieldName string
	As        string
}

type FTSearchSortBy struct {
	FieldName string
	Asc       bool
	Desc      bool
}

type FTDropIndexOptions struct {
	DeleteDocs bool
}

type FTExplainOptions struct {
	Dialect string
}

type IndexErrors struct {
	IndexingFailures     int `redis:"indexing failures"`
	LastIndexingError    string
	LastIndexingErrorKey string
}

type FTAttribute struct {
	Identifier      string
	Attribute       string
	Type            string
	Weight          float64
	Sortable        bool
	NoStem          bool
	NoIndex         bool
	UNF             bool
	PhoneticMatcher string
	CaseSensitive   bool
	WithSuffixtrie  bool
}

type CursorStats struct {
	GlobalIdle    int
	GlobalTotal   int
	IndexCapacity int
	IndexTotal    int
}

type FieldStatistic struct {
	Identifier  string
	Attribute   string
	IndexErrors IndexErrors
}

type GCStats struct {
	BytesCollected       int    `redis:"bytes_collected"`
	TotalMsRun           int    `redis:"total_ms_run"`
	TotalCycles          int    `redis:"total_cycles"`
	AverageCycleTimeMs   string `redis:"average_cycle_time_ms"`
	LastRunTimeMs        int    `redis:"last_run_time_ms"`
	GCNumericTreesMissed int    `redis:"gc_numeric_trees_missed"`
	GCBlocksDenied       int    `redis:"gc_blocks_denied"`
}

type IndexDefinition struct {
	KeyType      string
	Prefixes     []string
	DefaultScore float64
}

type FTInfoResult struct {
	IndexErrors              IndexErrors      `redis:"Index Errors"`
	Attributes               []FTAttribute    `redis:"attributes"`
	BytesPerRecordAvg        string           `redis:"bytes_per_record_avg"`
	Cleaning                 int              `redis:"cleaning"`
	CursorStats              CursorStats      `redis:"cursor_stats"`
	DialectStats             map[string]int   `redis:"dialect_stats"`
	DocTableSizeMB           float64          `redis:"doc_table_size_mb"`
	FieldStatistics          []FieldStatistic `redis:"field statistics"`
	GCStats                  GCStats          `redis:"gc_stats"`
	GeoshapesSzMB            float64          `redis:"geoshapes_sz_mb"`
	HashIndexingFailures     int              `redis:"hash_indexing_failures"`
	IndexDefinition          IndexDefinition  `redis:"index_definition"`
	IndexName                string           `redis:"index_name"`
	IndexOptions             []string         `redis:"index_options"`
	Indexing                 int              `redis:"indexing"`
	InvertedSzMB             float64          `redis:"inverted_sz_mb"`
	KeyTableSizeMB           float64          `redis:"key_table_size_mb"`
	MaxDocID                 int              `redis:"max_doc_id"`
	NumDocs                  int              `redis:"num_docs"`
	NumRecords               int              `redis:"num_records"`
	NumTerms                 int              `redis:"num_terms"`
	NumberOfUses             int              `redis:"number_of_uses"`
	OffsetBitsPerRecordAvg   string           `redis:"offset_bits_per_record_avg"`
	OffsetVectorsSzMB        float64          `redis:"offset_vectors_sz_mb"`
	OffsetsPerTermAvg        string           `redis:"offsets_per_term_avg"`
	PercentIndexed           float64          `redis:"percent_indexed"`
	RecordsPerDocAvg         string           `redis:"records_per_doc_avg"`
	SortableValuesSizeMB     float64          `redis:"sortable_values_size_mb"`
	TagOverheadSzMB          float64          `redis:"tag_overhead_sz_mb"`
	TextOverheadSzMB         float64          `redis:"text_overhead_sz_mb"`
	TotalIndexMemorySzMB     float64          `redis:"total_index_memory_sz_mb"`
	TotalIndexingTime        int              `redis:"total_indexing_time"`
	TotalInvertedIndexBlocks int              `redis:"total_inverted_index_blocks"`
	VectorIndexSzMB          float64          `redis:"vector_index_sz_mb"`
}

type FTInfoCmd struct {
	baseCmd[FTInfoResult]
}

// Ref: https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1143
func parseFTInfo(data map[string]interface{}) (FTInfoResult, error) {
	var ftInfo FTInfoResult
	// Manually parse each field from the map
	if indexErrors, ok := data["Index Errors"].([]interface{}); ok {
		ftInfo.IndexErrors = IndexErrors{
			IndexingFailures:     ToInteger(indexErrors[1]),
			LastIndexingError:    ToString(indexErrors[3]),
			LastIndexingErrorKey: ToString(indexErrors[5]),
		}
	}

	if attributes, ok := data["attributes"].([]interface{}); ok {
		for _, attr := range attributes {
			if attrMap, ok := attr.([]interface{}); ok {
				att := FTAttribute{}
				for i := 0; i < len(attrMap); i++ {
					if ToLower(ToString(attrMap[i])) == "attribute" {
						att.Attribute = ToString(attrMap[i+1])
						continue
					}
					if ToLower(ToString(attrMap[i])) == "identifier" {
						att.Identifier = ToString(attrMap[i+1])
						continue
					}
					if ToLower(ToString(attrMap[i])) == "type" {
						att.Type = ToString(attrMap[i+1])
						continue
					}
					if ToLower(ToString(attrMap[i])) == "weight" {
						att.Weight = ToFloat(attrMap[i+1])
						continue
					}
					if ToLower(ToString(attrMap[i])) == "nostem" {
						att.NoStem = true
						continue
					}
					if ToLower(ToString(attrMap[i])) == "sortable" {
						att.Sortable = true
						continue
					}
					if ToLower(ToString(attrMap[i])) == "noindex" {
						att.NoIndex = true
						continue
					}
					if ToLower(ToString(attrMap[i])) == "unf" {
						att.UNF = true
						continue
					}
					if ToLower(ToString(attrMap[i])) == "phonetic" {
						att.PhoneticMatcher = ToString(attrMap[i+1])
						continue
					}
					if ToLower(ToString(attrMap[i])) == "case_sensitive" {
						att.CaseSensitive = true
						continue
					}
					if ToLower(ToString(attrMap[i])) == "withsuffixtrie" {
						att.WithSuffixtrie = true
						continue
					}

				}
				ftInfo.Attributes = append(ftInfo.Attributes, att)
			}
		}
	}

	ftInfo.BytesPerRecordAvg = ToString(data["bytes_per_record_avg"])
	ftInfo.Cleaning = ToInteger(data["cleaning"])

	if cursorStats, ok := data["cursor_stats"].([]interface{}); ok {
		ftInfo.CursorStats = CursorStats{
			GlobalIdle:    ToInteger(cursorStats[1]),
			GlobalTotal:   ToInteger(cursorStats[3]),
			IndexCapacity: ToInteger(cursorStats[5]),
			IndexTotal:    ToInteger(cursorStats[7]),
		}
	}

	if dialectStats, ok := data["dialect_stats"].([]interface{}); ok {
		ftInfo.DialectStats = make(map[string]int)
		for i := 0; i < len(dialectStats); i += 2 {
			ftInfo.DialectStats[ToString(dialectStats[i])] = ToInteger(dialectStats[i+1])
		}
	}

	ftInfo.DocTableSizeMB = ToFloat(data["doc_table_size_mb"])

	if fieldStats, ok := data["field statistics"].([]interface{}); ok {
		for _, stat := range fieldStats {
			if statMap, ok := stat.([]interface{}); ok {
				ftInfo.FieldStatistics = append(ftInfo.FieldStatistics, FieldStatistic{
					Identifier: ToString(statMap[1]),
					Attribute:  ToString(statMap[3]),
					IndexErrors: IndexErrors{
						IndexingFailures:     ToInteger(statMap[5].([]interface{})[1]),
						LastIndexingError:    ToString(statMap[5].([]interface{})[3]),
						LastIndexingErrorKey: ToString(statMap[5].([]interface{})[5]),
					},
				})
			}
		}
	}

	if gcStats, ok := data["gc_stats"].([]interface{}); ok {
		ftInfo.GCStats = GCStats{}
		for i := 0; i < len(gcStats); i += 2 {
			if ToLower(ToString(gcStats[i])) == "bytes_collected" {
				ftInfo.GCStats.BytesCollected = ToInteger(gcStats[i+1])
				continue
			}
			if ToLower(ToString(gcStats[i])) == "total_ms_run" {
				ftInfo.GCStats.TotalMsRun = ToInteger(gcStats[i+1])
				continue
			}
			if ToLower(ToString(gcStats[i])) == "total_cycles" {
				ftInfo.GCStats.TotalCycles = ToInteger(gcStats[i+1])
				continue
			}
			if ToLower(ToString(gcStats[i])) == "average_cycle_time_ms" {
				ftInfo.GCStats.AverageCycleTimeMs = ToString(gcStats[i+1])
				continue
			}
			if ToLower(ToString(gcStats[i])) == "last_run_time_ms" {
				ftInfo.GCStats.LastRunTimeMs = ToInteger(gcStats[i+1])
				continue
			}
			if ToLower(ToString(gcStats[i])) == "gc_numeric_trees_missed" {
				ftInfo.GCStats.GCNumericTreesMissed = ToInteger(gcStats[i+1])
				continue
			}
			if ToLower(ToString(gcStats[i])) == "gc_blocks_denied" {
				ftInfo.GCStats.GCBlocksDenied = ToInteger(gcStats[i+1])
				continue
			}
		}
	}

	ftInfo.GeoshapesSzMB = ToFloat(data["geoshapes_sz_mb"])
	ftInfo.HashIndexingFailures = ToInteger(data["hash_indexing_failures"])

	if indexDef, ok := data["index_definition"].([]interface{}); ok {
		ftInfo.IndexDefinition = IndexDefinition{
			KeyType:      ToString(indexDef[1]),
			Prefixes:     ToStringSlice(indexDef[3]),
			DefaultScore: ToFloat(indexDef[5]),
		}
	}

	ftInfo.IndexName = ToString(data["index_name"])
	ftInfo.IndexOptions = ToStringSlice(data["index_options"].([]interface{}))
	ftInfo.Indexing = ToInteger(data["indexing"])
	ftInfo.InvertedSzMB = ToFloat(data["inverted_sz_mb"])
	ftInfo.KeyTableSizeMB = ToFloat(data["key_table_size_mb"])
	ftInfo.MaxDocID = ToInteger(data["max_doc_id"])
	ftInfo.NumDocs = ToInteger(data["num_docs"])
	ftInfo.NumRecords = ToInteger(data["num_records"])
	ftInfo.NumTerms = ToInteger(data["num_terms"])
	ftInfo.NumberOfUses = ToInteger(data["number_of_uses"])
	ftInfo.OffsetBitsPerRecordAvg = ToString(data["offset_bits_per_record_avg"])
	ftInfo.OffsetVectorsSzMB = ToFloat(data["offset_vectors_sz_mb"])
	ftInfo.OffsetsPerTermAvg = ToString(data["offsets_per_term_avg"])
	ftInfo.PercentIndexed = ToFloat(data["percent_indexed"])
	ftInfo.RecordsPerDocAvg = ToString(data["records_per_doc_avg"])
	ftInfo.SortableValuesSizeMB = ToFloat(data["sortable_values_size_mb"])
	ftInfo.TagOverheadSzMB = ToFloat(data["tag_overhead_sz_mb"])
	ftInfo.TextOverheadSzMB = ToFloat(data["text_overhead_sz_mb"])
	ftInfo.TotalIndexMemorySzMB = ToFloat(data["total_index_memory_sz_mb"])
	ftInfo.TotalIndexingTime = ToInteger(data["total_indexing_time"])
	ftInfo.TotalInvertedIndexBlocks = ToInteger(data["total_inverted_index_blocks"])
	ftInfo.VectorIndexSzMB = ToFloat(data["vector_index_sz_mb"])

	return ftInfo, nil
}

func (cmd *FTInfoCmd) from(res rueidis.RedisResult) {
	if err := res.Error(); err != nil {
		cmd.SetErr(err)
		return
	}
	m, err := res.AsMap()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	anyM, err := res.ToAny()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetRawVal(anyM)
	anyMap := make(map[string]any, len(m))
	for k, v := range m {
		anyMap[k], err = v.ToAny()
		if err != nil {
			cmd.SetErr(err)
			return
		}
	}
	ftInfoResult, err := parseFTInfo(anyMap)
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetVal(ftInfoResult)
}

func newFTInfoCmd(res rueidis.RedisResult) *FTInfoCmd {
	cmd := &FTInfoCmd{}
	cmd.from(res)
	return cmd
}

type FTSpellCheckOptions struct {
	Distance int
	Terms    *FTSpellCheckTerms
	Dialect  int
}

type FTSpellCheckTerms struct {
	Inclusion  string // Either "INCLUDE" or "EXCLUDE"
	Dictionary string
	Terms      []interface{}
}

type SpellCheckResult struct {
	Term        string
	Suggestions []SpellCheckSuggestion
}

type SpellCheckSuggestion struct {
	Score      float64
	Suggestion string
}

type FTSpellCheckCmd struct{ baseCmd[[]SpellCheckResult] }

func (cmd *FTSpellCheckCmd) Val() []SpellCheckResult {
	return cmd.val
}

func (cmd *FTSpellCheckCmd) Result() ([]SpellCheckResult, error) {
	return cmd.Val(), cmd.Err()
}

func (cmd *FTSpellCheckCmd) from(res rueidis.RedisResult) {
	if err := res.Error(); err != nil {
		cmd.SetErr(err)
		return
	}
	msg, err := res.ToMessage()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if !(msg.IsMap() || msg.IsArray()) {
		panic("res should be either map(RESP3) or array(RESP2)")
	}
	if msg.IsMap() {
		// is RESP3 map
		m, err := msg.ToMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		anyM, err := msg.ToAny()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		cmd.SetRawVal(anyM)
		spellCheckResults := []SpellCheckResult{}
		result := m["results"]
		resultMap, err := result.ToMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		for k, v := range resultMap {
			result := SpellCheckResult{}
			result.Term = k
			suggestions, err := v.ToArray()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			for _, suggestion := range suggestions {
				// map key: suggestion, score
				sugMap, err := suggestion.ToMap()
				if err != nil {
					cmd.SetErr(err)
					return
				}
				for _k, _v := range sugMap {
					score, err := _v.ToFloat64()
					if err != nil {
						cmd.SetErr(err)
						return
					}
					result.Suggestions = append(result.Suggestions, SpellCheckSuggestion{Suggestion: _k, Score: score})
				}
			}
			spellCheckResults = append(spellCheckResults, result)
		}
		cmd.SetVal(spellCheckResults)
		return
	}
	// is RESP2 array
	arr, err := msg.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	anyRes, err := msg.ToAny()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetRawVal(anyRes)
	AnyArr := make([]any, 0, len(arr))
	for _, e := range arr {
		anyE, err := e.ToAny()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		AnyArr = append(AnyArr, anyE)
	}
	result, err := parseFTSpellCheck(AnyArr)
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetVal(result)
}

func newFTSpellCheckCmd(res rueidis.RedisResult) *FTSpellCheckCmd {
	cmd := &FTSpellCheckCmd{}
	cmd.from(res)
	return cmd
}

func parseFTSpellCheck(data []interface{}) ([]SpellCheckResult, error) {
	results := make([]SpellCheckResult, 0, len(data))

	for _, termData := range data {
		termInfo, ok := termData.([]interface{})
		if !ok || len(termInfo) != 3 {
			return nil, fmt.Errorf("invalid term format")
		}

		term, ok := termInfo[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid term format")
		}

		suggestionsData, ok := termInfo[2].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid suggestions format")
		}

		suggestions := make([]SpellCheckSuggestion, 0, len(suggestionsData))
		for _, suggestionData := range suggestionsData {
			suggestionInfo, ok := suggestionData.([]interface{})
			if !ok || len(suggestionInfo) != 2 {
				return nil, fmt.Errorf("invalid suggestion format")
			}

			scoreStr, ok := suggestionInfo[0].(string)
			if !ok {
				return nil, fmt.Errorf("invalid suggestion score format")
			}
			score, err := strconv.ParseFloat(scoreStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid suggestion score value")
			}

			suggestion, ok := suggestionInfo[1].(string)
			if !ok {
				return nil, fmt.Errorf("invalid suggestion format")
			}

			suggestions = append(suggestions, SpellCheckSuggestion{
				Score:      score,
				Suggestion: suggestion,
			})
		}

		results = append(results, SpellCheckResult{
			Term:        term,
			Suggestions: suggestions,
		})
	}

	return results, nil
}

type Document struct {
	ID      string
	Score   *float64
	Payload *string
	SortKey *string
	Fields  map[string]string
}

type FTSearchResult struct {
	Total int64
	Docs  []Document
}

type FTSearchOptions struct {
	NoContent       bool
	Verbatim        bool
	NoStopWords     bool
	WithScores      bool
	WithPayloads    bool
	WithSortKeys    bool
	Filters         []FTSearchFilter
	GeoFilter       []FTSearchGeoFilter
	InKeys          []interface{}
	InFields        []interface{}
	Return          []FTSearchReturn
	Slop            int
	Timeout         int
	InOrder         bool
	Language        string
	Expander        string
	Scorer          string
	ExplainScore    bool
	Payload         string
	SortBy          []FTSearchSortBy
	SortByWithCount bool
	LimitOffset     int
	Limit           int
	Params          map[string]interface{}
	DialectVersion  int
}

type FTSearchCmd struct {
	baseCmd[FTSearchResult]
	options *FTSearchOptions
}

// Ref: https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1541
func (cmd *FTSearchCmd) from(res rueidis.RedisResult) {
	if err := res.Error(); err != nil {
		cmd.SetErr(err)
		return
	}
	anyRes, err := res.ToAny()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetRawVal(anyRes)
	msg, err := res.ToMessage()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if !(msg.IsMap() || msg.IsArray()) {
		panic("res should be either map(RESP3) or array(RESP2)")
	}
	if msg.IsMap() {
		// is RESP3 map
		m, err := msg.ToMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		totalResultsMsg, ok := m["total_results"]
		if !ok {
			cmd.SetErr(fmt.Errorf(`result map should contain key "total_results"`))
		}
		totalResults, err := totalResultsMsg.AsInt64()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		resultsMsg, ok := m["results"]
		if !ok {
			cmd.SetErr(fmt.Errorf(`result map should contain key "results"`))
		}
		resultsArr, err := resultsMsg.ToArray()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		ftSearchResult := FTSearchResult{Total: totalResults, Docs: make([]Document, 0, len(resultsArr))}
		for _, result := range resultsArr {
			resultMap, err := result.ToMap()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			doc := Document{}
			for k, v := range resultMap {
				switch k {
				case "id":
					idStr, err := v.ToString()
					if err != nil {
						cmd.SetErr(err)
						return
					}
					doc.ID = idStr
				case "extra_attributes":
					// doc.ID = resultArr[i+1].String()
					strMap, err := v.AsStrMap()
					if err != nil {
						cmd.SetErr(err)
						return
					}
					doc.Fields = strMap
				case "score":
					score, err := v.AsFloat64()
					if err != nil {
						cmd.SetErr(err)
						return
					}
					doc.Score = &score
				case "payload":
					if !v.IsNil() {
						payload, err := v.ToString()
						if err != nil {
							cmd.SetErr(err)
							return
						}
						doc.Payload = &payload
					}
				case "sortkey":
					if !v.IsNil() {
						sortKey, err := v.ToString()
						if err != nil {
							cmd.SetErr(err)
							return
						}
						doc.SortKey = &sortKey
					}
				}
			}

			ftSearchResult.Docs = append(ftSearchResult.Docs, doc)
		}
		cmd.SetVal(ftSearchResult)
		return
	}
	// is RESP2 array
	data, err := msg.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if len(data) < 1 {
		cmd.SetErr(fmt.Errorf("unexpected search result format"))
		return
	}
	total, err := data[0].AsInt64()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	var results []Document
	for i := 1; i < len(data); {
		docID, err := data[i].ToString()
		if err != nil {
			cmd.SetErr(fmt.Errorf("invalid total results format: %w", err))
			return
		}
		doc := Document{
			ID:     docID,
			Fields: make(map[string]string),
		}
		i++
		if cmd.options != nil {
			if cmd.options.NoContent {
				results = append(results, doc)
				continue
			}
			if cmd.options.WithScores && i < len(data) {
				scoreStr, err := data[i].ToString()
				if err != nil {
					cmd.SetErr(fmt.Errorf("invalid score format: %w", err))
					return
				}
				score, err := strconv.ParseFloat(scoreStr, 64)
				if err != nil {
					cmd.SetErr(fmt.Errorf("invalid score format: %w", err))
					return
				}
				doc.Score = &score
				i++
			}
			if cmd.options.WithPayloads && i < len(data) {
				payload, err := data[i].ToString()
				if err != nil {
					cmd.SetErr(fmt.Errorf("invalid payload format: %w", err))
					return
				}
				doc.Payload = &payload
				i++
			}
			if cmd.options.WithSortKeys && i < len(data) {
				sortKey, err := data[i].ToString()
				if err != nil {
					cmd.SetErr(fmt.Errorf("invalid payload format: %w", err))
					return
				}
				doc.SortKey = &sortKey
				i++
			}
		}
		if i < len(data) {
			fields, err := data[i].ToArray()
			if err != nil {
				cmd.SetErr(fmt.Errorf("invalid document fields format: %w", err))
				return
			}
			for j := 0; j < len(fields); j += 2 {
				key, err := fields[j].ToString()
				if err != nil {
					cmd.SetErr(fmt.Errorf("invalid field key format: %w", err))
					return
				}
				value, err := fields[j+1].ToString()
				if err != nil {
					cmd.SetErr(fmt.Errorf("invalid field value format: %w", err))
					return
				}
				doc.Fields[key] = value
			}
			i++
		}
		results = append(results, doc)
	}
	cmd.SetVal(FTSearchResult{
		Total: total,
		Docs:  results,
	})
}

func newFTSearchCmd(res rueidis.RedisResult, options *FTSearchOptions) *FTSearchCmd {
	cmd := &FTSearchCmd{options: options}
	cmd.from(res)
	return cmd
}

type FTSynUpdateOptions struct {
	SkipInitialScan bool
}

type FTSynDumpCmd struct {
	baseCmd[[]FTSynDumpResult]
}

func (cmd *FTSynDumpCmd) from(res rueidis.RedisResult) {
	if err := res.Error(); err != nil {
		cmd.SetErr(err)
		return
	}
	anyRes, err := res.ToAny()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	cmd.SetRawVal(anyRes)
	msg, err := res.ToMessage()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	if !(msg.IsMap() || msg.IsArray()) {
		panic("res should be either map(RESP3) or array(RESP2)")
	}
	if msg.IsMap() {
		// is RESP3 map
		m, err := msg.ToMap()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		results := make([]FTSynDumpResult, 0, len(m))
		for term, synMsg := range m {
			synonyms, err := synMsg.AsStrSlice()
			if err != nil {
				cmd.SetErr(err)
				return
			}
			results = append(results, FTSynDumpResult{Term: term, Synonyms: synonyms})
		}
		cmd.SetVal(results)
		return
	}
	// is RESP2 array
	arr, err := msg.ToArray()
	if err != nil {
		cmd.SetErr(err)
		return
	}
	results := make([]FTSynDumpResult, 0, len(arr)/2)
	for i := 0; i < len(arr); i += 2 {
		term, err := arr[i].ToString()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		synonyms, err := arr[i+1].AsStrSlice()
		if err != nil {
			cmd.SetErr(err)
			return
		}
		results = append(results, FTSynDumpResult{
			Term:     term,
			Synonyms: synonyms,
		})
	}
	cmd.SetVal(results)
}

func newFTSynDumpCmd(res rueidis.RedisResult) *FTSynDumpCmd {
	cmd := &FTSynDumpCmd{}
	cmd.from(res)
	return cmd
}

type FTSynDumpResult struct {
	Term     string
	Synonyms []string
}
