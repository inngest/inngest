# Parallel queue claiming

`inngest start` and `inngest dev` support an experimental
`--queue-claim-concurrency` setting. It limits simultaneous claim attempts within
each function partition on a server. The default is `1`, which keeps best-effort
FIFO dispatch. The value must be at least `1` and no greater than `--queue-workers`.

For a busy function whose queue claims are waiting on Redis or Valkey round trips,
start by trying `8`, alongside your existing server options:

```sh
inngest start --queue-workers 300 --queue-claim-concurrency 8
```

The same setting is available as `INNGEST_QUEUE_CLAIM_CONCURRENCY=8`, or as
`queue-claim-concurrency: 8` in a YAML configuration file. JSON configuration uses
the same key. Explicit CLI flags override environment variables, which override
configuration files. Use the same setting on each server sharing the queue.

This bounds claim attempts, not executing steps. Worker counts and function/key
concurrency limits still apply. Partition ownership, atomic item leases, retry
handling and step dependencies use the same paths. System queues keep their
existing claiming behavior.

Values above `1` relax best-effort dispatch order: later jobs in a fetched batch
can claim capacity before older or higher-priority jobs. A busy partition can take
workers faster and increase latency for other functions. More simultaneous claims
can also increase load on Redis/Valkey and the configuration database. This does
not provide exactly-once execution or change the need for idempotent side effects.

Compare throughput, queue delay, competing functions' latency, and Redis/Valkey
and database load with the setting enabled. Network latency alone does not predict
the gain; saturated workers, an overloaded database or a slow SDK may still limit
throughput. Increasing claim concurrency is not a substitute for those checks.

To restore best-effort FIFO claiming, set the value to `1` and restart each server.
No queue or database migration is needed. The setting only changes future claims;
it cannot undo earlier out-of-order dispatch.
