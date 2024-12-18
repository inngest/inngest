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
	"context"
	"errors"
	"runtime"
	"time"
	"unsafe"

	"github.com/redis/rueidis"
)

// Pipeliner is a mechanism to realise Redis Pipeline technique.
//
// Pipelining is a technique to extremely speed up processing by packing
// operations to batches, send them at once to Redis and read a replies in a
// single step.
// See https://redis.io/topics/pipelining
//
// Pay attention, that Pipeline is not a transaction, so you can get unexpected
// results in case of big pipelines and small read/write timeouts.
// Redis client has retransmission logic in case of timeouts, pipeline
// can be retransmitted and commands can be executed more then once.
// To avoid this: it is good idea to use reasonable bigger read/write timeouts
// depends on your batch size and/or use TxPipeline.
type Pipeliner interface {
	CoreCmdable

	// Len is to obtain the number of commands in the pipeline that have not yet been executed.
	Len() int

	// Do is an API for executing any command.
	// If a certain Redis command is not yet supported, you can use Do to execute it.
	Do(ctx context.Context, args ...interface{}) *Cmd

	// Discard is to discard all commands in the cache that have not yet been executed.
	Discard()

	// Exec is to send all the commands buffered in the pipeline to the redis-server.
	Exec(ctx context.Context) ([]Cmder, error)
}

var _ Pipeliner = (*Pipeline)(nil)

type proxyresult struct {
	err error
	val rueidis.RedisMessage
}

var placeholder = proxyresult{err: errors.New("the pipeline has not been executed")}

type proxy struct {
	rueidis.Client
	cmds []rueidis.Completed
}

func (p *proxy) Do(_ context.Context, cmd rueidis.Completed) rueidis.RedisResult {
	p.cmds = append(p.cmds, cmd)
	return *(*rueidis.RedisResult)(unsafe.Pointer(&placeholder))
}

func newPipeline(real rueidis.Client) *Pipeline {
	return &Pipeline{comp: Compat{client: &proxy{Client: real}, maxp: runtime.GOMAXPROCS(0), pOnly: true}}
}

// Pipeline implements pipelining as described in
// http://redis.io/topics/pipelining.
// Please note: it is not safe for concurrent use by multiple goroutines.
type Pipeline struct {
	comp Compat
	rets []Cmder
}

