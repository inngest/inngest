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
	"encoding"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/internal/cmds"
	"github.com/redis/rueidis/internal/util"
)

const (
	KeepTTL           = -1
	BitCountIndexByte = "BYTE"
	BitCountIndexBit  = "BIT"
)

var Nil = rueidis.Nil

type Cmdable interface {
	CoreCmdable
	Cache(ttl time.Duration) CacheCompat

	Subscribe(ctx context.Context, channels ...string) PubSub
	PSubscribe(ctx context.Context, patterns ...string) PubSub
	SSubscribe(ctx context.Context, channels ...string) PubSub

	Watch(ctx context.Context, fn func(Tx) error, keys ...string) error
}

type CoreCmdable interface {
	Command(ctx context.Context) *CommandsInfoCmd
	CommandList(ctx context.Context, filter FilterBy) *StringSliceCmd
	CommandGetKeys(ctx context.Context, commands ...any) *StringSliceCmd
	CommandGetKeysAndFlags(ctx context.Context, commands ...any) *KeyFlagsCmd
	ClientGetName(ctx context.Context) *StringCmd
	Echo(ctx context.Context, message any) *StringCmd
	Ping(ctx context.Context) *StatusCmd
	Quit(ctx context.Context) *StatusCmd
	Del(ctx context.Context, keys ...string) *IntCmd
	Unlink(ctx context.Context, keys ...string) *IntCmd
	Dump(ctx context.Context, key string) *StringCmd
	Exists(ctx context.Context, keys ...string) *IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *BoolCmd
	ExpireAt(ctx context.Context, key string, tm time.Time) *BoolCmd
	ExpireTime(ctx context.Context, key string) *DurationCmd
	ExpireNX(ctx context.Context, key string, expiration time.Duration) *BoolCmd
	ExpireXX(ctx context.Context, key string, expiration time.Duration) *BoolCmd
	ExpireGT(ctx context.Context, key string, expiration time.Duration) *BoolCmd
	ExpireLT(ctx context.Context, key string, expiration time.Duration) *BoolCmd
	Keys(ctx context.Context, pattern string) *StringSliceCmd
	Migrate(ctx context.Context, host string, port int64, key string, db int64, timeout time.Duration) *StatusCmd
	Move(ctx context.Context, key string, db int64) *BoolCmd
	ObjectRefCount(ctx context.Context, key string) *IntCmd
	ObjectEncoding(ctx context.Context, key string) *StringCmd
	ObjectIdleTime(ctx context.Context, key string) *DurationCmd
	Persist(ctx context.Context, key string) *BoolCmd
	PExpire(ctx context.Context, key string, expiration time.Duration) *BoolCmd
	PExpireAt(ctx context.Context, key string, tm time.Time) *BoolCmd
	PExpireTime(ctx context.Context, key string) *DurationCmd
	PTTL(ctx context.Context, key string) *DurationCmd
	RandomKey(ctx context.Context) *StringCmd
	Rename(ctx context.Context, key, newkey string) *StatusCmd
	RenameNX(ctx context.Context, key, newkey string) *BoolCmd
	Restore(ctx context.Context, key string, ttl time.Duration, value string) *StatusCmd
	RestoreReplace(ctx context.Context, key string, ttl time.Duration, value string) *StatusCmd
	Sort(ctx context.Context, key string, sort Sort) *StringSliceCmd
	SortRO(ctx context.Context, key string, sort Sort) *StringSliceCmd
	SortStore(ctx context.Context, key, store string, sort Sort) *IntCmd
	SortInterfaces(ctx context.Context, key string, sort Sort) *SliceCmd
	Touch(ctx context.Context, keys ...string) *IntCmd
	TTL(ctx context.Context, key string) *DurationCmd
	Type(ctx context.Context, key string) *StatusCmd
	Append(ctx context.Context, key, value string) *IntCmd
	Decr(ctx context.Context, key string) *IntCmd
	DecrBy(ctx context.Context, key string, decrement int64) *IntCmd
	Get(ctx context.Context, key string) *StringCmd
	GetRange(ctx context.Context, key string, start, end int64) *StringCmd
	GetSet(ctx context.Context, key string, value any) *StringCmd
	GetEx(ctx context.Context, key string, expiration time.Duration) *StringCmd
	GetDel(ctx context.Context, key string) *StringCmd
	Incr(ctx context.Context, key string) *IntCmd
	IncrBy(ctx context.Context, key string, value int64) *IntCmd
	IncrByFloat(ctx context.Context, key string, value float64) *FloatCmd
	MGet(ctx context.Context, keys ...string) *SliceCmd
	MSet(ctx context.Context, values ...any) *StatusCmd
	MSetNX(ctx context.Context, values ...any) *BoolCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd
	SetArgs(ctx context.Context, key string, value any, a SetArgs) *StatusCmd
	SetEX(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd
	SetNX(ctx context.Context, key string, value any, expiration time.Duration) *BoolCmd
	SetXX(ctx context.Context, key string, value any, expiration time.Duration) *BoolCmd
	SetRange(ctx context.Context, key string, offset int64, value string) *IntCmd
	StrLen(ctx context.Context, key string) *IntCmd
	Copy(ctx context.Context, sourceKey string, destKey string, db int64, replace bool) *IntCmd

	GetBit(ctx context.Context, key string, offset int64) *IntCmd
	SetBit(ctx context.Context, key string, offset int64, value int64) *IntCmd
	BitCount(ctx context.Context, key string, bitCount *BitCount) *IntCmd
	BitOpAnd(ctx context.Context, destKey string, keys ...string) *IntCmd
	BitOpOr(ctx context.Context, destKey string, keys ...string) *IntCmd
	BitOpXor(ctx context.Context, destKey string, keys ...string) *IntCmd
	BitOpNot(ctx context.Context, destKey string, key string) *IntCmd
	BitPos(ctx context.Context, key string, bit int64, pos ...int64) *IntCmd
	BitPosSpan(ctx context.Context, key string, bit int64, start, end int64, span string) *IntCmd
	BitField(ctx context.Context, key string, args ...any) *IntSliceCmd
	// TODO BitFieldRO(ctx context.Context, key string, values ...interface{}) *IntSliceCmd

	Scan(ctx context.Context, cursor uint64, match string, count int64) *ScanCmd
	ScanType(ctx context.Context, cursor uint64, match string, count int64, keyType string) *ScanCmd
	SScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd
	HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd
	// TODO HScanNoValues(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd
	ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd

	HDel(ctx context.Context, key string, fields ...string) *IntCmd
	HExists(ctx context.Context, key, field string) *BoolCmd
	HGet(ctx context.Context, key, field string) *StringCmd
	HGetAll(ctx context.Context, key string) *StringStringMapCmd
	HIncrBy(ctx context.Context, key, field string, incr int64) *IntCmd
	HIncrByFloat(ctx context.Context, key, field string, incr float64) *FloatCmd
	HKeys(ctx context.Context, key string) *StringSliceCmd
	HLen(ctx context.Context, key string) *IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *SliceCmd
	HSet(ctx context.Context, key string, values ...any) *IntCmd
	HMSet(ctx context.Context, key string, values ...any) *BoolCmd
	HSetNX(ctx context.Context, key, field string, value any) *BoolCmd
	HVals(ctx context.Context, key string) *StringSliceCmd
	HRandField(ctx context.Context, key string, count int64) *StringSliceCmd
	HRandFieldWithValues(ctx context.Context, key string, count int64) *KeyValueSliceCmd
	HExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *IntSliceCmd
	HExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd
	HPExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *IntSliceCmd
	HPExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd
	HExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *IntSliceCmd
	HExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd
	HPExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *IntSliceCmd
	HPExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd
	HPersist(ctx context.Context, key string, fields ...string) *IntSliceCmd
	HExpireTime(ctx context.Context, key string, fields ...string) *IntSliceCmd
	HPExpireTime(ctx context.Context, key string, fields ...string) *IntSliceCmd
	HTTL(ctx context.Context, key string, fields ...string) *IntSliceCmd
	HPTTL(ctx context.Context, key string, fields ...string) *IntSliceCmd

	BLPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd
	BLMPop(ctx context.Context, timeout time.Duration, direction string, count int64, keys ...string) *KeyValuesCmd
	BRPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd
	BRPopLPush(ctx context.Context, source, destination string, timeout time.Duration) *StringCmd
	// TODO LCS(ctx context.Context, q *LCSQuery) *LCSCmd
	LIndex(ctx context.Context, key string, index int64) *StringCmd
	LInsert(ctx context.Context, key, op string, pivot, value any) *IntCmd
	LInsertBefore(ctx context.Context, key string, pivot, value any) *IntCmd
	LInsertAfter(ctx context.Context, key string, pivot, value any) *IntCmd
	LLen(ctx context.Context, key string) *IntCmd
	LMPop(ctx context.Context, direction string, count int64, keys ...string) *KeyValuesCmd
	LPop(ctx context.Context, key string) *StringCmd
	LPopCount(ctx context.Context, key string, count int64) *StringSliceCmd
	LPos(ctx context.Context, key string, value string, args LPosArgs) *IntCmd
	LPosCount(ctx context.Context, key string, value string, count int64, args LPosArgs) *IntSliceCmd
	LPush(ctx context.Context, key string, values ...any) *IntCmd
	LPushX(ctx context.Context, key string, values ...any) *IntCmd
	LRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd
	LRem(ctx context.Context, key string, count int64, value any) *IntCmd
	LSet(ctx context.Context, key string, index int64, value any) *StatusCmd
	LTrim(ctx context.Context, key string, start, stop int64) *StatusCmd
	RPop(ctx context.Context, key string) *StringCmd
	RPopCount(ctx context.Context, key string, count int64) *StringSliceCmd
	RPopLPush(ctx context.Context, source, destination string) *StringCmd
	RPush(ctx context.Context, key string, values ...any) *IntCmd
	RPushX(ctx context.Context, key string, values ...any) *IntCmd
	LMove(ctx context.Context, source, destination, srcpos, destpos string) *StringCmd
	BLMove(ctx context.Context, source, destination, srcpos, destpos string, timeout time.Duration) *StringCmd

	SAdd(ctx context.Context, key string, members ...any) *IntCmd
	SCard(ctx context.Context, key string) *IntCmd
	SDiff(ctx context.Context, keys ...string) *StringSliceCmd
	SDiffStore(ctx context.Context, destination string, keys ...string) *IntCmd
	SInter(ctx context.Context, keys ...string) *StringSliceCmd
	SInterCard(ctx context.Context, limit int64, keys ...string) *IntCmd
	SInterStore(ctx context.Context, destination string, keys ...string) *IntCmd
	SIsMember(ctx context.Context, key string, member any) *BoolCmd
	SMIsMember(ctx context.Context, key string, members ...any) *BoolSliceCmd
	SMembers(ctx context.Context, key string) *StringSliceCmd
	SMembersMap(ctx context.Context, key string) *StringStructMapCmd
	SMove(ctx context.Context, source, destination string, member any) *BoolCmd
	SPop(ctx context.Context, key string) *StringCmd
	SPopN(ctx context.Context, key string, count int64) *StringSliceCmd
	SRandMember(ctx context.Context, key string) *StringCmd
	SRandMemberN(ctx context.Context, key string, count int64) *StringSliceCmd
	SRem(ctx context.Context, key string, members ...any) *IntCmd
	SUnion(ctx context.Context, keys ...string) *StringSliceCmd
	SUnionStore(ctx context.Context, destination string, keys ...string) *IntCmd

	XAdd(ctx context.Context, a XAddArgs) *StringCmd
	XDel(ctx context.Context, stream string, ids ...string) *IntCmd
	XLen(ctx context.Context, stream string) *IntCmd
	XRange(ctx context.Context, stream, start, stop string) *XMessageSliceCmd
	XRangeN(ctx context.Context, stream, start, stop string, count int64) *XMessageSliceCmd
	XRevRange(ctx context.Context, stream string, start, stop string) *XMessageSliceCmd
	XRevRangeN(ctx context.Context, stream string, start, stop string, count int64) *XMessageSliceCmd
	XRead(ctx context.Context, a XReadArgs) *XStreamSliceCmd
	XReadStreams(ctx context.Context, streams ...string) *XStreamSliceCmd
	XGroupCreate(ctx context.Context, stream, group, start string) *StatusCmd
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) *StatusCmd
	XGroupSetID(ctx context.Context, stream, group, start string) *StatusCmd
	XGroupDestroy(ctx context.Context, stream, group string) *IntCmd
	XGroupCreateConsumer(ctx context.Context, stream, group, consumer string) *IntCmd
	XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *IntCmd
	XReadGroup(ctx context.Context, a XReadGroupArgs) *XStreamSliceCmd
	XAck(ctx context.Context, stream, group string, ids ...string) *IntCmd
	XPending(ctx context.Context, stream, group string) *XPendingCmd
	XPendingExt(ctx context.Context, a XPendingExtArgs) *XPendingExtCmd
	XClaim(ctx context.Context, a XClaimArgs) *XMessageSliceCmd
	XClaimJustID(ctx context.Context, a XClaimArgs) *StringSliceCmd
	XAutoClaim(ctx context.Context, a XAutoClaimArgs) *XAutoClaimCmd
	XAutoClaimJustID(ctx context.Context, a XAutoClaimArgs) *XAutoClaimJustIDCmd
	XTrimMaxLen(ctx context.Context, key string, maxLen int64) *IntCmd
	XTrimMaxLenApprox(ctx context.Context, key string, maxLen, limit int64) *IntCmd
	XTrimMinID(ctx context.Context, key string, minID string) *IntCmd
	XTrimMinIDApprox(ctx context.Context, key string, minID string, limit int64) *IntCmd
	XInfoGroups(ctx context.Context, key string) *XInfoGroupsCmd
	XInfoStream(ctx context.Context, key string) *XInfoStreamCmd
	XInfoStreamFull(ctx context.Context, key string, count int64) *XInfoStreamFullCmd
	XInfoConsumers(ctx context.Context, key string, group string) *XInfoConsumersCmd

	BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) *ZWithKeyCmd
	BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) *ZWithKeyCmd
	BZMPop(ctx context.Context, timeout time.Duration, order string, count int64, keys ...string) *ZSliceWithKeyCmd

	ZAdd(ctx context.Context, key string, members ...Z) *IntCmd
	ZAddLT(ctx context.Context, key string, members ...Z) *IntCmd
	ZAddGT(ctx context.Context, key string, members ...Z) *IntCmd
	ZAddNX(ctx context.Context, key string, members ...Z) *IntCmd
	ZAddXX(ctx context.Context, key string, members ...Z) *IntCmd
	ZAddArgs(ctx context.Context, key string, args ZAddArgs) *IntCmd
	ZAddArgsIncr(ctx context.Context, key string, args ZAddArgs) *FloatCmd
	ZCard(ctx context.Context, key string) *IntCmd
	ZCount(ctx context.Context, key, min, max string) *IntCmd
	ZLexCount(ctx context.Context, key, min, max string) *IntCmd
	ZIncrBy(ctx context.Context, key string, increment float64, member string) *FloatCmd
	ZInter(ctx context.Context, store ZStore) *StringSliceCmd
	ZInterWithScores(ctx context.Context, store ZStore) *ZSliceCmd
	ZInterCard(ctx context.Context, limit int64, keys ...string) *IntCmd
	ZInterStore(ctx context.Context, destination string, store ZStore) *IntCmd
	ZMPop(ctx context.Context, order string, count int64, keys ...string) *ZSliceWithKeyCmd
	ZMScore(ctx context.Context, key string, members ...string) *FloatSliceCmd
	ZPopMax(ctx context.Context, key string, count ...int64) *ZSliceCmd
	ZPopMin(ctx context.Context, key string, count ...int64) *ZSliceCmd
	ZRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd
	ZRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd
	ZRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd
	ZRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd
	ZRangeArgs(ctx context.Context, z ZRangeArgs) *StringSliceCmd
	ZRangeArgsWithScores(ctx context.Context, z ZRangeArgs) *ZSliceCmd
	ZRangeStore(ctx context.Context, dst string, z ZRangeArgs) *IntCmd
	ZRank(ctx context.Context, key, member string) *IntCmd
	ZRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd
	ZRem(ctx context.Context, key string, members ...any) *IntCmd
	ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *IntCmd
	ZRemRangeByScore(ctx context.Context, key, min, max string) *IntCmd
	ZRemRangeByLex(ctx context.Context, key, min, max string) *IntCmd
	ZRevRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd
	ZRevRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd
	ZRevRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd
	ZRevRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd
	ZRevRank(ctx context.Context, key, member string) *IntCmd
	ZRevRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd
	ZScore(ctx context.Context, key, member string) *FloatCmd
	ZUnionStore(ctx context.Context, dest string, store ZStore) *IntCmd
	ZRandMember(ctx context.Context, key string, count int64) *StringSliceCmd
	ZRandMemberWithScores(ctx context.Context, key string, count int64) *ZSliceCmd
	ZUnion(ctx context.Context, store ZStore) *StringSliceCmd
	ZUnionWithScores(ctx context.Context, store ZStore) *ZSliceCmd
	ZDiff(ctx context.Context, keys ...string) *StringSliceCmd
	ZDiffWithScores(ctx context.Context, keys ...string) *ZSliceCmd
	ZDiffStore(ctx context.Context, destination string, keys ...string) *IntCmd

	PFAdd(ctx context.Context, key string, els ...any) *IntCmd
	PFCount(ctx context.Context, keys ...string) *IntCmd
	PFMerge(ctx context.Context, dest string, keys ...string) *StatusCmd

	BgRewriteAOF(ctx context.Context) *StatusCmd
	BgSave(ctx context.Context) *StatusCmd
	ClientKill(ctx context.Context, ipPort string) *StatusCmd
	ClientKillByFilter(ctx context.Context, keys ...string) *IntCmd
	ClientList(ctx context.Context) *StringCmd
	// TODO ClientInfo(ctx context.Context) *ClientInfoCmd
	ClientPause(ctx context.Context, dur time.Duration) *BoolCmd
	ClientUnpause(ctx context.Context) *BoolCmd
	ClientID(ctx context.Context) *IntCmd
	ClientUnblock(ctx context.Context, id int64) *IntCmd
	ClientUnblockWithError(ctx context.Context, id int64) *IntCmd
	ConfigGet(ctx context.Context, parameter string) *StringStringMapCmd
	ConfigResetStat(ctx context.Context) *StatusCmd
	ConfigSet(ctx context.Context, parameter, value string) *StatusCmd
	ConfigRewrite(ctx context.Context) *StatusCmd
	DBSize(ctx context.Context) *IntCmd
	FlushAll(ctx context.Context) *StatusCmd
	FlushAllAsync(ctx context.Context) *StatusCmd
	FlushDB(ctx context.Context) *StatusCmd
	FlushDBAsync(ctx context.Context) *StatusCmd
	Info(ctx context.Context, section ...string) *StringCmd
	LastSave(ctx context.Context) *IntCmd
	Save(ctx context.Context) *StatusCmd
	Shutdown(ctx context.Context) *StatusCmd
	ShutdownSave(ctx context.Context) *StatusCmd
	ShutdownNoSave(ctx context.Context) *StatusCmd
	// TODO SlaveOf(ctx context.Context, host, port string) *StatusCmd
	// TODO SlowLogGet(ctx context.Context, num int64) *SlowLogCmd
	Time(ctx context.Context) *TimeCmd
	DebugObject(ctx context.Context, key string) *StringCmd
	ReadOnly(ctx context.Context) *StatusCmd
	ReadWrite(ctx context.Context) *StatusCmd
	MemoryUsage(ctx context.Context, key string, samples ...int64) *IntCmd

	Eval(ctx context.Context, script string, keys []string, args ...any) *Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd
	EvalRO(ctx context.Context, script string, keys []string, args ...any) *Cmd
	EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd
	ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd
	ScriptFlush(ctx context.Context) *StatusCmd
	ScriptKill(ctx context.Context) *StatusCmd
	ScriptLoad(ctx context.Context, script string) *StringCmd

	FunctionLoad(ctx context.Context, code string) *StringCmd
	FunctionLoadReplace(ctx context.Context, code string) *StringCmd
	FunctionDelete(ctx context.Context, libName string) *StringCmd
	FunctionFlush(ctx context.Context) *StringCmd
	FunctionKill(ctx context.Context) *StringCmd
	FunctionFlushAsync(ctx context.Context) *StringCmd
	FunctionList(ctx context.Context, q FunctionListQuery) *FunctionListCmd
	FunctionDump(ctx context.Context) *StringCmd
	FunctionRestore(ctx context.Context, libDump string) *StringCmd
	// TODO FunctionStats(ctx context.Context) *FunctionStatsCmd
	FCall(ctx context.Context, function string, keys []string, args ...any) *Cmd
	FCallRO(ctx context.Context, function string, keys []string, args ...any) *Cmd

	Publish(ctx context.Context, channel string, message any) *IntCmd
	SPublish(ctx context.Context, channel string, message any) *IntCmd
	PubSubChannels(ctx context.Context, pattern string) *StringSliceCmd
	PubSubNumSub(ctx context.Context, channels ...string) *StringIntMapCmd
	PubSubNumPat(ctx context.Context) *IntCmd
	PubSubShardChannels(ctx context.Context, pattern string) *StringSliceCmd
	PubSubShardNumSub(ctx context.Context, channels ...string) *StringIntMapCmd

	// TODO ClusterMyShardID(ctx context.Context) *StringCmd
	ClusterSlots(ctx context.Context) *ClusterSlotsCmd
	ClusterShards(ctx context.Context) *ClusterShardsCmd
	// TODO ClusterLinks(ctx context.Context) *ClusterLinksCmd
	ClusterNodes(ctx context.Context) *StringCmd
	ClusterMeet(ctx context.Context, host string, port int64) *StatusCmd
	ClusterForget(ctx context.Context, nodeID string) *StatusCmd
	ClusterReplicate(ctx context.Context, nodeID string) *StatusCmd
	ClusterResetSoft(ctx context.Context) *StatusCmd
	ClusterResetHard(ctx context.Context) *StatusCmd
	ClusterInfo(ctx context.Context) *StringCmd
	ClusterKeySlot(ctx context.Context, key string) *IntCmd
	ClusterGetKeysInSlot(ctx context.Context, slot int64, count int64) *StringSliceCmd
	ClusterCountFailureReports(ctx context.Context, nodeID string) *IntCmd
	ClusterCountKeysInSlot(ctx context.Context, slot int64) *IntCmd
	ClusterDelSlots(ctx context.Context, slots ...int64) *StatusCmd
	ClusterDelSlotsRange(ctx context.Context, min, max int64) *StatusCmd
	ClusterSaveConfig(ctx context.Context) *StatusCmd
	ClusterSlaves(ctx context.Context, nodeID string) *StringSliceCmd
	ClusterFailover(ctx context.Context) *StatusCmd
	ClusterAddSlots(ctx context.Context, slots ...int64) *StatusCmd
	ClusterAddSlotsRange(ctx context.Context, min, max int64) *StatusCmd
	// TODO ReadOnly(ctx context.Context) *StatusCmd
	// TODO ReadWrite(ctx context.Context) *StatusCmd

	GeoAdd(ctx context.Context, key string, geoLocation ...GeoLocation) *IntCmd
	GeoPos(ctx context.Context, key string, members ...string) *GeoPosCmd
	GeoRadius(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *GeoLocationCmd
	GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *IntCmd
	GeoRadiusByMember(ctx context.Context, key, member string, query GeoRadiusQuery) *GeoLocationCmd
	GeoRadiusByMemberStore(ctx context.Context, key, member string, query GeoRadiusQuery) *IntCmd
	GeoSearch(ctx context.Context, key string, q GeoSearchQuery) *StringSliceCmd
	GeoSearchLocation(ctx context.Context, key string, q GeoSearchLocationQuery) *GeoLocationCmd
	GeoSearchStore(ctx context.Context, key, store string, q GeoSearchStoreQuery) *IntCmd
	GeoDist(ctx context.Context, key string, member1, member2, unit string) *FloatCmd
	GeoHash(ctx context.Context, key string, members ...string) *StringSliceCmd

	ACLDryRun(ctx context.Context, username string, command ...any) *StringCmd
	// TODO ACLLog(ctx context.Context, count int64) *ACLLogCmd
	// TODO ACLLogReset(ctx context.Context) *StatusCmd

	// TODO ModuleLoadex(ctx context.Context, conf *ModuleLoadexConfig) *StringCmd
	GearsCmdable
	ProbabilisticCmdable
	TimeseriesCmdable
	JSONCmdable
	SearchCmdable
}

type SearchCmdable interface {
	FT_List(ctx context.Context) *StringSliceCmd
	FTAggregate(ctx context.Context, index string, query string) *MapStringInterfaceCmd
	FTAggregateWithArgs(ctx context.Context, index string, query string, options *FTAggregateOptions) *AggregateCmd
	FTAliasAdd(ctx context.Context, index string, alias string) *StatusCmd
	FTAliasDel(ctx context.Context, alias string) *StatusCmd
	FTAliasUpdate(ctx context.Context, index string, alias string) *StatusCmd
	FTAlter(ctx context.Context, index string, skipInitalScan bool, definition []interface{}) *StatusCmd
	FTConfigGet(ctx context.Context, option string) *MapMapStringInterfaceCmd
	FTConfigSet(ctx context.Context, option string, value interface{}) *StatusCmd
	FTCreate(ctx context.Context, index string, options *FTCreateOptions, schema ...*FieldSchema) *StatusCmd
	FTCursorDel(ctx context.Context, index string, cursorId int) *StatusCmd
	FTCursorRead(ctx context.Context, index string, cursorId int, count int) *MapStringInterfaceCmd
	FTDictAdd(ctx context.Context, dict string, term ...interface{}) *IntCmd
	FTDictDel(ctx context.Context, dict string, term ...interface{}) *IntCmd
	FTDictDump(ctx context.Context, dict string) *StringSliceCmd
	FTDropIndex(ctx context.Context, index string) *StatusCmd
	FTDropIndexWithArgs(ctx context.Context, index string, options *FTDropIndexOptions) *StatusCmd
	FTExplain(ctx context.Context, index string, query string) *StringCmd
	FTExplainWithArgs(ctx context.Context, index string, query string, options *FTExplainOptions) *StringCmd
	FTInfo(ctx context.Context, index string) *FTInfoCmd
	FTSpellCheck(ctx context.Context, index string, query string) *FTSpellCheckCmd
	FTSpellCheckWithArgs(ctx context.Context, index string, query string, options *FTSpellCheckOptions) *FTSpellCheckCmd
	FTSearch(ctx context.Context, index string, query string) *FTSearchCmd
	FTSearchWithArgs(ctx context.Context, index string, query string, options *FTSearchOptions) *FTSearchCmd
	FTSynDump(ctx context.Context, index string) *FTSynDumpCmd
	FTSynUpdate(ctx context.Context, index string, synGroupId interface{}, terms []interface{}) *StatusCmd
	FTSynUpdateWithArgs(ctx context.Context, index string, synGroupId interface{}, options *FTSynUpdateOptions, terms []interface{}) *StatusCmd
	FTTagVals(ctx context.Context, index string, field string) *StringSliceCmd
}

