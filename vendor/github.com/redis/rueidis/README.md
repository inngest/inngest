# rueidis

[![Go Reference](https://pkg.go.dev/badge/github.com/redis/rueidis.svg)](https://pkg.go.dev/github.com/redis/rueidis)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/redis/rueidis/tree/main.svg?style=shield)](https://dl.circleci.com/status-badge/redirect/gh/redis/rueidis/tree/main)
[![Go Report Card](https://goreportcard.com/badge/github.com/redis/rueidis)](https://goreportcard.com/report/github.com/redis/rueidis)
[![codecov](https://codecov.io/gh/redis/rueidis/branch/master/graph/badge.svg?token=wGTB8GdY06)](https://codecov.io/gh/redis/rueidis)

A fast Golang Redis client that does auto pipelining and supports client side caching.

## Features

* [Auto pipelining for non-blocking redis commands](https://redis.io/docs/manual/pipelining/)
* [Client Side Caching in RESP3](https://redis.io/docs/manual/client-side-caching/)
* Pub/Sub, Sharded Pub/Sub, Streams
* Redis Cluster, Sentinel, RedisJSON, RedisBloom, RediSearch, RedisTimeseries, etc.
* [Generic Object Mapping with client side caching](./om)
* [Cache-Aside Pattern with client side caching](./rueidisaside)
* [Distributed Locks with client side caching](./rueidislock)
* [Helpers for writing tests with rueidis mock](./mock)
* [OpenTelemetry integration](./rueidisotel)
* [Hooks and other integrations](./rueidishook)
* [Go-redis like API adapter](./rueidiscompat) by [@418Coffee](https://github.com/418Coffee)

## Getting Started

```golang
package main

import (
	"context"
	"github.com/redis/rueidis"
)

func main() {
	client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{"127.0.0.1:6379"}})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx := context.Background()
	// SET key val NX
	err = client.Do(ctx, client.B().Set().Key("key").Value("val").Nx().Build()).Error()
	// HGETALL hm
	hm, err := client.Do(ctx, client.B().Hgetall().Key("hm").Build()).AsStrMap()
}
```

Checkout more examples: [Command Response Cheatsheet](https://github.com/redis/rueidis#command-response-cheatsheet)

## Developer Friendly Command Builder

`client.B()` is the builder entrypoint to construct a redis command:

![Developer friendly command builder](https://user-images.githubusercontent.com/2727535/209358313-39000aee-eaa4-42e1-9748-0d3836c1264f.gif)\
<sub>_Recorded by @FZambia [Improving Centrifugo Redis Engine throughput and allocation efficiency with Rueidis Go library
](https://centrifugal.dev/blog/2022/12/20/improving-redis-engine-performance)_</sub>

Once a command is built, use either `client.Do()` or `client.DoMulti()` to send it to redis.

**Constructed commands will be recycled to underlying `sync.Pool` by default and you ❗️SHOULD NOT❗️ reuse them across multiple `client.Do()` or `client.DoMulti()` calls.**
To reuse a command, use `Pin()` after `Build()` and it will prevent the command being recycled. 

## [Auto Pipelining](https://redis.io/docs/manual/pipelining/)

All concurrent non-blocking redis commands (such as `GET`, `SET`) are automatically pipelined through connections,
which reduces the overall round trips and system calls, and gets higher throughput. You can easily get the benefit
of [pipelining technique](https://redis.io/docs/manual/pipelining/) by just calling `client.Do()` from multiple goroutines.
For example:

```go
func BenchmarkPipelining(b *testing.B, client rueidis.Client) {
	// the below client.Do() operations will be issued from
	// multiple goroutines and thus will be pipelined automatically.
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client.Do(context.Background(), client.B().Get().Key("k").Build()).ToString()
		}
	})
}
```

### Benchmark comparison with go-redis v9

Comparing to go-redis, Rueidis has higher throughput across 1, 8, and 64 parallelism settings.

It is even able to achieve ~14x throughput over go-redis in a local benchmark of Macbook Pro 16" M1 Pro 2021. (see `parallelism(64)-key(16)-value(64)-10`)

![client_test_set](https://github.com/rueian/rueidis-benchmark/blob/master/client_test_set_10.png)

Benchmark source code: https://github.com/rueian/rueidis-benchmark

A benchmark result performed on two GCP n2-highcpu-2 machines also shows that rueidis can achieve higher throughput with lower latencies: https://github.com/redis/rueidis/pull/93

### Pipelining Bulk Operations Manually

Though all concurrent non-blocking commands are automatically pipelined, you can still pipeline commands manually with `DoMulti()`:

``` golang
cmds := make(rueidis.Commands, 0, 10)
for i := 0; i < 10; i++ {
    cmds = append(cmds, client.B().Set().Key("key").Value("value").Build())
}
for _, resp := range client.DoMulti(ctx, cmds...) {
    if err := resp.Error(); err != nil {
        panic(err)
    }
}
```

## [Client Side Caching](https://redis.io/docs/manual/client-side-caching/)

The opt-in mode of [server-assisted client side caching](https://redis.io/docs/manual/client-side-caching/) is enabled by default, and can be used by calling `DoCache()` or `DoMultiCache()` with
pairs of a readonly command and a client side TTL.

```golang
client.DoCache(ctx, client.B().Hmget().Key("myhash").Field("1", "2").Cache(), time.Minute).ToArray()
client.DoMultiCache(ctx,
    rueidis.CT(client.B().Get().Key("k1").Cache(), 1*time.Minute),
    rueidis.CT(client.B().Get().Key("k2").Cache(), 2*time.Minute))
```

Cached responses will be invalidated when being notified by redis or their client side ttl is reached.

### Benchmark

Client Side Caching can boost read throughput just like **having a redis replica right inside your application**:

![client_test_get](https://github.com/rueian/rueidis-benchmark/blob/master/client_test_get_10.png)

Benchmark source code: https://github.com/rueian/rueidis-benchmark

### Client Side Caching Helpers

Use `CacheTTL()` to check the remaining client side TTL in seconds:

```golang
client.DoCache(ctx, client.B().Get().Key("k1").Cache(), time.Minute).CacheTTL() == 60
```

Use `IsCacheHit()` to verify that if the response came from the client side memory:

```golang
client.DoCache(ctx, client.B().Get().Key("k1").Cache(), time.Minute).IsCacheHit() == true
```

If the OpenTelemetry is enabled by the `rueidisotel.WithClient(client)`, then there are also two metrics instrumented:
* rueidis_do_cache_miss
* rueidis_do_cache_hits

### MGET/JSON.MGET Client Side Caching Helpers

`rueidis.MGetCache` and `rueidis.JsonMGetCache` are handy helpers fetching multiple keys across different slots through the client side caching.
They will first group keys by slot to build `MGET` or `JSON.MGET` commands respectively and then send requests with only cache missed keys to redis nodes.

### Broadcast Mode Client Side Caching

Although the default is opt-in mode, you can use broadcast mode by specifying your prefixes in `ClientOption.ClientTrackingOptions`:

```go
client, err := rueidis.NewClient(rueidis.ClientOption{
	InitAddress:           []string{"127.0.0.1:6379"},
	ClientTrackingOptions: []string{"PREFIX", "prefix1:", "PREFIX", "prefix2:", "BCAST"},
})
if err != nil {
	panic(err)
}
client.DoCache(ctx, client.B().Get().Key("prefix1:1").Cache(), time.Minute).IsCacheHit() == false
client.DoCache(ctx, client.B().Get().Key("prefix1:1").Cache(), time.Minute).IsCacheHit() == true
```

Please make sure that commands passed to `DoCache()` and `DoMultiCache()` are covered by your prefixes.
Otherwise, their client-side cache will not be invalidated by redis.

### Disable Client Side Caching

Some Redis provider doesn't support client-side caching, ex. Google Cloud Memorystore.
You can disable client-side caching by setting `ClientOption.DisableCache` to `true`.
This will also fall back `client.DoCache()` and `client.DoMultiCache()` to `client.Do()` and `client.DoMulti()`.

## Context Cancellation

`client.Do()`, `client.DoMulti()`, `client.DoCache()` and `client.DoMultiCache()` can return early if the context is canceled or the deadline is reached.

```golang
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
client.Do(ctx, client.B().Set().Key("key").Value("val").Nx().Build()).Error() == context.DeadlineExceeded
```

Please note that though operations can return early, the command is likely sent already.

## Pub/Sub

To receive messages from channels, `client.Receive()` should be used. It supports `SUBSCRIBE`, `PSUBSCRIBE` and Redis 7.0's `SSUBSCRIBE`:

```golang
err = client.Receive(context.Background(), client.B().Subscribe().Channel("ch1", "ch2").Build(), func(msg rueidis.PubSubMessage) {
    // handle the msg
})
```

The provided handler will be called with received message.

It is important to note that `client.Receive()` will keep blocking and return only when the following cases:
1. return `nil` when received any unsubscribe/punsubscribe message related to the provided `subscribe` command.
2. return `rueidis.ErrClosing` when the client is closed manually.
3. return `ctx.Err()` when the `ctx` is done.
4. return non-nil `err` when the provided `subscribe` command failed.

While the `client.Receive()` call is blocking, the `Client` is still able to accept other concurrent requests,
and they are sharing the same tcp connection. If your message handler may take some time to complete, it is recommended
to use the `client.Receive()` inside a `client.Dedicated()` for not blocking other concurrent requests.

### Alternative PubSub Hooks

The `client.Receive()` requires users to provide a subscription command in advance.
There is an alternative `Dedicatedclient.SetPubSubHooks()` allows users to subscribe/unsubscribe channels later.

```golang
c, cancel := client.Dedicate()
defer cancel()

wait := c.SetPubSubHooks(rueidis.PubSubHooks{
	OnMessage: func(m rueidis.PubSubMessage) {
		// Handle message. This callback will be called sequentially, but in another goroutine.
	}
})
c.Do(ctx, c.B().Subscribe().Channel("ch").Build())
err := <-wait // disconnected with err
```

If the hooks are not nil, the above `wait` channel is guaranteed to be close when the hooks will not be called anymore,
and produce at most one error describing the reason. Users can use this channel to detect disconnection.

## CAS Pattern

To do a CAS operation (`WATCH` + `MULTI` + `EXEC`), a dedicated connection should be used, because there should be no
unintentional write commands between `WATCH` and `EXEC`. Otherwise, the `EXEC` may not fail as expected.

```golang
client.Dedicated(func(c rueidis.DedicatedClient) error {
    // watch keys first
    c.Do(ctx, c.B().Watch().Key("k1", "k2").Build())
    // perform read here
    c.Do(ctx, c.B().Mget().Key("k1", "k2").Build())
    // perform write with MULTI EXEC
    c.DoMulti(
        ctx,
        c.B().Multi().Build(),
        c.B().Set().Key("k1").Value("1").Build(),
        c.B().Set().Key("k2").Value("2").Build(),
        c.B().Exec().Build(),
    )
    return nil
})

```

Or use `Dedicate()` and invoke `cancel()` when finished to put the connection back to the pool.

``` golang
c, cancel := client.Dedicate()
defer cancel()

c.Do(ctx, c.B().Watch().Key("k1", "k2").Build())
// do the rest CAS operations with the `client` who occupying a connection 
```

However, occupying a connection is not good in terms of throughput. It is better to use Lua script to perform
optimistic locking instead.

## Memory Consumption Consideration

Each underlying connection in rueidis allocates a ring buffer for pipelining.
Its size is controlled by the `ClientOption.RingScaleEachConn` and the default value is 10 which results into each ring of size 2^10.

If you have many rueidis connections, you may find that they occupy quite amount of memory.
In that case, you may consider reducing `ClientOption.RingScaleEachConn` to 8 or 9 at the cost of potential throughput degradation.

## Lua Script

The `NewLuaScript` or `NewLuaScriptReadOnly` will create a script which is safe for concurrent usage.

When calling the `script.Exec`, it will try sending `EVALSHA` first and fallback to `EVAL` if the server returns `NOSCRIPT`.

```golang
script := rueidis.NewLuaScript("return {KEYS[1],KEYS[2],ARGV[1],ARGV[2]}")
// the script.Exec is safe for concurrent call
list, err := script.Exec(ctx, client, []string{"k1", "k2"}, []string{"a1", "a2"}).ToArray()
```

## Redis Cluster, Single Redis and Sentinel

To connect to a redis cluster, the `NewClient` should be used:

```golang
client, err := rueidis.NewClient(rueidis.ClientOption{
    InitAddress: []string{"127.0.0.1:7001", "127.0.0.1:7002", "127.0.0.1:7003"},
    ShuffleInit: true,
})
```

To connect to a single redis node, still use the `NewClient` with one InitAddress

```golang
client, err := rueidis.NewClient(rueidis.ClientOption{
    InitAddress: []string{"127.0.0.1:6379"},
})
```

To connect to sentinels, specify the required master set name:

```golang
client, err := rueidis.NewClient(rueidis.ClientOption{
    InitAddress: []string{"127.0.0.1:26379", "127.0.0.1:26380", "127.0.0.1:26381"},
    Sentinel: rueidis.SentinelOption{
        MasterSet: "my_master",
    },
})
```

## Arbitrary Command

If you want to construct commands that are absent from the command builder, you can use `client.B().Arbitrary()`:

```golang
// This will result into [ANY CMD k1 k2 a1 a2]
client.B().Arbitrary("ANY", "CMD").Keys("k1", "k2").Args("a1", "a2").Build()
```

## Working with JSON, Raw `[]byte`, and Vector Similarity Search

The command builder treats all the parameters as Redis strings, which are binary safe. This means that users can store `[]byte`
directly into Redis without conversion. And the `rueidis.BinaryString` helper can convert `[]byte` to `string` without copy. For example:

```golang
client.B().Set().Key("b").Value(rueidis.BinaryString([]byte{...})).Build()
```

Treating all the parameters as Redis strings also means that the command builder doesn't do any quoting, conversion automatically for users.

When working with RedisJSON, users frequently need to prepare JSON string in Redis string. And `rueidis.JSON` can help:

```golang
client.B().JsonSet().Key("j").Path("$.myStrField").Value(rueidis.JSON("str")).Build()
// equivalent to
client.B().JsonSet().Key("j").Path("$.myStrField").Value(`"str"`).Build()
```

When working with vector similarity search, users can use `rueidis.VectorString32` and `rueidis.VectorString64` to build queries:

```golang
cmd := client.B().FtSearch().Index("idx").Query("*=>[KNN 5 @vec $V]").
    Params().Nargs(2).NameValue().NameValue("V", rueidis.VectorString64([]float64{...})).
    Dialect(2).Build()
n, resp, err := client.Do(ctx, cmd).AsFtSearch()
```

## Command Response Cheatsheet

It is hard to remember what message type is returned from redis and which parsing method should be used with. So, here is some common examples:

```golang
// GET
client.Do(ctx, client.B().Get().Key("k").Build()).ToString()
client.Do(ctx, client.B().Get().Key("k").Build()).AsInt64()
// MGET
client.Do(ctx, client.B().Mget().Key("k1", "k2").Build()).ToArray()
// SET
client.Do(ctx, client.B().Set().Key("k").Value("v").Build()).Error()
// INCR
client.Do(ctx, client.B().Incr().Key("k").Build()).AsInt64()
// HGET
client.Do(ctx, client.B().Hget().Key("k").Field("f").Build()).ToString()
// HMGET
client.Do(ctx, client.B().Hmget().Key("h").Field("a", "b").Build()).ToArray()
// HGETALL
client.Do(ctx, client.B().Hgetall().Key("h").Build()).AsStrMap()
// ZRANGE
client.Do(ctx, client.B().Zrange().Key("k").Min("1").Max("2").Build()).AsStrSlice()
// ZRANK
client.Do(ctx, client.B().Zrank().Key("k").Member("m").Build()).AsInt64()
// ZSCORE
client.Do(ctx, client.B().Zscore().Key("k").Member("m").Build()).AsFloat64()
// ZRANGE
client.Do(ctx, client.B().Zrange().Key("k").Min("0").Max("-1").Build()).AsStrSlice()
client.Do(ctx, client.B().Zrange().Key("k").Min("0").Max("-1").Withscores().Build()).AsZScores()
// ZPOPMIN
client.Do(ctx, client.B().Zpopmin().Key("k").Build()).AsZScore()
client.Do(ctx, client.B().Zpopmin().Key("myzset").Count(2).Build()).AsZScores()
// SCARD
client.Do(ctx, client.B().Scard().Key("k").Build()).AsInt64()
// SMEMBERS
client.Do(ctx, client.B().Smembers().Key("k").Build()).AsStrSlice()
// LINDEX
client.Do(ctx, client.B().Lindex().Key("k").Index(0).Build()).ToString()
// LPOP
client.Do(ctx, client.B().Lpop().Key("k").Build()).ToString()
client.Do(ctx, client.B().Lpop().Key("k").Count(2).Build()).AsStrSlice()
// FT.SEARCH
client.Do(ctx, client.B().FtSearch().Index("idx").Query("@f:v").Build()).AsFtSearch()
// GEOSEARCH
client.Do(ctx, client.B().Geosearch().Key("k").Fromlonlat(1, 1).Bybox(1).Height(1).Km().Build()).AsGeosearch()
```

## Supporting Go mod 1.18

To support the old Go 1.18 at least until Go 1.21 comes, there will be a special build tagged with `-go1.18` for each release.

Such releases remove `RedisResult.AsBytes()` and other related functionalities provided by later go versions.

```
# go.mod
module mymodule

go 1.18

require github.com/redis/rueidis v1.0.14-go1.18
```