func (c *Pipeline) Command(ctx context.Context) *CommandsInfoCmd {
	ret := c.comp.Command(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CommandList(ctx context.Context, filter FilterBy) *StringSliceCmd {
	ret := c.comp.CommandList(ctx, filter)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CommandGetKeys(ctx context.Context, commands ...any) *StringSliceCmd {
	ret := c.comp.CommandGetKeys(ctx, commands...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CommandGetKeysAndFlags(ctx context.Context, commands ...any) *KeyFlagsCmd {
	ret := c.comp.CommandGetKeysAndFlags(ctx, commands...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientGetName(ctx context.Context) *StringCmd {
	ret := c.comp.ClientGetName(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Echo(ctx context.Context, message any) *StringCmd {
	ret := c.comp.Echo(ctx, message)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Ping(ctx context.Context) *StatusCmd {
	ret := c.comp.Ping(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Quit(ctx context.Context) *StatusCmd {
	ret := c.comp.Quit(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Del(ctx context.Context, keys ...string) *IntCmd {
	ret := c.comp.Del(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Unlink(ctx context.Context, keys ...string) *IntCmd {
	ret := c.comp.Unlink(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Dump(ctx context.Context, key string) *StringCmd {
	ret := c.comp.Dump(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Exists(ctx context.Context, keys ...string) *IntCmd {
	ret := c.comp.Exists(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Expire(ctx context.Context, key string, expiration time.Duration) *BoolCmd {
	ret := c.comp.Expire(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ExpireAt(ctx context.Context, key string, tm time.Time) *BoolCmd {
	ret := c.comp.ExpireAt(ctx, key, tm)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ExpireTime(ctx context.Context, key string) *DurationCmd {
	ret := c.comp.ExpireTime(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ExpireNX(ctx context.Context, key string, expiration time.Duration) *BoolCmd {
	ret := c.comp.ExpireNX(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ExpireXX(ctx context.Context, key string, expiration time.Duration) *BoolCmd {
	ret := c.comp.ExpireXX(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ExpireGT(ctx context.Context, key string, expiration time.Duration) *BoolCmd {
	ret := c.comp.ExpireGT(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ExpireLT(ctx context.Context, key string, expiration time.Duration) *BoolCmd {
	ret := c.comp.ExpireLT(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Keys(ctx context.Context, pattern string) *StringSliceCmd {
	ret := c.comp.Keys(ctx, pattern)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Migrate(ctx context.Context, host string, port int64, key string, db int64, timeout time.Duration) *StatusCmd {
	ret := c.comp.Migrate(ctx, host, port, key, db, timeout)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Move(ctx context.Context, key string, db int64) *BoolCmd {
	ret := c.comp.Move(ctx, key, db)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ObjectRefCount(ctx context.Context, key string) *IntCmd {
	ret := c.comp.ObjectRefCount(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ObjectEncoding(ctx context.Context, key string) *StringCmd {
	ret := c.comp.ObjectEncoding(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ObjectIdleTime(ctx context.Context, key string) *DurationCmd {
	ret := c.comp.ObjectIdleTime(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Persist(ctx context.Context, key string) *BoolCmd {
	ret := c.comp.Persist(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PExpire(ctx context.Context, key string, expiration time.Duration) *BoolCmd {
	ret := c.comp.PExpire(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PExpireAt(ctx context.Context, key string, tm time.Time) *BoolCmd {
	ret := c.comp.PExpireAt(ctx, key, tm)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PExpireTime(ctx context.Context, key string) *DurationCmd {
	ret := c.comp.PExpireTime(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PTTL(ctx context.Context, key string) *DurationCmd {
	ret := c.comp.PTTL(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RandomKey(ctx context.Context) *StringCmd {
	ret := c.comp.RandomKey(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Rename(ctx context.Context, key, newkey string) *StatusCmd {
	ret := c.comp.Rename(ctx, key, newkey)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RenameNX(ctx context.Context, key, newkey string) *BoolCmd {
	ret := c.comp.RenameNX(ctx, key, newkey)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Restore(ctx context.Context, key string, ttl time.Duration, value string) *StatusCmd {
	ret := c.comp.Restore(ctx, key, ttl, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RestoreReplace(ctx context.Context, key string, ttl time.Duration, value string) *StatusCmd {
	ret := c.comp.RestoreReplace(ctx, key, ttl, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Sort(ctx context.Context, key string, sort Sort) *StringSliceCmd {
	ret := c.comp.Sort(ctx, key, sort)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SortRO(ctx context.Context, key string, sort Sort) *StringSliceCmd {
	ret := c.comp.SortRO(ctx, key, sort)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SortStore(ctx context.Context, key, store string, sort Sort) *IntCmd {
	ret := c.comp.SortStore(ctx, key, store, sort)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SortInterfaces(ctx context.Context, key string, sort Sort) *SliceCmd {
	ret := c.comp.SortInterfaces(ctx, key, sort)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Touch(ctx context.Context, keys ...string) *IntCmd {
	ret := c.comp.Touch(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TTL(ctx context.Context, key string) *DurationCmd {
	ret := c.comp.TTL(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Type(ctx context.Context, key string) *StatusCmd {
	ret := c.comp.Type(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Append(ctx context.Context, key, value string) *IntCmd {
	ret := c.comp.Append(ctx, key, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Decr(ctx context.Context, key string) *IntCmd {
	ret := c.comp.Decr(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) DecrBy(ctx context.Context, key string, decrement int64) *IntCmd {
	ret := c.comp.DecrBy(ctx, key, decrement)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Get(ctx context.Context, key string) *StringCmd {
	ret := c.comp.Get(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GetRange(ctx context.Context, key string, start, end int64) *StringCmd {
	ret := c.comp.GetRange(ctx, key, start, end)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GetSet(ctx context.Context, key string, value any) *StringCmd {
	ret := c.comp.GetSet(ctx, key, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GetEx(ctx context.Context, key string, expiration time.Duration) *StringCmd {
	ret := c.comp.GetEx(ctx, key, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GetDel(ctx context.Context, key string) *StringCmd {
	ret := c.comp.GetDel(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Incr(ctx context.Context, key string) *IntCmd {
	ret := c.comp.Incr(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) IncrBy(ctx context.Context, key string, value int64) *IntCmd {
	ret := c.comp.IncrBy(ctx, key, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) IncrByFloat(ctx context.Context, key string, value float64) *FloatCmd {
	ret := c.comp.IncrByFloat(ctx, key, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) MGet(ctx context.Context, keys ...string) *SliceCmd {
	ret := c.comp.MGet(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) MSet(ctx context.Context, values ...any) *StatusCmd {
	ret := c.comp.MSet(ctx, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) MSetNX(ctx context.Context, values ...any) *BoolCmd {
	ret := c.comp.MSetNX(ctx, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Set(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd {
	ret := c.comp.Set(ctx, key, value, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SetArgs(ctx context.Context, key string, value any, a SetArgs) *StatusCmd {
	ret := c.comp.SetArgs(ctx, key, value, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SetEX(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd {
	ret := c.comp.SetEX(ctx, key, value, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *BoolCmd {
	ret := c.comp.SetNX(ctx, key, value, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SetXX(ctx context.Context, key string, value any, expiration time.Duration) *BoolCmd {
	ret := c.comp.SetXX(ctx, key, value, expiration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SetRange(ctx context.Context, key string, offset int64, value string) *IntCmd {
	ret := c.comp.SetRange(ctx, key, offset, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) StrLen(ctx context.Context, key string) *IntCmd {
	ret := c.comp.StrLen(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Copy(ctx context.Context, sourceKey string, destKey string, db int64, replace bool) *IntCmd {
	ret := c.comp.Copy(ctx, sourceKey, destKey, db, replace)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GetBit(ctx context.Context, key string, offset int64) *IntCmd {
	ret := c.comp.GetBit(ctx, key, offset)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SetBit(ctx context.Context, key string, offset int64, value int64) *IntCmd {
	ret := c.comp.SetBit(ctx, key, offset, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitCount(ctx context.Context, key string, bitCount *BitCount) *IntCmd {
	ret := c.comp.BitCount(ctx, key, bitCount)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitOpAnd(ctx context.Context, destKey string, keys ...string) *IntCmd {
	ret := c.comp.BitOpAnd(ctx, destKey, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitOpOr(ctx context.Context, destKey string, keys ...string) *IntCmd {
	ret := c.comp.BitOpOr(ctx, destKey, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitOpXor(ctx context.Context, destKey string, keys ...string) *IntCmd {
	ret := c.comp.BitOpXor(ctx, destKey, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitOpNot(ctx context.Context, destKey string, key string) *IntCmd {
	ret := c.comp.BitOpNot(ctx, destKey, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitPos(ctx context.Context, key string, bit int64, pos ...int64) *IntCmd {
	ret := c.comp.BitPos(ctx, key, bit, pos...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitPosSpan(ctx context.Context, key string, bit int64, start, end int64, span string) *IntCmd {
	ret := c.comp.BitPosSpan(ctx, key, bit, start, end, span)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BitField(ctx context.Context, key string, args ...any) *IntSliceCmd {
	ret := c.comp.BitField(ctx, key, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Scan(ctx context.Context, cursor uint64, match string, count int64) *ScanCmd {
	ret := c.comp.Scan(ctx, cursor, match, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ScanType(ctx context.Context, cursor uint64, match string, count int64, keyType string) *ScanCmd {
	ret := c.comp.ScanType(ctx, cursor, match, count, keyType)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd {
	ret := c.comp.SScan(ctx, key, cursor, match, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd {
	ret := c.comp.HScan(ctx, key, cursor, match, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd {
	ret := c.comp.ZScan(ctx, key, cursor, match, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HDel(ctx context.Context, key string, fields ...string) *IntCmd {
	ret := c.comp.HDel(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HExists(ctx context.Context, key, field string) *BoolCmd {
	ret := c.comp.HExists(ctx, key, field)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HGet(ctx context.Context, key, field string) *StringCmd {
	ret := c.comp.HGet(ctx, key, field)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HGetAll(ctx context.Context, key string) *StringStringMapCmd {
	ret := c.comp.HGetAll(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HIncrBy(ctx context.Context, key, field string, incr int64) *IntCmd {
	ret := c.comp.HIncrBy(ctx, key, field, incr)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HIncrByFloat(ctx context.Context, key, field string, incr float64) *FloatCmd {
	ret := c.comp.HIncrByFloat(ctx, key, field, incr)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HKeys(ctx context.Context, key string) *StringSliceCmd {
	ret := c.comp.HKeys(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HLen(ctx context.Context, key string) *IntCmd {
	ret := c.comp.HLen(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HMGet(ctx context.Context, key string, fields ...string) *SliceCmd {
	ret := c.comp.HMGet(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HSet(ctx context.Context, key string, values ...any) *IntCmd {
	ret := c.comp.HSet(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HMSet(ctx context.Context, key string, values ...any) *BoolCmd {
	ret := c.comp.HMSet(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HSetNX(ctx context.Context, key, field string, value any) *BoolCmd {
	ret := c.comp.HSetNX(ctx, key, field, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HVals(ctx context.Context, key string) *StringSliceCmd {
	ret := c.comp.HVals(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HRandField(ctx context.Context, key string, count int64) *StringSliceCmd {
	ret := c.comp.HRandField(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HRandFieldWithValues(ctx context.Context, key string, count int64) *KeyValueSliceCmd {
	ret := c.comp.HRandFieldWithValues(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *IntSliceCmd {
	ret := c.comp.HExpire(ctx, key, expiration, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	ret := c.comp.HExpireWithArgs(ctx, key, expiration, expirationArgs, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *IntSliceCmd {
	ret := c.comp.HPExpire(ctx, key, expiration, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	ret := c.comp.HPExpireWithArgs(ctx, key, expiration, expirationArgs, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *IntSliceCmd {
	ret := c.comp.HExpireAt(ctx, key, tm, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	ret := c.comp.HExpireAtWithArgs(ctx, key, tm, expirationArgs, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *IntSliceCmd {
	ret := c.comp.HPExpireAt(ctx, key, tm, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	ret := c.comp.HPExpireAtWithArgs(ctx, key, tm, expirationArgs, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPersist(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	ret := c.comp.HPersist(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HExpireTime(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	ret := c.comp.HExpireTime(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPExpireTime(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	ret := c.comp.HPExpireTime(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HTTL(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	ret := c.comp.HTTL(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) HPTTL(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	ret := c.comp.HPTTL(ctx, key, fields...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BLPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd {
	ret := c.comp.BLPop(ctx, timeout, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BLMPop(ctx context.Context, timeout time.Duration, direction string, count int64, keys ...string) *KeyValuesCmd {
	ret := c.comp.BLMPop(ctx, timeout, direction, count, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd {
	ret := c.comp.BRPop(ctx, timeout, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BRPopLPush(ctx context.Context, source, destination string, timeout time.Duration) *StringCmd {
	ret := c.comp.BRPopLPush(ctx, source, destination, timeout)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LIndex(ctx context.Context, key string, index int64) *StringCmd {
	ret := c.comp.LIndex(ctx, key, index)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LInsert(ctx context.Context, key, op string, pivot, value any) *IntCmd {
	ret := c.comp.LInsert(ctx, key, op, pivot, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LInsertBefore(ctx context.Context, key string, pivot, value any) *IntCmd {
	ret := c.comp.LInsertBefore(ctx, key, pivot, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LInsertAfter(ctx context.Context, key string, pivot, value any) *IntCmd {
	ret := c.comp.LInsertAfter(ctx, key, pivot, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LLen(ctx context.Context, key string) *IntCmd {
	ret := c.comp.LLen(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LMPop(ctx context.Context, direction string, count int64, keys ...string) *KeyValuesCmd {
	ret := c.comp.LMPop(ctx, direction, count, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LPop(ctx context.Context, key string) *StringCmd {
	ret := c.comp.LPop(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LPopCount(ctx context.Context, key string, count int64) *StringSliceCmd {
	ret := c.comp.LPopCount(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LPos(ctx context.Context, key string, value string, args LPosArgs) *IntCmd {
	ret := c.comp.LPos(ctx, key, value, args)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LPosCount(ctx context.Context, key string, value string, count int64, args LPosArgs) *IntSliceCmd {
	ret := c.comp.LPosCount(ctx, key, value, count, args)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LPush(ctx context.Context, key string, values ...any) *IntCmd {
	ret := c.comp.LPush(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LPushX(ctx context.Context, key string, values ...any) *IntCmd {
	ret := c.comp.LPushX(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	ret := c.comp.LRange(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LRem(ctx context.Context, key string, count int64, value any) *IntCmd {
	ret := c.comp.LRem(ctx, key, count, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LSet(ctx context.Context, key string, index int64, value any) *StatusCmd {
	ret := c.comp.LSet(ctx, key, index, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LTrim(ctx context.Context, key string, start, stop int64) *StatusCmd {
	ret := c.comp.LTrim(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RPop(ctx context.Context, key string) *StringCmd {
	ret := c.comp.RPop(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RPopCount(ctx context.Context, key string, count int64) *StringSliceCmd {
	ret := c.comp.RPopCount(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RPopLPush(ctx context.Context, source, destination string) *StringCmd {
	ret := c.comp.RPopLPush(ctx, source, destination)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RPush(ctx context.Context, key string, values ...any) *IntCmd {
	ret := c.comp.RPush(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) RPushX(ctx context.Context, key string, values ...any) *IntCmd {
	ret := c.comp.RPushX(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LMove(ctx context.Context, source, destination, srcpos, destpos string) *StringCmd {
	ret := c.comp.LMove(ctx, source, destination, srcpos, destpos)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BLMove(ctx context.Context, source, destination, srcpos, destpos string, timeout time.Duration) *StringCmd {
	ret := c.comp.BLMove(ctx, source, destination, srcpos, destpos, timeout)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SAdd(ctx context.Context, key string, members ...any) *IntCmd {
	ret := c.comp.SAdd(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SCard(ctx context.Context, key string) *IntCmd {
	ret := c.comp.SCard(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SDiff(ctx context.Context, keys ...string) *StringSliceCmd {
	ret := c.comp.SDiff(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SDiffStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	ret := c.comp.SDiffStore(ctx, destination, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SInter(ctx context.Context, keys ...string) *StringSliceCmd {
	ret := c.comp.SInter(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SInterCard(ctx context.Context, limit int64, keys ...string) *IntCmd {
	ret := c.comp.SInterCard(ctx, limit, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SInterStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	ret := c.comp.SInterStore(ctx, destination, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SIsMember(ctx context.Context, key string, member any) *BoolCmd {
	ret := c.comp.SIsMember(ctx, key, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SMIsMember(ctx context.Context, key string, members ...any) *BoolSliceCmd {
	ret := c.comp.SMIsMember(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SMembers(ctx context.Context, key string) *StringSliceCmd {
	ret := c.comp.SMembers(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SMembersMap(ctx context.Context, key string) *StringStructMapCmd {
	ret := c.comp.SMembersMap(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SMove(ctx context.Context, source, destination string, member any) *BoolCmd {
	ret := c.comp.SMove(ctx, source, destination, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SPop(ctx context.Context, key string) *StringCmd {
	ret := c.comp.SPop(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SPopN(ctx context.Context, key string, count int64) *StringSliceCmd {
	ret := c.comp.SPopN(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SRandMember(ctx context.Context, key string) *StringCmd {
	ret := c.comp.SRandMember(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SRandMemberN(ctx context.Context, key string, count int64) *StringSliceCmd {
	ret := c.comp.SRandMemberN(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SRem(ctx context.Context, key string, members ...any) *IntCmd {
	ret := c.comp.SRem(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SUnion(ctx context.Context, keys ...string) *StringSliceCmd {
	ret := c.comp.SUnion(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SUnionStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	ret := c.comp.SUnionStore(ctx, destination, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XAdd(ctx context.Context, a XAddArgs) *StringCmd {
	ret := c.comp.XAdd(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XDel(ctx context.Context, stream string, ids ...string) *IntCmd {
	ret := c.comp.XDel(ctx, stream, ids...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XLen(ctx context.Context, stream string) *IntCmd {
	ret := c.comp.XLen(ctx, stream)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XRange(ctx context.Context, stream, start, stop string) *XMessageSliceCmd {
	ret := c.comp.XRange(ctx, stream, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XRangeN(ctx context.Context, stream, start, stop string, count int64) *XMessageSliceCmd {
	ret := c.comp.XRangeN(ctx, stream, start, stop, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XRevRange(ctx context.Context, stream string, start, stop string) *XMessageSliceCmd {
	ret := c.comp.XRevRange(ctx, stream, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XRevRangeN(ctx context.Context, stream string, start, stop string, count int64) *XMessageSliceCmd {
	ret := c.comp.XRevRangeN(ctx, stream, start, stop, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XRead(ctx context.Context, a XReadArgs) *XStreamSliceCmd {
	ret := c.comp.XRead(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XReadStreams(ctx context.Context, streams ...string) *XStreamSliceCmd {
	ret := c.comp.XReadStreams(ctx, streams...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XGroupCreate(ctx context.Context, stream, group, start string) *StatusCmd {
	ret := c.comp.XGroupCreate(ctx, stream, group, start)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XGroupCreateMkStream(ctx context.Context, stream, group, start string) *StatusCmd {
	ret := c.comp.XGroupCreateMkStream(ctx, stream, group, start)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XGroupSetID(ctx context.Context, stream, group, start string) *StatusCmd {
	ret := c.comp.XGroupSetID(ctx, stream, group, start)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XGroupDestroy(ctx context.Context, stream, group string) *IntCmd {
	ret := c.comp.XGroupDestroy(ctx, stream, group)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XGroupCreateConsumer(ctx context.Context, stream, group, consumer string) *IntCmd {
	ret := c.comp.XGroupCreateConsumer(ctx, stream, group, consumer)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *IntCmd {
	ret := c.comp.XGroupDelConsumer(ctx, stream, group, consumer)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XReadGroup(ctx context.Context, a XReadGroupArgs) *XStreamSliceCmd {
	ret := c.comp.XReadGroup(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XAck(ctx context.Context, stream, group string, ids ...string) *IntCmd {
	ret := c.comp.XAck(ctx, stream, group, ids...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XPending(ctx context.Context, stream, group string) *XPendingCmd {
	ret := c.comp.XPending(ctx, stream, group)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XPendingExt(ctx context.Context, a XPendingExtArgs) *XPendingExtCmd {
	ret := c.comp.XPendingExt(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XClaim(ctx context.Context, a XClaimArgs) *XMessageSliceCmd {
	ret := c.comp.XClaim(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XClaimJustID(ctx context.Context, a XClaimArgs) *StringSliceCmd {
	ret := c.comp.XClaimJustID(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XAutoClaim(ctx context.Context, a XAutoClaimArgs) *XAutoClaimCmd {
	ret := c.comp.XAutoClaim(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XAutoClaimJustID(ctx context.Context, a XAutoClaimArgs) *XAutoClaimJustIDCmd {
	ret := c.comp.XAutoClaimJustID(ctx, a)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XTrimMaxLen(ctx context.Context, key string, maxLen int64) *IntCmd {
	ret := c.comp.XTrimMaxLen(ctx, key, maxLen)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XTrimMaxLenApprox(ctx context.Context, key string, maxLen, limit int64) *IntCmd {
	ret := c.comp.XTrimMaxLenApprox(ctx, key, maxLen, limit)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XTrimMinID(ctx context.Context, key string, minID string) *IntCmd {
	ret := c.comp.XTrimMinID(ctx, key, minID)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XTrimMinIDApprox(ctx context.Context, key string, minID string, limit int64) *IntCmd {
	ret := c.comp.XTrimMinIDApprox(ctx, key, minID, limit)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XInfoGroups(ctx context.Context, key string) *XInfoGroupsCmd {
	ret := c.comp.XInfoGroups(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XInfoStream(ctx context.Context, key string) *XInfoStreamCmd {
	ret := c.comp.XInfoStream(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XInfoStreamFull(ctx context.Context, key string, count int64) *XInfoStreamFullCmd {
	ret := c.comp.XInfoStreamFull(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) XInfoConsumers(ctx context.Context, key string, group string) *XInfoConsumersCmd {
	ret := c.comp.XInfoConsumers(ctx, key, group)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) *ZWithKeyCmd {
	ret := c.comp.BZPopMax(ctx, timeout, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) *ZWithKeyCmd {
	ret := c.comp.BZPopMin(ctx, timeout, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BZMPop(ctx context.Context, timeout time.Duration, order string, count int64, keys ...string) *ZSliceWithKeyCmd {
	ret := c.comp.BZMPop(ctx, timeout, order, count, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAdd(ctx context.Context, key string, members ...Z) *IntCmd {
	ret := c.comp.ZAdd(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAddLT(ctx context.Context, key string, members ...Z) *IntCmd {
	ret := c.comp.ZAddLT(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAddGT(ctx context.Context, key string, members ...Z) *IntCmd {
	ret := c.comp.ZAddGT(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAddNX(ctx context.Context, key string, members ...Z) *IntCmd {
	ret := c.comp.ZAddNX(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAddXX(ctx context.Context, key string, members ...Z) *IntCmd {
	ret := c.comp.ZAddXX(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAddArgs(ctx context.Context, key string, args ZAddArgs) *IntCmd {
	ret := c.comp.ZAddArgs(ctx, key, args)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZAddArgsIncr(ctx context.Context, key string, args ZAddArgs) *FloatCmd {
	ret := c.comp.ZAddArgsIncr(ctx, key, args)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZCard(ctx context.Context, key string) *IntCmd {
	ret := c.comp.ZCard(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZCount(ctx context.Context, key, min, max string) *IntCmd {
	ret := c.comp.ZCount(ctx, key, min, max)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZLexCount(ctx context.Context, key, min, max string) *IntCmd {
	ret := c.comp.ZLexCount(ctx, key, min, max)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZIncrBy(ctx context.Context, key string, increment float64, member string) *FloatCmd {
	ret := c.comp.ZIncrBy(ctx, key, increment, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZInter(ctx context.Context, store ZStore) *StringSliceCmd {
	ret := c.comp.ZInter(ctx, store)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZInterWithScores(ctx context.Context, store ZStore) *ZSliceCmd {
	ret := c.comp.ZInterWithScores(ctx, store)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZInterCard(ctx context.Context, limit int64, keys ...string) *IntCmd {
	ret := c.comp.ZInterCard(ctx, limit, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZInterStore(ctx context.Context, destination string, store ZStore) *IntCmd {
	ret := c.comp.ZInterStore(ctx, destination, store)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZMPop(ctx context.Context, order string, count int64, keys ...string) *ZSliceWithKeyCmd {
	ret := c.comp.ZMPop(ctx, order, count, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZMScore(ctx context.Context, key string, members ...string) *FloatSliceCmd {
	ret := c.comp.ZMScore(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZPopMax(ctx context.Context, key string, count ...int64) *ZSliceCmd {
	ret := c.comp.ZPopMax(ctx, key, count...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZPopMin(ctx context.Context, key string, count ...int64) *ZSliceCmd {
	ret := c.comp.ZPopMin(ctx, key, count...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	ret := c.comp.ZRange(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd {
	ret := c.comp.ZRangeWithScores(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	ret := c.comp.ZRangeByScore(ctx, key, opt)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	ret := c.comp.ZRangeByLex(ctx, key, opt)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd {
	ret := c.comp.ZRangeByScoreWithScores(ctx, key, opt)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeArgs(ctx context.Context, z ZRangeArgs) *StringSliceCmd {
	ret := c.comp.ZRangeArgs(ctx, z)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeArgsWithScores(ctx context.Context, z ZRangeArgs) *ZSliceCmd {
	ret := c.comp.ZRangeArgsWithScores(ctx, z)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRangeStore(ctx context.Context, dst string, z ZRangeArgs) *IntCmd {
	ret := c.comp.ZRangeStore(ctx, dst, z)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRank(ctx context.Context, key, member string) *IntCmd {
	ret := c.comp.ZRank(ctx, key, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd {
	ret := c.comp.ZRankWithScore(ctx, key, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRem(ctx context.Context, key string, members ...any) *IntCmd {
	ret := c.comp.ZRem(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *IntCmd {
	ret := c.comp.ZRemRangeByRank(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRemRangeByScore(ctx context.Context, key, min, max string) *IntCmd {
	ret := c.comp.ZRemRangeByScore(ctx, key, min, max)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRemRangeByLex(ctx context.Context, key, min, max string) *IntCmd {
	ret := c.comp.ZRemRangeByLex(ctx, key, min, max)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	ret := c.comp.ZRevRange(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd {
	ret := c.comp.ZRevRangeWithScores(ctx, key, start, stop)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	ret := c.comp.ZRevRangeByScore(ctx, key, opt)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	ret := c.comp.ZRevRangeByLex(ctx, key, opt)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd {
	ret := c.comp.ZRevRangeByScoreWithScores(ctx, key, opt)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRank(ctx context.Context, key, member string) *IntCmd {
	ret := c.comp.ZRevRank(ctx, key, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRevRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd {
	ret := c.comp.ZRevRankWithScore(ctx, key, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZScore(ctx context.Context, key, member string) *FloatCmd {
	ret := c.comp.ZScore(ctx, key, member)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZUnionStore(ctx context.Context, dest string, store ZStore) *IntCmd {
	ret := c.comp.ZUnionStore(ctx, dest, store)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRandMember(ctx context.Context, key string, count int64) *StringSliceCmd {
	ret := c.comp.ZRandMember(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZRandMemberWithScores(ctx context.Context, key string, count int64) *ZSliceCmd {
	ret := c.comp.ZRandMemberWithScores(ctx, key, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZUnion(ctx context.Context, store ZStore) *StringSliceCmd {
	ret := c.comp.ZUnion(ctx, store)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZUnionWithScores(ctx context.Context, store ZStore) *ZSliceCmd {
	ret := c.comp.ZUnionWithScores(ctx, store)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZDiff(ctx context.Context, keys ...string) *StringSliceCmd {
	ret := c.comp.ZDiff(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZDiffWithScores(ctx context.Context, keys ...string) *ZSliceCmd {
	ret := c.comp.ZDiffWithScores(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ZDiffStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	ret := c.comp.ZDiffStore(ctx, destination, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PFAdd(ctx context.Context, key string, els ...any) *IntCmd {
	ret := c.comp.PFAdd(ctx, key, els...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PFCount(ctx context.Context, keys ...string) *IntCmd {
	ret := c.comp.PFCount(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PFMerge(ctx context.Context, dest string, keys ...string) *StatusCmd {
	ret := c.comp.PFMerge(ctx, dest, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BgRewriteAOF(ctx context.Context) *StatusCmd {
	ret := c.comp.BgRewriteAOF(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BgSave(ctx context.Context) *StatusCmd {
	ret := c.comp.BgSave(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientKill(ctx context.Context, ipPort string) *StatusCmd {
	ret := c.comp.ClientKill(ctx, ipPort)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientKillByFilter(ctx context.Context, keys ...string) *IntCmd {
	ret := c.comp.ClientKillByFilter(ctx, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientList(ctx context.Context) *StringCmd {
	ret := c.comp.ClientList(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientPause(ctx context.Context, dur time.Duration) *BoolCmd {
	ret := c.comp.ClientPause(ctx, dur)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientUnpause(ctx context.Context) *BoolCmd {
	ret := c.comp.ClientUnpause(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientID(ctx context.Context) *IntCmd {
	ret := c.comp.ClientID(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientUnblock(ctx context.Context, id int64) *IntCmd {
	ret := c.comp.ClientUnblock(ctx, id)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClientUnblockWithError(ctx context.Context, id int64) *IntCmd {
	ret := c.comp.ClientUnblockWithError(ctx, id)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ConfigGet(ctx context.Context, parameter string) *StringStringMapCmd {
	ret := c.comp.ConfigGet(ctx, parameter)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ConfigResetStat(ctx context.Context) *StatusCmd {
	ret := c.comp.ConfigResetStat(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ConfigSet(ctx context.Context, parameter, value string) *StatusCmd {
	ret := c.comp.ConfigSet(ctx, parameter, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ConfigRewrite(ctx context.Context) *StatusCmd {
	ret := c.comp.ConfigRewrite(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) DBSize(ctx context.Context) *IntCmd {
	ret := c.comp.DBSize(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FlushAll(ctx context.Context) *StatusCmd {
	ret := c.comp.FlushAll(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FlushAllAsync(ctx context.Context) *StatusCmd {
	ret := c.comp.FlushAllAsync(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FlushDB(ctx context.Context) *StatusCmd {
	ret := c.comp.FlushDB(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FlushDBAsync(ctx context.Context) *StatusCmd {
	ret := c.comp.FlushDBAsync(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Info(ctx context.Context, section ...string) *StringCmd {
	ret := c.comp.Info(ctx, section...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) LastSave(ctx context.Context) *IntCmd {
	ret := c.comp.LastSave(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Save(ctx context.Context) *StatusCmd {
	ret := c.comp.Save(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Shutdown(ctx context.Context) *StatusCmd {
	ret := c.comp.Shutdown(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ShutdownSave(ctx context.Context) *StatusCmd {
	ret := c.comp.ShutdownSave(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ShutdownNoSave(ctx context.Context) *StatusCmd {
	ret := c.comp.ShutdownNoSave(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Time(ctx context.Context) *TimeCmd {
	ret := c.comp.Time(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) DebugObject(ctx context.Context, key string) *StringCmd {
	ret := c.comp.DebugObject(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ReadOnly(ctx context.Context) *StatusCmd {
	ret := c.comp.ReadOnly(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ReadWrite(ctx context.Context) *StatusCmd {
	ret := c.comp.ReadWrite(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) MemoryUsage(ctx context.Context, key string, samples ...int64) *IntCmd {
	ret := c.comp.MemoryUsage(ctx, key, samples...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Eval(ctx context.Context, script string, keys []string, args ...any) *Cmd {
	ret := c.comp.Eval(ctx, script, keys, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd {
	ret := c.comp.EvalSha(ctx, sha1, keys, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) EvalRO(ctx context.Context, script string, keys []string, args ...any) *Cmd {
	ret := c.comp.EvalRO(ctx, script, keys, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd {
	ret := c.comp.EvalShaRO(ctx, sha1, keys, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd {
	ret := c.comp.ScriptExists(ctx, hashes...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ScriptFlush(ctx context.Context) *StatusCmd {
	ret := c.comp.ScriptFlush(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ScriptKill(ctx context.Context) *StatusCmd {
	ret := c.comp.ScriptKill(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ScriptLoad(ctx context.Context, script string) *StringCmd {
	ret := c.comp.ScriptLoad(ctx, script)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionLoad(ctx context.Context, code string) *StringCmd {
	ret := c.comp.FunctionLoad(ctx, code)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionLoadReplace(ctx context.Context, code string) *StringCmd {
	ret := c.comp.FunctionLoadReplace(ctx, code)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionDelete(ctx context.Context, libName string) *StringCmd {
	ret := c.comp.FunctionDelete(ctx, libName)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionFlush(ctx context.Context) *StringCmd {
	ret := c.comp.FunctionFlush(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionKill(ctx context.Context) *StringCmd {
	ret := c.comp.FunctionKill(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionFlushAsync(ctx context.Context) *StringCmd {
	ret := c.comp.FunctionFlushAsync(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionList(ctx context.Context, q FunctionListQuery) *FunctionListCmd {
	ret := c.comp.FunctionList(ctx, q)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionDump(ctx context.Context) *StringCmd {
	ret := c.comp.FunctionDump(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FunctionRestore(ctx context.Context, libDump string) *StringCmd {
	ret := c.comp.FunctionRestore(ctx, libDump)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FCall(ctx context.Context, function string, keys []string, args ...any) *Cmd {
	ret := c.comp.FCall(ctx, function, keys, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FCallRO(ctx context.Context, function string, keys []string, args ...any) *Cmd {
	ret := c.comp.FCallRO(ctx, function, keys, args...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) Publish(ctx context.Context, channel string, message any) *IntCmd {
	ret := c.comp.Publish(ctx, channel, message)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) SPublish(ctx context.Context, channel string, message any) *IntCmd {
	ret := c.comp.SPublish(ctx, channel, message)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PubSubChannels(ctx context.Context, pattern string) *StringSliceCmd {
	ret := c.comp.PubSubChannels(ctx, pattern)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PubSubNumSub(ctx context.Context, channels ...string) *StringIntMapCmd {
	ret := c.comp.PubSubNumSub(ctx, channels...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PubSubNumPat(ctx context.Context) *IntCmd {
	ret := c.comp.PubSubNumPat(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PubSubShardChannels(ctx context.Context, pattern string) *StringSliceCmd {
	ret := c.comp.PubSubShardChannels(ctx, pattern)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) PubSubShardNumSub(ctx context.Context, channels ...string) *StringIntMapCmd {
	ret := c.comp.PubSubShardNumSub(ctx, channels...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterSlots(ctx context.Context) *ClusterSlotsCmd {
	ret := c.comp.ClusterSlots(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterShards(ctx context.Context) *ClusterShardsCmd {
	ret := c.comp.ClusterShards(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterNodes(ctx context.Context) *StringCmd {
	ret := c.comp.ClusterNodes(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterMeet(ctx context.Context, host string, port int64) *StatusCmd {
	ret := c.comp.ClusterMeet(ctx, host, port)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterForget(ctx context.Context, nodeID string) *StatusCmd {
	ret := c.comp.ClusterForget(ctx, nodeID)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterReplicate(ctx context.Context, nodeID string) *StatusCmd {
	ret := c.comp.ClusterReplicate(ctx, nodeID)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterResetSoft(ctx context.Context) *StatusCmd {
	ret := c.comp.ClusterResetSoft(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterResetHard(ctx context.Context) *StatusCmd {
	ret := c.comp.ClusterResetHard(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterInfo(ctx context.Context) *StringCmd {
	ret := c.comp.ClusterInfo(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterKeySlot(ctx context.Context, key string) *IntCmd {
	ret := c.comp.ClusterKeySlot(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterGetKeysInSlot(ctx context.Context, slot int64, count int64) *StringSliceCmd {
	ret := c.comp.ClusterGetKeysInSlot(ctx, slot, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterCountFailureReports(ctx context.Context, nodeID string) *IntCmd {
	ret := c.comp.ClusterCountFailureReports(ctx, nodeID)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterCountKeysInSlot(ctx context.Context, slot int64) *IntCmd {
	ret := c.comp.ClusterCountKeysInSlot(ctx, slot)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterDelSlots(ctx context.Context, slots ...int64) *StatusCmd {
	ret := c.comp.ClusterDelSlots(ctx, slots...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterDelSlotsRange(ctx context.Context, min, max int64) *StatusCmd {
	ret := c.comp.ClusterDelSlotsRange(ctx, min, max)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterSaveConfig(ctx context.Context) *StatusCmd {
	ret := c.comp.ClusterSaveConfig(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterSlaves(ctx context.Context, nodeID string) *StringSliceCmd {
	ret := c.comp.ClusterSlaves(ctx, nodeID)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterFailover(ctx context.Context) *StatusCmd {
	ret := c.comp.ClusterFailover(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterAddSlots(ctx context.Context, slots ...int64) *StatusCmd {
	ret := c.comp.ClusterAddSlots(ctx, slots...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ClusterAddSlotsRange(ctx context.Context, min, max int64) *StatusCmd {
	ret := c.comp.ClusterAddSlotsRange(ctx, min, max)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoAdd(ctx context.Context, key string, geoLocation ...GeoLocation) *IntCmd {
	ret := c.comp.GeoAdd(ctx, key, geoLocation...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoPos(ctx context.Context, key string, members ...string) *GeoPosCmd {
	ret := c.comp.GeoPos(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoRadius(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *GeoLocationCmd {
	ret := c.comp.GeoRadius(ctx, key, longitude, latitude, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *IntCmd {
	ret := c.comp.GeoRadiusStore(ctx, key, longitude, latitude, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoRadiusByMember(ctx context.Context, key, member string, query GeoRadiusQuery) *GeoLocationCmd {
	ret := c.comp.GeoRadiusByMember(ctx, key, member, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoRadiusByMemberStore(ctx context.Context, key, member string, query GeoRadiusQuery) *IntCmd {
	ret := c.comp.GeoRadiusByMemberStore(ctx, key, member, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoSearch(ctx context.Context, key string, q GeoSearchQuery) *StringSliceCmd {
	ret := c.comp.GeoSearch(ctx, key, q)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoSearchLocation(ctx context.Context, key string, q GeoSearchLocationQuery) *GeoLocationCmd {
	ret := c.comp.GeoSearchLocation(ctx, key, q)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoSearchStore(ctx context.Context, key, store string, q GeoSearchStoreQuery) *IntCmd {
	ret := c.comp.GeoSearchStore(ctx, key, store, q)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoDist(ctx context.Context, key string, member1, member2, unit string) *FloatCmd {
	ret := c.comp.GeoDist(ctx, key, member1, member2, unit)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) GeoHash(ctx context.Context, key string, members ...string) *StringSliceCmd {
	ret := c.comp.GeoHash(ctx, key, members...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) ACLDryRun(ctx context.Context, username string, command ...any) *StringCmd {
	ret := c.comp.ACLDryRun(ctx, username, command...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFunctionLoad(ctx context.Context, lib string) *StatusCmd {
	ret := c.comp.TFunctionLoad(ctx, lib)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFunctionLoadArgs(ctx context.Context, lib string, options *TFunctionLoadOptions) *StatusCmd {
	ret := c.comp.TFunctionLoadArgs(ctx, lib, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFunctionDelete(ctx context.Context, libName string) *StatusCmd {
	ret := c.comp.TFunctionDelete(ctx, libName)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFunctionList(ctx context.Context) *MapStringInterfaceSliceCmd {
	ret := c.comp.TFunctionList(ctx)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFunctionListArgs(ctx context.Context, options *TFunctionListOptions) *MapStringInterfaceSliceCmd {
	ret := c.comp.TFunctionListArgs(ctx, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFCall(ctx context.Context, libName string, funcName string, numKeys int) *Cmd {
	ret := c.comp.TFCall(ctx, libName, funcName, numKeys)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFCallArgs(ctx context.Context, libName string, funcName string, numKeys int, options *TFCallOptions) *Cmd {
	ret := c.comp.TFCallArgs(ctx, libName, funcName, numKeys, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFCallASYNC(ctx context.Context, libName string, funcName string, numKeys int) *Cmd {
	ret := c.comp.TFCallASYNC(ctx, libName, funcName, numKeys)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TFCallASYNCArgs(ctx context.Context, libName string, funcName string, numKeys int, options *TFCallOptions) *Cmd {
	ret := c.comp.TFCallASYNCArgs(ctx, libName, funcName, numKeys, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFAdd(ctx context.Context, key string, element interface{}) *BoolCmd {
	ret := c.comp.BFAdd(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFCard(ctx context.Context, key string) *IntCmd {
	ret := c.comp.BFCard(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFExists(ctx context.Context, key string, element interface{}) *BoolCmd {
	ret := c.comp.BFExists(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfo(ctx context.Context, key string) *BFInfoCmd {
	ret := c.comp.BFInfo(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfoArg(ctx context.Context, key, option string) *BFInfoCmd {
	ret := c.comp.BFInfoArg(ctx, key, option)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfoCapacity(ctx context.Context, key string) *BFInfoCmd {
	ret := c.comp.BFInfoCapacity(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfoSize(ctx context.Context, key string) *BFInfoCmd {
	ret := c.comp.BFInfoSize(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfoFilters(ctx context.Context, key string) *BFInfoCmd {
	ret := c.comp.BFInfoFilters(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfoItems(ctx context.Context, key string) *BFInfoCmd {
	ret := c.comp.BFInfoItems(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInfoExpansion(ctx context.Context, key string) *BFInfoCmd {
	ret := c.comp.BFInfoExpansion(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFInsert(ctx context.Context, key string, options *BFInsertOptions, elements ...interface{}) *BoolSliceCmd {
	ret := c.comp.BFInsert(ctx, key, options, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFMAdd(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	ret := c.comp.BFMAdd(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFMExists(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	ret := c.comp.BFMExists(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFReserve(ctx context.Context, key string, errorRate float64, capacity int64) *StatusCmd {
	ret := c.comp.BFReserve(ctx, key, errorRate, capacity)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFReserveExpansion(ctx context.Context, key string, errorRate float64, capacity, expansion int64) *StatusCmd {
	ret := c.comp.BFReserveExpansion(ctx, key, errorRate, capacity, expansion)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFReserveNonScaling(ctx context.Context, key string, errorRate float64, capacity int64) *StatusCmd {
	ret := c.comp.BFReserveNonScaling(ctx, key, errorRate, capacity)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFReserveWithArgs(ctx context.Context, key string, options *BFReserveOptions) *StatusCmd {
	ret := c.comp.BFReserveWithArgs(ctx, key, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFScanDump(ctx context.Context, key string, iterator int64) *ScanDumpCmd {
	ret := c.comp.BFScanDump(ctx, key, iterator)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) BFLoadChunk(ctx context.Context, key string, iterator int64, data interface{}) *StatusCmd {
	ret := c.comp.BFLoadChunk(ctx, key, iterator, data)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFAdd(ctx context.Context, key string, element interface{}) *BoolCmd {
	ret := c.comp.CFAdd(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFAddNX(ctx context.Context, key string, element interface{}) *BoolCmd {
	ret := c.comp.CFAddNX(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFCount(ctx context.Context, key string, element interface{}) *IntCmd {
	ret := c.comp.CFCount(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFDel(ctx context.Context, key string, element interface{}) *BoolCmd {
	ret := c.comp.CFDel(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFExists(ctx context.Context, key string, element interface{}) *BoolCmd {
	ret := c.comp.CFExists(ctx, key, element)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFInfo(ctx context.Context, key string) *CFInfoCmd {
	ret := c.comp.CFInfo(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFInsert(ctx context.Context, key string, options *CFInsertOptions, elements ...interface{}) *BoolSliceCmd {
	ret := c.comp.CFInsert(ctx, key, options, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFInsertNX(ctx context.Context, key string, options *CFInsertOptions, elements ...interface{}) *IntSliceCmd {
	ret := c.comp.CFInsertNX(ctx, key, options, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFMExists(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	ret := c.comp.CFMExists(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFReserve(ctx context.Context, key string, capacity int64) *StatusCmd {
	ret := c.comp.CFReserve(ctx, key, capacity)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFReserveWithArgs(ctx context.Context, key string, options *CFReserveOptions) *StatusCmd {
	ret := c.comp.CFReserveWithArgs(ctx, key, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFReserveExpansion(ctx context.Context, key string, capacity int64, expansion int64) *StatusCmd {
	ret := c.comp.CFReserveExpansion(ctx, key, capacity, expansion)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFReserveBucketSize(ctx context.Context, key string, capacity int64, bucketsize int64) *StatusCmd {
	ret := c.comp.CFReserveBucketSize(ctx, key, capacity, bucketsize)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFReserveMaxIterations(ctx context.Context, key string, capacity int64, maxiterations int64) *StatusCmd {
	ret := c.comp.CFReserveMaxIterations(ctx, key, capacity, maxiterations)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFScanDump(ctx context.Context, key string, iterator int64) *ScanDumpCmd {
	ret := c.comp.CFScanDump(ctx, key, iterator)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CFLoadChunk(ctx context.Context, key string, iterator int64, data interface{}) *StatusCmd {
	ret := c.comp.CFLoadChunk(ctx, key, iterator, data)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSIncrBy(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	ret := c.comp.CMSIncrBy(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSInfo(ctx context.Context, key string) *CMSInfoCmd {
	ret := c.comp.CMSInfo(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSInitByDim(ctx context.Context, key string, width, height int64) *StatusCmd {
	ret := c.comp.CMSInitByDim(ctx, key, width, height)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSInitByProb(ctx context.Context, key string, errorRate, probability float64) *StatusCmd {
	ret := c.comp.CMSInitByProb(ctx, key, errorRate, probability)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSMerge(ctx context.Context, destKey string, sourceKeys ...string) *StatusCmd {
	ret := c.comp.CMSMerge(ctx, destKey, sourceKeys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSMergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) *StatusCmd {
	ret := c.comp.CMSMergeWithWeight(ctx, destKey, sourceKeys)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) CMSQuery(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	ret := c.comp.CMSQuery(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKAdd(ctx context.Context, key string, elements ...interface{}) *StringSliceCmd {
	ret := c.comp.TopKAdd(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKCount(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	ret := c.comp.TopKCount(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKIncrBy(ctx context.Context, key string, elements ...interface{}) *StringSliceCmd {
	ret := c.comp.TopKIncrBy(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKInfo(ctx context.Context, key string) *TopKInfoCmd {
	ret := c.comp.TopKInfo(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKList(ctx context.Context, key string) *StringSliceCmd {
	ret := c.comp.TopKList(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKListWithCount(ctx context.Context, key string) *MapStringIntCmd {
	ret := c.comp.TopKListWithCount(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKQuery(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	ret := c.comp.TopKQuery(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKReserve(ctx context.Context, key string, k int64) *StatusCmd {
	ret := c.comp.TopKReserve(ctx, key, k)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TopKReserveWithOptions(ctx context.Context, key string, k int64, width, depth int64, decay float64) *StatusCmd {
	ret := c.comp.TopKReserveWithOptions(ctx, key, k, width, depth, decay)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestAdd(ctx context.Context, key string, elements ...float64) *StatusCmd {
	ret := c.comp.TDigestAdd(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestByRank(ctx context.Context, key string, rank ...uint64) *FloatSliceCmd {
	ret := c.comp.TDigestByRank(ctx, key, rank...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestByRevRank(ctx context.Context, key string, rank ...uint64) *FloatSliceCmd {
	ret := c.comp.TDigestByRevRank(ctx, key, rank...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestCDF(ctx context.Context, key string, elements ...float64) *FloatSliceCmd {
	ret := c.comp.TDigestCDF(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestCreate(ctx context.Context, key string) *StatusCmd {
	ret := c.comp.TDigestCreate(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestCreateWithCompression(ctx context.Context, key string, compression int64) *StatusCmd {
	ret := c.comp.TDigestCreateWithCompression(ctx, key, compression)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestInfo(ctx context.Context, key string) *TDigestInfoCmd {
	ret := c.comp.TDigestInfo(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestMax(ctx context.Context, key string) *FloatCmd {
	ret := c.comp.TDigestMax(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestMin(ctx context.Context, key string) *FloatCmd {
	ret := c.comp.TDigestMin(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestMerge(ctx context.Context, destKey string, options *TDigestMergeOptions, sourceKeys ...string) *StatusCmd {
	ret := c.comp.TDigestMerge(ctx, destKey, options, sourceKeys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestQuantile(ctx context.Context, key string, elements ...float64) *FloatSliceCmd {
	ret := c.comp.TDigestQuantile(ctx, key, elements...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestRank(ctx context.Context, key string, values ...float64) *IntSliceCmd {
	ret := c.comp.TDigestRank(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestReset(ctx context.Context, key string) *StatusCmd {
	ret := c.comp.TDigestReset(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestRevRank(ctx context.Context, key string, values ...float64) *IntSliceCmd {
	ret := c.comp.TDigestRevRank(ctx, key, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TDigestTrimmedMean(ctx context.Context, key string, lowCutQuantile, highCutQuantile float64) *FloatCmd {
	ret := c.comp.TDigestTrimmedMean(ctx, key, lowCutQuantile, highCutQuantile)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSAdd(ctx context.Context, key string, timestamp interface{}, value float64) *IntCmd {
	ret := c.comp.TSAdd(ctx, key, timestamp, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSAddWithArgs(ctx context.Context, key string, timestamp interface{}, value float64, options *TSOptions) *IntCmd {
	ret := c.comp.TSAddWithArgs(ctx, key, timestamp, value, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSCreate(ctx context.Context, key string) *StatusCmd {
	ret := c.comp.TSCreate(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSCreateWithArgs(ctx context.Context, key string, options *TSOptions) *StatusCmd {
	ret := c.comp.TSCreateWithArgs(ctx, key, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSAlter(ctx context.Context, key string, options *TSAlterOptions) *StatusCmd {
	ret := c.comp.TSAlter(ctx, key, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSCreateRule(ctx context.Context, sourceKey string, destKey string, aggregator Aggregator, bucketDuration int) *StatusCmd {
	ret := c.comp.TSCreateRule(ctx, sourceKey, destKey, aggregator, bucketDuration)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSCreateRuleWithArgs(ctx context.Context, sourceKey string, destKey string, aggregator Aggregator, bucketDuration int, options *TSCreateRuleOptions) *StatusCmd {
	ret := c.comp.TSCreateRuleWithArgs(ctx, sourceKey, destKey, aggregator, bucketDuration, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSIncrBy(ctx context.Context, Key string, timestamp float64) *IntCmd {
	ret := c.comp.TSIncrBy(ctx, Key, timestamp)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSIncrByWithArgs(ctx context.Context, key string, timestamp float64, options *TSIncrDecrOptions) *IntCmd {
	ret := c.comp.TSIncrByWithArgs(ctx, key, timestamp, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSDecrBy(ctx context.Context, Key string, timestamp float64) *IntCmd {
	ret := c.comp.TSDecrBy(ctx, Key, timestamp)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSDecrByWithArgs(ctx context.Context, key string, timestamp float64, options *TSIncrDecrOptions) *IntCmd {
	ret := c.comp.TSDecrByWithArgs(ctx, key, timestamp, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSDel(ctx context.Context, Key string, fromTimestamp int, toTimestamp int) *IntCmd {
	ret := c.comp.TSDel(ctx, Key, fromTimestamp, toTimestamp)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSDeleteRule(ctx context.Context, sourceKey string, destKey string) *StatusCmd {
	ret := c.comp.TSDeleteRule(ctx, sourceKey, destKey)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSGet(ctx context.Context, key string) *TSTimestampValueCmd {
	ret := c.comp.TSGet(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSGetWithArgs(ctx context.Context, key string, options *TSGetOptions) *TSTimestampValueCmd {
	ret := c.comp.TSGetWithArgs(ctx, key, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSInfo(ctx context.Context, key string) *MapStringInterfaceCmd {
	ret := c.comp.TSInfo(ctx, key)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSInfoWithArgs(ctx context.Context, key string, options *TSInfoOptions) *MapStringInterfaceCmd {
	ret := c.comp.TSInfoWithArgs(ctx, key, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMAdd(ctx context.Context, ktvSlices [][]interface{}) *IntSliceCmd {
	ret := c.comp.TSMAdd(ctx, ktvSlices)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSQueryIndex(ctx context.Context, filterExpr []string) *StringSliceCmd {
	ret := c.comp.TSQueryIndex(ctx, filterExpr)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSRevRange(ctx context.Context, key string, fromTimestamp int, toTimestamp int) *TSTimestampValueSliceCmd {
	ret := c.comp.TSRevRange(ctx, key, fromTimestamp, toTimestamp)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSRevRangeWithArgs(ctx context.Context, key string, fromTimestamp int, toTimestamp int, options *TSRevRangeOptions) *TSTimestampValueSliceCmd {
	ret := c.comp.TSRevRangeWithArgs(ctx, key, fromTimestamp, toTimestamp, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSRange(ctx context.Context, key string, fromTimestamp int, toTimestamp int) *TSTimestampValueSliceCmd {
	ret := c.comp.TSRange(ctx, key, fromTimestamp, toTimestamp)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSRangeWithArgs(ctx context.Context, key string, fromTimestamp int, toTimestamp int, options *TSRangeOptions) *TSTimestampValueSliceCmd {
	ret := c.comp.TSRangeWithArgs(ctx, key, fromTimestamp, toTimestamp, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMRange(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string) *MapStringSliceInterfaceCmd {
	ret := c.comp.TSMRange(ctx, fromTimestamp, toTimestamp, filterExpr)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMRangeWithArgs(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string, options *TSMRangeOptions) *MapStringSliceInterfaceCmd {
	ret := c.comp.TSMRangeWithArgs(ctx, fromTimestamp, toTimestamp, filterExpr, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMRevRange(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string) *MapStringSliceInterfaceCmd {
	ret := c.comp.TSMRevRange(ctx, fromTimestamp, toTimestamp, filterExpr)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMRevRangeWithArgs(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string, options *TSMRevRangeOptions) *MapStringSliceInterfaceCmd {
	ret := c.comp.TSMRevRangeWithArgs(ctx, fromTimestamp, toTimestamp, filterExpr, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMGet(ctx context.Context, filters []string) *MapStringSliceInterfaceCmd {
	ret := c.comp.TSMGet(ctx, filters)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) TSMGetWithArgs(ctx context.Context, filters []string, options *TSMGetOptions) *MapStringSliceInterfaceCmd {
	ret := c.comp.TSMGetWithArgs(ctx, filters, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrAppend(ctx context.Context, key, path string, values ...interface{}) *IntSliceCmd {
	ret := c.comp.JSONArrAppend(ctx, key, path, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrIndex(ctx context.Context, key, path string, value ...interface{}) *IntSliceCmd {
	ret := c.comp.JSONArrIndex(ctx, key, path, value...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrIndexWithArgs(ctx context.Context, key, path string, options *JSONArrIndexArgs, value ...interface{}) *IntSliceCmd {
	ret := c.comp.JSONArrIndexWithArgs(ctx, key, path, options, value...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrInsert(ctx context.Context, key, path string, index int64, values ...interface{}) *IntSliceCmd {
	ret := c.comp.JSONArrInsert(ctx, key, path, index, values...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrLen(ctx context.Context, key, path string) *IntSliceCmd {
	ret := c.comp.JSONArrLen(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrPop(ctx context.Context, key, path string, index int) *StringSliceCmd {
	ret := c.comp.JSONArrPop(ctx, key, path, index)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrTrim(ctx context.Context, key, path string) *IntSliceCmd {
	ret := c.comp.JSONArrTrim(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONArrTrimWithArgs(ctx context.Context, key, path string, options *JSONArrTrimArgs) *IntSliceCmd {
	ret := c.comp.JSONArrTrimWithArgs(ctx, key, path, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONClear(ctx context.Context, key, path string) *IntCmd {
	ret := c.comp.JSONClear(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONDebugMemory(ctx context.Context, key, path string) *IntCmd {
	ret := c.comp.JSONDebugMemory(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONDel(ctx context.Context, key, path string) *IntCmd {
	ret := c.comp.JSONDel(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONForget(ctx context.Context, key, path string) *IntCmd {
	ret := c.comp.JSONForget(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONGet(ctx context.Context, key string, paths ...string) *JSONCmd {
	ret := c.comp.JSONGet(ctx, key, paths...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONGetWithArgs(ctx context.Context, key string, options *JSONGetArgs, paths ...string) *JSONCmd {
	ret := c.comp.JSONGetWithArgs(ctx, key, options, paths...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONMerge(ctx context.Context, key, path string, value string) *StatusCmd {
	ret := c.comp.JSONMerge(ctx, key, path, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONMSetArgs(ctx context.Context, docs []JSONSetArgs) *StatusCmd {
	ret := c.comp.JSONMSetArgs(ctx, docs)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONMSet(ctx context.Context, params ...interface{}) *StatusCmd {
	ret := c.comp.JSONMSet(ctx, params...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONMGet(ctx context.Context, path string, keys ...string) *JSONSliceCmd {
	ret := c.comp.JSONMGet(ctx, path, keys...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONNumIncrBy(ctx context.Context, key, path string, value float64) *JSONCmd {
	ret := c.comp.JSONNumIncrBy(ctx, key, path, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONObjKeys(ctx context.Context, key, path string) *SliceCmd {
	ret := c.comp.JSONObjKeys(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONObjLen(ctx context.Context, key, path string) *IntPointerSliceCmd {
	ret := c.comp.JSONObjLen(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONSet(ctx context.Context, key, path string, value interface{}) *StatusCmd {
	ret := c.comp.JSONSet(ctx, key, path, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONSetMode(ctx context.Context, key, path string, value interface{}, mode string) *StatusCmd {
	ret := c.comp.JSONSetMode(ctx, key, path, value, mode)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONStrAppend(ctx context.Context, key, path, value string) *IntPointerSliceCmd {
	ret := c.comp.JSONStrAppend(ctx, key, path, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONStrLen(ctx context.Context, key, path string) *IntPointerSliceCmd {
	ret := c.comp.JSONStrLen(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONToggle(ctx context.Context, key, path string) *IntPointerSliceCmd {
	ret := c.comp.JSONToggle(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) JSONType(ctx context.Context, key, path string) *JSONSliceCmd {
	ret := c.comp.JSONType(ctx, key, path)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FT_List(ctx context.Context) *StringSliceCmd {
	ret := c.comp.FT_List(ctx)
	c.rets = append(c.rets, ret)
	return ret
}
func (c *Pipeline) FTAggregate(ctx context.Context, index string, query string) *MapStringInterfaceCmd {
	ret := c.comp.FTAggregate(ctx, index, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTAggregateWithArgs(ctx context.Context, index string, query string, options *FTAggregateOptions) *AggregateCmd {
	ret := c.comp.FTAggregateWithArgs(ctx, index, query, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTAliasAdd(ctx context.Context, index string, alias string) *StatusCmd {
	ret := c.comp.FTAliasAdd(ctx, index, alias)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTAliasDel(ctx context.Context, alias string) *StatusCmd {
	ret := c.comp.FTAliasDel(ctx, alias)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTAliasUpdate(ctx context.Context, index string, alias string) *StatusCmd {
	ret := c.comp.FTAliasUpdate(ctx, index, alias)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTAlter(ctx context.Context, index string, skipInitalScan bool, definition []interface{}) *StatusCmd {
	ret := c.comp.FTAlter(ctx, index, skipInitalScan, definition)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTConfigGet(ctx context.Context, option string) *MapMapStringInterfaceCmd {
	ret := c.comp.FTConfigGet(ctx, option)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTConfigSet(ctx context.Context, option string, value interface{}) *StatusCmd {
	ret := c.comp.FTConfigSet(ctx, option, value)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTCreate(ctx context.Context, index string, options *FTCreateOptions, schema ...*FieldSchema) *StatusCmd {
	ret := c.comp.FTCreate(ctx, index, options, schema...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTCursorDel(ctx context.Context, index string, cursorId int) *StatusCmd {
	ret := c.comp.FTCursorDel(ctx, index, cursorId)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTCursorRead(ctx context.Context, index string, cursorId int, count int) *MapStringInterfaceCmd {
	ret := c.comp.FTCursorRead(ctx, index, cursorId, count)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTDictAdd(ctx context.Context, dict string, term ...interface{}) *IntCmd {
	ret := c.comp.FTDictAdd(ctx, dict, term...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTDictDel(ctx context.Context, dict string, term ...interface{}) *IntCmd {
	ret := c.comp.FTDictDel(ctx, dict, term...)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTDictDump(ctx context.Context, dict string) *StringSliceCmd {
	ret := c.comp.FTDictDump(ctx, dict)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTDropIndex(ctx context.Context, index string) *StatusCmd {
	ret := c.comp.FTDropIndex(ctx, index)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTDropIndexWithArgs(ctx context.Context, index string, options *FTDropIndexOptions) *StatusCmd {
	ret := c.comp.FTDropIndexWithArgs(ctx, index, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTExplain(ctx context.Context, index string, query string) *StringCmd {
	ret := c.comp.FTExplain(ctx, index, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTExplainWithArgs(ctx context.Context, index string, query string, options *FTExplainOptions) *StringCmd {
	ret := c.comp.FTExplainWithArgs(ctx, index, query, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTInfo(ctx context.Context, index string) *FTInfoCmd {
	ret := c.comp.FTInfo(ctx, index)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSpellCheck(ctx context.Context, index string, query string) *FTSpellCheckCmd {
	ret := c.comp.FTSpellCheck(ctx, index, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSpellCheckWithArgs(ctx context.Context, index string, query string, options *FTSpellCheckOptions) *FTSpellCheckCmd {
	ret := c.comp.FTSpellCheckWithArgs(ctx, index, query, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSearch(ctx context.Context, index string, query string) *FTSearchCmd {
	ret := c.comp.FTSearch(ctx, index, query)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSearchWithArgs(ctx context.Context, index string, query string, options *FTSearchOptions) *FTSearchCmd {
	ret := c.comp.FTSearchWithArgs(ctx, index, query, options)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSynDump(ctx context.Context, index string) *FTSynDumpCmd {
	ret := c.comp.FTSynDump(ctx, index)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSynUpdate(ctx context.Context, index string, synGroupId interface{}, terms []interface{}) *StatusCmd {
	ret := c.comp.FTSynUpdate(ctx, index, synGroupId, terms)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTSynUpdateWithArgs(ctx context.Context, index string, synGroupId interface{}, options *FTSynUpdateOptions, terms []interface{}) *StatusCmd {
	ret := c.comp.FTSynUpdateWithArgs(ctx, index, synGroupId, options, terms)
	c.rets = append(c.rets, ret)
	return ret
}

func (c *Pipeline) FTTagVals(ctx context.Context, index string, field string) *StringSliceCmd {
	ret := c.comp.FTTagVals(ctx, index, field)
	c.rets = append(c.rets, ret)
	return ret
}

// Len returns the number of queued commands.
func (c *Pipeline) Len() int {
	return len(c.comp.client.(*proxy).cmds)
}

// Do queues the custom command for later execution.
func (c *Pipeline) Do(_ context.Context, args ...interface{}) *Cmd {
	ret := &Cmd{}
	if len(args) == 0 {
		ret.SetErr(errors.New("redis: please enter the command to be executed"))
		return ret
	}
	p := c.comp.client.(*proxy)
	command := p.B().Arbitrary(str(args[0]))
	if len(args) > 1 {
		command = command.Keys(str(args[1]))
		for _, a := range args[2:] {
			command = command.Args(str(a))
		}
	}
	p.cmds = append(p.cmds, command.Build())
	c.rets = append(c.rets, ret)
	return ret
}

// Discard resets the pipeline and discards queued commands.
func (c *Pipeline) Discard() {
	p := c.comp.client.(*proxy)
	p.cmds = nil
	c.rets = nil
}

// Exec executes all previously queued commands using one
// client-server roundtrip.
//
// Exec always returns list of commands and error of the first failed
// command if any.
func (c *Pipeline) Exec(ctx context.Context) ([]Cmder, error) {
	p := c.comp.client.(*proxy)
	if len(p.cmds) == 0 {
		return nil, nil
	}

	rets := c.rets
	cmds := p.cmds
	c.rets = nil
	p.cmds = nil

	var err error
	for i, r := range p.DoMulti(ctx, cmds...) {
		if r.NonRedisError() != nil {
			err = r.NonRedisError()
		}
		rets[i].from(r)
	}

	return rets, err
}

func (c *Pipeline) Pipelined(ctx context.Context, fn func(Pipeliner) error) ([]Cmder, error) {
	if err := fn(c); err != nil {
		return nil, err
	}
	return c.Exec(ctx)
}

func (c *Pipeline) Pipeline() Pipeliner {
	return c
}

func (c *Pipeline) TxPipelined(ctx context.Context, fn func(Pipeliner) error) ([]Cmder, error) {
	return c.Pipelined(ctx, fn)
}

func (c *Pipeline) TxPipeline() Pipeliner {
	return c
}