// https://github.com/redis/go-redis/blob/af4872cbd0de349855ce3f0978929c2f56eb995f/probabilistic.go#L10
type ProbabilisticCmdable interface {
	BFAdd(ctx context.Context, key string, element interface{}) *BoolCmd
	BFCard(ctx context.Context, key string) *IntCmd
	BFExists(ctx context.Context, key string, element interface{}) *BoolCmd
	BFInfo(ctx context.Context, key string) *BFInfoCmd
	BFInfoArg(ctx context.Context, key, option string) *BFInfoCmd
	BFInfoCapacity(ctx context.Context, key string) *BFInfoCmd
	BFInfoSize(ctx context.Context, key string) *BFInfoCmd
	BFInfoFilters(ctx context.Context, key string) *BFInfoCmd
	BFInfoItems(ctx context.Context, key string) *BFInfoCmd
	BFInfoExpansion(ctx context.Context, key string) *BFInfoCmd
	BFInsert(ctx context.Context, key string, options *BFInsertOptions, elements ...interface{}) *BoolSliceCmd
	BFMAdd(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd
	BFMExists(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd
	BFReserve(ctx context.Context, key string, errorRate float64, capacity int64) *StatusCmd
	BFReserveExpansion(ctx context.Context, key string, errorRate float64, capacity, expansion int64) *StatusCmd
	BFReserveNonScaling(ctx context.Context, key string, errorRate float64, capacity int64) *StatusCmd
	BFReserveWithArgs(ctx context.Context, key string, options *BFReserveOptions) *StatusCmd
	BFScanDump(ctx context.Context, key string, iterator int64) *ScanDumpCmd
	BFLoadChunk(ctx context.Context, key string, iterator int64, data interface{}) *StatusCmd

	CFAdd(ctx context.Context, key string, element interface{}) *BoolCmd
	CFAddNX(ctx context.Context, key string, element interface{}) *BoolCmd
	CFCount(ctx context.Context, key string, element interface{}) *IntCmd
	CFDel(ctx context.Context, key string, element interface{}) *BoolCmd
	CFExists(ctx context.Context, key string, element interface{}) *BoolCmd
	CFInfo(ctx context.Context, key string) *CFInfoCmd
	CFInsert(ctx context.Context, key string, options *CFInsertOptions, elements ...interface{}) *BoolSliceCmd
	CFInsertNX(ctx context.Context, key string, options *CFInsertOptions, elements ...interface{}) *IntSliceCmd
	CFMExists(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd
	CFReserve(ctx context.Context, key string, capacity int64) *StatusCmd
	CFReserveWithArgs(ctx context.Context, key string, options *CFReserveOptions) *StatusCmd
	CFReserveExpansion(ctx context.Context, key string, capacity int64, expansion int64) *StatusCmd
	CFReserveBucketSize(ctx context.Context, key string, capacity int64, bucketsize int64) *StatusCmd
	CFReserveMaxIterations(ctx context.Context, key string, capacity int64, maxiterations int64) *StatusCmd
	CFScanDump(ctx context.Context, key string, iterator int64) *ScanDumpCmd
	CFLoadChunk(ctx context.Context, key string, iterator int64, data interface{}) *StatusCmd

	CMSIncrBy(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd
	CMSInfo(ctx context.Context, key string) *CMSInfoCmd
	CMSInitByDim(ctx context.Context, key string, width, height int64) *StatusCmd
	CMSInitByProb(ctx context.Context, key string, errorRate, probability float64) *StatusCmd
	CMSMerge(ctx context.Context, destKey string, sourceKeys ...string) *StatusCmd
	CMSMergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) *StatusCmd
	CMSQuery(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd

	TopKAdd(ctx context.Context, key string, elements ...interface{}) *StringSliceCmd
	TopKCount(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd
	TopKIncrBy(ctx context.Context, key string, elements ...interface{}) *StringSliceCmd
	TopKInfo(ctx context.Context, key string) *TopKInfoCmd
	TopKList(ctx context.Context, key string) *StringSliceCmd
	TopKListWithCount(ctx context.Context, key string) *MapStringIntCmd
	TopKQuery(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd
	TopKReserve(ctx context.Context, key string, k int64) *StatusCmd
	TopKReserveWithOptions(ctx context.Context, key string, k int64, width, depth int64, decay float64) *StatusCmd

	TDigestAdd(ctx context.Context, key string, elements ...float64) *StatusCmd
	TDigestByRank(ctx context.Context, key string, rank ...uint64) *FloatSliceCmd
	TDigestByRevRank(ctx context.Context, key string, rank ...uint64) *FloatSliceCmd
	TDigestCDF(ctx context.Context, key string, elements ...float64) *FloatSliceCmd
	TDigestCreate(ctx context.Context, key string) *StatusCmd
	TDigestCreateWithCompression(ctx context.Context, key string, compression int64) *StatusCmd
	TDigestInfo(ctx context.Context, key string) *TDigestInfoCmd
	TDigestMax(ctx context.Context, key string) *FloatCmd
	TDigestMin(ctx context.Context, key string) *FloatCmd
	TDigestMerge(ctx context.Context, destKey string, options *TDigestMergeOptions, sourceKeys ...string) *StatusCmd
	TDigestQuantile(ctx context.Context, key string, elements ...float64) *FloatSliceCmd
	TDigestRank(ctx context.Context, key string, values ...float64) *IntSliceCmd
	TDigestReset(ctx context.Context, key string) *StatusCmd
	TDigestRevRank(ctx context.Context, key string, values ...float64) *IntSliceCmd
	TDigestTrimmedMean(ctx context.Context, key string, lowCutQuantile, highCutQuantile float64) *FloatCmd

	Pipeline() Pipeliner
	Pipelined(ctx context.Context, fn func(Pipeliner) error) ([]Cmder, error)

	TxPipeline() Pipeliner
	TxPipelined(ctx context.Context, fn func(Pipeliner) error) ([]Cmder, error)
}

// Align with go-redis
// https://github.com/redis/go-redis/blob/f994ff1cd96299a5c8029ae3403af7b17ef06e8a/gears_commands.go#L9-L19
type GearsCmdable interface {
	TFunctionLoad(ctx context.Context, lib string) *StatusCmd
	TFunctionLoadArgs(ctx context.Context, lib string, options *TFunctionLoadOptions) *StatusCmd
	TFunctionDelete(ctx context.Context, libName string) *StatusCmd
	TFunctionList(ctx context.Context) *MapStringInterfaceSliceCmd
	TFunctionListArgs(ctx context.Context, options *TFunctionListOptions) *MapStringInterfaceSliceCmd
	TFCall(ctx context.Context, libName string, funcName string, numKeys int) *Cmd
	TFCallArgs(ctx context.Context, libName string, funcName string, numKeys int, options *TFCallOptions) *Cmd
	TFCallASYNC(ctx context.Context, libName string, funcName string, numKeys int) *Cmd
	TFCallASYNCArgs(ctx context.Context, libName string, funcName string, numKeys int, options *TFCallOptions) *Cmd
}

type TimeseriesCmdable interface {
	TSAdd(ctx context.Context, key string, timestamp interface{}, value float64) *IntCmd
	TSAddWithArgs(ctx context.Context, key string, timestamp interface{}, value float64, options *TSOptions) *IntCmd
	TSCreate(ctx context.Context, key string) *StatusCmd
	TSCreateWithArgs(ctx context.Context, key string, options *TSOptions) *StatusCmd
	TSAlter(ctx context.Context, key string, options *TSAlterOptions) *StatusCmd
	TSCreateRule(ctx context.Context, sourceKey string, destKey string, aggregator Aggregator, bucketDuration int) *StatusCmd
	TSCreateRuleWithArgs(ctx context.Context, sourceKey string, destKey string, aggregator Aggregator, bucketDuration int, options *TSCreateRuleOptions) *StatusCmd
	TSIncrBy(ctx context.Context, Key string, timestamp float64) *IntCmd
	TSIncrByWithArgs(ctx context.Context, key string, timestamp float64, options *TSIncrDecrOptions) *IntCmd
	TSDecrBy(ctx context.Context, Key string, timestamp float64) *IntCmd
	TSDecrByWithArgs(ctx context.Context, key string, timestamp float64, options *TSIncrDecrOptions) *IntCmd
	TSDel(ctx context.Context, Key string, fromTimestamp int, toTimestamp int) *IntCmd
	TSDeleteRule(ctx context.Context, sourceKey string, destKey string) *StatusCmd
	TSGet(ctx context.Context, key string) *TSTimestampValueCmd
	TSGetWithArgs(ctx context.Context, key string, options *TSGetOptions) *TSTimestampValueCmd
	TSInfo(ctx context.Context, key string) *MapStringInterfaceCmd
	TSInfoWithArgs(ctx context.Context, key string, options *TSInfoOptions) *MapStringInterfaceCmd
	TSMAdd(ctx context.Context, ktvSlices [][]interface{}) *IntSliceCmd
	TSQueryIndex(ctx context.Context, filterExpr []string) *StringSliceCmd
	TSRevRange(ctx context.Context, key string, fromTimestamp int, toTimestamp int) *TSTimestampValueSliceCmd
	TSRevRangeWithArgs(ctx context.Context, key string, fromTimestamp int, toTimestamp int, options *TSRevRangeOptions) *TSTimestampValueSliceCmd
	TSRange(ctx context.Context, key string, fromTimestamp int, toTimestamp int) *TSTimestampValueSliceCmd
	TSRangeWithArgs(ctx context.Context, key string, fromTimestamp int, toTimestamp int, options *TSRangeOptions) *TSTimestampValueSliceCmd
	TSMRange(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string) *MapStringSliceInterfaceCmd
	TSMRangeWithArgs(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string, options *TSMRangeOptions) *MapStringSliceInterfaceCmd
	TSMRevRange(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string) *MapStringSliceInterfaceCmd
	TSMRevRangeWithArgs(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string, options *TSMRevRangeOptions) *MapStringSliceInterfaceCmd
	TSMGet(ctx context.Context, filters []string) *MapStringSliceInterfaceCmd
	TSMGetWithArgs(ctx context.Context, filters []string, options *TSMGetOptions) *MapStringSliceInterfaceCmd
}

type JSONCmdable interface {
	JSONArrAppend(ctx context.Context, key, path string, values ...interface{}) *IntSliceCmd
	JSONArrIndex(ctx context.Context, key, path string, value ...interface{}) *IntSliceCmd
	JSONArrIndexWithArgs(ctx context.Context, key, path string, options *JSONArrIndexArgs, value ...interface{}) *IntSliceCmd
	JSONArrInsert(ctx context.Context, key, path string, index int64, values ...interface{}) *IntSliceCmd
	JSONArrLen(ctx context.Context, key, path string) *IntSliceCmd
	JSONArrPop(ctx context.Context, key, path string, index int) *StringSliceCmd
	JSONArrTrim(ctx context.Context, key, path string) *IntSliceCmd
	JSONArrTrimWithArgs(ctx context.Context, key, path string, options *JSONArrTrimArgs) *IntSliceCmd
	JSONClear(ctx context.Context, key, path string) *IntCmd
	JSONDebugMemory(ctx context.Context, key, path string) *IntCmd
	JSONDel(ctx context.Context, key, path string) *IntCmd
	JSONForget(ctx context.Context, key, path string) *IntCmd
	JSONGet(ctx context.Context, key string, paths ...string) *JSONCmd
	JSONGetWithArgs(ctx context.Context, key string, options *JSONGetArgs, paths ...string) *JSONCmd
	JSONMerge(ctx context.Context, key, path string, value string) *StatusCmd
	JSONMSetArgs(ctx context.Context, docs []JSONSetArgs) *StatusCmd
	JSONMSet(ctx context.Context, params ...interface{}) *StatusCmd
	JSONMGet(ctx context.Context, path string, keys ...string) *JSONSliceCmd
	JSONNumIncrBy(ctx context.Context, key, path string, value float64) *JSONCmd
	JSONObjKeys(ctx context.Context, key, path string) *SliceCmd
	JSONObjLen(ctx context.Context, key, path string) *IntPointerSliceCmd
	JSONSet(ctx context.Context, key, path string, value interface{}) *StatusCmd
	JSONSetMode(ctx context.Context, key, path string, value interface{}, mode string) *StatusCmd
	JSONStrAppend(ctx context.Context, key, path, value string) *IntPointerSliceCmd
	JSONStrLen(ctx context.Context, key, path string) *IntPointerSliceCmd
	JSONToggle(ctx context.Context, key, path string) *IntPointerSliceCmd
	JSONType(ctx context.Context, key, path string) *JSONSliceCmd
}

var _ Cmdable = (*Compat)(nil)

type Compat struct {
	client rueidis.Client
	maxp   int
	pOnly  bool
}

// CacheCompat implements commands that support client-side caching.
type CacheCompat struct {
	client rueidis.Client
	ttl    time.Duration
}

func NewAdapter(client rueidis.Client) Cmdable {
	return &Compat{client: client, maxp: runtime.GOMAXPROCS(0)}
}

func (c *Compat) Cache(ttl time.Duration) CacheCompat {
	return CacheCompat{client: c.client, ttl: ttl}
}

func (c *Compat) Command(ctx context.Context) *CommandsInfoCmd {
	cmd := c.client.B().Command().Build()
	resp := c.client.Do(ctx, cmd)
	return newCommandsInfoCmd(resp)
}

type FilterBy struct {
	Module  string
	ACLCat  string
	Pattern string
}

func (c *Compat) CommandList(ctx context.Context, filter FilterBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if filter.Module != "" {
		resp = c.client.Do(ctx, c.client.B().CommandList().FilterbyModuleName(filter.Module).Build())
	} else if filter.Pattern != "" {
		resp = c.client.Do(ctx, c.client.B().CommandList().FilterbyPatternPattern(filter.Pattern).Build())
	} else if filter.ACLCat != "" {
		resp = c.client.Do(ctx, c.client.B().CommandList().FilterbyAclcatCategory(filter.ACLCat).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().CommandList().Build())
	}
	return newStringSliceCmd(resp)
}

func (c *Compat) CommandGetKeys(ctx context.Context, commands ...any) *StringSliceCmd {
	cmd := c.client.B().CommandGetkeys().Command(commands[0].(string)).Arg(argsToSlice(commands[1:])...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) CommandGetKeysAndFlags(ctx context.Context, commands ...any) *KeyFlagsCmd {
	cmd := c.client.B().CommandGetkeysandflags().Command(commands[0].(string)).Arg(argsToSlice(commands[1:])...).Build()
	resp := c.client.Do(ctx, cmd)
	return newKeyFlagsCmd(resp)
}

func (c *Compat) ClientGetName(ctx context.Context) *StringCmd {
	cmd := c.client.B().ClientGetname().Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) Echo(ctx context.Context, message any) *StringCmd {
	cmd := c.client.B().Echo().Message(str(message)).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) Ping(ctx context.Context) *StatusCmd {
	cmd := c.client.B().Ping().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) Quit(ctx context.Context) *StatusCmd {
	cmd := c.client.B().Quit().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) Del(ctx context.Context, keys ...string) *IntCmd {
	cmd := c.client.B().Del().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Unlink(ctx context.Context, keys ...string) *IntCmd {
	cmd := c.client.B().Unlink().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Dump(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Dump().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) Exists(ctx context.Context, keys ...string) *IntCmd {
	cmd := c.client.B().Exists().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Expire(ctx context.Context, key string, seconds time.Duration) *BoolCmd {
	cmd := c.client.B().Expire().Key(key).Seconds(formatSec(seconds)).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) ExpireAt(ctx context.Context, key string, timestamp time.Time) *BoolCmd {
	cmd := c.client.B().Expireat().Key(key).Timestamp(timestamp.Unix()).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}
func (c *Compat) ExpireTime(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().Expiretime().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newDurationCmd(resp, time.Second)
}

func (c *Compat) ExpireNX(ctx context.Context, key string, seconds time.Duration) *BoolCmd {
	cmd := c.client.B().Expire().Key(key).Seconds(formatSec(seconds)).Nx().Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) ExpireXX(ctx context.Context, key string, seconds time.Duration) *BoolCmd {
	cmd := c.client.B().Expire().Key(key).Seconds(formatSec(seconds)).Xx().Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) ExpireGT(ctx context.Context, key string, seconds time.Duration) *BoolCmd {
	cmd := c.client.B().Expire().Key(key).Seconds(formatSec(seconds)).Gt().Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) ExpireLT(ctx context.Context, key string, seconds time.Duration) *BoolCmd {
	cmd := c.client.B().Expire().Key(key).Seconds(formatSec(seconds)).Lt().Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) Keys(ctx context.Context, pattern string) *StringSliceCmd {
	var mu sync.Mutex
	ret := &StringSliceCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		res, err := c.Do(ctx, c.B().Keys().Pattern(pattern).Build()).AsStrSlice()
		if err == nil {
			mu.Lock()
			ret.val = append(ret.val, res...)
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) Migrate(ctx context.Context, host string, port int64, key string, db int64, timeout time.Duration) *StatusCmd {
	cmd := c.client.B().Migrate().Host(host).Port(port).Key(key).DestinationDb(db).Timeout(formatSec(timeout)).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) Move(ctx context.Context, key string, db int64) *BoolCmd {
	cmd := c.client.B().Move().Key(key).Db(db).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) ObjectRefCount(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().ObjectRefcount().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ObjectEncoding(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().ObjectEncoding().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) ObjectIdleTime(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().ObjectIdletime().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newDurationCmd(resp, time.Second)
}
func (c *Compat) Persist(ctx context.Context, key string) *BoolCmd {
	cmd := c.client.B().Persist().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) PExpire(ctx context.Context, key string, milliseconds time.Duration) *BoolCmd {
	cmd := c.client.B().Pexpire().Key(key).Milliseconds(formatMs(milliseconds)).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) PExpireAt(ctx context.Context, key string, millisecondsTimestamp time.Time) *BoolCmd {
	cmd := c.client.B().Pexpireat().Key(key).MillisecondsTimestamp(millisecondsTimestamp.UnixNano() / int64(time.Millisecond)).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) PExpireTime(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().Pexpiretime().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newDurationCmd(resp, time.Millisecond)
}

func (c *Compat) PTTL(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().Pttl().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newDurationCmd(resp, time.Millisecond)
}

func (c *Compat) RandomKey(ctx context.Context) *StringCmd {
	cmd := c.client.B().Randomkey().Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) Rename(ctx context.Context, key, newkey string) *StatusCmd {
	cmd := c.client.B().Rename().Key(key).Newkey(newkey).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) RenameNX(ctx context.Context, key, newkey string) *BoolCmd {
	cmd := c.client.B().Renamenx().Key(key).Newkey(newkey).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) Restore(ctx context.Context, key string, ttl time.Duration, serializedValue string) *StatusCmd {
	cmd := c.client.B().Restore().Key(key).Ttl(formatMs(ttl)).SerializedValue(serializedValue).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) RestoreReplace(ctx context.Context, key string, ttl time.Duration, serializedValue string) *StatusCmd {
	cmd := c.client.B().Restore().Key(key).Ttl(formatMs(ttl)).SerializedValue(serializedValue).Replace().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) sort(command, key string, sort Sort) cmds.Arbitrary {
	cmd := c.client.B().Arbitrary(command).Keys(key)
	if sort.By != "" {
		cmd = cmd.Args("BY", sort.By)
	}
	if sort.Offset != 0 || sort.Count != 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(sort.Offset, 10), strconv.FormatInt(sort.Count, 10))
	}
	for _, get := range sort.Get {
		cmd = cmd.Args("GET").Args(get)
	}
	switch order := strings.ToUpper(sort.Order); order {
	case "ASC", "DESC":
		cmd = cmd.Args(order)
	case "":
	default:
		panic(fmt.Sprintf("invalid sort order %s", sort.Order))
	}
	if sort.Alpha {
		cmd = cmd.Args("ALPHA")
	}
	return cmd
}

func (c *Compat) Sort(ctx context.Context, key string, sort Sort) *StringSliceCmd {
	resp := c.client.Do(ctx, c.sort("SORT", key, sort).Build())
	return newStringSliceCmd(resp)
}

func (c *Compat) SortRO(ctx context.Context, key string, sort Sort) *StringSliceCmd {
	resp := c.client.Do(ctx, c.sort("SORT_RO", key, sort).Build())
	return newStringSliceCmd(resp)
}

func (c *Compat) SortStore(ctx context.Context, key, store string, sort Sort) *IntCmd {
	resp := c.client.Do(ctx, c.sort("SORT", key, sort).Args("STORE", store).Build())
	return newIntCmd(resp)
}

func (c *Compat) SortInterfaces(ctx context.Context, key string, sort Sort) *SliceCmd {
	resp := c.client.Do(ctx, c.sort("SORT", key, sort).Build())
	return newSliceCmd(resp, false)
}

func (c *Compat) Touch(ctx context.Context, keys ...string) *IntCmd {
	cmd := c.client.B().Touch().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) TTL(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().Ttl().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newDurationCmd(resp, time.Second)
}

func (c *Compat) Type(ctx context.Context, key string) *StatusCmd {
	cmd := c.client.B().Type().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) Append(ctx context.Context, key, value string) *IntCmd {
	cmd := c.client.B().Append().Key(key).Value(value).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Decr(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Decr().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) DecrBy(ctx context.Context, key string, decrement int64) *IntCmd {
	cmd := c.client.B().Decrby().Key(key).Decrement(decrement).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Get(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Get().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) GetRange(ctx context.Context, key string, start, end int64) *StringCmd {
	cmd := c.client.B().Getrange().Key(key).Start(start).End(end).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) GetSet(ctx context.Context, key string, value any) *StringCmd {
	cmd := c.client.B().Getset().Key(key).Value(str(value)).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

// GetEx An expiration of zero removes the TTL associated with the key (i.e. GETEX key persist).
// Requires Redis >= 6.2.0.
func (c *Compat) GetEx(ctx context.Context, key string, expiration time.Duration) *StringCmd {
	var resp rueidis.RedisResult
	if expiration > 0 {
		if usePrecise(expiration) {
			resp = c.client.Do(ctx, c.client.B().Getex().Key(key).PxMilliseconds(formatMs(expiration)).Build())
		} else {
			resp = c.client.Do(ctx, c.client.B().Getex().Key(key).ExSeconds(formatSec(expiration)).Build())
		}
	} else {
		resp = c.client.Do(ctx, c.client.B().Getex().Key(key).Build())
	}
	return newStringCmd(resp)
}

func (c *Compat) GetDel(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Getdel().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) Incr(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Incr().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) IncrBy(ctx context.Context, key string, increment int64) *IntCmd {
	cmd := c.client.B().Incrby().Key(key).Increment(increment).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) IncrByFloat(ctx context.Context, key string, increment float64) *FloatCmd {
	cmd := c.client.B().Incrbyfloat().Key(key).Increment(increment).Build()
	resp := c.client.Do(ctx, cmd)
	return newFloatCmd(resp)
}

func (c *Compat) MGet(ctx context.Context, keys ...string) *SliceCmd {
	cmd := c.client.B().Mget().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newSliceCmd(resp, false, keys...)
}

func (c *Compat) MSet(ctx context.Context, values ...any) *StatusCmd {
	partial := c.client.B().Mset().KeyValue()

	args := argsToSlice(values)
	for i := 0; i < len(args); i += 2 {
		partial = partial.KeyValue(args[i], args[i+1])
	}
	cmd := partial.Build()

	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) MSetNX(ctx context.Context, values ...any) *BoolCmd {
	partial := c.client.B().Msetnx().KeyValue()

	args := argsToSlice(values)
	for i := 0; i < len(args); i += 2 {
		partial = partial.KeyValue(args[i], args[i+1])
	}
	cmd := partial.Build()

	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

// Set key value [expiration]
//
// For no expiration use 0.
//
// For KEEPTTL use -1.
//
// For more options, use SetArgs.
func (c *Compat) Set(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd {
	var resp rueidis.RedisResult
	if expiration > 0 {
		if usePrecise(expiration) {
			resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).PxMilliseconds(formatMs(expiration)).Build())
		} else {
			resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).ExSeconds(formatSec(expiration)).Build())
		}
	} else if expiration == KeepTTL {
		resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Keepttl().Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Build())
	}
	return newStatusCmd(resp)
}

func (c *Compat) SetArgs(ctx context.Context, key string, value any, a SetArgs) *StatusCmd {
	cmd := c.client.B().Arbitrary("SET").Keys(key).Args(str(value))
	if a.KeepTTL {
		cmd = cmd.Args("KEEPTTL")
	}
	if !a.ExpireAt.IsZero() {
		cmd = cmd.Args("EXAT", strconv.FormatInt(a.ExpireAt.Unix(), 10))
	}
	if a.TTL > 0 {
		if usePrecise(a.TTL) {
			cmd = cmd.Args("PX", strconv.FormatInt(formatMs(a.TTL), 10))
		} else {
			cmd = cmd.Args("EX", strconv.FormatInt(formatSec(a.TTL), 10))
		}
	}
	switch mode := strings.ToUpper(a.Mode); mode {
	case "XX", "NX":
		cmd = cmd.Args(mode)
	case "":
	default:
		panic(fmt.Sprintf("invalid mode for SET: %s", a.Mode))
	}
	if a.Get {
		cmd = cmd.Args("GET")
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newStatusCmd(resp)
}

func (c *Compat) SetEX(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd {
	cmd := c.client.B().Setex().Key(key).Seconds(formatSec(expiration)).Value(str(value)).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *BoolCmd {
	var resp rueidis.RedisResult

	switch expiration {
	case 0:
		resp = c.client.Do(ctx, c.client.B().Setnx().Key(key).Value(str(value)).Build())
	case KeepTTL:
		resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Nx().Keepttl().Build())
	default:
		if usePrecise(expiration) {
			resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Nx().PxMilliseconds(formatMs(expiration)).Build())
		} else {
			resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Nx().ExSeconds(formatSec(expiration)).Build())
		}
	}

	return newBoolCmd(resp)
}

func (c *Compat) SetXX(ctx context.Context, key string, value any, expiration time.Duration) *BoolCmd {
	var resp rueidis.RedisResult
	if expiration > 0 {
		if usePrecise(expiration) {
			resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Xx().PxMilliseconds(formatMs(expiration)).Build())
		} else {
			resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Xx().ExSeconds(formatSec(expiration)).Build())
		}
	} else if expiration == KeepTTL {
		resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Xx().Keepttl().Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Set().Key(key).Value(str(value)).Xx().Build())
	}
	return newBoolCmd(resp)
}

func (c *Compat) SetRange(ctx context.Context, key string, offset int64, value string) *IntCmd {
	cmd := c.client.B().Setrange().Key(key).Offset(offset).Value(value).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) StrLen(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Strlen().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Copy(ctx context.Context, source string, destination string, db int64, replace bool) *IntCmd {
	var resp rueidis.RedisResult
	if replace {
		resp = c.client.Do(ctx, c.client.B().Copy().Source(source).Destination(destination).Db(db).Replace().Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Copy().Source(source).Destination(destination).Db(db).Build())
	}
	return newIntCmd(resp)
}

func (c *Compat) GetBit(ctx context.Context, key string, offset int64) *IntCmd {
	cmd := c.client.B().Getbit().Key(key).Offset(offset).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) SetBit(ctx context.Context, key string, offset int64, value int64) *IntCmd {
	cmd := c.client.B().Setbit().Key(key).Offset(offset).Value(value).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) BitCount(ctx context.Context, key string, bitCount *BitCount) *IntCmd {

	var resp rueidis.RedisResult
	if bitCount == nil {
		resp = c.client.Do(ctx, c.client.B().Bitcount().Key(key).Build())
		return newIntCmd(resp)
	}

	if bitCount.Unit == "" {
		resp = c.client.Do(ctx, c.client.B().Bitcount().Key(key).Start(bitCount.Start).End(bitCount.End).Build())
		return newIntCmd(resp)
	}

	switch bitCount.Unit {
	case BitCountIndexByte:
		resp = c.client.Do(ctx, c.client.B().Bitcount().Key(key).Start(bitCount.Start).End(bitCount.End).Byte().Build())
	case BitCountIndexBit:
		resp = c.client.Do(ctx, c.client.B().Bitcount().Key(key).Start(bitCount.Start).End(bitCount.End).Bit().Build())
	}
	return newIntCmd(resp)
}

func (c *Compat) BitOpAnd(ctx context.Context, destKey string, keys ...string) *IntCmd {
	cmd := c.client.B().Bitop().And().Destkey(destKey).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) BitOpOr(ctx context.Context, destKey string, keys ...string) *IntCmd {
	cmd := c.client.B().Bitop().Or().Destkey(destKey).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) BitOpXor(ctx context.Context, destKey string, keys ...string) *IntCmd {
	cmd := c.client.B().Bitop().Xor().Destkey(destKey).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) BitOpNot(ctx context.Context, destKey string, key string) *IntCmd {
	cmd := c.client.B().Bitop().Not().Destkey(destKey).Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) BitPos(ctx context.Context, key string, bit int64, pos ...int64) *IntCmd {
	var resp rueidis.RedisResult
	switch len(pos) {
	case 0:
		resp = c.client.Do(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Build())
	case 1:
		resp = c.client.Do(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(pos[0]).Build())
	case 2:
		resp = c.client.Do(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(pos[0]).End(pos[1]).Build())
	default:
		panic("too many arguments")
	}
	return newIntCmd(resp)
}

func (c *Compat) BitPosSpan(ctx context.Context, key string, bit, start, end int64, span string) *IntCmd {
	var resp rueidis.RedisResult
	if strings.ToLower(span) == "bit" {
		resp = c.client.Do(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(start).End(end).Bit().Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(start).End(end).Byte().Build())
	}
	return newIntCmd(resp)
}

func (c *Compat) BitField(ctx context.Context, key string, args ...any) *IntSliceCmd {
	cmd := c.client.B().Arbitrary("BITFIELD").Keys(key)
	for _, v := range args {
		cmd = cmd.Args(str(v))
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntSliceCmd(resp)
}

func (c *Compat) Scan(ctx context.Context, cursor uint64, match string, count int64) *ScanCmd {
	cmd := c.client.B().Arbitrary("SCAN", strconv.FormatInt(int64(cursor), 10))
	if match != "" {
		cmd = cmd.Args("MATCH", match)
	}
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	resp := c.client.Do(ctx, cmd.ReadOnly())
	return newScanCmd(resp)
}

func (c *Compat) ScanType(ctx context.Context, cursor uint64, match string, count int64, keyType string) *ScanCmd {
	cmd := c.client.B().Arbitrary("SCAN", strconv.FormatInt(int64(cursor), 10))
	if match != "" {
		cmd = cmd.Args("MATCH", match)
	}
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	resp := c.client.Do(ctx, cmd.Args("TYPE", keyType).ReadOnly())
	return newScanCmd(resp)
}

func (c *Compat) SScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd {
	cmd := c.client.B().Arbitrary("SSCAN").Keys(key).Args(strconv.FormatInt(int64(cursor), 10))
	if match != "" {
		cmd = cmd.Args("MATCH", match)
	}
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	resp := c.client.Do(ctx, cmd.ReadOnly())
	return newScanCmd(resp)
}

func (c *Compat) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd {
	cmd := c.client.B().Arbitrary("HSCAN").Keys(key).Args(strconv.FormatInt(int64(cursor), 10))
	if match != "" {
		cmd = cmd.Args("MATCH", match)
	}
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	resp := c.client.Do(ctx, cmd.ReadOnly())
	return newScanCmd(resp)
}

func (c *Compat) ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) *ScanCmd {
	cmd := c.client.B().Arbitrary("ZSCAN").Keys(key).Args(strconv.FormatInt(int64(cursor), 10))
	if match != "" {
		cmd = cmd.Args("MATCH", match)
	}
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	resp := c.client.Do(ctx, cmd.ReadOnly())
	return newScanCmd(resp)
}

func (c *Compat) HDel(ctx context.Context, key string, fields ...string) *IntCmd {
	cmd := c.client.B().Hdel().Key(key).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) HExists(ctx context.Context, key, field string) *BoolCmd {
	cmd := c.client.B().Hexists().Key(key).Field(field).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) HGet(ctx context.Context, key, field string) *StringCmd {
	cmd := c.client.B().Hget().Key(key).Field(field).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) HGetAll(ctx context.Context, key string) *StringStringMapCmd {
	cmd := c.client.B().Hgetall().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringStringMapCmd(resp)
}

func (c *Compat) HIncrBy(ctx context.Context, key, field string, incr int64) *IntCmd {
	cmd := c.client.B().Hincrby().Key(key).Field(field).Increment(incr).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) HIncrByFloat(ctx context.Context, key, field string, incr float64) *FloatCmd {
	cmd := c.client.B().Hincrbyfloat().Key(key).Field(field).Increment(incr).Build()
	resp := c.client.Do(ctx, cmd)
	return newFloatCmd(resp)
}

func (c *Compat) HKeys(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().Hkeys().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) HLen(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Hlen().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) HMGet(ctx context.Context, key string, fields ...string) *SliceCmd {
	cmd := c.client.B().Hmget().Key(key).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newSliceCmd(resp, false, fields...)
}

// HSet requires Redis v4 for multiple field/value pairs support.
func (c *Compat) HSet(ctx context.Context, key string, values ...any) *IntCmd {
	partial := c.client.B().Hset().Key(key).FieldValue()

	args := argsToSlice(values)
	for i := 0; i < len(args); i += 2 {
		partial = partial.FieldValue(args[i], args[i+1])
	}
	cmd := partial.Build()

	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

// HMSet is a deprecated version of HSet left for compatibility with Redis 3.
func (c *Compat) HMSet(ctx context.Context, key string, values ...any) *BoolCmd {
	partial := c.client.B().Hset().Key(key).FieldValue()

	args := argsToSlice(values)
	for i := 0; i < len(args); i += 2 {
		partial = partial.FieldValue(args[i], args[i+1])
	}
	cmd := partial.Build()

	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) HSetNX(ctx context.Context, key, field string, value any) *BoolCmd {
	cmd := c.client.B().Hsetnx().Key(key).Field(field).Value(str(value)).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) HVals(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().Hvals().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) HRandField(ctx context.Context, key string, count int64) *StringSliceCmd {
	return newStringSliceCmd(c.client.Do(ctx, c.client.B().Hrandfield().Key(key).Count(count).Build()))
}

func (c *Compat) HRandFieldWithValues(ctx context.Context, key string, count int64) *KeyValueSliceCmd {
	return newKeyValueSliceCmd(c.client.Do(ctx, c.client.B().Hrandfield().Key(key).Count(count).Withvalues().Build()))
}

func (c *Compat) HExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hexpire().Key(key).Seconds(formatSec(expiration)).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	var cmd rueidis.Completed
	if expirationArgs.NX {
		cmd = c.client.B().Hexpire().Key(key).Seconds(formatSec(expiration)).Nx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.XX {
		cmd = c.client.B().Hexpire().Key(key).Seconds(formatSec(expiration)).Xx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.GT {
		cmd = c.client.B().Hexpire().Key(key).Seconds(formatSec(expiration)).Gt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.LT {
		cmd = c.client.B().Hexpire().Key(key).Seconds(formatSec(expiration)).Lt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else {
		cmd = c.client.B().Hexpire().Key(key).Seconds(formatSec(expiration)).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	}
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hpexpire().Key(key).Milliseconds(formatMs(expiration)).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	var cmd rueidis.Completed
	if expirationArgs.NX {
		cmd = c.client.B().Hpexpire().Key(key).Milliseconds(formatMs(expiration)).Nx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.XX {
		cmd = c.client.B().Hpexpire().Key(key).Milliseconds(formatMs(expiration)).Xx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.GT {
		cmd = c.client.B().Hpexpire().Key(key).Milliseconds(formatMs(expiration)).Gt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.LT {
		cmd = c.client.B().Hpexpire().Key(key).Milliseconds(formatMs(expiration)).Lt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else {
		cmd = c.client.B().Hpexpire().Key(key).Milliseconds(formatMs(expiration)).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	}
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hexpireat().Key(key).UnixTimeSeconds(tm.Unix()).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	var cmd rueidis.Completed
	if expirationArgs.NX {
		cmd = c.client.B().Hexpireat().Key(key).UnixTimeSeconds(tm.Unix()).Nx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.XX {
		cmd = c.client.B().Hexpireat().Key(key).UnixTimeSeconds(tm.Unix()).Xx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.GT {
		cmd = c.client.B().Hexpireat().Key(key).UnixTimeSeconds(tm.Unix()).Gt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.LT {
		cmd = c.client.B().Hexpireat().Key(key).UnixTimeSeconds(tm.Unix()).Lt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else {
		cmd = c.client.B().Hexpireat().Key(key).UnixTimeSeconds(tm.Unix()).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	}
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hpexpireat().Key(key).UnixTimeMilliseconds(tm.UnixMilli()).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs HExpireArgs, fields ...string) *IntSliceCmd {
	var cmd rueidis.Completed
	if expirationArgs.NX {
		cmd = c.client.B().Hpexpireat().Key(key).UnixTimeMilliseconds(tm.UnixMilli()).Nx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.XX {
		cmd = c.client.B().Hpexpireat().Key(key).UnixTimeMilliseconds(tm.UnixMilli()).Xx().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.GT {
		cmd = c.client.B().Hpexpireat().Key(key).UnixTimeMilliseconds(tm.UnixMilli()).Gt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else if expirationArgs.LT {
		cmd = c.client.B().Hpexpireat().Key(key).UnixTimeMilliseconds(tm.UnixMilli()).Lt().Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	} else {
		cmd = c.client.B().Hpexpireat().Key(key).UnixTimeMilliseconds(tm.UnixMilli()).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	}
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPersist(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hpersist().Key(key).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HExpireTime(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hexpiretime().Key(key).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPExpireTime(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hpexpiretime().Key(key).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HTTL(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Httl().Key(key).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) HPTTL(ctx context.Context, key string, fields ...string) *IntSliceCmd {
	cmd := c.client.B().Hpttl().Key(key).Fields().Numfields(int64(len(fields))).Field(fields...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntSliceCmd(resp)
}

func (c *Compat) BLPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd {
	cmd := c.client.B().Blpop().Key(keys...).Timeout(float64(formatSec(timeout))).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) BLMPop(ctx context.Context, timeout time.Duration, direction string, count int64, keys ...string) *KeyValuesCmd {
	cmd := c.client.B().Arbitrary("BLMPOP", strconv.FormatInt(formatSec(timeout), 10), strconv.Itoa(len(keys))).Keys(keys...)
	cmd = cmd.Args(direction)
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	return newKeyValuesCmd(c.client.Do(ctx, cmd.Blocking()))
}

func (c *Compat) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd {
	cmd := c.client.B().Brpop().Key(keys...).Timeout(float64(formatSec(timeout))).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) BRPopLPush(ctx context.Context, source, destination string, timeout time.Duration) *StringCmd {
	cmd := c.client.B().Brpoplpush().Source(source).Destination(destination).Timeout(float64(formatSec(timeout))).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) LIndex(ctx context.Context, key string, index int64) *StringCmd {
	cmd := c.client.B().Lindex().Key(key).Index(index).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) LInsert(ctx context.Context, key, op string, pivot, element any) *IntCmd {
	var resp rueidis.RedisResult
	switch strings.ToUpper(op) {
	case "BEFORE":
		resp = c.client.Do(ctx, c.client.B().Linsert().Key(key).Before().Pivot(str(pivot)).Element(str(element)).Build())
	case "AFTER":
		resp = c.client.Do(ctx, c.client.B().Linsert().Key(key).After().Pivot(str(pivot)).Element(str(element)).Build())
	default:
		panic(fmt.Sprintf("Invalid op argument value: %s", op))
	}
	return newIntCmd(resp)
}

func (c *Compat) LInsertBefore(ctx context.Context, key string, pivot, element any) *IntCmd {
	cmd := c.client.B().Linsert().Key(key).Before().Pivot(str(pivot)).Element(str(element)).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LInsertAfter(ctx context.Context, key string, pivot, element any) *IntCmd {
	cmd := c.client.B().Linsert().Key(key).After().Pivot(str(pivot)).Element(str(element)).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LLen(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Llen().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LMPop(ctx context.Context, direction string, count int64, keys ...string) *KeyValuesCmd {
	cmd := c.client.B().Arbitrary("LMPOP", strconv.Itoa(len(keys))).Keys(keys...)
	cmd = cmd.Args(direction)
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	return newKeyValuesCmd(c.client.Do(ctx, cmd.Build()))
}

func (c *Compat) LPop(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Lpop().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) LPopCount(ctx context.Context, key string, count int64) *StringSliceCmd {
	cmd := c.client.B().Lpop().Key(key).Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) LPos(ctx context.Context, key string, element string, a LPosArgs) *IntCmd {
	cmd := c.client.B().Arbitrary("LPOS").Keys(key).Args(element)
	if a.Rank != 0 {
		cmd = cmd.Args("RANK", strconv.FormatInt(a.Rank, 10))
	}
	if a.MaxLen != 0 {
		cmd = cmd.Args("MAXLEN", strconv.FormatInt(a.MaxLen, 10))
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

func (c *Compat) LPosCount(ctx context.Context, key string, element string, count int64, a LPosArgs) *IntSliceCmd {
	cmd := c.client.B().Arbitrary("LPOS").Keys(key).Args(element).Args("COUNT", strconv.FormatInt(count, 10))
	if a.Rank != 0 {
		cmd = cmd.Args("RANK", strconv.FormatInt(a.Rank, 10))
	}
	if a.MaxLen != 0 {
		cmd = cmd.Args("MAXLEN", strconv.FormatInt(a.MaxLen, 10))
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntSliceCmd(resp)
}

func (c *Compat) LPush(ctx context.Context, key string, elements ...any) *IntCmd {
	cmd := c.client.B().Lpush().Key(key).Element(argsToSlice(elements)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LPushX(ctx context.Context, key string, elements ...any) *IntCmd {
	cmd := c.client.B().Lpushx().Key(key).Element(argsToSlice(elements)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	cmd := c.client.B().Lrange().Key(key).Start(start).Stop(stop).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) LRem(ctx context.Context, key string, count int64, element any) *IntCmd {
	cmd := c.client.B().Lrem().Key(key).Count(count).Element(str(element)).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LSet(ctx context.Context, key string, index int64, element any) *StatusCmd {
	cmd := c.client.B().Lset().Key(key).Index(index).Element(str(element)).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) LTrim(ctx context.Context, key string, start, stop int64) *StatusCmd {
	cmd := c.client.B().Ltrim().Key(key).Start(start).Stop(stop).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) RPop(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Rpop().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) RPopCount(ctx context.Context, key string, count int64) *StringSliceCmd {
	cmd := c.client.B().Rpop().Key(key).Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) RPopLPush(ctx context.Context, source, destination string) *StringCmd {
	cmd := c.client.B().Rpoplpush().Source(source).Destination(destination).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) RPush(ctx context.Context, key string, elements ...any) *IntCmd {
	cmd := c.client.B().Rpush().Key(key).Element(argsToSlice(elements)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) RPushX(ctx context.Context, key string, elements ...any) *IntCmd {
	cmd := c.client.B().Rpushx().Key(key).Element(argsToSlice(elements)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) LMove(ctx context.Context, source, destination, srcpos, destpos string) *StringCmd {
	resp := c.client.Do(ctx, c.client.B().Arbitrary("LMOVE").Keys(source, destination).Args(srcpos, destpos).Build())
	return newStringCmd(resp)
}

func (c *Compat) BLMove(ctx context.Context, source, destination, srcpos, destpos string, timeout time.Duration) *StringCmd {
	resp := c.client.Do(ctx, c.client.B().Arbitrary("BLMOVE").Keys(source, destination).Args(srcpos, destpos, strconv.FormatFloat(float64(formatSec(timeout)), 'f', -1, 64)).Blocking())
	return newStringCmd(resp)
}

func (c *Compat) SAdd(ctx context.Context, key string, members ...any) *IntCmd {
	cmd := c.client.B().Sadd().Key(key).Member()
	for _, m := range argsToSlice(members) {
		cmd = cmd.Member(str(m))
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

func (c *Compat) SCard(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Scard().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) SDiff(ctx context.Context, keys ...string) *StringSliceCmd {
	cmd := c.client.B().Sdiff().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) SDiffStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	cmd := c.client.B().Sdiffstore().Destination(destination).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) SInter(ctx context.Context, keys ...string) *StringSliceCmd {
	cmd := c.client.B().Sinter().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) SInterCard(ctx context.Context, limit int64, keys ...string) *IntCmd {
	return newIntCmd(c.client.Do(ctx, c.client.B().Sintercard().Numkeys(int64(len(keys))).Key(keys...).Limit(limit).Build()))
}

func (c *Compat) SInterStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	cmd := c.client.B().Sinterstore().Destination(destination).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) SIsMember(ctx context.Context, key string, member any) *BoolCmd {
	cmd := c.client.B().Sismember().Key(key).Member(str(member)).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) SMIsMember(ctx context.Context, key string, members ...any) *BoolSliceCmd {
	cmd := c.client.B().Smismember().Key(key).Member(argsToSlice(members)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolSliceCmd(resp)
}

func (c *Compat) SMembers(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().Smembers().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) SMembersMap(ctx context.Context, key string) *StringStructMapCmd {
	cmd := c.client.B().Smembers().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringStructMapCmd(resp)
}

func (c *Compat) SMove(ctx context.Context, source, destination string, member any) *BoolCmd {
	cmd := c.client.B().Smove().Source(source).Destination(destination).Member(str(member)).Build()
	resp := c.client.Do(ctx, cmd)
	return newBoolCmd(resp)
}

func (c *Compat) SPop(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Spop().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) SPopN(ctx context.Context, key string, count int64) *StringSliceCmd {
	cmd := c.client.B().Spop().Key(key).Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) SRandMember(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Srandmember().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) SRandMemberN(ctx context.Context, key string, count int64) *StringSliceCmd {
	cmd := c.client.B().Srandmember().Key(key).Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) SRem(ctx context.Context, key string, members ...any) *IntCmd {
	cmd := c.client.B().Srem().Key(key).Member(argsToSlice(members)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) SUnion(ctx context.Context, keys ...string) *StringSliceCmd {
	cmd := c.client.B().Sunion().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) SUnionStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	cmd := c.client.B().Sunionstore().Destination(destination).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XAdd(ctx context.Context, a XAddArgs) *StringCmd {
	cmd := c.client.B().Arbitrary("XADD").Keys(a.Stream)
	if a.NoMkStream {
		cmd = cmd.Args("NOMKSTREAM")
	}
	switch {
	case a.MaxLen > 0:
		if a.Approx {
			cmd = cmd.Args("MAXLEN", "~", strconv.FormatInt(a.MaxLen, 10))
		} else {
			cmd = cmd.Args("MAXLEN", strconv.FormatInt(a.MaxLen, 10))
		}
	case a.MinID != "":
		if a.Approx {
			cmd = cmd.Args("MINID", "~", a.MinID)
		} else {
			cmd = cmd.Args("MINID", a.MinID)
		}
	}
	if a.Limit > 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(a.Limit, 10))
	}
	if a.ID != "" {
		cmd = cmd.Args(a.ID)
	} else {
		cmd = cmd.Args("*")
	}
	cmd = cmd.Args(argToSlice(a.Values)...)
	resp := c.client.Do(ctx, cmd.Build())
	return newStringCmd(resp)
}

func (c *Compat) XDel(ctx context.Context, stream string, ids ...string) *IntCmd {
	cmd := c.client.B().Xdel().Key(stream).Id(ids...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XLen(ctx context.Context, stream string) *IntCmd {
	cmd := c.client.B().Xlen().Key(stream).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XRange(ctx context.Context, stream, start, stop string) *XMessageSliceCmd {
	cmd := c.client.B().Xrange().Key(stream).Start(start).End(stop).Build()
	resp := c.client.Do(ctx, cmd)
	return newXMessageSliceCmd(resp)
}

func (c *Compat) XRangeN(ctx context.Context, stream, start, stop string, count int64) *XMessageSliceCmd {
	cmd := c.client.B().Xrange().Key(stream).Start(start).End(stop).Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newXMessageSliceCmd(resp)
}

func (c *Compat) XRevRange(ctx context.Context, stream, stop, start string) *XMessageSliceCmd {
	cmd := c.client.B().Xrevrange().Key(stream).End(stop).Start(start).Build()
	resp := c.client.Do(ctx, cmd)
	return newXMessageSliceCmd(resp)
}

func (c *Compat) XRevRangeN(ctx context.Context, stream, stop, start string, count int64) *XMessageSliceCmd {
	cmd := c.client.B().Xrevrange().Key(stream).End(stop).Start(start).Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newXMessageSliceCmd(resp)
}

func (c *Compat) XRead(ctx context.Context, a XReadArgs) *XStreamSliceCmd {
	cmd := c.client.B().Arbitrary("XREAD")
	if a.Count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(a.Count, 10))
	}
	if a.Block >= 0 {
		cmd = cmd.Args("BLOCK", strconv.FormatInt(formatMs(a.Block), 10))
	}
	cmd = cmd.Args("STREAMS")
	cmd = cmd.Keys(a.Streams[:len(a.Streams)/2]...).Args(a.Streams[len(a.Streams)/2:]...)
	var resp rueidis.RedisResult
	if a.Block >= 0 {
		resp = c.client.Do(ctx, cmd.Blocking())
	} else {
		resp = c.client.Do(ctx, cmd.Build())
	}
	return newXStreamSliceCmd(resp)
}

func (c *Compat) XReadStreams(ctx context.Context, streams ...string) *XStreamSliceCmd {
	return c.XRead(ctx, XReadArgs{Streams: streams, Block: -1})
}

func (c *Compat) XGroupCreate(ctx context.Context, stream, group, start string) *StatusCmd {
	cmd := c.client.B().XgroupCreate().Key(stream).Group(group).Id(start).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) XGroupCreateMkStream(ctx context.Context, stream, group, start string) *StatusCmd {
	cmd := c.client.B().XgroupCreate().Key(stream).Group(group).Id(start).Mkstream().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) XGroupSetID(ctx context.Context, stream, group, start string) *StatusCmd {
	cmd := c.client.B().XgroupSetid().Key(stream).Group(group).Id(start).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) XGroupDestroy(ctx context.Context, stream, group string) *IntCmd {
	cmd := c.client.B().XgroupDestroy().Key(stream).Group(group).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XGroupCreateConsumer(ctx context.Context, stream, group, consumer string) *IntCmd {
	cmd := c.client.B().XgroupCreateconsumer().Key(stream).Group(group).Consumer(consumer).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *IntCmd {
	cmd := c.client.B().XgroupDelconsumer().Key(stream).Group(group).Consumername(consumer).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XReadGroup(ctx context.Context, a XReadGroupArgs) *XStreamSliceCmd {
	cmd := c.client.B().Arbitrary("XREADGROUP")
	cmd = cmd.Args("GROUP", a.Group, a.Consumer)
	if a.Count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(a.Count, 10))
	}
	if a.Block >= 0 {
		cmd = cmd.Args("BLOCK", strconv.FormatInt(formatMs(a.Block), 10))
	}
	if a.NoAck {
		cmd = cmd.Args("NOACK")
	}
	cmd = cmd.Args("STREAMS")
	cmd = cmd.Keys(a.Streams[:len(a.Streams)/2]...).Args(a.Streams[len(a.Streams)/2:]...)
	var resp rueidis.RedisResult
	if a.Block >= 0 {
		resp = c.client.Do(ctx, cmd.Blocking())
	} else {
		resp = c.client.Do(ctx, cmd.Build())
	}
	return newXStreamSliceCmd(resp)
}

func (c *Compat) XAck(ctx context.Context, stream, group string, ids ...string) *IntCmd {
	cmd := c.client.B().Xack().Key(stream).Group(group).Id(ids...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) XPending(ctx context.Context, stream, group string) *XPendingCmd {
	cmd := c.client.B().Xpending().Key(stream).Group(group).Build()
	resp := c.client.Do(ctx, cmd)
	return newXPendingCmd(resp)
}

func (c *Compat) XPendingExt(ctx context.Context, a XPendingExtArgs) *XPendingExtCmd {
	cmd := c.client.B().Arbitrary("XPENDING").Keys(a.Stream).Args(a.Group)
	if a.Idle != 0 {
		cmd = cmd.Args("IDLE", strconv.FormatInt(formatMs(a.Idle), 10))
	}
	cmd = cmd.Args(a.Start, a.End, strconv.FormatInt(a.Count, 10))
	if a.Consumer != "" {
		cmd = cmd.Args(a.Consumer)
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newXPendingExtCmd(resp)
}

func (c *Compat) XClaim(ctx context.Context, a XClaimArgs) *XMessageSliceCmd {
	cmd := c.client.B().Xclaim().Key(a.Stream).Group(a.Group).Consumer(a.Consumer).MinIdleTime(strconv.FormatInt(formatMs(a.MinIdle), 10)).Id(a.Messages...).Build()
	resp := c.client.Do(ctx, cmd)
	return newXMessageSliceCmd(resp)
}

func (c *Compat) XClaimJustID(ctx context.Context, a XClaimArgs) *StringSliceCmd {
	cmd := c.client.B().Xclaim().Key(a.Stream).Group(a.Group).Consumer(a.Consumer).MinIdleTime(strconv.FormatInt(formatMs(a.MinIdle), 10)).Id(a.Messages...).Justid().Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) XAutoClaim(ctx context.Context, a XAutoClaimArgs) *XAutoClaimCmd {
	var resp rueidis.RedisResult
	if a.Count > 0 {
		resp = c.client.Do(ctx, c.client.B().Xautoclaim().Key(a.Stream).Group(a.Group).Consumer(a.Consumer).MinIdleTime(strconv.FormatInt(formatMs(a.MinIdle), 10)).Start(a.Start).Count(a.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Xautoclaim().Key(a.Stream).Group(a.Group).Consumer(a.Consumer).MinIdleTime(strconv.FormatInt(formatMs(a.MinIdle), 10)).Start(a.Start).Build())
	}
	return newXAutoClaimCmd(resp)
}

func (c *Compat) XAutoClaimJustID(ctx context.Context, a XAutoClaimArgs) *XAutoClaimJustIDCmd {
	var resp rueidis.RedisResult
	if a.Count > 0 {
		resp = c.client.Do(ctx, c.client.B().Xautoclaim().Key(a.Stream).Group(a.Group).Consumer(a.Consumer).MinIdleTime(strconv.FormatInt(formatMs(a.MinIdle), 10)).Start(a.Start).Count(a.Count).Justid().Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Xautoclaim().Key(a.Stream).Group(a.Group).Consumer(a.Consumer).MinIdleTime(strconv.FormatInt(formatMs(a.MinIdle), 10)).Start(a.Start).Justid().Build())
	}
	return newXAutoClaimJustIDCmd(resp)
}

// xTrim If approx is true, add the "~" parameter, otherwise it is the default "=" (redis default).
// example:
//
//	XTRIM key MAXLEN/MINID threshold LIMIT limit.
//	XTRIM key MAXLEN/MINID ~ threshold LIMIT limit.
//
// The redis-server version is lower than 6.2, please set limit to 0.
func (c *Compat) xTrim(ctx context.Context, key, strategy string,
	approx bool, threshold string, limit int64) *IntCmd {
	cmd := c.client.B().Arbitrary("XTRIM").Keys(key).Args(strategy)
	if approx {
		cmd = cmd.Args("~")
	}
	cmd = cmd.Args(threshold)
	if limit > 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(limit, 10))
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

// XTrimMaxLen No `~` rules are used, `limit` cannot be used.
// cmd: XTRIM key MAXLEN maxLen
func (c *Compat) XTrimMaxLen(ctx context.Context, key string, maxLen int64) *IntCmd {
	return c.xTrim(ctx, key, "MAXLEN", false, strconv.FormatInt(maxLen, 10), 0)
}

// XTrimMaxLenApprox LIMIT has a bug, please confirm it and use it.
// issue: https://github.com/redis/redis/issues/9046
// cmd: XTRIM key MAXLEN ~ maxLen LIMIT limit
func (c *Compat) XTrimMaxLenApprox(ctx context.Context, key string, maxLen, limit int64) *IntCmd {
	return c.xTrim(ctx, key, "MAXLEN", true, strconv.FormatInt(maxLen, 10), limit)
}

// XTrimMinID No `~` rules are used, `limit` cannot be used.
// cmd: XTRIM key MINID minID
func (c *Compat) XTrimMinID(ctx context.Context, key string, minID string) *IntCmd {
	return c.xTrim(ctx, key, "MINID", false, minID, 0)
}

// XTrimMinIDApprox LIMIT has a bug, please confirm it and use it.
// issue: https://github.com/redis/redis/issues/9046
// cmd: XTRIM key MINID ~ minID LIMIT limit
func (c *Compat) XTrimMinIDApprox(ctx context.Context, key string, minID string, limit int64) *IntCmd {
	return c.xTrim(ctx, key, "MINID", true, minID, limit)
}

func (c *Compat) XInfoGroups(ctx context.Context, key string) *XInfoGroupsCmd {
	cmd := c.client.B().XinfoGroups().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newXInfoGroupsCmd(resp)
}

func (c *Compat) XInfoStream(ctx context.Context, key string) *XInfoStreamCmd {
	cmd := c.client.B().XinfoStream().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newXInfoStreamCmd(resp)
}

func (c *Compat) XInfoStreamFull(ctx context.Context, key string, count int64) *XInfoStreamFullCmd {
	cmd := c.client.B().XinfoStream().Key(key).Full().Count(count).Build()
	resp := c.client.Do(ctx, cmd)
	return newXInfoStreamFullCmd(resp)
}

func (c *Compat) XInfoConsumers(ctx context.Context, key, group string) *XInfoConsumersCmd {
	cmd := c.client.B().XinfoConsumers().Key(key).Group(group).Build()
	resp := c.client.Do(ctx, cmd)
	return newXInfoConsumersCmd(resp)
}

// BZPopMax Redis `BZPOPMAX key [key ...] timeout` command.
func (c *Compat) BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) *ZWithKeyCmd {
	cmd := c.client.B().Bzpopmax().Key(keys...).Timeout(float64(formatSec(timeout))).Build()
	resp := c.client.Do(ctx, cmd)
	return newZWithKeyCmd(resp)
}

// BZPopMin Redis `BZPOPMIN key [key ...] timeout` command.
func (c *Compat) BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) *ZWithKeyCmd {
	cmd := c.client.B().Bzpopmin().Key(keys...).Timeout(float64(formatSec(timeout))).Build()
	resp := c.client.Do(ctx, cmd)
	return newZWithKeyCmd(resp)
}

func (c *Compat) BZMPop(ctx context.Context, timeout time.Duration, order string, count int64, keys ...string) *ZSliceWithKeyCmd {
	cmd := c.client.B().Arbitrary("BZMPOP", strconv.FormatInt(formatSec(timeout), 10), strconv.Itoa(len(keys))).Keys(keys...)
	cmd = cmd.Args(order)
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	return newZSliceWithKeyCmd(c.client.Do(ctx, cmd.Blocking()))
}

// ZAdd Redis `ZADD key score member [score member ...]` command.
func (c *Compat) ZAdd(ctx context.Context, key string, members ...Z) *IntCmd {
	return newIntCmd(c.zAddArgs(ctx, key, false, ZAddArgs{Members: members}))
}

// ZAddNX Redis `ZADD key NX score member [score member ...]` command.
func (c *Compat) ZAddNX(ctx context.Context, key string, members ...Z) *IntCmd {
	return newIntCmd(c.zAddArgs(ctx, key, false, ZAddArgs{Members: members, NX: true}))
}

// ZAddXX Redis `ZADD key XX score member [score member ...]` command.
func (c *Compat) ZAddXX(ctx context.Context, key string, members ...Z) *IntCmd {
	return newIntCmd(c.zAddArgs(ctx, key, false, ZAddArgs{Members: members, XX: true}))
}

func (c *Compat) ZAddLT(ctx context.Context, key string, members ...Z) *IntCmd {
	return newIntCmd(c.zAddArgs(ctx, key, false, ZAddArgs{Members: members, LT: true}))
}

func (c *Compat) ZAddGT(ctx context.Context, key string, members ...Z) *IntCmd {
	return newIntCmd(c.zAddArgs(ctx, key, false, ZAddArgs{Members: members, GT: true}))
}

func (c *Compat) zAddArgs(ctx context.Context, key string, incr bool, args ZAddArgs) rueidis.RedisResult {
	cmd := c.client.B().Arbitrary("ZADD").Keys(key)
	// The GT, LT and NX options are mutually exclusive.
	if args.NX {
		cmd = cmd.Args("NX")
	} else {
		if args.XX {
			cmd = cmd.Args("XX")
		}
		if args.GT {
			cmd = cmd.Args("GT")
		} else if args.LT {
			cmd = cmd.Args("LT")
		}
	}
	if args.Ch {
		cmd = cmd.Args("CH")
	}
	if incr {
		cmd = cmd.Args("INCR")
	}
	for _, v := range args.Members {
		cmd = cmd.Args(strconv.FormatFloat(v.Score, 'f', -1, 64), v.Member)
	}
	resp := c.client.Do(ctx, cmd.Build())
	return resp
}

func (c *Compat) ZAddArgs(ctx context.Context, key string, args ZAddArgs) *IntCmd {
	return newIntCmd(c.zAddArgs(ctx, key, false, args))
}

func (c *Compat) ZAddArgsIncr(ctx context.Context, key string, args ZAddArgs) *FloatCmd {
	return newFloatCmd(c.zAddArgs(ctx, key, true, args))
}

func (c *Compat) ZCard(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Zcard().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZCount(ctx context.Context, key, min, max string) *IntCmd {
	cmd := c.client.B().Zcount().Key(key).Min(min).Max(max).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZLexCount(ctx context.Context, key, min, max string) *IntCmd {
	cmd := c.client.B().Zlexcount().Key(key).Min(min).Max(max).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZIncrBy(ctx context.Context, key string, increment float64, member string) *FloatCmd {
	cmd := c.client.B().Zincrby().Key(key).Increment(increment).Member(member).Build()
	resp := c.client.Do(ctx, cmd)
	return newFloatCmd(resp)
}

func zstore(cmd cmds.Arbitrary, store ZStore) cmds.Arbitrary {
	cmd = cmd.Args(strconv.Itoa(len(store.Keys))).Keys(store.Keys...)
	if len(store.Weights) > 0 {
		cmd = cmd.Args("WEIGHTS")
		for _, w := range store.Weights {
			cmd = cmd.Args(strconv.FormatInt(w, 10))
		}
	}
	if store.Aggregate != "" {
		cmd = cmd.Args("AGGREGATE", store.Aggregate)
	}
	return cmd
}

func (c *Compat) ZInter(ctx context.Context, store ZStore) *StringSliceCmd {
	return newStringSliceCmd(c.client.Do(ctx, zstore(c.client.B().Arbitrary("ZINTER"), store).ReadOnly()))
}

func (c *Compat) ZInterWithScores(ctx context.Context, store ZStore) *ZSliceCmd {
	return newZSliceCmd(c.client.Do(ctx, zstore(c.client.B().Arbitrary("ZINTER"), store).Args("WITHSCORES").ReadOnly()))
}

func (c *Compat) ZInterCard(ctx context.Context, limit int64, keys ...string) *IntCmd {
	return newIntCmd(c.client.Do(ctx, c.client.B().Zintercard().Numkeys(int64(len(keys))).Key(keys...).Limit(limit).Build()))
}

func (c *Compat) ZInterStore(ctx context.Context, destination string, store ZStore) *IntCmd {
	return newIntCmd(c.client.Do(ctx, zstore(c.client.B().Arbitrary("ZINTERSTORE").Keys(destination), store).Build()))
}

func (c *Compat) ZMPop(ctx context.Context, order string, count int64, keys ...string) *ZSliceWithKeyCmd {
	cmd := c.client.B().Arbitrary("ZMPOP", strconv.Itoa(len(keys))).Keys(keys...)
	cmd = cmd.Args(order)
	if count > 0 {
		cmd = cmd.Args("COUNT", strconv.FormatInt(count, 10))
	}
	return newZSliceWithKeyCmd(c.client.Do(ctx, cmd.Build()))
}

func (c *Compat) ZMScore(ctx context.Context, key string, members ...string) *FloatSliceCmd {
	cmd := c.client.B().Zmscore().Key(key).Member(members...).Build()
	resp := c.client.Do(ctx, cmd)
	return newFloatSliceCmd(resp)
}

func (c *Compat) ZPopMax(ctx context.Context, key string, count ...int64) *ZSliceCmd {
	var resp rueidis.RedisResult
	switch len(count) {
	case 0:
		resp = c.client.Do(ctx, c.client.B().Zpopmax().Key(key).Build())
	case 1:
		resp = c.client.Do(ctx, c.client.B().Zpopmax().Key(key).Count(count[0]).Build())
		return newZSliceCmd(resp)
	default:
		panic("too many arguments")
	}
	return newZSliceSingleCmd(resp)
}

func (c *Compat) ZPopMin(ctx context.Context, key string, count ...int64) *ZSliceCmd {
	var resp rueidis.RedisResult
	switch len(count) {
	case 0:
		resp = c.client.Do(ctx, c.client.B().Zpopmin().Key(key).Build())
	case 1:
		resp = c.client.Do(ctx, c.client.B().Zpopmin().Key(key).Count(count[0]).Build())
		return newZSliceCmd(resp)
	default:
		panic("too many arguments")
	}
	return newZSliceSingleCmd(resp)
}

func (c *Compat) zRangeArgs(withScores bool, z ZRangeArgs) rueidis.Completed {
	cmd := c.client.B().Arbitrary("ZRANGE").Keys(z.Key)
	if z.Rev && (z.ByScore || z.ByLex) {
		cmd = cmd.Args(str(z.Stop), str(z.Start))
	} else {
		cmd = cmd.Args(str(z.Start), str(z.Stop))
	}
	if z.ByScore {
		cmd = cmd.Args("BYSCORE")
	} else if z.ByLex {
		cmd = cmd.Args("BYLEX")
	}
	if z.Rev {
		cmd = cmd.Args("REV")
	}
	if z.Offset != 0 || z.Count != 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(z.Offset, 10), strconv.FormatInt(z.Count, 10))
	}
	if withScores {
		cmd = cmd.Args("WITHSCORES")
	}
	return cmd.Build()
}

func (c *Compat) ZRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	cmd := c.zRangeArgs(false, ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  stop,
	})
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd {
	cmd := c.zRangeArgs(true, ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  stop,
	})
	resp := c.client.Do(ctx, cmd)
	return newZSliceCmd(resp)
}

func (c *Compat) ZRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.Do(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Limit(opt.Offset, opt.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Build())
	}
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.Do(ctx, c.client.B().Zrangebylex().Key(key).Min(opt.Min).Max(opt.Max).Limit(opt.Offset, opt.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Zrangebylex().Key(key).Min(opt.Min).Max(opt.Max).Build())
	}
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.Do(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Withscores().Limit(opt.Offset, opt.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Withscores().Build())
	}
	return newZSliceCmd(resp)
}

func (c *Compat) ZRangeArgs(ctx context.Context, z ZRangeArgs) *StringSliceCmd {
	cmd := c.zRangeArgs(false, z)
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRangeArgsWithScores(ctx context.Context, z ZRangeArgs) *ZSliceCmd {
	cmd := c.zRangeArgs(true, z)
	resp := c.client.Do(ctx, cmd)
	return newZSliceCmd(resp)
}

func (c *Compat) ZRangeStore(ctx context.Context, dst string, z ZRangeArgs) *IntCmd {
	cmd := c.client.B().Arbitrary("ZRANGESTORE").Keys(dst, z.Key)
	if z.Rev && (z.ByScore || z.ByLex) {
		cmd = cmd.Args(str(z.Stop), str(z.Start))
	} else {
		cmd = cmd.Args(str(z.Start), str(z.Stop))
	}
	if z.ByScore {
		cmd = cmd.Args("BYSCORE")
	} else if z.ByLex {
		cmd = cmd.Args("BYLEX")
	}
	if z.Rev {
		cmd = cmd.Args("REV")
	}
	if z.Offset != 0 || z.Count != 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(z.Offset, 10), strconv.FormatInt(z.Count, 10))
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

func (c *Compat) ZRank(ctx context.Context, key, member string) *IntCmd {
	cmd := c.client.B().Zrank().Key(key).Member(member).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd {
	cmd := c.client.B().Zrank().Key(key).Member(member).Withscore().Build()
	resp := c.client.Do(ctx, cmd)
	return newRankWithScoreCmd(resp)
}

func (c *Compat) ZRem(ctx context.Context, key string, members ...any) *IntCmd {
	cmd := c.client.B().Zrem().Key(key).Member(argsToSlice(members)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *IntCmd {
	cmd := c.client.B().Zremrangebyrank().Key(key).Start(start).Stop(stop).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}
func (c *Compat) ZRemRangeByScore(ctx context.Context, key, min, max string) *IntCmd {
	cmd := c.client.B().Zremrangebyscore().Key(key).Min(min).Max(max).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZRemRangeByLex(ctx context.Context, key string, min, max string) *IntCmd {
	cmd := c.client.B().Zremrangebylex().Key(key).Min(min).Max(max).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZRevRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	cmd := c.client.B().Zrevrange().Key(key).Start(start).Stop(stop).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd {
	cmd := c.client.B().Zrevrange().Key(key).Start(start).Stop(stop).Withscores().Build()
	resp := c.client.Do(ctx, cmd)
	return newZSliceCmd(resp)
}

func (c *Compat) ZRevRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.Do(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Limit(opt.Offset, opt.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Build())
	}
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRevRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.Do(ctx, c.client.B().Zrevrangebylex().Key(key).Max(opt.Max).Min(opt.Min).Limit(opt.Offset, opt.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Zrevrangebylex().Key(key).Max(opt.Max).Min(opt.Min).Build())
	}
	return newStringSliceCmd(resp)
}

func (c *Compat) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.Do(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Withscores().Limit(opt.Offset, opt.Count).Build())
	} else {
		resp = c.client.Do(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Withscores().Build())
	}
	return newZSliceCmd(resp)
}

func (c *Compat) ZRevRank(ctx context.Context, key, member string) *IntCmd {
	cmd := c.client.B().Zrevrank().Key(key).Member(member).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ZRevRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd {
	cmd := c.client.B().Zrevrank().Key(key).Member(member).Withscore().Build()
	resp := c.client.Do(ctx, cmd)
	return newRankWithScoreCmd(resp)
}

func (c *Compat) ZScore(ctx context.Context, key, member string) *FloatCmd {
	cmd := c.client.B().Zscore().Key(key).Member(member).Build()
	resp := c.client.Do(ctx, cmd)
	return newFloatCmd(resp)
}

func (c *Compat) ZUnionStore(ctx context.Context, dest string, store ZStore) *IntCmd {
	return newIntCmd(c.client.Do(ctx, zstore(c.client.B().Arbitrary("ZUNIONSTORE").Keys(dest), store).Build()))
}

func (c *Compat) ZUnion(ctx context.Context, store ZStore) *StringSliceCmd {
	return newStringSliceCmd(c.client.Do(ctx, zstore(c.client.B().Arbitrary("ZUNION"), store).ReadOnly()))
}

func (c *Compat) ZUnionWithScores(ctx context.Context, store ZStore) *ZSliceCmd {
	return newZSliceCmd(c.client.Do(ctx, zstore(c.client.B().Arbitrary("ZUNION"), store).Args("WITHSCORES").ReadOnly()))
}

func (c *Compat) ZRandMember(ctx context.Context, key string, count int64) *StringSliceCmd {
	return newStringSliceCmd(c.client.Do(ctx, c.client.B().Zrandmember().Key(key).Count(count).Build()))
}

func (c *Compat) ZRandMemberWithScores(ctx context.Context, key string, count int64) *ZSliceCmd {
	return newZSliceCmd(c.client.Do(ctx, c.client.B().Zrandmember().Key(key).Count(count).Withscores().Build()))
}

func (c *Compat) ZDiff(ctx context.Context, keys ...string) *StringSliceCmd {
	cmd := c.client.B().Zdiff().Numkeys(int64(len(keys))).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) ZDiffWithScores(ctx context.Context, keys ...string) *ZSliceCmd {
	cmd := c.client.B().Zdiff().Numkeys(int64(len(keys))).Key(keys...).Withscores().Build()
	resp := c.client.Do(ctx, cmd)
	return newZSliceCmd(resp)
}

func (c *Compat) ZDiffStore(ctx context.Context, destination string, keys ...string) *IntCmd {
	cmd := c.client.B().Zdiffstore().Destination(destination).Numkeys(int64(len(keys))).Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) PFAdd(ctx context.Context, key string, els ...any) *IntCmd {
	cmd := c.client.B().Pfadd().Key(key).Element(argsToSlice(els)...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) PFCount(ctx context.Context, keys ...string) *IntCmd {
	cmd := c.client.B().Pfcount().Key(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) PFMerge(ctx context.Context, dest string, keys ...string) *StatusCmd {
	cmd := c.client.B().Pfmerge().Destkey(dest).Sourcekey(keys...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) BgRewriteAOF(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Bgrewriteaof().Build()
	})
}

func (c *Compat) BgSave(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Bgsave().Build()
	})
}

func (c *Compat) ClientKill(ctx context.Context, ipPort string) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ClientKill().IpPort(ipPort).Build()
	})
}

func (c *Compat) ClientKillByFilter(ctx context.Context, keys ...string) *IntCmd {
	return c.doIntCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Arbitrary("CLIENT", "KILL").Args(keys...).Build()
	})
}

func (c *Compat) ClientList(ctx context.Context) *StringCmd {
	cmd := c.client.B().ClientList().Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) ClientPause(ctx context.Context, dur time.Duration) *BoolCmd {
	var mu sync.Mutex
	ret := &BoolCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		res, err := c.Do(ctx, c.B().ClientPause().Timeout(formatSec(dur)).Build()).ToString()
		if err == nil {
			mu.Lock()
			ret.val = res == "OK"
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) ClientUnpause(ctx context.Context) *BoolCmd {
	var mu sync.Mutex
	ret := &BoolCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		res, err := c.Do(ctx, c.B().ClientUnpause().Build()).ToString()
		if err == nil {
			mu.Lock()
			ret.val = res == "OK"
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) ClientID(ctx context.Context) *IntCmd {
	return newIntCmd(c.client.Do(ctx, c.client.B().ClientId().Build()))
}

func (c *Compat) ClientUnblock(ctx context.Context, id int64) *IntCmd {
	return newIntCmd(c.client.Do(ctx, c.client.B().ClientUnblock().ClientId(id).Build()))
}

func (c *Compat) ClientUnblockWithError(ctx context.Context, id int64) *IntCmd {
	return newIntCmd(c.client.Do(ctx, c.client.B().ClientUnblock().ClientId(id).Error().Build()))
}

func (c *Compat) ConfigGet(ctx context.Context, parameter string) *StringStringMapCmd {
	cmd := c.client.B().ConfigGet().Parameter(parameter).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringStringMapCmd(resp)
}

func (c *Compat) ConfigResetStat(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ConfigResetstat().Build()
	})
}

func (c *Compat) ConfigSet(ctx context.Context, parameter, value string) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ConfigSet().ParameterValue().ParameterValue(parameter, value).Build()
	})
}

func (c *Compat) ConfigRewrite(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ConfigRewrite().Build()
	})
}

func (c *Compat) DBSize(ctx context.Context) *IntCmd {
	return c.doIntCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Dbsize().Build()
	})
}

func (c *Compat) FlushAll(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Flushall().Build()
	})
}

func (c *Compat) FlushAllAsync(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Flushall().Async().Build()
	})
}

func (c *Compat) FlushDB(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Flushdb().Build()
	})
}

func (c *Compat) FlushDBAsync(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Flushdb().Async().Build()
	})
}

func (c *Compat) Info(ctx context.Context, section ...string) *StringCmd {
	cmd := c.client.B().Info().Section(section...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) LastSave(ctx context.Context) *IntCmd {
	cmd := c.client.B().Lastsave().Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) Save(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Save().Build()
	})
}

func (c *Compat) Shutdown(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Shutdown().Build()
	})
}

func (c *Compat) ShutdownSave(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Shutdown().Save().Build()
	})
}

func (c *Compat) ShutdownNoSave(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Shutdown().Nosave().Build()
	})
}

func (c *Compat) Time(ctx context.Context) *TimeCmd {
	cmd := c.client.B().Time().Build()
	resp := c.client.Do(ctx, cmd)
	return newTimeCmd(resp)
}

func (c *Compat) DebugObject(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().DebugObject().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) ReadOnly(ctx context.Context) *StatusCmd {
	cmd := c.client.B().Readonly().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ReadWrite(ctx context.Context) *StatusCmd {
	cmd := c.client.B().Readwrite().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) MemoryUsage(ctx context.Context, key string, samples ...int64) *IntCmd {
	var resp rueidis.RedisResult
	switch len(samples) {
	case 0:
		resp = c.client.Do(ctx, c.client.B().MemoryUsage().Key(key).Build())
	case 1:
		resp = c.client.Do(ctx, c.client.B().MemoryUsage().Key(key).Samples(samples[0]).Build())
	default:
		panic("too many arguments")
	}
	return newIntCmd(resp)
}

func (c *Compat) Eval(ctx context.Context, script string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().Eval().Script(script).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Build()
	return newCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().Evalsha().Sha1(sha1).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Build()
	return newCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) EvalRO(ctx context.Context, script string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().EvalRo().Script(script).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Build()
	return newCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().EvalshaRo().Sha1(sha1).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Build()
	return newCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd {
	var mu sync.Mutex
	ret := &BoolSliceCmd{}
	ret.val = make([]bool, len(hashes))
	for i := range hashes {
		ret.val[i] = true
	}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		res, err := c.Do(ctx, c.B().ScriptExists().Sha1(hashes...).Build()).ToArray()
		if err == nil {
			mu.Lock()
			for i, v := range res {
				if b, _ := v.ToInt64(); b == 0 {
					ret.val[i] = false
				}
			}
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) ScriptFlush(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ScriptFlush().Build()
	})
}

func (c *Compat) ScriptKill(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ScriptKill().Build()
	})
}

func (c *Compat) ScriptLoad(ctx context.Context, script string) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ScriptLoad().Script(script).Build()
	})
}

func (c *Compat) FunctionLoad(ctx context.Context, code string) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionLoad().FunctionCode(code).Build()
	})
}

func (c *Compat) FunctionLoadReplace(ctx context.Context, code string) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionLoad().Replace().FunctionCode(code).Build()
	})
}

func (c *Compat) FunctionDelete(ctx context.Context, libName string) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionDelete().LibraryName(libName).Build()
	})
}

func (c *Compat) FunctionFlush(ctx context.Context) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionFlush().Build()
	})
}

func (c *Compat) FunctionKill(ctx context.Context) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionKill().Build()
	})
}

func (c *Compat) FunctionFlushAsync(ctx context.Context) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionFlush().Async().Build()
	})
}

func (c *Compat) FunctionList(ctx context.Context, q FunctionListQuery) *FunctionListCmd {
	cmd := c.client.B().Arbitrary("FUNCTION", "LIST")
	if q.LibraryNamePattern != "" {
		cmd = cmd.Args("LIBRARYNAME", q.LibraryNamePattern)
	}
	if q.WithCode {
		cmd = cmd.Args("WITHCODE")
	}
	return newFunctionListCmd(c.client.Do(ctx, cmd.Build()))
}

func (c *Compat) FunctionDump(ctx context.Context) *StringCmd {
	return newStringCmd(c.client.Do(ctx, c.client.B().FunctionDump().Build()))
}

func (c *Compat) FunctionRestore(ctx context.Context, libDump string) *StringCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().FunctionRestore().SerializedValue(libDump).Build()
	})
}

func (c *Compat) FCall(ctx context.Context, function string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().Fcall().Function(function).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Build()
	return newCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) FCallRO(ctx context.Context, function string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().FcallRo().Function(function).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Build()
	return newCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) Publish(ctx context.Context, channel string, message any) *IntCmd {
	cmd := c.client.B().Publish().Channel(channel).Message(str(message)).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) SPublish(ctx context.Context, channel string, message any) *IntCmd {
	cmd := c.client.B().Spublish().Channel(channel).Message(str(message)).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) PubSubChannels(ctx context.Context, pattern string) *StringSliceCmd {
	cmd := c.client.B().PubsubChannels().Pattern(pattern).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) PubSubNumSub(ctx context.Context, channels ...string) *StringIntMapCmd {
	cmd := c.client.B().PubsubNumsub().Channel(channels...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringIntMapCmd(resp)
}

func (c *Compat) PubSubNumPat(ctx context.Context) *IntCmd {
	cmd := c.client.B().PubsubNumpat().Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) PubSubShardChannels(ctx context.Context, pattern string) *StringSliceCmd {
	cmd := c.client.B().PubsubShardchannels().Pattern(pattern).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) PubSubShardNumSub(ctx context.Context, channels ...string) *StringIntMapCmd {
	cmd := c.client.B().PubsubShardnumsub().Channel(channels...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringIntMapCmd(resp)
}

func (c *Compat) ClusterSlots(ctx context.Context) *ClusterSlotsCmd {
	cmd := c.client.B().ClusterSlots().Build()
	resp := c.client.Do(ctx, cmd)
	return newClusterSlotsCmd(resp)
}

func (c *Compat) ClusterShards(ctx context.Context) *ClusterShardsCmd {
	cmd := c.client.B().ClusterShards().Build()
	resp := c.client.Do(ctx, cmd)
	return newClusterShardsCmd(resp)
}

func (c *Compat) ClusterNodes(ctx context.Context) *StringCmd {
	cmd := c.client.B().ClusterNodes().Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) ClusterMeet(ctx context.Context, host string, port int64) *StatusCmd {
	cmd := c.client.B().ClusterMeet().Ip(host).Port(port).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterForget(ctx context.Context, nodeID string) *StatusCmd {
	cmd := c.client.B().ClusterForget().NodeId(nodeID).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterReplicate(ctx context.Context, nodeID string) *StatusCmd {
	cmd := c.client.B().ClusterReplicate().NodeId(nodeID).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterResetSoft(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ClusterReset().Soft().Build()
	})
}

func (c *Compat) ClusterResetHard(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ClusterReset().Hard().Build()
	})
}

func (c *Compat) ClusterInfo(ctx context.Context) *StringCmd {
	cmd := c.client.B().ClusterInfo().Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) ClusterKeySlot(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().ClusterKeyslot().Key(key).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ClusterGetKeysInSlot(ctx context.Context, slot int64, count int64) *StringSliceCmd {
	var mu sync.Mutex
	ret := &StringSliceCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		cmd := c.B().ClusterGetkeysinslot().Slot(slot).Count(count).Build()
		resp, err := c.Do(ctx, cmd).AsStrSlice()
		if err == nil {
			mu.Lock()
			ret.val = append(ret.val, resp...)
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) ClusterCountFailureReports(ctx context.Context, nodeID string) *IntCmd {
	cmd := c.client.B().ClusterCountFailureReports().NodeId(nodeID).Build()
	resp := c.client.Do(ctx, cmd)
	return newIntCmd(resp)
}

func (c *Compat) ClusterCountKeysInSlot(ctx context.Context, slot int64) *IntCmd {
	var mu sync.Mutex
	ret := &IntCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		cmd := c.B().ClusterCountkeysinslot().Slot(slot).Build()
		resp, err := c.Do(ctx, cmd).AsInt64()
		if err == nil {
			mu.Lock()
			ret.val += resp
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) ClusterDelSlots(ctx context.Context, slots ...int64) *StatusCmd {
	cmd := c.client.B().ClusterDelslots().Slot(slots...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterDelSlotsRange(ctx context.Context, min, max int64) *StatusCmd {
	cmd := c.client.B().ClusterDelslotsrange().StartSlotEndSlot().StartSlotEndSlot(min, max).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterSaveConfig(ctx context.Context) *StatusCmd {
	return c.doStringCmdPrimaries(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().ClusterSaveconfig().Build()
	})
}

func (c *Compat) ClusterSlaves(ctx context.Context, nodeID string) *StringSliceCmd {
	cmd := c.client.B().ClusterSlaves().NodeId(nodeID).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) ClusterFailover(ctx context.Context) *StatusCmd {
	cmd := c.client.B().ClusterFailover().Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterAddSlots(ctx context.Context, slots ...int64) *StatusCmd {
	cmd := c.client.B().ClusterAddslots().Slot(slots...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) ClusterAddSlotsRange(ctx context.Context, min, max int64) *StatusCmd {
	cmd := c.client.B().ClusterAddslotsrange().StartSlotEndSlot().StartSlotEndSlot(min, max).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) GeoAdd(ctx context.Context, key string, geoLocation ...GeoLocation) *IntCmd {
	cmd := c.client.B().Geoadd().Key(key).LongitudeLatitudeMember()
	for _, loc := range geoLocation {
		cmd = cmd.LongitudeLatitudeMember(loc.Longitude, loc.Latitude, loc.Name)
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

func (c *Compat) GeoPos(ctx context.Context, key string, members ...string) *GeoPosCmd {
	cmd := c.client.B().Geopos().Key(key).Member(members...).Build()
	resp := c.client.Do(ctx, cmd)
	return newGeoPosCmd(resp)
}

// GeoRadius is a read-only GEORADIUS_RO command.
func (c *Compat) GeoRadius(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *GeoLocationCmd {
	cmd := c.client.B().Arbitrary("GEORADIUS_RO").Keys(key).Args(strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	if query.Store != "" || query.StoreDist != "" {
		panic("GeoRadius does not support Store or StoreDist")
	}
	cmd = cmd.Args(query.args()...)
	resp := c.client.Do(ctx, cmd.Build())
	return newGeoLocationCmd(resp)
}

// GeoRadiusStore is a writing GEORADIUS command.
func (c *Compat) GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *IntCmd {
	cmd := c.client.B().Arbitrary("GEORADIUS").Keys(key).Args(strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	if query.Store == "" && query.StoreDist == "" {
		panic("GeoRadiusStore requires Store or StoreDist")
	}
	cmd = cmd.Args(query.args()...)
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

// GeoRadiusByMember is a read-only GEORADIUSBYMEMBER_RO command.
func (c *Compat) GeoRadiusByMember(ctx context.Context, key, member string, query GeoRadiusQuery) *GeoLocationCmd {
	cmd := c.client.B().Arbitrary("GEORADIUSBYMEMBER_RO").Keys(key).Args(member)
	if query.Store != "" || query.StoreDist != "" {
		panic("GeoRadiusByMember does not support Store or StoreDist")
	}
	cmd = cmd.Args(query.args()...)
	resp := c.client.Do(ctx, cmd.Build())
	return newGeoLocationCmd(resp)
}

// GeoRadiusByMemberStore is a writing GEORADIUSBYMEMBER command.
func (c *Compat) GeoRadiusByMemberStore(ctx context.Context, key, member string, query GeoRadiusQuery) *IntCmd {
	cmd := c.client.B().Arbitrary("GEORADIUSBYMEMBER").Keys(key).Args(member)
	if query.Store == "" && query.StoreDist == "" {
		panic("GeoRadiusByMemberStore requires Store or StoreDist")
	}
	cmd = cmd.Args(query.args()...)
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

func (c *Compat) GeoSearch(ctx context.Context, key string, q GeoSearchQuery) *StringSliceCmd {
	cmd := c.client.B().Arbitrary("GEOSEARCH").Keys(key)
	cmd = cmd.Args(q.args()...)
	resp := c.client.Do(ctx, cmd.Build())
	return newStringSliceCmd(resp)
}

func (c *Compat) GeoSearchLocation(ctx context.Context, key string, q GeoSearchLocationQuery) *GeoLocationCmd {
	cmd := c.client.B().Arbitrary("GEOSEARCH").Keys(key)
	cmd = cmd.Args(q.args()...)
	resp := c.client.Do(ctx, cmd.Build())
	return newGeoLocationCmd(resp)
}

func (c *Compat) GeoSearchStore(ctx context.Context, src, dest string, q GeoSearchStoreQuery) *IntCmd {
	cmd := c.client.B().Arbitrary("GEOSEARCHSTORE").Keys(dest, src)
	cmd = cmd.Args(q.args()...)
	if q.StoreDist {
		cmd = cmd.Args("STOREDIST")
	}
	resp := c.client.Do(ctx, cmd.Build())
	return newIntCmd(resp)
}

func (c *Compat) GeoDist(ctx context.Context, key, member1, member2, unit string) *FloatCmd {
	var resp rueidis.RedisResult
	switch strings.ToUpper(unit) {
	case "M":
		resp = c.client.Do(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).M().Build())
	case "MI":
		resp = c.client.Do(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).Mi().Build())
	case "FT":
		resp = c.client.Do(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).Ft().Build())
	case "KM", "":
		resp = c.client.Do(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).Km().Build())
	default:
		panic(fmt.Sprintf("invalid unit %s", unit))
	}
	return newFloatCmd(resp)
}

func (c *Compat) GeoHash(ctx context.Context, key string, members ...string) *StringSliceCmd {
	cmd := c.client.B().Geohash().Key(key).Member(members...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringSliceCmd(resp)
}

func (c *Compat) ACLDryRun(ctx context.Context, username string, command ...any) *StringCmd {
	cmd := c.client.B().AclDryrun().Username(username).Command(command[0].(string)).Arg(argsToSlice(command[1:])...).Build()
	resp := c.client.Do(ctx, cmd)
	return newStringCmd(resp)
}

func (c *Compat) doPrimaries(ctx context.Context, fn func(c rueidis.Client) error) error {
	if c.pOnly {
		return fn(c.client)
	}
	var firsterr atomic.Value
	util.ParallelVals(c.maxp, c.client.Nodes(), func(client rueidis.Client) {
		msgs, err := client.Do(ctx, client.B().Role().Build()).ToArray()
		if err == nil {
			if role, _ := msgs[0].ToString(); role == "master" {
				err = fn(client)
			}
		}
		if err != nil {
			firsterr.CompareAndSwap(nil, err)
		}
	})
	if v := firsterr.Load(); v != nil {
		return v.(error)
	}
	return nil
}

func (c *Compat) doStringCmdPrimaries(ctx context.Context, fn func(c rueidis.Client) rueidis.Completed) *StringCmd {
	var mu sync.Mutex
	ret := &StringCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		res, err := c.Do(ctx, fn(c)).ToString()
		if err == nil {
			mu.Lock()
			ret.val = res
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) doIntCmdPrimaries(ctx context.Context, fn func(c rueidis.Client) rueidis.Completed) *IntCmd {
	var mu sync.Mutex
	ret := &IntCmd{}
	ret.err = c.doPrimaries(ctx, func(c rueidis.Client) error {
		res, err := c.Do(ctx, fn(c)).ToInt64()
		if err == nil {
			mu.Lock()
			ret.val += res
			mu.Unlock()
		}
		return err
	})
	return ret
}

func (c *Compat) TFunctionLoad(ctx context.Context, lib string) *StatusCmd {
	cmd := c.client.B().TfunctionLoad().LibraryCode(lib).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

// FIXME: should check nil of options
func (c *Compat) TFunctionLoadArgs(ctx context.Context, lib string, options *TFunctionLoadOptions) *StatusCmd {
	b := c.client.B()
	var cmd cmds.Completed
	if options.Replace {
		cmd = b.TfunctionLoad().Replace().Config(options.Config).LibraryCode(lib).Build()
	} else {
		cmd = b.TfunctionLoad().Config(options.Config).LibraryCode(lib).Build()
	}
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) TFunctionDelete(ctx context.Context, libName string) *StatusCmd {
	cmd := c.client.B().TfunctionDelete().LibraryName(libName).Build()
	resp := c.client.Do(ctx, cmd)
	return newStatusCmd(resp)
}

func (c *Compat) TFunctionList(ctx context.Context) *MapStringInterfaceSliceCmd {
	cmd := c.client.B().TfunctionList().Build()
	resp := c.client.Do(ctx, cmd)
	return newMapStringInterfaceSliceCmd(resp)
}

func (c *Compat) TFunctionListArgs(ctx context.Context, options *TFunctionListOptions) *MapStringInterfaceSliceCmd {
	cmd := c.client.B().TfunctionList()
	if options.Library != "" {
		cmd.LibraryName(options.Library)
	}
	if options.Withcode {
		cmd.Withcode()
	}
	if options.Verbose > 0 {
		cmd.Verbose()
		for i := 0; i < options.Verbose; i++ {
			cmd.V()
		}
	}
	cmdCompleted := cmd.Build()
	resp := c.client.Do(ctx, cmdCompleted)
	return newMapStringInterfaceSliceCmd(resp)
}

func (c *Compat) TFCall(ctx context.Context, libName string, funcName string, numKeys int) *Cmd {
	cmd := c.client.
		B().
		Tfcall().
		LibraryFunction(fmt.Sprintf("%s.%s", libName, funcName)).
		Numkeys(int64(numKeys)).
		Build()
	resp := c.client.Do(ctx, cmd)
	return newCmd(resp)
}

func (c *Compat) TFCallArgs(ctx context.Context, libName string, funcName string, numKeys int, options *TFCallOptions) *Cmd {
	cmd := c.client.
		B().
		Tfcall().
		LibraryFunction(fmt.Sprintf("%s.%s", libName, funcName)).
		Numkeys(int64(numKeys)).
		Key(options.Keys...).
		Arg(options.Arguments...).
		Build()
	resp := c.client.Do(ctx, cmd)
	return newCmd(resp)
}

func (c *Compat) TFCallASYNC(ctx context.Context, libName string, funcName string, numKeys int) *Cmd {
	cmd := c.client.
		B().
		Tfcallasync().
		LibraryFunction(fmt.Sprintf("%s.%s", libName, funcName)).
		Numkeys(int64(numKeys)).
		Build()
	resp := c.client.Do(ctx, cmd)
	return newCmd(resp)
}

func (c *Compat) TFCallASYNCArgs(ctx context.Context, libName string, funcName string, numKeys int, options *TFCallOptions) *Cmd {
	cmd := c.client.
		B().
		Tfcallasync().
		LibraryFunction(fmt.Sprintf("%s.%s", libName, funcName)).
		Numkeys(int64(numKeys)).
		Key(options.Keys...).
		Arg(options.Arguments...).
		Build()
	resp := c.client.Do(ctx, cmd)
	return newCmd(resp)
}

func (c *Compat) BFAdd(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().BfAdd().Key(key).Item(str(element)).Build()
	return newBoolCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFCard(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().BfCard().Key(key).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFExists(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().BfExists().Key(key).Item(str(element)).Build()
	return newBoolCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInfo(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Build()
	return newBFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInfoArg(ctx context.Context, key, option string) *BFInfoCmd {
	switch option {
	case "CAPACITY":
		return c.BFInfoCapacity(ctx, key)
	case "SIZE":
		return c.BFInfoSize(ctx, key)
	case "FILTERS":
		return c.BFInfoFilters(ctx, key)
	case "ITEMS":
		return c.BFInfoItems(ctx, key)
	case "EXPANSION":
		return c.BFInfoExpansion(ctx, key)
	default:
		panic(fmt.Sprintf("unknown option %v", option))
	}
}

func (c *Compat) BFInfoCapacity(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Capacity().Build()
	return newBFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInfoSize(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Size().Build()
	return newBFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInfoFilters(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Filters().Build()
	return newBFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInfoItems(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Items().Build()
	return newBFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInfoExpansion(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Expansion().Build()
	return newBFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFInsert(ctx context.Context, key string, options *BFInsertOptions, elements ...interface{}) *BoolSliceCmd {
	_cmd := c.client.B().
		BfInsert().
		Key(key).
		Capacity(options.Capacity).
		Error(options.Error).
		Expansion(options.Expansion)
	if options.NonScaling {
		_cmd.Nonscaling()
	}
	if options.NoCreate {
		_cmd.Nocreate()
	}
	items := _cmd.Items()
	for _, e := range elements {
		items.Item(str(e))
	}
	cmd := (cmds.BfInsertItem)(items).Build()
	return newBoolSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFMAdd(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	cmd := c.client.B().BfMadd().Key(key)
	var last cmds.BfMaddItem
	for _, e := range elements {
		last = cmd.Item(str(e))
	}
	return newBoolSliceCmd(c.client.Do(ctx, last.Build()))
}

func (c *Compat) BFMExists(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	cmd := c.client.B().BfMexists().Key(key)
	var last cmds.BfMexistsItem
	for _, e := range elements {
		last = cmd.Item(str(e))
	}
	return newBoolSliceCmd(c.client.Do(ctx, last.Build()))
}

func (c *Compat) BFReserve(ctx context.Context, key string, errorRate float64, capacity int64) *StatusCmd {
	cmd := c.client.B().BfReserve().Key(key).ErrorRate(errorRate).Capacity(capacity).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFReserveExpansion(ctx context.Context, key string, errorRate float64, capacity, expansion int64) *StatusCmd {
	cmd := c.client.B().BfReserve().Key(key).ErrorRate(errorRate).Capacity(capacity).Expansion(expansion).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFReserveNonScaling(ctx context.Context, key string, errorRate float64, capacity int64) *StatusCmd {
	cmd := c.client.B().BfReserve().Key(key).ErrorRate(errorRate).Capacity(capacity).Nonscaling().Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFReserveWithArgs(ctx context.Context, key string, options *BFReserveOptions) *StatusCmd {
	cmd := c.client.B().BfReserve().Key(key).ErrorRate(options.Error).Capacity(options.Capacity).Expansion(options.Expansion)
	if options.NonScaling {
		cmd.Nonscaling()
	}
	return newStatusCmd(c.client.Do(ctx, cmd.Build()))
}

func (c *Compat) BFScanDump(ctx context.Context, key string, iterator int64) *ScanDumpCmd {
	cmd := c.client.B().BfScandump().Key(key).Iterator(iterator).Build()
	return newScanDumpCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) BFLoadChunk(ctx context.Context, key string, iterator int64, data interface{}) *StatusCmd {
	cmd := c.client.B().BfLoadchunk().Key(key).Iterator(iterator).Data(str(data)).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFAdd(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().CfAdd().Key(key).Item(str(element)).Build()
	return newBoolCmd(c.client.Do(ctx, cmd))
}
func (c *Compat) CFAddNX(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().CfAddnx().Key(key).Item(str(element)).Build()
	return newBoolCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFCount(ctx context.Context, key string, element interface{}) *IntCmd {
	cmd := c.client.B().CfCount().Key(key).Item(str(element)).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFDel(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().CfDel().Key(key).Item(str(element)).Build()
	return newBoolCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFExists(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().CfExists().Key(key).Item(str(element)).Build()
	return newBoolCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFInfo(ctx context.Context, key string) *CFInfoCmd {
	cmd := c.client.B().CfInfo().Key(key).Build()
	return newCFInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFInsert(ctx context.Context, key string, options *CFInsertOptions, elements ...interface{}) *BoolSliceCmd {
	_cmd := c.client.B().CfInsert().Key(key)
	if options != nil {
		_cmd.Capacity(options.Capacity)
		if options.NoCreate {
			_cmd.Nocreate()
		}
	}
	items := _cmd.Items()
	for _, e := range elements {
		items.Item(str(e))
	}
	cmd := (cmds.CfInsertItem)(items).Build()
	return newBoolSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFInsertNX(ctx context.Context, key string, options *CFInsertOptions, elements ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().CfInsertnx().Key(key).Capacity(options.Capacity)
	if options.NoCreate {
		_cmd.Nocreate()
	}
	items := _cmd.Items()
	for _, e := range elements {
		items.Item(str(e))
	}
	cmd := (cmds.CfInsertItem)(items).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFMExists(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	_cmd := c.client.B().CfMexists().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.CfMexistsItem)(_cmd).Build()
	return newBoolSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFReserve(ctx context.Context, key string, capacity int64) *StatusCmd {
	cmd := c.client.B().CfReserve().Key(key).Capacity(capacity).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFReserveWithArgs(ctx context.Context, key string, options *CFReserveOptions) *StatusCmd {
	cmd := c.client.B().
		CfReserve().
		Key(key).
		Capacity(options.Capacity).
		Bucketsize(options.BucketSize).
		Maxiterations(options.MaxIterations).
		Expansion(options.Expansion).
		Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFReserveExpansion(ctx context.Context, key string, capacity int64, expansion int64) *StatusCmd {
	cmd := c.client.B().
		CfReserve().
		Key(key).
		Capacity(capacity).
		Expansion(expansion).
		Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFReserveBucketSize(ctx context.Context, key string, capacity int64, bucketsize int64) *StatusCmd {
	cmd := c.client.B().
		CfReserve().
		Key(key).
		Capacity(capacity).
		Bucketsize(bucketsize).
		Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFReserveMaxIterations(ctx context.Context, key string, capacity int64, maxiterations int64) *StatusCmd {
	cmd := c.client.B().
		CfReserve().
		Key(key).
		Capacity(capacity).
		Maxiterations(maxiterations).
		Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFScanDump(ctx context.Context, key string, iterator int64) *ScanDumpCmd {
	cmd := c.client.B().CfScandump().Key(key).Iterator(iterator).Build()
	return newScanDumpCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CFLoadChunk(ctx context.Context, key string, iterator int64, data interface{}) *StatusCmd {
	cmd := c.client.B().CfLoadchunk().Key(key).Iterator(iterator).Data(str(data)).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSIncrBy(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().CmsIncrby().Key(key)
	for i := 0; i < len(elements); i += 2 {
		_cmd.Item(str(elements[i])).Increment((int64)(elements[i+1].(int)))
	}
	cmd := (cmds.CmsIncrbyItemsIncrement)(_cmd).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSInfo(ctx context.Context, key string) *CMSInfoCmd {
	cmd := c.client.B().CmsInfo().Key(key).Build()
	return newCMSInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSInitByDim(ctx context.Context, key string, width, height int64) *StatusCmd {
	cmd := c.client.B().CmsInitbydim().Key(key).Width(width).Depth(height).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSInitByProb(ctx context.Context, key string, errorRate, probability float64) *StatusCmd {
	cmd := c.client.B().CmsInitbyprob().Key(key).Error(errorRate).Probability(probability).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSMerge(ctx context.Context, destKey string, sourceKeys ...string) *StatusCmd {
	cmd := c.client.B().CmsMerge().Destination(destKey).Numkeys((int64)(len(sourceKeys))).Source(sourceKeys...).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSMergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) *StatusCmd {
	_cmd := c.client.B().CmsMerge().Destination(destKey).Numkeys((int64)(len(sourceKeys)))
	keys := make([]string, 0, len(sourceKeys))
	for k := range sourceKeys {
		keys = append(keys, k)
	}
	for _, k := range keys {
		_cmd.Source(k)
	}
	wCmd := (cmds.CmsMergeSource)(_cmd).Weights()
	for _, k := range keys {
		// weight should be integer
		// we converts int64 to float64 to avoid API breaking change
		wCmd.Weight((float64)(sourceKeys[k]))
	}
	cmd := (cmds.CmsMergeWeightWeight)(wCmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) CMSQuery(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().CmsQuery().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.CmsQueryItem)(_cmd).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKAdd(ctx context.Context, key string, elements ...interface{}) *StringSliceCmd {
	_cmd := c.client.B().TopkAdd().Key(key)
	for _, e := range elements {
		_cmd.Items(str(e))
	}
	cmd := (cmds.TopkAddItems)(_cmd).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKCount(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().TopkCount().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.TopkCountItem)(_cmd).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKIncrBy(ctx context.Context, key string, elements ...interface{}) *StringSliceCmd {
	_cmd := c.client.B().TopkIncrby().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.TopkIncrbyItemsIncrement)(_cmd).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKInfo(ctx context.Context, key string) *TopKInfoCmd {
	cmd := c.client.B().TopkInfo().Key(key).Build()
	return newTopKInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKList(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().TopkList().Key(key).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKListWithCount(ctx context.Context, key string) *MapStringIntCmd {
	cmd := c.client.B().TopkList().Key(key).Withcount().Build()
	return newMapStringIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKQuery(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	_cmd := c.client.B().TopkQuery().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.TopkQueryItem)(_cmd).Build()
	return newBoolSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKReserve(ctx context.Context, key string, k int64) *StatusCmd {
	cmd := c.client.B().TopkReserve().Key(key).Topk(k).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TopKReserveWithOptions(ctx context.Context, key string, k int64, width, depth int64, decay float64) *StatusCmd {
	cmd := c.client.B().TopkReserve().Key(key).Topk(k).Width(width).Depth(depth).Decay(decay).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestAdd(ctx context.Context, key string, elements ...float64) *StatusCmd {
	_cmd := c.client.B().TdigestAdd().Key(key)
	for _, e := range elements {
		_cmd.Value(e)
	}
	cmd := (cmds.TdigestAddValuesValue)(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestByRank(ctx context.Context, key string, rank ...uint64) *FloatSliceCmd {
	_cmd := c.client.B().TdigestByrank().Key(key)
	for _, r := range rank {
		_cmd.Rank((float64)(r))
	}
	cmd := (cmds.TdigestByrankRank)(_cmd).Build()
	return newFloatSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestByRevRank(ctx context.Context, key string, rank ...uint64) *FloatSliceCmd {
	_cmd := c.client.B().TdigestByrevrank().Key(key)
	for _, r := range rank {
		_cmd.ReverseRank((float64)(r))
	}
	cmd := (cmds.TdigestByrevrankReverseRank)(_cmd).Build()
	return newFloatSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestCDF(ctx context.Context, key string, elements ...float64) *FloatSliceCmd {
	cmd := c.client.B().TdigestCdf().Key(key).Value(elements...).Build()
	return newFloatSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestCreate(ctx context.Context, key string) *StatusCmd {
	cmd := c.client.B().TdigestCreate().Key(key).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))

}
func (c *Compat) TDigestCreateWithCompression(ctx context.Context, key string, compression int64) *StatusCmd {
	cmd := c.client.B().TdigestCreate().Key(key).Compression(compression).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestInfo(ctx context.Context, key string) *TDigestInfoCmd {
	cmd := c.client.B().TdigestInfo().Key(key).Build()
	return newTDigestInfoCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestMax(ctx context.Context, key string) *FloatCmd {
	cmd := c.client.B().TdigestMax().Key(key).Build()
	return newFloatCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestMin(ctx context.Context, key string) *FloatCmd {
	cmd := c.client.B().TdigestMin().Key(key).Build()
	return newFloatCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestMerge(ctx context.Context, destKey string, options *TDigestMergeOptions, sourceKeys ...string) *StatusCmd {
	_cmd := c.client.B().TdigestMerge().DestinationKey(destKey).Numkeys(int64(len(sourceKeys))).SourceKey(sourceKeys...).Compression(options.Compression)
	if options.Override {
		_cmd.Override()
	}
	cmd := (cmds.TdigestMergeOverride)(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestQuantile(ctx context.Context, key string, elements ...float64) *FloatSliceCmd {
	cmd := c.client.B().TdigestQuantile().Key(key).Quantile(elements...).Build()
	return newFloatSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestRank(ctx context.Context, key string, values ...float64) *IntSliceCmd {
	cmd := c.client.B().TdigestRank().Key(key).Value(values...).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestReset(ctx context.Context, key string) *StatusCmd {
	cmd := c.client.B().TdigestReset().Key(key).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestRevRank(ctx context.Context, key string, values ...float64) *IntSliceCmd {
	cmd := c.client.B().TdigestRevrank().Key(key).Value(values...).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) TDigestTrimmedMean(ctx context.Context, key string, lowCutQuantile, highCutQuantile float64) *FloatCmd {
	cmd := c.client.B().TdigestTrimmedMean().Key(key).LowCutQuantile(lowCutQuantile).HighCutQuantile(highCutQuantile).Build()
	return newFloatCmd(c.client.Do(ctx, cmd))
}

// TSAdd - Adds one or more observations to a t-digest sketch.
// For more information - https://redis.io/commands/ts.add/
func (c *Compat) TSAdd(ctx context.Context, key string, timestamp interface{}, value float64) *IntCmd {
	cmd := c.client.B().TsAdd().Key(key).Timestamp(str(timestamp)).Value(value).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSAddWithArgs - Adds one or more observations to a t-digest sketch.
// This function also allows for specifying additional options such as:
// Retention, ChunkSize, Encoding, DuplicatePolicy and Labels.
// For more information - https://redis.io/commands/ts.add/
func (c *Compat) TSAddWithArgs(ctx context.Context, key string, timestamp interface{}, value float64, options *TSOptions) *IntCmd {
	_cmd := c.client.B().
		TsAdd().
		Key(key).
		Timestamp(str(timestamp)).
		Value(value)
	if options.ChunkSize != 0 {
		_cmd.ChunkSize(int64(options.ChunkSize))
	}
	if options.Retention != 0 {
		_cmd.Retention(int64(options.Retention))
	}
	switch options.Encoding {
	case "COMPRESSED", "":
		_cmd.EncodingCompressed()
	case "UNCOMPRESSED":
		_cmd.EncodingUncompressed()
	}
	if options.DuplicatePolicy != "" {
		switch options.DuplicatePolicy {
		case "BLOCK", "block":
			_cmd.OnDuplicateBlock()
		case "FIRST", "first":
			_cmd.OnDuplicateFirst()
		case "LAST", "last":
			_cmd.OnDuplicateLast()
		case "MIN", "min":
			_cmd.OnDuplicateMin()
		case "MAX", "max":
			_cmd.OnDuplicateMax()
		case "SUM", "sum":
			_cmd.OnDuplicateSum()
		default:
			panic(fmt.Sprintf("invalid duplicate policy, want one of [BLOCK|FIRST|LAST|MIN|MAX|SUM], got %v", options.DuplicatePolicy))
		}
	}
	if options.Labels != nil {
		labels := (cmds.TsAddOnDuplicateBlock)(_cmd).Labels()
		for k, v := range options.Labels {
			labels.Labels(k, v)
		}
		_cmd = cmds.TsAddValue(labels)
	}
	cmd := _cmd.Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSCreate - Creates a new time-series key.
// For more information - https://redis.io/commands/ts.create/
func (c *Compat) TSCreate(ctx context.Context, key string) *StatusCmd {
	cmd := c.client.B().TsCreate().Key(key).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// TSCreateWithArgs - Creates a new time-series key with additional options.
// This function allows for specifying additional options such as:
// Retention, ChunkSize, Encoding, DuplicatePolicy and Labels.
// For more information - https://redis.io/commands/ts.create/
func (c *Compat) TSCreateWithArgs(ctx context.Context, key string, options *TSOptions) *StatusCmd {
	_cmd := c.client.B().TsCreate().Key(key)
	if options.Retention != 0 {
		_cmd.Retention(int64(options.Retention))
	}
	if options.ChunkSize != 0 {
		_cmd.ChunkSize(int64(options.ChunkSize))
	}
	if options.Encoding != "" {
		_cmd.EncodingCompressed()
	} else {
		_cmd.EncodingUncompressed()
	}
	if options.DuplicatePolicy != "" {
		switch options.DuplicatePolicy {
		case "BLOCK", "block":
			_cmd.DuplicatePolicyBlock()
		case "FIRST", "first":
			_cmd.DuplicatePolicyFirst()
		case "LAST", "last":
			_cmd.DuplicatePolicyLast()
		case "MIN", "min":
			_cmd.DuplicatePolicyMin()
		case "MAX", "max":
			_cmd.DuplicatePolicyMax()
		case "SUM", "sum":
			_cmd.DuplicatePolicySum()
		default:
			panic(fmt.Sprintf("invalid duplicate policy, want one of [BLOCK|FIRST|LAST|MIN|MAX|SUM], got %v", options.DuplicatePolicy))
		}
	}
	// var cmd cmds.Completed
	if options.Labels != nil {
		labels := _cmd.Labels()
		for k, v := range options.Labels {
			labels.Labels(k, v)
		}
		_cmd = cmds.TsCreateKey(labels)
	}
	cmd := (cmds.TsCreateKey)(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// TSAlter - Alters an existing time-series key with additional options.
// This function allows for specifying additional options such as:
// Retention, ChunkSize and DuplicatePolicy.
// For more information - https://redis.io/commands/ts.alter/
func (c *Compat) TSAlter(ctx context.Context, key string, options *TSAlterOptions) *StatusCmd {
	_cmd := c.client.B().TsAlter().Key(key)
	if options != nil {
		if options.Retention != 0 {
			_cmd.Retention(int64(options.Retention))
		}
		if options.ChunkSize != 0 {
			_cmd.ChunkSize(int64(options.ChunkSize))
		}
		if options.DuplicatePolicy != "" {
			switch options.DuplicatePolicy {
			case "BLOCK", "block":
				_cmd.DuplicatePolicyBlock()
			case "FIRST", "first":
				_cmd.DuplicatePolicyFirst()
			case "LAST", "last":
				_cmd.DuplicatePolicyLast()
			case "MIN", "min":
				_cmd.DuplicatePolicyMin()
			case "MAX", "max":
				_cmd.DuplicatePolicyMax()
			case "SUM", "sum":
				_cmd.DuplicatePolicySum()
			default:
				panic(fmt.Sprintf("invalid duplicate policy, want one of [BLOCK|FIRST|LAST|MIN|MAX|SUM], got %v", options.DuplicatePolicy))
			}
		}
		if options.Labels != nil {
			labels := _cmd.Labels()
			for label, value := range options.Labels {
				labels.Labels(label, value)
			}
			_cmd = cmds.TsAlterKey(labels)
		}
	}
	cmd := _cmd.Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// TSCreateRule - Creates a compaction rule from sourceKey to destKey.
// For more information - https://redis.io/commands/ts.createrule/
func (c *Compat) TSCreateRule(ctx context.Context, sourceKey string, destKey string, aggregator Aggregator, bucketDuration int) *StatusCmd {
	_cmd := c.client.B().TsCreaterule().Sourcekey(sourceKey).Destkey(destKey)
	var cmd cmds.Completed
	switch aggregator {
	case Avg:
		cmd = _cmd.AggregationAvg().Bucketduration(int64(bucketDuration)).Build()
	case Sum:
		cmd = _cmd.AggregationSum().Bucketduration(int64(bucketDuration)).Build()
	case Min:
		cmd = _cmd.AggregationMin().Bucketduration(int64(bucketDuration)).Build()
	case Max:
		cmd = _cmd.AggregationMax().Bucketduration(int64(bucketDuration)).Build()
	case Range:
		cmd = _cmd.AggregationRange().Bucketduration(int64(bucketDuration)).Build()
	case Count:
		cmd = _cmd.AggregationCount().Bucketduration(int64(bucketDuration)).Build()
	case First:
		cmd = _cmd.AggregationFirst().Bucketduration(int64(bucketDuration)).Build()
	case Last:
		cmd = _cmd.AggregationLast().Bucketduration(int64(bucketDuration)).Build()
	case StdP:
		cmd = _cmd.AggregationStdP().Bucketduration(int64(bucketDuration)).Build()
	case StdS:
		cmd = _cmd.AggregationStdS().Bucketduration(int64(bucketDuration)).Build()
	case VarP:
		cmd = _cmd.AggregationVarP().Bucketduration(int64(bucketDuration)).Build()
	case VarS:
		cmd = _cmd.AggregationVarS().Bucketduration(int64(bucketDuration)).Build()
	case Twa:
		cmd = _cmd.AggregationTwa().Bucketduration(int64(bucketDuration)).Build()
	}
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// TSCreateRuleWithArgs - Creates a compaction rule from sourceKey to destKey with additional option.
// This function allows for specifying additional option such as:
// AlignTimestamp.
// For more information - https://redis.io/commands/ts.createrule/
func (c *Compat) TSCreateRuleWithArgs(ctx context.Context, sourceKey string, destKey string, aggregator Aggregator, bucketDuration int, options *TSCreateRuleOptions) *StatusCmd {
	_cmd := c.client.B().TsCreaterule().Sourcekey(sourceKey).Destkey(destKey)
	var duration cmds.TsCreateruleBucketduration
	switch aggregator {
	case Avg:
		duration = _cmd.AggregationAvg().Bucketduration(int64(bucketDuration))
	case Sum:
		duration = _cmd.AggregationSum().Bucketduration(int64(bucketDuration))
	case Min:
		duration = _cmd.AggregationMin().Bucketduration(int64(bucketDuration))
	case Max:
		duration = _cmd.AggregationMax().Bucketduration(int64(bucketDuration))
	case Range:
		duration = _cmd.AggregationRange().Bucketduration(int64(bucketDuration))
	case Count:
		duration = _cmd.AggregationCount().Bucketduration(int64(bucketDuration))
	case First:
		duration = _cmd.AggregationFirst().Bucketduration(int64(bucketDuration))
	case Last:
		duration = _cmd.AggregationLast().Bucketduration(int64(bucketDuration))
	case StdP:
		duration = _cmd.AggregationStdP().Bucketduration(int64(bucketDuration))
	case StdS:
		duration = _cmd.AggregationStdS().Bucketduration(int64(bucketDuration))
	case VarP:
		duration = _cmd.AggregationVarP().Bucketduration(int64(bucketDuration))
	case VarS:
		duration = _cmd.AggregationVarS().Bucketduration(int64(bucketDuration))
	case Twa:
		duration = _cmd.AggregationTwa().Bucketduration(int64(bucketDuration))
	}
	if options != nil && options.AlignTimestamp != 0 {
		duration.Aligntimestamp(options.AlignTimestamp)
	}
	cmd := duration.Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// TSIncrBy - Increments the value of a time-series key by the specified timestamp.
// For more information - https://redis.io/commands/ts.incrby/
// FIXME: timestamp should be addend
func (c *Compat) TSIncrBy(ctx context.Context, Key string, timestamp float64) *IntCmd {
	cmd := c.client.B().TsIncrby().Key(Key).Value(timestamp).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSIncrByWithArgs - Increments the value of a time-series key by the specified timestamp with additional options.
// This function allows for specifying additional options such as:
// Timestamp, Retention, ChunkSize, Uncompressed and Labels.
// For more information - https://redis.io/commands/ts.incrby/
// FIXME: timestamp should be addend
func (c *Compat) TSIncrByWithArgs(ctx context.Context, key string, timestamp float64, options *TSIncrDecrOptions) *IntCmd {
	_cmd := c.client.B().TsIncrby().Key(key).Value(timestamp)
	if options != nil {
		if options.Timestamp != 0 {
			_cmd.Timestamp(str(options.Timestamp))
		}
		if options.Retention != 0 {
			_cmd.Retention(int64(options.Retention))
		}
		if options.ChunkSize != 0 {
			_cmd.ChunkSize(int64(options.ChunkSize))
		}
		if options.Uncompressed {
			_cmd.Uncompressed()
		}
		if options.Labels != nil {
			_cmd.Labels()
			for label, value := range options.Labels {
				(cmds.TsIncrbyLabels)(_cmd).Labels(label, value)
			}
		}
	}
	cmd := (cmds.TsIncrbyLabels)(_cmd).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSDecrBy - Decrements the value of a time-series key by the specified timestamp.
// For more information - https://redis.io/commands/ts.decrby/
// FIXME: timestamp should be subtrahend
func (c *Compat) TSDecrBy(ctx context.Context, Key string, timestamp float64) *IntCmd {
	cmd := c.client.B().TsDecrby().Key(Key).Value(timestamp).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSDecrByWithArgs - Decrements the value of a time-series key by the specified timestamp with additional options.
// This function allows for specifying additional options such as:
// Timestamp, Retention, ChunkSize, Uncompressed and Labels.
// For more information - https://redis.io/commands/ts.decrby/
func (c *Compat) TSDecrByWithArgs(ctx context.Context, key string, timestamp float64, options *TSIncrDecrOptions) *IntCmd {
	_cmd := c.client.B().TsDecrby().Key(key).Value(timestamp)
	if options != nil {
		if options.Timestamp != 0 {
			_cmd.Timestamp(str(options.Timestamp))
		}
		if options.Retention != 0 {
			_cmd.Retention(int64(options.Retention))
		}
		if options.ChunkSize != 0 {
			_cmd.ChunkSize(int64(options.ChunkSize))
		}
		if options.Uncompressed {
			_cmd.Uncompressed()
		}
		if options.Labels != nil {
			_cmd.Labels()
			for label, value := range options.Labels {
				(cmds.TsDecrbyLabels)(_cmd).Labels(label, value)
			}
		}
	}
	cmd := (cmds.TsDecrbyLabels)(_cmd).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSDel - Deletes a range of samples from a time-series key.
// For more information - https://redis.io/commands/ts.del/
func (c *Compat) TSDel(ctx context.Context, Key string, fromTimestamp int, toTimestamp int) *IntCmd {
	cmd := c.client.B().TsDel().Key(Key).FromTimestamp(int64(fromTimestamp)).ToTimestamp(int64(toTimestamp)).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// TSDeleteRule - Deletes a compaction rule from sourceKey to destKey.
// For more information - https://redis.io/commands/ts.deleterule/
func (c *Compat) TSDeleteRule(ctx context.Context, sourceKey string, destKey string) *StatusCmd {
	cmd := c.client.B().TsDeleterule().Sourcekey(sourceKey).Destkey(destKey).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// TSGetWithArgs - Gets the last sample of a time-series key with additional option.
// This function allows for specifying additional option such as:
// Latest.
// For more information - https://redis.io/commands/ts.get/
func (c *Compat) TSGetWithArgs(ctx context.Context, key string, options *TSGetOptions) *TSTimestampValueCmd {
	_cmd := c.client.B().TsGet().Key(key)
	if options != nil && options.Latest {
		_cmd.Latest()
	}
	return newTSTimestampValueCmd(c.client.Do(ctx, _cmd.Build()))
}

// TSGet - Gets the last sample of a time-series key.
// For more information - https://redis.io/commands/ts.get/
func (c *Compat) TSGet(ctx context.Context, key string) *TSTimestampValueCmd {
	cmd := c.client.B().TsGet().Key(key).Build()
	return newTSTimestampValueCmd(c.client.Do(ctx, cmd))
}

// TSInfo - Returns information about a time-series key.
// For more information - https://redis.io/commands/ts.info/
func (c *Compat) TSInfo(ctx context.Context, key string) *MapStringInterfaceCmd {
	cmd := c.client.B().TsInfo().Key(key).Build()
	return newMapStringInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSInfoWithArgs - Returns information about a time-series key with additional option.
// This function allows for specifying additional option such as:
// Debug.
// For more information - https://redis.io/commands/ts.info/
func (c *Compat) TSInfoWithArgs(ctx context.Context, key string, options *TSInfoOptions) *MapStringInterfaceCmd {
	_cmd := c.client.B().TsInfo().Key(key)
	if options != nil && options.Debug {
		// FIXME: should not accept arg, just append "DEBUG"
		_cmd.Debug("DEBUG")
	}
	cmd := (cmds.TsInfoKey)(_cmd).Build()
	return newMapStringInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSMAdd - Adds multiple samples to multiple time-series keys.
// For more information - https://redis.io/commands/ts.madd/
func (c *Compat) TSMAdd(ctx context.Context, ktvSlices [][]interface{}) *IntSliceCmd {
	_cmd := c.client.B().TsMadd().KeyTimestampValue()
	for _, ktv := range ktvSlices {
		tstmp, err := toInt64(int64(ktv[1].(int)))
		if err != nil {
			panic(err)
		}
		val, err := toFloat64(int64(ktv[2].(int)))
		if err != nil {
			panic(err)
		}
		_cmd.KeyTimestampValue(str(ktv[0]), tstmp, val)
	}
	cmd := (cmds.TsMaddKeyTimestampValue)(_cmd).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

// TSQueryIndex - Returns all the keys matching the filter expression.
// For more information - https://redis.io/commands/ts.queryindex/
func (c *Compat) TSQueryIndex(ctx context.Context, filterExpr []string) *StringSliceCmd {
	cmd := c.client.B().TsQueryindex().Filter(filterExpr...).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

// TSRevRange - Returns a range of samples from a time-series key in reverse order.
// For more information - https://redis.io/commands/ts.revrange/
func (c *Compat) TSRevRange(ctx context.Context, key string, fromTimestamp int, toTimestamp int) *TSTimestampValueSliceCmd {
	cmd := c.client.B().TsRevrange().Key(key).Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp)).Build()
	return newTSTimestampValueSliceCmd(c.client.Do(ctx, cmd))
}

// TSRevRangeWithArgs - Returns a range of samples from a time-series key in reverse order with additional options.
// This function allows for specifying additional options such as:
// Latest, FilterByTS, FilterByValue, Count, Align, Aggregator,
// BucketDuration, BucketTimestamp and Empty.
// For more information - https://redis.io/commands/ts.revrange/
func (c *Compat) TSRevRangeWithArgs(ctx context.Context, key string, fromTimestamp int, toTimestamp int, options *TSRevRangeOptions) *TSTimestampValueSliceCmd {
	_cmd := c.client.B().TsRevrange().Key(key).Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp))
	if options != nil {
		if options.Latest {
			_cmd.Latest()
		}
		if options.FilterByTS != nil {
			tss := make([]int64, 0, len(options.FilterByTS))
			for _, ts := range options.FilterByTS {
				tss = append(tss, int64(ts))
			}
			_cmd.FilterByTs(tss...)
		}
		if options.FilterByValue != nil {
			if len(options.FilterByValue) != 2 {
				panic(fmt.Sprintf("wrong number of arguments in options.FilterByValue, expect min, max, got %v", options.FilterByValue))
			}
			_cmd.FilterByValue(float64(options.FilterByValue[0]), float64(options.FilterByValue[1]))
		}
		if options.Count != 0 {
			_cmd.Count(int64(options.Count))
		}
		if options.Align != nil {
			_cmd.Align(str(options.Align))
		}
		if options.Aggregator != 0 {
			switch options.Aggregator {
			case Invalid:
				break
			case Avg:
				_cmd.AggregationAvg()
			case Sum:
				_cmd.AggregationSum()
			case Min:
				_cmd.AggregationMin()
			case Max:
				_cmd.AggregationMax()
			case Range:
				_cmd.AggregationRange()
			case Count:
				_cmd.AggregationCount()
			case First:
				_cmd.AggregationFirst()
			case Last:
				_cmd.AggregationLast()
			case StdP:
				_cmd.AggregationStdP()
			case StdS:
				_cmd.AggregationStdS()
			case VarP:
				_cmd.AggregationVarP()
			case VarS:
				_cmd.AggregationVarS()
			case Twa:
				_cmd.AggregationTwa()
			}
		}
		if options.BucketDuration != 0 {
			cmds.TsRevrangeAggregationAggregationAvg(_cmd).Bucketduration(int64(options.BucketDuration))
		}
		if options.BucketTimestamp != nil {
			cmds.TsRevrangeAggregationBucketduration(_cmd).Buckettimestamp(str(options.BucketTimestamp))
		}
		if options.Empty {
			cmds.TsRevrangeAggregationBuckettimestamp(_cmd).Empty()
		}
	}
	cmd := cmds.TsRevrangeTotimestamp(_cmd).Build()
	return newTSTimestampValueSliceCmd(c.client.Do(ctx, cmd))
}

// TSRange - Returns a range of samples from a time-series key.
// For more information - https://redis.io/commands/ts.range/
func (c *Compat) TSRange(ctx context.Context, key string, fromTimestamp int, toTimestamp int) *TSTimestampValueSliceCmd {
	cmd := c.client.B().TsRange().Key(key).Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp)).Build()
	return newTSTimestampValueSliceCmd(c.client.Do(ctx, cmd))
}

// TSRangeWithArgs - Returns a range of samples from a time-series key with additional options.
// This function allows for specifying additional options such as:
// Latest, FilterByTS, FilterByValue, Count, Align, Aggregator,
// BucketDuration, BucketTimestamp and Empty.
// For more information - https://redis.io/commands/ts.range/
func (c *Compat) TSRangeWithArgs(ctx context.Context, key string, fromTimestamp int, toTimestamp int, options *TSRangeOptions) *TSTimestampValueSliceCmd {
	_cmd := c.client.B().TsRange().Key(key).Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp))
	if options != nil {
		if options.Latest {
			_cmd.Latest()
		}
		if options.FilterByTS != nil {
			tss := make([]int64, 0, len(options.FilterByTS))
			for _, ts := range options.FilterByTS {
				tss = append(tss, int64(ts))
			}
			_cmd.FilterByTs(tss...)
		}
		if options.FilterByValue != nil {
			if len(options.FilterByValue) != 2 {
				panic(fmt.Sprintf("wrong number of arguments in options.FilterByValue, expect min, max, got %v", options.FilterByValue))
			}
			_cmd.FilterByValue(float64(options.FilterByValue[0]), float64(options.FilterByValue[1]))
		}
		if options.Count != 0 {
			_cmd.Count(int64(options.Count))
		}
		if options.Align != nil {
			_cmd.Align(str(options.Align))
		}
		if options.Aggregator != 0 {
			switch options.Aggregator {
			case Invalid:
				break
			case Avg:
				_cmd.AggregationAvg()
			case Sum:
				_cmd.AggregationSum()
			case Min:
				_cmd.AggregationMin()
			case Max:
				_cmd.AggregationMax()
			case Range:
				_cmd.AggregationRange()
			case Count:
				_cmd.AggregationCount()
			case First:
				_cmd.AggregationFirst()
			case Last:
				_cmd.AggregationLast()
			case StdP:
				_cmd.AggregationStdP()
			case StdS:
				_cmd.AggregationStdS()
			case VarP:
				_cmd.AggregationVarP()
			case VarS:
				_cmd.AggregationVarS()
			case Twa:
				_cmd.AggregationTwa()
			}
		}
		if options.BucketDuration != 0 {
			cmds.TsRangeAggregationAggregationAvg(_cmd).Bucketduration(int64(options.BucketDuration))
		}
		if options.BucketTimestamp != nil {
			cmds.TsRangeAggregationBucketduration(_cmd).Buckettimestamp(str(options.BucketTimestamp))
		}
		if options.Empty {
			cmds.TsRangeAggregationBuckettimestamp(_cmd).Empty()
		}
	}
	cmd := cmds.TsRangeTotimestamp(_cmd).Build()
	return newTSTimestampValueSliceCmd(c.client.Do(ctx, cmd))
}

// TSMRange - Returns a range of samples from multiple time-series keys.
// For more information - https://redis.io/commands/ts.mrange/
func (c *Compat) TSMRange(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string) *MapStringSliceInterfaceCmd {
	cmd := c.client.B().TsMrange().Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp)).Filter(filterExpr...).Build()
	return newMapStringSliceInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSMRangeWithArgs - Returns a range of samples from multiple time-series keys with additional options.
// This function allows for specifying additional options such as:
// Latest, FilterByTS, FilterByValue, WithLabels, SelectedLabels,
// Count, Align, Aggregator, BucketDuration, BucketTimestamp,
// Empty, GroupByLabel and Reducer.
// For more information - https://redis.io/commands/ts.mrange/
func (c *Compat) TSMRangeWithArgs(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string, options *TSMRangeOptions) *MapStringSliceInterfaceCmd {
	_cmd := c.client.B().TsMrange().Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp))
	if options != nil {
		if options.Latest {
			_cmd.Latest()
		}
		if options.FilterByTS != nil {
			tss := make([]int64, 0, len(options.FilterByTS))
			for _, ts := range options.FilterByTS {
				tss = append(tss, int64(ts))
			}
			_cmd.FilterByTs(tss...)
		}
		if options.FilterByValue != nil {
			if len(options.FilterByValue) != 2 {
				panic(fmt.Sprintf("wrong number of arguments in options.FilterByValue, expect min, max, got %v", options.FilterByValue))
			}
			_cmd.FilterByValue(float64(options.FilterByValue[0]), float64(options.FilterByValue[1]))
		}
		if options.WithLabels {
			_cmd.Withlabels()
		}
		if options.SelectedLabels != nil {
			labels := make([]string, 0, len(options.SelectedLabels))
			for _, l := range options.SelectedLabels {
				labels = append(labels, str(l))
			}
			_cmd.SelectedLabels(labels)
		}
		if options.Count != 0 {
			_cmd.Count(int64(options.Count))
		}
		if options.Align != nil {
			_cmd.Align(str(options.Align))
		}
		if options.Aggregator != 0 {
			switch options.Aggregator {
			case Invalid:
				break
			case Avg:
				_cmd.AggregationAvg()
			case Sum:
				_cmd.AggregationSum()
			case Min:
				_cmd.AggregationMin()
			case Max:
				_cmd.AggregationMax()
			case Range:
				_cmd.AggregationRange()
			case Count:
				_cmd.AggregationCount()
			case First:
				_cmd.AggregationFirst()
			case Last:
				_cmd.AggregationLast()
			case StdP:
				_cmd.AggregationStdP()
			case StdS:
				_cmd.AggregationStdS()
			case VarP:
				_cmd.AggregationVarP()
			case VarS:
				_cmd.AggregationVarS()
			case Twa:
				_cmd.AggregationTwa()
			}
		}
		if options.BucketDuration != 0 {
			cmds.TsMrangeAggregationAggregationAvg(_cmd).Bucketduration(int64(options.BucketDuration))
		}
		if options.BucketTimestamp != nil {
			cmds.TsMrangeAggregationBucketduration(_cmd).Buckettimestamp(str(options.BucketTimestamp))
		}
		if options.Empty {
			cmds.TsMrangeAggregationBuckettimestamp(_cmd).Empty()
		}
		cmds.TsMrangeTotimestamp(_cmd).Filter(filterExpr...)
		if options.GroupByLabel != nil {
			// FIXME: Wrong API definition: REDUCE
			cmds.TsMrangeFilter(_cmd).Groupby(str(options.GroupByLabel), "REDUCE", str(options.Reducer))
		}
	}
	cmd := (cmds.TsMrangeFilter)(_cmd).Build()
	return newMapStringSliceInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSMRevRange - Returns a range of samples from multiple time-series keys in reverse order.
// For more information - https://redis.io/commands/ts.mrevrange/
func (c *Compat) TSMRevRange(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string) *MapStringSliceInterfaceCmd {
	cmd := c.client.B().TsMrange().Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp)).Filter(filterExpr...).Build()
	return newMapStringSliceInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSMRevRangeWithArgs - Returns a range of samples from multiple time-series keys in reverse order with additional options.
// This function allows for specifying additional options such as:
// Latest, FilterByTS, FilterByValue, WithLabels, SelectedLabels,
// Count, Align, Aggregator, BucketDuration, BucketTimestamp,
// Empty, GroupByLabel and Reducer.
// For more information - https://redis.io/commands/ts.mrevrange/
func (c *Compat) TSMRevRangeWithArgs(ctx context.Context, fromTimestamp int, toTimestamp int, filterExpr []string, options *TSMRevRangeOptions) *MapStringSliceInterfaceCmd {
	_cmd := c.client.B().TsMrevrange().Fromtimestamp(str(fromTimestamp)).Totimestamp(str(toTimestamp))
	if options != nil {
		if options.Latest {
			_cmd.Latest()
		}
		if options.FilterByTS != nil {
			tss := make([]int64, 0, len(options.FilterByTS))
			for _, ts := range options.FilterByTS {
				tss = append(tss, int64(ts))
			}
			_cmd.FilterByTs(tss...)
		}
		if options.FilterByValue != nil {
			if len(options.FilterByValue) != 2 {
				panic(fmt.Sprintf("wrong number of arguments in options.FilterByValue, expect min, max, got %v", options.FilterByValue))
			}
			_cmd.FilterByValue(float64(options.FilterByValue[0]), float64(options.FilterByValue[1]))
		}
		if options.WithLabels {
			_cmd.Withlabels()
		}
		if options.SelectedLabels != nil {
			labels := make([]string, 0, len(options.SelectedLabels))
			for _, l := range options.SelectedLabels {
				labels = append(labels, str(l))
			}
			_cmd.SelectedLabels(labels)
		}
		if options.Count != 0 {
			_cmd.Count(int64(options.Count))
		}
		if options.Align != nil {
			_cmd.Align(str(options.Align))
		}
		if options.Aggregator != 0 {
			switch options.Aggregator {
			case Invalid:
				break
			case Avg:
				_cmd.AggregationAvg()
			case Sum:
				_cmd.AggregationSum()
			case Min:
				_cmd.AggregationMin()
			case Max:
				_cmd.AggregationMax()
			case Range:
				_cmd.AggregationRange()
			case Count:
				_cmd.AggregationCount()
			case First:
				_cmd.AggregationFirst()
			case Last:
				_cmd.AggregationLast()
			case StdP:
				_cmd.AggregationStdP()
			case StdS:
				_cmd.AggregationStdS()
			case VarP:
				_cmd.AggregationVarP()
			case VarS:
				_cmd.AggregationVarS()
			case Twa:
				_cmd.AggregationTwa()
			}
		}
		if options.BucketDuration != 0 {
			cmds.TsMrevrangeAggregationAggregationAvg(_cmd).Bucketduration(int64(options.BucketDuration))
		}
		if options.BucketTimestamp != nil {
			cmds.TsMrevrangeAggregationBucketduration(_cmd).Buckettimestamp(str(options.BucketTimestamp))
		}
		if options.Empty {
			cmds.TsMrevrangeAggregationBuckettimestamp(_cmd).Empty()
		}
		cmds.TsMrevrangeTotimestamp(_cmd).Filter(filterExpr...)
		if options.GroupByLabel != nil {
			// FIXME: Wrong API definition: REDUCE
			cmds.TsMrevrangeFilter(_cmd).Groupby(str(options.GroupByLabel), "REDUCE", str(options.Reducer))
		}
	}
	cmd := (cmds.TsMrevrangeFilter)(_cmd).Build()
	return newMapStringSliceInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSMGet - Returns the last sample of multiple time-series keys.
// For more information - https://redis.io/commands/ts.mget/
func (c *Compat) TSMGet(ctx context.Context, filters []string) *MapStringSliceInterfaceCmd {
	cmd := c.client.B().TsMget().Filter(filters...).Build()
	return newMapStringSliceInterfaceCmd(c.client.Do(ctx, cmd))
}

// TSMGetWithArgs - Returns the last sample of multiple time-series keys with additional options.
// This function allows for specifying additional options such as:
// Latest, WithLabels and SelectedLabels.
// For more information - https://redis.io/commands/ts.mget/
func (c *Compat) TSMGetWithArgs(ctx context.Context, filters []string, options *TSMGetOptions) *MapStringSliceInterfaceCmd {
	_cmd := c.client.B().TsMget()
	if options != nil {
		if options.Latest {
			_cmd.Latest()
		}
		if options.WithLabels {
			_cmd.Withlabels()
		}
		if options.SelectedLabels != nil {
			labels := make([]string, 0, len(options.SelectedLabels))
			for _, l := range options.SelectedLabels {
				labels = append(labels, str(l))
			}
			_cmd.SelectedLabels(labels)
		}
	}
	cmd := _cmd.Filter(filters...).Build()
	return newMapStringSliceInterfaceCmd(c.client.Do(ctx, cmd))
}

// JSONArrAppend adds the provided JSON values to the end of the array at the given path.
// For more information, see https://redis.io/commands/json.arrappend
func (c *Compat) JSONArrAppend(ctx context.Context, key, path string, values ...interface{}) *IntSliceCmd {
	cmd := c.client.B().JsonArrappend().Key(key).Path(path).Value(argToSlice(values)...).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

// JSONArrIndex searches for the first occurrence of the provided JSON value in the array at the given path.
// For more information, see https://redis.io/commands/json.arrindex
// NOTE: value should have the format value start [stop]
func (c *Compat) JSONArrIndex(ctx context.Context, key, path string, value ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().JsonArrindex().Key(key).Path(path)
	switch len(value) {
	case 1:
		// format: value
		_cmd.Value(str(value[0]))
	case 2:
		// format: value start
		_cmd.Value(str(value[0])).Start((int64)(value[1].(int)))
	case 3:
		// format: value start stop
		_cmd.Value(str(value[0])).Start((int64)(value[1].(int))).Stop((int64)(value[2].(int)))
	default:
		panic(fmt.Sprintf("the format of value should be value [start [stop]], got %v", value))
	}
	cmd := (cmds.JsonArrindexValue)(_cmd).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

// JSONArrIndex searches for the first occurrence of the provided JSON value in the array at the given path.
// For more information, see https://redis.io/commands/json.arrindex
func (c *Compat) JSONArrIndexWithArgs(ctx context.Context, key, path string, options *JSONArrIndexArgs, value ...interface{}) *IntSliceCmd {
	// FIXME: why value has 1..N ?
	_cmd := c.client.B().JsonArrindex().Key(key).Path(path).Value(str(value[0]))
	if options != nil {
		if options.Start != 0 {
			_cmd.Start(int64(options.Start))
		} else {
			_cmd.Start(int64(0))
		}
		if options.Stop != nil {
			(cmds.JsonArrindexStartStart)(_cmd).Stop(int64(*options.Stop))
		}
	}
	cmd := _cmd.Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONArrInsert(ctx context.Context, key, path string, index int64, values ...interface{}) *IntSliceCmd {
	valStrs := make([]string, 0, len(values))
	for _, val := range values {
		valStrs = append(valStrs, str(val))
	}
	cmd := c.client.B().JsonArrinsert().Key(key).Path(path).Index(index).Value(valStrs...).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONArrLen(ctx context.Context, key, path string) *IntSliceCmd {
	cmd := c.client.B().JsonArrlen().Key(key).Path(path).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONArrPop(ctx context.Context, key, path string, index int) *StringSliceCmd {
	cmd := c.client.B().JsonArrpop().Key(key).Path(path).Index(int64(index)).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONArrTrim(ctx context.Context, key, path string) *IntSliceCmd {
	// both default value of start and stop are 0
	// Ref: https://redis.io/commands/json.arrtrim/
	cmd := c.client.B().JsonArrtrim().Key(key).Path(path).Start(0).Stop(0).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONArrTrimWithArgs(ctx context.Context, key, path string, options *JSONArrTrimArgs) *IntSliceCmd {
	_cmd := c.client.B().JsonArrtrim().Key(key).Path(path)
	if options != nil {
		if options.Start != 0 {
			_cmd.Start(int64(options.Start))
		} else {
			_cmd.Start(0)
		}
		if options.Stop != nil {
			(cmds.JsonArrtrimStart)(_cmd).Stop(int64(*options.Stop))
		} else {
			(cmds.JsonArrtrimStart)(_cmd).Stop(0)
		}
	}
	cmd := (cmds.JsonArrtrimStop)(_cmd).Build()
	return newIntSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONClear(ctx context.Context, key, path string) *IntCmd {
	cmd := c.client.B().JsonClear().Key(key).Path(path).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONDebugMemory(ctx context.Context, key, path string) *IntCmd {
	cmd := c.client.B().JsonDebugMemory().Key(key).Path(path).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONDel(ctx context.Context, key, path string) *IntCmd {
	cmd := c.client.B().JsonDel().Key(key).Path(path).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONForget(ctx context.Context, key, path string) *IntCmd {
	cmd := c.client.B().JsonForget().Key(key).Path(path).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONGet(ctx context.Context, key string, paths ...string) *JSONCmd {
	cmd := c.client.B().JsonGet().Key(key).Path(paths...).Build()
	return newJSONCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONGetWithArgs(ctx context.Context, key string, options *JSONGetArgs, paths ...string) *JSONCmd {
	// _cmd := c.client.B().JsonGet().Key(key).Path(paths...)
	_cmd := c.client.B().JsonGet().Key(key)
	if options != nil {
		if options.Indent != "" {
			_cmd.Indent(options.Indent)
		}
		if options.Newline != "" {
			_cmd.Newline(options.Newline)
		}
		if options.Space != "" {
			_cmd.Space(options.Space)
		}
	}
	cmd := _cmd.Path(paths...).Build()
	return newJSONCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONMerge(ctx context.Context, key, path string, value string) *StatusCmd {
	cmd := c.client.B().JsonMerge().Key(key).Path(path).Value(value).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONMSetArgs(ctx context.Context, docs []JSONSetArgs) *StatusCmd {
	_cmd := c.client.B().JsonMset()
	for _, doc := range docs {
		_cmd.Key(doc.Key).Path(doc.Path).Value(str(doc.Value))
	}
	cmd := (cmds.JsonMsetTripletValue)(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONMSet(ctx context.Context, params ...interface{}) *StatusCmd {
	_cmd := c.client.B().JsonMset()
	for i := 0; i < len(params); i += 3 {
		_cmd.Key(str(params[i])).Path(str(params[i+1])).Value(str(params[i+2]))
	}
	cmd := (cmds.JsonMsetTripletValue)(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONMGet(ctx context.Context, path string, keys ...string) *JSONSliceCmd {
	cmd := c.client.B().JsonMget().Key(keys...).Path(path).Build()
	return newJSONSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONNumIncrBy(ctx context.Context, key, path string, value float64) *JSONCmd {
	cmd := c.client.B().JsonNumincrby().Key(key).Path(path).Value(value).Build()
	return newJSONCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONObjKeys(ctx context.Context, key, path string) *SliceCmd {
	cmd := c.client.B().JsonObjkeys().Key(key).Path(path).Build()
	return newSliceCmd(c.client.Do(ctx, cmd), true)
}

func (c *Compat) JSONObjLen(ctx context.Context, key, path string) *IntPointerSliceCmd {
	cmd := c.client.B().JsonObjlen().Key(key).Path(path).Build()
	return newIntPointerSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONSet(ctx context.Context, key, path string, value interface{}) *StatusCmd {
	cmd := c.client.B().JsonSet().Key(key).Path(path).Value(str(value)).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// JSONSetMode sets the JSON value at the given path in the given key and allows the mode to be set
// (the mode value must be "XX" or "NX"). The value must be something that can be marshaled to JSON (using encoding/JSON) unless
// the argument is a string or []byte when we assume that it can be passed directly as JSON.
// For more information, see https://redis.io/commands/json.set
func (c *Compat) JSONSetMode(ctx context.Context, key, path string, value interface{}, mode string) *StatusCmd {
	_cmd := c.client.B().JsonSet().Key(key).Path(path).Value(str(value))
	switch mode {
	case "XX":
		_cmd.Xx()
	case "NX":
		_cmd.Nx()
	default:
		panic(`the mode value must be "XX" or "NX"`)
	}
	cmd := (cmds.JsonSetValue)(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONStrAppend(ctx context.Context, key, path, value string) *IntPointerSliceCmd {
	cmd := c.client.B().JsonStrappend().Key(key).Path(path).Value(value).Build()
	return newIntPointerSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONStrLen(ctx context.Context, key, path string) *IntPointerSliceCmd {
	cmd := c.client.B().JsonStrlen().Key(key).Path(path).Build()
	return newIntPointerSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONToggle(ctx context.Context, key, path string) *IntPointerSliceCmd {
	cmd := c.client.B().JsonToggle().Key(key).Path(path).Build()
	return newIntPointerSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) JSONType(ctx context.Context, key, path string) *JSONSliceCmd {
	cmd := c.client.B().JsonType().Key(key).Path(path).Build()
	return newJSONSliceCmd(c.client.Do(ctx, cmd))
}

func (c *Compat) Subscribe(ctx context.Context, channels ...string) PubSub {
	p := newPubSub(c.client)
	_ = p.Subscribe(ctx, channels...)
	return p
}

func (c *Compat) SSubscribe(ctx context.Context, channels ...string) PubSub {
	p := newPubSub(c.client)
	_ = p.SSubscribe(ctx, channels...)
	return p
}

func (c *Compat) PSubscribe(ctx context.Context, patterns ...string) PubSub {
	p := newPubSub(c.client)
	_ = p.PSubscribe(ctx, patterns...)
	return p
}

func (c *Compat) Pipelined(ctx context.Context, fn func(Pipeliner) error) ([]Cmder, error) {
	return newPipeline(c.client).Pipelined(ctx, fn)
}

func (c *Compat) Pipeline() Pipeliner {
	return newPipeline(c.client)
}

func (c *Compat) TxPipelined(ctx context.Context, fn func(Pipeliner) error) ([]Cmder, error) {
	return newTxPipeline(c.client).Pipelined(ctx, fn)
}

func (c *Compat) TxPipeline() Pipeliner {
	return newTxPipeline(c.client)
}

func (c *Compat) Watch(ctx context.Context, fn func(Tx) error, keys ...string) error {
	dc, cancel := c.client.Dedicate()
	defer cancel()
	tx := newTx(dc, cancel)
	if err := tx.Watch(ctx, keys...).Err(); err != nil {
		return err
	}
	return fn(newTx(dc, cancel))
}

func (c *Compat) FT_List(ctx context.Context) *StringSliceCmd {
	cmd := c.client.B().FtList().Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

// FTAggregate - Performs a search query on an index and applies a series of aggregate transformations to the result.
// The 'index' parameter specifies the index to search, and the 'query' parameter specifies the search query.
// For more information, please refer to the Redis documentation:
// [FT.AGGREGATE]: (https://redis.io/commands/ft.aggregate/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L473
func (c *Compat) FTAggregate(ctx context.Context, index string, query string) *MapStringInterfaceCmd {
	cmd := c.client.B().FtAggregate().Index(index).Query(query).Build()
	return newMapStringInterfaceCmd(c.client.Do(ctx, cmd))
}

// FTAggregateWithArgs - Performs a search query on an index and applies a series of aggregate transformations to the result.
// The 'index' parameter specifies the index to search, and the 'query' parameter specifies the search query.
// This function also allows for specifying additional options such as: Verbatim, LoadAll, Load, Timeout, GroupBy, SortBy, SortByMax, Apply, LimitOffset, Limit, Filter, WithCursor, Params, and DialectVersion.
// For more information, please refer to the Redis documentation:
// [FT.AGGREGATE]: (https://redis.io/commands/ft.aggregate/)
// see: go-redis v9.7.0: https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L671
func (c *Compat) FTAggregateWithArgs(ctx context.Context, index string, query string, options *FTAggregateOptions) *AggregateCmd {
	_cmd := cmds.Incomplete(c.client.B().FtAggregate().Index(index).Query(query))
	if options != nil {
		// [VERBATIM]
		if options.Verbatim {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Verbatim())
		}
		// [LOAD count field [field ...]]
		if options.LoadAll {
			// LOAD *
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).LoadAll())
		} else {
			// LOAD
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Load(int64(len(options.Load))))
			fields := make([]string, 0, len(options.Load))
			for _, l := range options.Load {
				fields = append(fields, l.Field)
			}
			_cmd = cmds.Incomplete(cmds.FtAggregateOpLoadLoad(_cmd).Field(fields...))
		}
		// [TIMEOUT timeout]
		if options.Timeout > 0 {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Timeout(int64(options.Timeout)))
		}
		// [ GROUPBY nargs property [property ...] [ REDUCE function nargs arg [arg ...] [AS name] [ REDUCE function nargs arg [arg ...] [AS name] ...]] ...]]
		if options.GroupBy != nil {
			for _, groupBy := range options.GroupBy {
				_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).
					Groupby(int64(len(options.GroupBy))).
					Property(argsToSlice(groupBy.Fields)...))
				for _, reduce := range groupBy.Reduce {
					_cmd = cmds.Incomplete(cmds.FtAggregateOpGroupbyProperty(_cmd).
						Reduce(reduce.Reducer.String()).
						Nargs(int64(len(reduce.Args))).
						Arg(argsToSlice(reduce.Args)...))
					if reduce.As != "" {
						_cmd = cmds.Incomplete(cmds.FtAggregateOpGroupbyReduceArg(_cmd).As(reduce.As))
					}
				}
			}
		}
		// [ SORTBY nargs [ property ASC | DESC [ property ASC | DESC ...]] [MAX num] [WITHCOUNT]
		if options.SortBy != nil {
			var numOfArgs int64 = 0
			// count number of args to be passed in to cmds.FtAggregateQuery(_cmd).Sortby()
			for _, sortBy := range options.SortBy {
				numOfArgs++
				if sortBy.Asc && sortBy.Desc {
					panic("FT.AGGREGATE: ASC and DESC are mutually exclusive")
				}
				if sortBy.Asc || sortBy.Desc {
					numOfArgs++
				}
			}
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Sortby(numOfArgs))
			for _, sortBy := range options.SortBy {
				_cmd = cmds.Incomplete(cmds.FtAggregateOpSortbySortby(_cmd).Property(sortBy.FieldName))
				if sortBy.Asc && sortBy.Desc {
					panic("FT.AGGREGATE: ASC and DESC are mutually exclusive")
				}
				if sortBy.Asc {
					// ASC
					_cmd = cmds.Incomplete(cmds.FtAggregateOpSortbyFieldsProperty(_cmd).Asc())
				}
				if sortBy.Desc {
					// DESC
					_cmd = cmds.Incomplete(cmds.FtAggregateOpSortbyFieldsProperty(_cmd).Desc())
				}
			}
		}
		if options.SortByMax > 0 {
			_cmd = cmds.Incomplete(cmds.FtAggregateOpSortbySortby(_cmd).Max(int64(options.SortByMax)))
		}
		// FIXME: go-redis doesn't provide WITHCOUNT option

		// [ APPLY expression AS name [ APPLY expression AS name ...]]
		if options.Apply != nil {
			for _, apply := range options.Apply {
				_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Apply(apply.Field).As(apply.As))
			}
		}
		// [ LIMIT offset num]
		if options.LimitOffset > 0 {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Limit().OffsetNum(int64(options.Limit), int64(options.LimitOffset)))
		}
		// [FILTER filter]
		if options.Filter != "" {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Filter(options.Filter))
		}
		// [ WITHCURSOR [COUNT read_size] [MAXIDLE idle_time]]
		if options.WithCursor {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Withcursor())
			if options.WithCursorOptions != nil {
				if options.WithCursorOptions.Count > 0 {
					_cmd = cmds.Incomplete(cmds.FtAggregateCursorWithcursor(_cmd).Count(int64(options.WithCursorOptions.Count)))
				}
				if options.WithCursorOptions.MaxIdle > 0 {
					_cmd = cmds.Incomplete(cmds.FtAggregateCursorWithcursor(_cmd).Maxidle(int64(options.WithCursorOptions.MaxIdle)))
				}
			}
		}
		// [ PARAMS nargs name value [ name value ...]]
		if options.Params != nil {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Params().Nargs(int64(len(options.Params) * 2)).NameValue())
			for name, val := range options.Params {
				_cmd = cmds.Incomplete(cmds.FtAggregateParamsNameValue(_cmd).NameValue(name, str(val)))
			}
		}
		// [ADDSCORES]: NOTE: go-redis doesn't implement this option.
		// [DIALECT dialect]
		if options.DialectVersion > 0 {
			_cmd = cmds.Incomplete(cmds.FtAggregateQuery(_cmd).Dialect(int64(options.DialectVersion)))
		}
	}
	cmd := cmds.FtAggregateQuery(_cmd).Build()
	return newAggregateCmd(c.client.Do(ctx, cmd))
}

// FTAliasAdd - Adds an alias to an index.
// The 'index' parameter specifies the index to which the alias is added, and the 'alias' parameter specifies the alias.
// For more information, please refer to the Redis documentation:
// [FT.ALIASADD]: (https://redis.io/commands/ft.aliasadd/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L782
func (c *Compat) FTAliasAdd(ctx context.Context, index string, alias string) *StatusCmd {
	cmd := c.client.B().FtAliasadd().Alias(alias).Index(index).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTAliasDel - Removes an alias from an index.
// The 'alias' parameter specifies the alias to be removed.
// For more information, please refer to the Redis documentation:
// [FT.ALIASDEL]: (https://redis.io/commands/ft.aliasdel/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L793
func (c *Compat) FTAliasDel(ctx context.Context, alias string) *StatusCmd {
	cmd := c.client.B().FtAliasdel().Alias(alias).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTAliasUpdate - Updates an alias to an index.
// The 'index' parameter specifies the index to which the alias is updated, and the 'alias' parameter specifies the alias.
// If the alias already exists for a different index, it updates the alias to point to the specified index instead.
// For more information, please refer to the Redis documentation:
// [FT.ALIASUPDATE]: (https://redis.io/commands/ft.aliasupdate/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L804
func (c *Compat) FTAliasUpdate(ctx context.Context, index string, alias string) *StatusCmd {
	cmd := c.client.B().FtAliasupdate().Alias(alias).Index(index).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTAlter - Alters the definition of an existing index.
// The 'index' parameter specifies the index to alter, and the 'skipInitialScan' parameter specifies whether to skip the initial scan.
// The 'definition' parameter specifies the new definition for the index.
// For more information, please refer to the Redis documentation:
// [FT.ALTER]: (https://redis.io/commands/ft.alter/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L815
func (c *Compat) FTAlter(ctx context.Context, index string, skipInitalScan bool, definition []interface{}) *StatusCmd {
	_cmd := cmds.Incomplete(c.client.B().FtAlter().Index(index))
	if skipInitalScan {
		_cmd = cmds.Incomplete(cmds.FtAlterIndex(_cmd).Skipinitialscan())
	}
	if len(definition) != 2 {
		panic("definition should contain attribute and options")
	}
	cmd := cmds.FtAlterIndex(_cmd).Schema().Add().Field(str(definition[0])).Options(str(definition[1])).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTConfigGet - Retrieves the value of a RediSearch configuration parameter.
// The 'option' parameter specifies the configuration parameter to retrieve.
// For more information, please refer to the Redis documentation:
// [FT.CONFIG GET]: (https://redis.io/commands/ft.config-get/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L831
func (c *Compat) FTConfigGet(ctx context.Context, option string) *MapMapStringInterfaceCmd {
	cmd := c.client.B().FtConfigGet().Option(option).Build()
	return newMapMapStringInterfaceCmd(c.client.Do(ctx, cmd))
}

// FTConfigSet - Sets the value of a RediSearch configuration parameter.
// The 'option' parameter specifies the configuration parameter to set, and the 'value' parameter specifies the new value.
// For more information, please refer to the Redis documentation:
// [FT.CONFIG SET]: (https://redis.io/commands/ft.config-set/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L841
func (c *Compat) FTConfigSet(ctx context.Context, option string, value interface{}) *StatusCmd {
	cmd := c.client.B().FtConfigSet().Option(option).Value(str(value)).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTCreate - Creates a new index with the given options and schema.
// The 'index' parameter specifies the name of the index to create.
// The 'options' parameter specifies various options for the index, such as:
// whether to index hashes or JSONs, prefixes, filters, default language, score, score field, payload field, etc.
// The 'schema' parameter specifies the schema for the index, which includes the field name, field type, etc.
// For more information, please refer to the Redis documentation:
// [FT.CREATE]: (https://redis.io/commands/ft.create/)
// FTCreate aligns with go-redis v9.7.0.
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L854
func (c *Compat) FTCreate(ctx context.Context, index string, options *FTCreateOptions, schema ...*FieldSchema) *StatusCmd {
	_cmd := cmds.Incomplete(c.client.B().FtCreate().Index(index))
	if options != nil {
		// [ON HASH | JSON]
		if options.OnHash {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).OnHash())
		}
		if options.OnJSON {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).OnJson())
		}
		// [PREFIX count prefix [prefix ...]]
		_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Prefix(int64(len(options.Prefix))).Prefix(argsToSlice(options.Prefix)...))
		// [FILTER {filter}]
		if options.Filter != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Filter(options.Filter))
		}
		// [LANGUAGE default_lang]
		if options.DefaultLanguage != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Language(options.DefaultLanguage))
		}
		// [LANGUAGE_FIELD lang_attribute]
		if options.LanguageField != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).LanguageField(options.LanguageField))
		}
		// [SCORE default_score]
		if options.Score != 0 {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Score(options.Score))
		}
		// [SCORE_FIELD score_attribute]
		if options.ScoreField != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).ScoreField(options.ScoreField))
		}
		// [PAYLOAD_FIELD payload_attribute]
		if options.PayloadField != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).PayloadField(options.PayloadField))
		}
		// [MAXTEXTFIELDS]
		// FIXME: in go-reids, FTCreateOptions.MaxTextFields should be bool, not int
		if options.MaxTextFields > 0 {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Maxtextfields())
		}
		// [TEMPORARY seconds]
		// FIXME: reudis: Temporary should not be float64
		if options.Temporary > 0 {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Temporary(float64(options.Temporary)))
		}
		// [NOOFFSETS]
		if options.NoOffsets {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Nooffsets())
		}
		// [NOHL]
		if options.NoHL {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Nohl())
		}
		// [NOFIELDS]
		if options.NoFields {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Nofields())
		}
		// [NOFREQS]
		if options.NoFreqs {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Nofreqs())
		}
		// [STOPWORDS count [stopword ...]]
		if len(options.StopWords) > 0 {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Stopwords(int64(len(options.StopWords))).Stopword(argsToSlice(options.StopWords)...))
		}
		// [SKIPINITIALSCAN]
		if options.SkipInitialScan {
			_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Skipinitialscan())
		}
	}
	_cmd = cmds.Incomplete(cmds.FtCreateIndex(_cmd).Schema())
	// 	SCHEMA field_name [AS alias] TEXT | TAG | NUMERIC | GEO | VECTOR | GEOSHAPE [ SORTABLE [UNF]]
	//   [NOINDEX] [ field_name [AS alias] TEXT | TAG | NUMERIC | GEO | VECTOR | GEOSHAPE [ SORTABLE [UNF]] [NOINDEX] ...]
	for _, sc := range schema {
		if sc.FieldName == "" || sc.FieldType == SearchFieldTypeInvalid {
			panic("FT.CREATE: SCHEMA FieldName and FieldType are required")
		}
		_cmd = cmds.Incomplete(cmds.FtCreateSchema(_cmd).FieldName(sc.FieldName))
		if sc.As != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldName(_cmd).As(sc.As))
		}
		switch sc.FieldType {
		case SearchFieldTypeNumeric:
			_cmd = cmds.Incomplete(cmds.FtCreateFieldAs(_cmd).Numeric())
		case SearchFieldTypeTag:
			_cmd = cmds.Incomplete(cmds.FtCreateFieldAs(_cmd).Tag())
		case SearchFieldTypeText:
			_cmd = cmds.Incomplete(cmds.FtCreateFieldAs(_cmd).Text())
		case SearchFieldTypeGeo:
			_cmd = cmds.Incomplete(cmds.FtCreateFieldAs(_cmd).Geo())
		case SearchFieldTypeVector:
			// Ref: https://redis.io/docs/latest/develop/interact/search-and-query/advanced-concepts/vectors/#create-a-vector-index
			if sc.VectorArgs == nil {
				panic("FT.CREATE: SCHEMA VectorArgs cannot be nil")
			}
			if sc.VectorArgs.FlatOptions != nil && sc.VectorArgs.HNSWOptions != nil {
				panic("FT.CREATE: SCHEMA VectorArgs FlatOptions and HNSWOptions are mutually exclusive")
			}
			var args []any
			algorithm := ""
			if sc.VectorArgs.FlatOptions != nil {
				algorithm = "FLAT"
				if sc.VectorArgs.FlatOptions.Type == "" || sc.VectorArgs.FlatOptions.Dim == 0 || sc.VectorArgs.FlatOptions.DistanceMetric == "" {
					panic("FT.CREATE: Type, Dim and DistanceMetric are required for VECTOR FLAT")
				}
				flatArgs := []interface{}{
					"TYPE", sc.VectorArgs.FlatOptions.Type,
					"DIM", sc.VectorArgs.FlatOptions.Dim,
					"DISTANCE_METRIC", sc.VectorArgs.FlatOptions.DistanceMetric,
				}
				if sc.VectorArgs.FlatOptions.InitialCapacity > 0 {
					flatArgs = append(flatArgs, "INITIAL_CAP", sc.VectorArgs.FlatOptions.InitialCapacity)
				}
				if sc.VectorArgs.FlatOptions.BlockSize > 0 {
					flatArgs = append(flatArgs, "BLOCK_SIZE", sc.VectorArgs.FlatOptions.BlockSize)
				}
				args = flatArgs
			}
			if sc.VectorArgs.HNSWOptions != nil {
				algorithm = "HNSW"
				if sc.VectorArgs.HNSWOptions.Type == "" || sc.VectorArgs.HNSWOptions.Dim == 0 || sc.VectorArgs.HNSWOptions.DistanceMetric == "" {
					panic("FT.CREATE: Type, Dim and DistanceMetric are required for VECTOR HNSW")
				}
				hnswArgs := []interface{}{
					"TYPE", sc.VectorArgs.HNSWOptions.Type,
					"DIM", sc.VectorArgs.HNSWOptions.Dim,
					"DISTANCE_METRIC", sc.VectorArgs.HNSWOptions.DistanceMetric,
				}
				if sc.VectorArgs.HNSWOptions.InitialCapacity > 0 {
					hnswArgs = append(hnswArgs, "INITIAL_CAP", sc.VectorArgs.HNSWOptions.InitialCapacity)
				}
				if sc.VectorArgs.HNSWOptions.MaxEdgesPerNode > 0 {
					hnswArgs = append(hnswArgs, "M", sc.VectorArgs.HNSWOptions.MaxEdgesPerNode)
				}
				if sc.VectorArgs.HNSWOptions.MaxAllowedEdgesPerNode > 0 {
					hnswArgs = append(hnswArgs, "EF_CONSTRUCTION", sc.VectorArgs.HNSWOptions.MaxAllowedEdgesPerNode)
				}
				if sc.VectorArgs.HNSWOptions.EFRunTime > 0 {
					hnswArgs = append(hnswArgs, "EF_RUNTIME", sc.VectorArgs.HNSWOptions.EFRunTime)
				}
				if sc.VectorArgs.HNSWOptions.Epsilon > 0 {
					hnswArgs = append(hnswArgs, "EPSILON", sc.VectorArgs.HNSWOptions.Epsilon)
				}
				args = hnswArgs
			}
			_cmd = cmds.Incomplete(cmds.FtCreateFieldAs(_cmd).Vector(algorithm, int64(len(args)), argsToSlice(args)...))

		case SearchFieldTypeGeoShape:
			if sc.GeoShapeFieldType == "" {
				panic("FT.CREATE: GeoShapeFieldType cannot be empty while SCHEMA FieldType is GEOSHAPE")
			}
			_cmd = cmds.Incomplete(cmds.FtCreateFieldAs(_cmd).Geoshape().FieldName(sc.GeoShapeFieldType))
		default:
			panic(fmt.Sprintf("unexpected SearchFieldType: %s", sc.FieldType.String()))
		}
		if sc.NoStem {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Nostem())
		}
		if sc.Sortable {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Sortable())
		}
		if sc.UNF {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldOptionSortableSortable(_cmd).Unf())
		}
		if sc.NoIndex {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Noindex())
		}
		// FIXME: redis doc: PHONETIC not in EBNF definition
		if sc.PhoneticMatcher != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Phonetic(sc.PhoneticMatcher))
		}
		if sc.Weight > 0 {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Weight(sc.Weight))
		}
		if sc.Separator != "" {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Separator(sc.Separator))
		}
		if sc.CaseSensitive {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Casesensitive())
		}
		if sc.WithSuffixtrie {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Withsuffixtrie())
		}
		if sc.IndexEmpty {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Indexempty())
		}
		if sc.IndexMissing {
			_cmd = cmds.Incomplete(cmds.FtCreateFieldFieldTypeText(_cmd).Indexmissing())
		}
	}
	cmd := cmds.FtCreateFieldFieldTypeText(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTCursorDel - Deletes a cursor from an existing index.
// The 'index' parameter specifies the index from which to delete the cursor, and the 'cursorId' parameter specifies the ID of the cursor to delete.
// For more information, please refer to the Redis documentation:
// [FT.CURSOR DEL]: (https://redis.io/commands/ft.cursor-del/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1032
func (c *Compat) FTCursorDel(ctx context.Context, index string, cursorId int) *StatusCmd {
	cmd := c.client.B().FtCursorDel().Index(index).CursorId(int64(cursorId)).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTCursorRead - Reads the next results from an existing cursor.
// The 'index' parameter specifies the index from which to read the cursor, the 'cursorId' parameter specifies the ID of the cursor to read, and the 'count' parameter specifies the number of results to read.
// For more information, please refer to the Redis documentation:
// [FT.CURSOR READ]: (https://redis.io/commands/ft.cursor-read/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1042
func (c *Compat) FTCursorRead(ctx context.Context, index string, cursorId int, count int) *MapStringInterfaceCmd {
	cmd := c.client.B().FtCursorRead().Index(index).CursorId(int64(cursorId)).Count(int64(count)).Build()
	return newMapStringInterfaceCmd(c.client.Do(ctx, cmd))
}

// FTDictAdd - Adds terms to a dictionary.
// The 'dict' parameter specifies the dictionary to which to add the terms, and the 'term' parameter specifies the terms to add.
// For more information, please refer to the Redis documentation:
// [FT.DICTADD]: (https://redis.io/commands/ft.dictadd/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1056
func (c *Compat) FTDictAdd(ctx context.Context, dict string, term ...interface{}) *IntCmd {
	cmd := c.client.B().FtDictadd().Dict(dict).Term(argsToSlice(term)...).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// FTDictDel - Deletes terms from a dictionary.
// The 'dict' parameter specifies the dictionary from which to delete the terms, and the 'term' parameter specifies the terms to delete.
// For more information, please refer to the Redis documentation:
// [FT.DICTDEL]: (https://redis.io/commands/ft.dictdel/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1068
func (c *Compat) FTDictDel(ctx context.Context, dict string, term ...interface{}) *IntCmd {
	cmd := c.client.B().FtDictdel().Dict(dict).Term(argsToSlice(term)...).Build()
	return newIntCmd(c.client.Do(ctx, cmd))
}

// FTDictDump - Returns all terms in the specified dictionary.
// The 'dict' parameter specifies the dictionary from which to return the terms.
// For more information, please refer to the Redis documentation:
// [FT.DICTDUMP]: (https://redis.io/commands/ft.dictdump/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1080
func (c *Compat) FTDictDump(ctx context.Context, dict string) *StringSliceCmd {
	cmd := c.client.B().FtDictdump().Dict(dict).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

// FTDropIndex - Deletes an index.
// The 'index' parameter specifies the index to delete.
// For more information, please refer to the Redis documentation:
// [FT.DROPINDEX]: (https://redis.io/commands/ft.dropindex/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1090
func (c *Compat) FTDropIndex(ctx context.Context, index string) *StatusCmd {
	cmd := c.client.B().FtDropindex().Index(index).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTDropIndexWithArgs - Deletes an index with options.
// The 'index' parameter specifies the index to delete, and the 'options' parameter specifies the DeleteDocs option for docs deletion.
// For more information, please refer to the Redis documentation:
// [FT.DROPINDEX]: (https://redis.io/commands/ft.dropindex/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1101
func (c *Compat) FTDropIndexWithArgs(ctx context.Context, index string, options *FTDropIndexOptions) *StatusCmd {
	_cmd := cmds.Incomplete(c.client.B().FtDropindex().Index(index))
	if options.DeleteDocs {
		_cmd = cmds.Incomplete(cmds.FtDropindexIndex(_cmd).Dd())
	}
	cmd := cmds.FtDropindexIndex(_cmd).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTExplain - Returns the execution plan for a complex query.
// The 'index' parameter specifies the index to query, and the 'query' parameter specifies the query string.
// For more information, please refer to the Redis documentation:
// [FT.EXPLAIN]: (https://redis.io/commands/ft.explain/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1117
func (c *Compat) FTExplain(ctx context.Context, index string, query string) *StringCmd {
	cmd := c.client.B().FtExplain().Index(index).Query(query).Build()
	return newStringCmd(c.client.Do(ctx, cmd))
}

// FTExplainWithArgs - Returns the execution plan for a complex query with options.
// The 'index' parameter specifies the index to query, the 'query' parameter specifies the query string, and the 'options' parameter specifies the Dialect for the query.
// For more information, please refer to the Redis documentation:
// [FT.EXPLAIN]: (https://redis.io/commands/ft.explain/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1127
func (c *Compat) FTExplainWithArgs(ctx context.Context, index string, query string, options *FTExplainOptions) *StringCmd {
	_cmd := cmds.Incomplete(c.client.B().FtExplain().Index(index).Query(query))
	if options != nil {
		dialectVersion, err := strconv.ParseInt(options.Dialect, 10, 64)
		if err != nil {
			panic(fmt.Errorf("failed to parse dialect version: %v", err))
		}
		_cmd = cmds.Incomplete(cmds.FtExplainQuery(_cmd).Dialect(dialectVersion))
	}
	cmd := cmds.FtExplainQuery(_cmd).Build()
	return newStringCmd(c.client.Do(ctx, cmd))
}

// FTInfo - Retrieves information about an index.
// The 'index' parameter specifies the index to retrieve information about.
// For more information, please refer to the Redis documentation:
// [FT.INFO]: (https://redis.io/commands/ft.info/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1393
func (c *Compat) FTInfo(ctx context.Context, index string) *FTInfoCmd {
	cmd := c.client.B().FtInfo().Index(index).Build()
	return newFTInfoCmd(c.client.Do(ctx, cmd))
}

// FTSpellCheck - Checks a query string for spelling errors.
// For more details about spellcheck query please follow:
// https://redis.io/docs/interact/search-and-query/advanced-concepts/spellcheck/
// For more information, please refer to the Redis documentation:
// [FT.SPELLCHECK]: (https://redis.io/commands/ft.spellcheck/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1404
func (c *Compat) FTSpellCheck(ctx context.Context, index string, query string) *FTSpellCheckCmd {
	cmd := c.client.B().FtSpellcheck().Index(index).Query(query).Build()
	return newFTSpellCheckCmd(c.client.Do(ctx, cmd))
}

// FTSpellCheckWithArgs - Checks a query string for spelling errors with additional options.
// For more details about spellcheck query please follow:
// https://redis.io/docs/interact/search-and-query/advanced-concepts/spellcheck/
// For more information, please refer to the Redis documentation:
// [FT.SPELLCHECK]: (https://redis.io/commands/ft.spellcheck/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1416
func (c *Compat) FTSpellCheckWithArgs(ctx context.Context, index string, query string, options *FTSpellCheckOptions) *FTSpellCheckCmd {
	_cmd := cmds.Incomplete(c.client.B().FtSpellcheck().Index(index).Query(query))
	if options != nil {
		if options.Distance > 0 {
			_cmd = cmds.Incomplete(cmds.FtSpellcheckQuery(_cmd).Distance(int64(options.Distance)))
		}
		if options.Terms != nil {
			if options.Terms.Inclusion != "INCLUDE" && options.Terms.Inclusion != "EXCLUDE" {
				panic("Inclusion should be either INCLUDE or EXCLUDE")
			}
			if options.Terms.Inclusion == "INCLUDE" {
				_cmd = cmds.Incomplete(cmds.FtSpellcheckQuery(_cmd).TermsInclude().Dictionary(options.Terms.Dictionary).Terms(argsToSlice(options.Terms.Terms)...))
			} else {
				_cmd = cmds.Incomplete(cmds.FtSpellcheckQuery(_cmd).TermsExclude().Dictionary(options.Terms.Dictionary).Terms(argsToSlice(options.Terms.Terms)...))
			}
		}
		if options.Dialect > 0 {
			_cmd = cmds.Incomplete(cmds.FtSpellcheckQuery(_cmd).Dialect(int64(options.Dialect)))
		}
	}
	cmd := cmds.FtSpellcheckQuery(_cmd).Build()
	return newFTSpellCheckCmd(c.client.Do(ctx, cmd))
}

// FTSearch - Executes a search query on an index.
// The 'index' parameter specifies the index to search, and the 'query' parameter specifies the search query.
// For more information, please refer to the Redis documentation:
// [FT.SEARCH]: (https://redis.io/commands/ft.search/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1679
func (c *Compat) FTSearch(ctx context.Context, index string, query string) *FTSearchCmd {
	cmd := c.client.B().FtSearch().Index(index).Query(query).Build()
	return newFTSearchCmd(c.client.Do(ctx, cmd), nil)
}

// FTSearchWithArgs - Executes a search query on an index with additional options.
// The 'index' parameter specifies the index to search, the 'query' parameter specifies the search query,
// and the 'options' parameter specifies additional options for the search.
// For more information, please refer to the Redis documentation:
// [FT.SEARCH]: (https://redis.io/commands/ft.search/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1802
func (c *Compat) FTSearchWithArgs(ctx context.Context, index string, query string, options *FTSearchOptions) *FTSearchCmd {
	_cmd := cmds.Incomplete(c.client.B().FtSearch().Index(index).Query(query))
	if options != nil {
		// [NOCONTENT]
		if options.NoContent {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Nocontent())
		}
		// [VERBATIM]
		if options.Verbatim {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Verbatim())
		}
		// [NOSTOPWORDS]
		if options.NoStopWords {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Nostopwords())
		}
		// [WITHSCORES]
		if options.WithScores {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Withscores())
		}
		// [WITHPAYLOADS]
		if options.WithPayloads {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Withpayloads())
		}
		// [WITHSORTKEYS]
		if options.WithSortKeys {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Withsortkeys())
		}
		// [FILTER numeric_field min max [ FILTER numeric_field min max ...]]
		for _, filter := range options.Filters {
			min, err := strconv.ParseFloat(str(filter.Min), 64)
			if err != nil {
				panic(fmt.Sprintf("failed to parse min %v to float64", filter.Min))
			}
			max, err := strconv.ParseFloat(str(filter.Max), 64)
			if err != nil {
				panic(fmt.Sprintf("failed to parse max %v to float64", filter.Max))
			}
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Filter(str(filter.FieldName)).Min(min).Max(max))
		}
		//  [GEOFILTER geo_field lon lat radius m | km | mi | ft [ GEOFILTER geo_field lon lat radius m | km | mi | ft ...]]
		for _, filter := range options.GeoFilter {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Geofilter(filter.FieldName).Lon(filter.Longitude).Lat(filter.Latitude).Radius(filter.Radius))
			switch filter.Unit {
			case "m":
				_cmd = cmds.Incomplete(cmds.FtSearchGeoFilterRadius(_cmd).M())
			case "km":
				_cmd = cmds.Incomplete(cmds.FtSearchGeoFilterRadius(_cmd).Km())
			case "mi":
				_cmd = cmds.Incomplete(cmds.FtSearchGeoFilterRadius(_cmd).Mi())
			case "ft":
				_cmd = cmds.Incomplete(cmds.FtSearchGeoFilterRadius(_cmd).Ft())
			default:
				panic(fmt.Sprintf("invalid unit, want m | km | mi | ft, got %v", filter.Unit))
			}
		}
		// [INKEYS count key [key ...]]
		if len(options.InKeys) > 0 {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Inkeys(str(len(options.InKeys))).Key(argsToSlice(options.InKeys)...))
		}
		// [ INFIELDS count field [field ...]]
		if len(options.InFields) > 0 {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Infields(str(len(options.InFields))).Field(argsToSlice(options.InFields)...))
		}
		// [RETURN count identifier [AS property] [ identifier [AS property] ...]]
		if len(options.Return) > 0 {
			var numOfArgs int64 = 0
			for _, re := range options.Return {
				numOfArgs++
				if re.As != "" {
					numOfArgs += 2
				}
			}
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Return(str(numOfArgs)))
			for _, re := range options.Return {
				_cmd = cmds.Incomplete(cmds.FtSearchReturnReturn(_cmd).Identifier(re.FieldName))
				if re.As != "" {
					_cmd = cmds.Incomplete(cmds.FtSearchReturnIdentifiersIdentifier(_cmd).As(re.As))
				}
			}
		}
		// FIXME: go-redis doesn't implement SUMMARIZE option
		// [SUMMARIZE [ FIELDS count field [field ...]] [FRAGS num] [LEN fragsize] [SEPARATOR separator]]
		// [SLOP slop]
		if options.Slop > 0 {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Slop(int64(options.Slop)))
		}
		// [TIMEOUT timeout]
		if options.Timeout > 0 {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Timeout(int64(options.Timeout)))
		}
		// [INORDER]
		if options.InOrder {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Inorder())
		}
		// [LANGUAGE language]
		if options.Language != "" {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Language(options.Language))
		}
		// [EXPANDER expander]
		if options.Expander != "" {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Expander(options.Expander))
		}
		// [SCORER scorer]
		if options.Scorer != "" {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Scorer(options.Scorer))
		}
		// [EXPLAINSCORE]
		if options.ExplainScore {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Explainscore())
		}
		// [PAYLOAD payload]
		if options.Payload != "" {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Payload(options.Payload))
		}
		// [SORTBY sortby [ ASC | DESC] [WITHCOUNT]]
		if options.SortBy != nil {
			if len(options.SortBy) != 1 {
				panic(fmt.Sprintf("options.SortBy can only have 1 element, got %v", len(options.SortBy)))
			}
			sortBy := options.SortBy[0]
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Sortby(sortBy.FieldName))
			if sortBy.Asc == sortBy.Desc && sortBy.Asc {
				panic("options.SortBy[0] should specify either ASC or DESC")
			}
			if sortBy.Asc {
				_cmd = cmds.Incomplete(cmds.FtSearchSortbySortby(_cmd).Asc())
			} else {
				_cmd = cmds.Incomplete(cmds.FtSearchSortbySortby(_cmd).Desc())
			}
			if options.SortByWithCount {
				_cmd = cmds.Incomplete(cmds.FtSearchSortbySortby(_cmd).Withcount())
			}
		}
		// [LIMIT offset num]
		if options.LimitOffset >= 0 && options.Limit > 0 {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Limit().OffsetNum(int64(options.Limit), int64(options.LimitOffset)))
		}
		// [PARAMS nargs name value [ name value ...]]
		if options.Params != nil {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Params().Nargs(int64(len(options.Params) * 2)))
			for name, val := range options.Params {
				_cmd = cmds.Incomplete(cmds.FtSearchParamsNargs(_cmd).NameValue().NameValue(name, str(val)))
			}
		}
		// [DIALECT dialect]
		if options.DialectVersion > 0 {
			_cmd = cmds.Incomplete(cmds.FtSearchQuery(_cmd).Dialect(int64(options.DialectVersion)))
		}
	}
	cmd := cmds.FtSearchQuery(_cmd).Build()
	return newFTSearchCmd(c.client.Do(ctx, cmd), options)
}

// FTSynDump - Dumps the contents of a synonym group.
// The 'index' parameter specifies the index to dump.
// For more information, please refer to the Redis documentation:
// [FT.SYNDUMP]: (https://redis.io/commands/ft.syndump/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1987
func (c *Compat) FTSynDump(ctx context.Context, index string) *FTSynDumpCmd {
	cmd := c.client.B().FtSyndump().Index(index).Build()
	return newFTSynDumpCmd(c.client.Do(ctx, cmd))
}

// FTSynUpdate - Creates or updates a synonym group with additional terms.
// The 'index' parameter specifies the index to update, the 'synGroupId' parameter specifies the synonym group id, and the 'terms' parameter specifies the additional terms.
// For more information, please refer to the Redis documentation:
// [FT.SYNUPDATE]: (https://redis.io/commands/ft.synupdate/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L1997
func (c *Compat) FTSynUpdate(ctx context.Context, index string, synGroupId interface{}, terms []interface{}) *StatusCmd {
	cmd := c.client.B().FtSynupdate().Index(index).SynonymGroupId(str(synGroupId)).Term(argToSlice(terms)...).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTSynUpdateWithArgs - Creates or updates a synonym group with additional terms and options.
// The 'index' parameter specifies the index to update, the 'synGroupId' parameter specifies the synonym group id, the 'options' parameter specifies additional options for the update, and the 'terms' parameter specifies the additional terms.
// For more information, please refer to the Redis documentation:
// [FT.SYNUPDATE]: (https://redis.io/commands/ft.synupdate/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L2009
func (c *Compat) FTSynUpdateWithArgs(ctx context.Context, index string, synGroupId interface{}, options *FTSynUpdateOptions, terms []interface{}) *StatusCmd {
	_cmd := cmds.Incomplete(c.client.B().FtSynupdate().Index(index).SynonymGroupId(str(synGroupId)))
	if options != nil {
		if options.SkipInitialScan {
			_cmd = cmds.Incomplete(cmds.FtSynupdateSynonymGroupId(_cmd).Skipinitialscan())
		}
	}
	cmd := cmds.FtSynupdateSynonymGroupId(_cmd).Term(argsToSlice(terms)...).Build()
	return newStatusCmd(c.client.Do(ctx, cmd))
}

// FTTagVals - Returns all distinct values indexed in a tag field.
// The 'index' parameter specifies the index to check, and the 'field' parameter specifies the tag field to retrieve values from.
// For more information, please refer to the Redis documentation:
// [FT.TAGVALS]: (https://redis.io/commands/ft.tagvals/)
// see go-redis v9.7.0 https://github.com/redis/go-redis/blob/v9.7.0/search_commands.go#L2024
func (c *Compat) FTTagVals(ctx context.Context, index string, field string) *StringSliceCmd {
	cmd := c.client.B().FtTagvals().Index(index).FieldName(field).Build()
	return newStringSliceCmd(c.client.Do(ctx, cmd))
}

func (c CacheCompat) BitCount(ctx context.Context, key string, bitCount *BitCount) *IntCmd {
	var resp rueidis.RedisResult
	if bitCount == nil {
		resp = c.client.DoCache(ctx, c.client.B().Bitcount().Key(key).Cache(), c.ttl)
		return newIntCmd(resp)
	}

	if bitCount.Unit == "" {
		resp = c.client.DoCache(ctx, c.client.B().Bitcount().Key(key).Start(bitCount.Start).End(bitCount.End).Cache(), c.ttl)
		return newIntCmd(resp)
	}

	switch bitCount.Unit {
	case BitCountIndexByte:
		resp = c.client.DoCache(ctx, c.client.B().Bitcount().Key(key).Start(bitCount.Start).End(bitCount.End).Byte().Cache(), c.ttl)
	case BitCountIndexBit:
		resp = c.client.DoCache(ctx, c.client.B().Bitcount().Key(key).Start(bitCount.Start).End(bitCount.End).Bit().Cache(), c.ttl)
	}
	return newIntCmd(resp)
}

func (c CacheCompat) BitPos(ctx context.Context, key string, bit int64, pos ...int64) *IntCmd {
	var resp rueidis.RedisResult
	switch len(pos) {
	case 0:
		resp = c.client.DoCache(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Cache(), c.ttl)
	case 1:
		resp = c.client.DoCache(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(pos[0]).Cache(), c.ttl)
	case 2:
		resp = c.client.DoCache(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(pos[0]).End(pos[1]).Cache(), c.ttl)
	default:
		panic("too many arguments")
	}
	return newIntCmd(resp)
}

func (c CacheCompat) BitPosSpan(ctx context.Context, key string, bit, start, end int64, span string) *IntCmd {
	var resp rueidis.RedisResult
	if strings.ToLower(span) == "bit" {
		resp = c.client.DoCache(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(start).End(end).Bit().Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Bitpos().Key(key).Bit(bit).Start(start).End(end).Byte().Cache(), c.ttl)
	}
	return newIntCmd(resp)
}

func (c CacheCompat) EvalRO(ctx context.Context, script string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().EvalRo().Script(script).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Cache()
	return newCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c CacheCompat) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().EvalshaRo().Sha1(sha1).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Cache()
	return newCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c CacheCompat) FCallRO(ctx context.Context, function string, keys []string, args ...any) *Cmd {
	cmd := c.client.B().FcallRo().Function(function).Numkeys(int64(len(keys))).Key(keys...).Arg(argsToSlice(args)...).Cache()
	return newCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c CacheCompat) GeoDist(ctx context.Context, key, member1, member2, unit string) *FloatCmd {
	var resp rueidis.RedisResult
	switch strings.ToUpper(unit) {
	case "M":
		resp = c.client.DoCache(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).M().Cache(), c.ttl)
	case "MI":
		resp = c.client.DoCache(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).Mi().Cache(), c.ttl)
	case "FT":
		resp = c.client.DoCache(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).Ft().Cache(), c.ttl)
	case "KM", "":
		resp = c.client.DoCache(ctx, c.client.B().Geodist().Key(key).Member1(member1).Member2(member2).Km().Cache(), c.ttl)
	default:
		panic(fmt.Sprintf("invalid unit %s", unit))
	}
	return newFloatCmd(resp)
}

func (c CacheCompat) GeoHash(ctx context.Context, key string, members ...string) *StringSliceCmd {
	cmd := c.client.B().Geohash().Key(key).Member(members...).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) GeoPos(ctx context.Context, key string, members ...string) *GeoPosCmd {
	cmd := c.client.B().Geopos().Key(key).Member(members...).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newGeoPosCmd(resp)
}

// GeoRadius is a read-only GEORADIUS_RO command.
func (c CacheCompat) GeoRadius(ctx context.Context, key string, longitude, latitude float64, query GeoRadiusQuery) *GeoLocationCmd {
	cmd := c.client.B().Arbitrary("GEORADIUS_RO").Keys(key).Args(strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	if query.Store != "" || query.StoreDist != "" {
		panic("GeoRadius does not support Store or StoreDist")
	}
	cmd = cmd.Args(query.args()...)
	resp := c.client.DoCache(ctx, rueidis.Cacheable(cmd.Build()), c.ttl)
	return newGeoLocationCmd(resp)
}

// GeoRadiusByMember is a read-only GEORADIUSBYMEMBER_RO command.
func (c CacheCompat) GeoRadiusByMember(ctx context.Context, key, member string, query GeoRadiusQuery) *GeoLocationCmd {
	cmd := c.client.B().Arbitrary("GEORADIUSBYMEMBER_RO").Keys(key).Args(member)
	if query.Store != "" || query.StoreDist != "" {
		panic("GeoRadiusByMember does not support Store or StoreDist")
	}
	cmd = cmd.Args(query.args()...)
	resp := c.client.DoCache(ctx, rueidis.Cacheable(cmd.Build()), c.ttl)
	return newGeoLocationCmd(resp)
}

func (c CacheCompat) GeoSearch(ctx context.Context, key string, q GeoSearchQuery) *StringSliceCmd {
	cmd := c.client.B().Arbitrary("GEOSEARCH").Keys(key)
	cmd = cmd.Args(q.args()...)
	resp := c.client.DoCache(ctx, rueidis.Cacheable(cmd.Build()), c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) Get(ctx context.Context, key string) *StringCmd {
	cmd := c.client.B().Get().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringCmd(resp)
}

func (c CacheCompat) GetBit(ctx context.Context, key string, offset int64) *IntCmd {
	cmd := c.client.B().Getbit().Key(key).Offset(offset).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) GetRange(ctx context.Context, key string, start, end int64) *StringCmd {
	cmd := c.client.B().Getrange().Key(key).Start(start).End(end).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringCmd(resp)
}

func (c CacheCompat) HExists(ctx context.Context, key, field string) *BoolCmd {
	cmd := c.client.B().Hexists().Key(key).Field(field).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBoolCmd(resp)
}

func (c CacheCompat) HGet(ctx context.Context, key, field string) *StringCmd {
	cmd := c.client.B().Hget().Key(key).Field(field).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringCmd(resp)
}

func (c CacheCompat) HGetAll(ctx context.Context, key string) *StringStringMapCmd {
	cmd := c.client.B().Hgetall().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringStringMapCmd(resp)
}

func (c CacheCompat) HKeys(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().Hkeys().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) HLen(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Hlen().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) HMGet(ctx context.Context, key string, fields ...string) *SliceCmd {
	cmd := c.client.B().Hmget().Key(key).Field(fields...).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newSliceCmd(resp, false, fields...)
}

func (c CacheCompat) HVals(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().Hvals().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) LIndex(ctx context.Context, key string, index int64) *StringCmd {
	cmd := c.client.B().Lindex().Key(key).Index(index).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringCmd(resp)
}

func (c CacheCompat) LLen(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Llen().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) LPos(ctx context.Context, key string, element string, a LPosArgs) *IntCmd {
	cmd := c.client.B().Arbitrary("LPOS").Keys(key).Args(element)
	if a.Rank != 0 {
		cmd = cmd.Args("RANK", strconv.FormatInt(a.Rank, 10))
	}
	if a.MaxLen != 0 {
		cmd = cmd.Args("MAXLEN", strconv.FormatInt(a.MaxLen, 10))
	}
	resp := c.client.DoCache(ctx, rueidis.Cacheable(cmd.Build()), c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) LRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	cmd := c.client.B().Lrange().Key(key).Start(start).Stop(stop).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) PTTL(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().Pttl().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newDurationCmd(resp, time.Millisecond)
}

func (c CacheCompat) SCard(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Scard().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) SIsMember(ctx context.Context, key string, member any) *BoolCmd {
	cmd := c.client.B().Sismember().Key(key).Member(str(member)).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBoolCmd(resp)
}

func (c CacheCompat) SMIsMember(ctx context.Context, key string, members ...any) *BoolSliceCmd {
	cmd := c.client.B().Smismember().Key(key).Member(argsToSlice(members)...).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBoolSliceCmd(resp)
}

func (c CacheCompat) SMembers(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().Smembers().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) SortRO(ctx context.Context, key string, sort Sort) *StringSliceCmd {
	cmd := c.client.B().Arbitrary("SORT_RO").Keys(key)
	if sort.By != "" {
		cmd = cmd.Args("BY", sort.By)
	}
	if sort.Offset != 0 || sort.Count != 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(sort.Offset, 10), strconv.FormatInt(sort.Count, 10))
	}
	for _, g := range sort.Get {
		cmd = cmd.Args("GET").Args(g)
	}
	switch order := strings.ToUpper(sort.Order); order {
	case "ASC", "DESC":
		cmd = cmd.Args(order)
	case "":
	default:
		panic(fmt.Sprintf("invalid sort order %s", sort.Order))
	}
	if sort.Alpha {
		cmd = cmd.Args("ALPHA")
	}
	resp := c.client.DoCache(ctx, rueidis.Cacheable(cmd.Build()), c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) StrLen(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Strlen().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) TTL(ctx context.Context, key string) *DurationCmd {
	cmd := c.client.B().Ttl().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newDurationCmd(resp, time.Second)
}

func (c CacheCompat) Type(ctx context.Context, key string) *StatusCmd {
	cmd := c.client.B().Type().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStatusCmd(resp)
}

func (c CacheCompat) ZCard(ctx context.Context, key string) *IntCmd {
	cmd := c.client.B().Zcard().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) ZCount(ctx context.Context, key, min, max string) *IntCmd {
	cmd := c.client.B().Zcount().Key(key).Min(min).Max(max).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) ZLexCount(ctx context.Context, key, min, max string) *IntCmd {
	cmd := c.client.B().Zlexcount().Key(key).Min(min).Max(max).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) ZMScore(ctx context.Context, key string, members ...string) *FloatSliceCmd {
	cmd := c.client.B().Zmscore().Key(key).Member(members...).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newFloatSliceCmd(resp)
}

func (c CacheCompat) zRangeArgs(withScores bool, z ZRangeArgs) rueidis.Cacheable {
	cmd := c.client.B().Arbitrary("ZRANGE").Keys(z.Key)
	if z.Rev && (z.ByScore || z.ByLex) {
		cmd = cmd.Args(str(z.Stop), str(z.Start))
	} else {
		cmd = cmd.Args(str(z.Start), str(z.Stop))
	}
	if z.ByScore {
		cmd = cmd.Args("BYSCORE")
	} else if z.ByLex {
		cmd = cmd.Args("BYLEX")
	}
	if z.Rev {
		cmd = cmd.Args("REV")
	}
	if z.Offset != 0 || z.Count != 0 {
		cmd = cmd.Args("LIMIT", strconv.FormatInt(z.Offset, 10), strconv.FormatInt(z.Count, 10))
	}
	if withScores {
		cmd = cmd.Args("WITHSCORES")
	}
	return rueidis.Cacheable(cmd.Build())
}

func (c CacheCompat) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd {
	cmd := c.zRangeArgs(true, ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  stop,
	})
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newZSliceCmd(resp)
}

func (c CacheCompat) ZRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.DoCache(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Limit(opt.Offset, opt.Count).Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Cache(), c.ttl)
	}
	return newStringSliceCmd(resp)
}

func (c CacheCompat) ZRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.DoCache(ctx, c.client.B().Zrangebylex().Key(key).Min(opt.Min).Max(opt.Max).Limit(opt.Offset, opt.Count).Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Zrangebylex().Key(key).Min(opt.Min).Max(opt.Max).Cache(), c.ttl)
	}
	return newStringSliceCmd(resp)
}

func (c CacheCompat) ZRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.DoCache(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Withscores().Limit(opt.Offset, opt.Count).Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Zrangebyscore().Key(key).Min(opt.Min).Max(opt.Max).Withscores().Cache(), c.ttl)
	}
	return newZSliceCmd(resp)
}

func (c CacheCompat) ZRangeArgs(ctx context.Context, z ZRangeArgs) *StringSliceCmd {
	cmd := c.zRangeArgs(false, z)
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) ZRangeArgsWithScores(ctx context.Context, z ZRangeArgs) *ZSliceCmd {
	cmd := c.zRangeArgs(true, z)
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newZSliceCmd(resp)
}

func (c CacheCompat) ZRank(ctx context.Context, key, member string) *IntCmd {
	cmd := c.client.B().Zrank().Key(key).Member(member).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) ZRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd {
	cmd := c.client.B().Zrank().Key(key).Member(member).Withscore().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newRankWithScoreCmd(resp)
}

func (c CacheCompat) ZRevRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	cmd := c.client.B().Zrevrange().Key(key).Start(start).Stop(stop).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newStringSliceCmd(resp)
}

func (c CacheCompat) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *ZSliceCmd {
	cmd := c.client.B().Zrevrange().Key(key).Start(start).Stop(stop).Withscores().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newZSliceCmd(resp)
}

func (c CacheCompat) ZRevRangeByScore(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.DoCache(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Limit(opt.Offset, opt.Count).Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Cache(), c.ttl)
	}
	return newStringSliceCmd(resp)
}

func (c CacheCompat) ZRevRangeByLex(ctx context.Context, key string, opt ZRangeBy) *StringSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.DoCache(ctx, c.client.B().Zrevrangebylex().Key(key).Max(opt.Max).Min(opt.Min).Limit(opt.Offset, opt.Count).Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Zrevrangebylex().Key(key).Max(opt.Max).Min(opt.Min).Cache(), c.ttl)
	}
	return newStringSliceCmd(resp)
}

func (c CacheCompat) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt ZRangeBy) *ZSliceCmd {
	var resp rueidis.RedisResult
	if opt.Offset != 0 || opt.Count != 0 {
		resp = c.client.DoCache(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Withscores().Limit(opt.Offset, opt.Count).Cache(), c.ttl)
	} else {
		resp = c.client.DoCache(ctx, c.client.B().Zrevrangebyscore().Key(key).Max(opt.Max).Min(opt.Min).Withscores().Cache(), c.ttl)
	}
	return newZSliceCmd(resp)
}

func (c CacheCompat) ZRevRank(ctx context.Context, key, member string) *IntCmd {
	cmd := c.client.B().Zrevrank().Key(key).Member(member).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newIntCmd(resp)
}

func (c CacheCompat) ZRevRankWithScore(ctx context.Context, key, member string) *RankWithScoreCmd {
	cmd := c.client.B().Zrevrank().Key(key).Member(member).Withscore().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newRankWithScoreCmd(resp)
}

func (c CacheCompat) ZScore(ctx context.Context, key, member string) *FloatCmd {
	cmd := c.client.B().Zscore().Key(key).Member(member).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newFloatCmd(resp)
}

func (c CacheCompat) BFExists(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().BfExists().Key(key).Item(str(element)).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBoolCmd(resp)
}

func (c CacheCompat) BFInfo(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBFInfoCmd(resp)
}

func (c CacheCompat) BFInfoArg(ctx context.Context, key, option string) *BFInfoCmd {
	switch option {
	case "CAPACITY":
		return c.BFInfoCapacity(ctx, key)
	case "SIZE":
		return c.BFInfoSize(ctx, key)
	case "FILTERS":
		return c.BFInfoFilters(ctx, key)
	case "ITEMS":
		return c.BFInfoItems(ctx, key)
	case "EXPANSION":
		return c.BFInfoExpansion(ctx, key)
	default:
		panic(fmt.Sprintf("unknown option %v", option))
	}
}

func (c CacheCompat) BFInfoCapacity(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Capacity().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBFInfoCmd(resp)
}

func (c CacheCompat) BFInfoSize(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Size().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBFInfoCmd(resp)
}

func (c CacheCompat) BFInfoFilters(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Filters().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBFInfoCmd(resp)
}

func (c CacheCompat) BFInfoItems(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Items().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBFInfoCmd(resp)
}

func (c CacheCompat) BFInfoExpansion(ctx context.Context, key string) *BFInfoCmd {
	cmd := c.client.B().BfInfo().Key(key).Expansion().Cache()
	resp := c.client.DoCache(ctx, cmd, c.ttl)
	return newBFInfoCmd(resp)
}

func (c *CacheCompat) CFCount(ctx context.Context, key string, element interface{}) *IntCmd {
	cmd := c.client.B().CfCount().Key(key).Item(str(element)).Cache()
	return newIntCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) CFExists(ctx context.Context, key string, element interface{}) *BoolCmd {
	cmd := c.client.B().CfExists().Key(key).Item(str(element)).Cache()
	return newBoolCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) CFInfo(ctx context.Context, key string) *CFInfoCmd {
	cmd := c.client.B().CfInfo().Key(key).Cache()
	return newCFInfoCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) CMSInfo(ctx context.Context, key string) *CMSInfoCmd {
	cmd := c.client.B().CmsInfo().Key(key).Cache()
	return newCMSInfoCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) CMSQuery(ctx context.Context, key string, elements ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().CmsQuery().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.CmsQueryItem)(_cmd).Cache()
	return newIntSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) TopKInfo(ctx context.Context, key string) *TopKInfoCmd {
	cmd := c.client.B().TopkInfo().Key(key).Cache()
	return newTopKInfoCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) TopKList(ctx context.Context, key string) *StringSliceCmd {
	cmd := c.client.B().TopkList().Key(key).Cache()
	return newStringSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) TopKQuery(ctx context.Context, key string, elements ...interface{}) *BoolSliceCmd {
	_cmd := c.client.B().TopkQuery().Key(key)
	for _, e := range elements {
		_cmd.Item(str(e))
	}
	cmd := (cmds.TopkQueryItem)(_cmd).Cache()
	return newBoolSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONArrIndex(ctx context.Context, key, path string, value ...interface{}) *IntSliceCmd {
	_cmd := c.client.B().JsonArrindex().Key(key).Path(path)
	switch len(value) {
	case 1:
		// format: value
		_cmd.Value(str(value[0]))
	case 2:
		// format: value start
		_cmd.Value(str(value[0])).Start((int64)(value[1].(int)))
	case 3:
		// format: value start stop
		_cmd.Value(str(value[0])).Start((int64)(value[1].(int))).Stop((int64)(value[2].(int)))
	default:
		panic(fmt.Sprintf("the format of value should be value [start [stop]], got %v", value))
	}
	cmd := (cmds.JsonArrindexValue)(_cmd).Cache()
	return newIntSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONArrLen(ctx context.Context, key, path string) *IntSliceCmd {
	cmd := c.client.B().JsonArrlen().Key(key).Path(path).Cache()
	return newIntSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONGet(ctx context.Context, key string, paths ...string) *JSONCmd {
	cmd := c.client.B().JsonGet().Key(key).Path(paths...).Cache()
	return newJSONCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONMGet(ctx context.Context, path string, keys ...string) *JSONSliceCmd {
	cmd := c.client.B().JsonMget().Key(keys...).Path(path).Cache()
	return newJSONSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONObjKeys(ctx context.Context, key, path string) *SliceCmd {
	cmd := c.client.B().JsonObjkeys().Key(key).Path(path).Cache()
	return newSliceCmd(c.client.DoCache(ctx, cmd, c.ttl), true)
}

func (c *CacheCompat) JSONObjLen(ctx context.Context, key, path string) *IntPointerSliceCmd {
	cmd := c.client.B().JsonObjlen().Key(key).Path(path).Cache()
	return newIntPointerSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONStrLen(ctx context.Context, key, path string) *IntPointerSliceCmd {
	cmd := c.client.B().JsonStrlen().Key(key).Path(path).Cache()
	return newIntPointerSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func (c *CacheCompat) JSONType(ctx context.Context, key, path string) *JSONSliceCmd {
	cmd := c.client.B().JsonType().Key(key).Path(path).Cache()
	return newJSONSliceCmd(c.client.DoCache(ctx, cmd, c.ttl))
}

func str(arg any) string {
	if arg == nil {
		return ""
	}
	switch v := arg.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case time.Duration:
		return strconv.FormatInt(v.Nanoseconds(), 10)
	case encoding.BinaryMarshaler:
		if data, err := v.MarshalBinary(); err == nil {
			return rueidis.BinaryString(data)
		}
	}
	return fmt.Sprint(arg)
}

func argsToSlice(src []any) []string {
	if len(src) == 1 {
		return argToSlice(src[0])
	}
	dst := make([]string, 0, len(src))
	for _, v := range src {
		dst = append(dst, str(v))
	}
	return dst
}

func argToSlice(arg any) []string {
	switch arg := arg.(type) {
	case []string:
		return arg
	case []any:
		dst := make([]string, 0, len(arg))
		for _, v := range arg {
			dst = append(dst, str(v))
		}
		return dst
	case map[string]any:
		dst := make([]string, 0, len(arg))
		for k, v := range arg {
			dst = append(dst, k, str(v))
		}
		return dst
	case map[string]string:
		dst := make([]string, 0, len(arg))
		for k, v := range arg {
			dst = append(dst, k, v)
		}
		return dst
	default:
		// scan struct field
		v := reflect.ValueOf(arg)
		if v.Type().Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil
			}
			v = v.Elem()
		}
		if v.Type().Kind() == reflect.Struct {
			return appendStructField(v)
		}
		return []string{str(arg)}
	}
}

func appendStructField(v reflect.Value) []string {
	typ := v.Type()
	dst := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("redis")
		if tag == "" || tag == "-" {
			continue
		}
		name, opt, _ := strings.Cut(tag, ",")
		if name == "" {
			continue
		}

		field := v.Field(i)

		// miss field
		if omitEmpty(opt) && isEmptyValue(field) {
			continue
		}

		// if its a nil pointer
		if field.Kind() == reflect.Pointer && field.IsNil() {
			continue
		}

		// if its a valid pointer
		if field.Kind() == reflect.Pointer && field.Elem().CanInterface() {
			dst = append(dst, name, str(field.Elem().Interface()))
			continue
		}

		if field.CanInterface() {
			dst = append(dst, name, str(field.Interface()))
		}
	}

	return dst
}

func omitEmpty(opt string) bool {
	for opt != "" {
		var name string
		name, opt, _ = strings.Cut(opt, ",")
		if name == "omitempty" {
			return true
		}
	}
	return false
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
