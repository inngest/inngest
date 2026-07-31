# Shard failure isolation specification

## Goal

A temporarily or permanently unavailable queue shard must affect only callers
that use that shard. Healthy shards must remain usable, and the presence of an
unavailable shard must not prevent queue components from initializing.

This change does not remove unavailable shards from the configured topology.
Routing and lookup continue to reflect configuration; an operation routed to an
unavailable shard fails at runtime with the shard-specific error.

## Terminology

- **Configured shard:** A shard present in the configured topology.
- **Available shard:** A configured shard whose backend can currently perform
  the requested operation.
- **Unavailable shard:** A configured shard whose backend returns an error while
  an operation is running. Unavailability may be temporary or permanent. Queue
  shard clients are constructed lazily and do not establish backend connectivity
  during registry construction.
- **Configuration error:** An invalid topology or shard definition that cannot be
  interpreted safely, such as an empty shard name, duplicate/conflicting shard
  definition, missing selector, or invalid shard-group configuration. These are
  not availability failures and may remain fatal during initialization.

## Required behavior

### Topology and initialization

1. Registry initialization must retain every valid configured shard without
   probing backend health. An unavailable configured shard remains registered by
   name and group, and its backend error is returned from runtime operations.
2. Initialization work that accesses a backend, including shard-group lease
   acquisition, must log each unavailable shard and continue trying the remaining
   configured shards. The log must include:
   - shard name;
   - shard group, when configured;
   - the underlying error;
   - an attribute identifying the phase as shard initialization.
3. Initialization succeeds when the topology is structurally valid, even if
   some or all configured shards are unavailable. Structural configuration
   errors remain initialization errors.
4. Queue processor construction must not probe shard health. An explicitly
   configured primary shard may be unavailable without making construction
   fail; attempts to use it fail at runtime.
5. In shard-group mode, lease acquisition must treat a shard-specific lease
   error as a failure of that candidate, log it, and continue trying the other
   shards in the group. One unavailable candidate must not abort the claim
   cycle. If no candidate can be leased, the processor keeps retrying until its
   context is cancelled rather than terminating because of one shard's error.

### Direct lookup and routing

1. `ByName`, `ByGroup`, and `Resolve` operate on configured topology, not current
   health. They must continue to return configured unavailable shards.
2. A call routed to an unavailable shard returns an error from that shard at
   runtime. There is no implicit fallback to another shard because doing so
   could violate data placement and routing guarantees.
3. A failure using shard A must not mutate, remove, mark unavailable, or cancel
   work against shard B.

### `ShardRegistry.ForEach`

1. `ForEach` snapshots the configured shard set and invokes the callback once
   for every shard in that snapshot.
2. Callbacks may run concurrently, but each callback must receive a context
   derived only from the caller's context. An error from one callback must not
   cancel sibling callbacks.
3. `ForEach` waits for all callbacks to finish unless the caller's context is
   cancelled.
4. Every callback error is logged and suppressed. The log must include:
   - shard name;
   - the underlying error;
   - an attribute identifying the operation as shard iteration.
5. After all callbacks complete, `ForEach` returns `nil` regardless of how many
   shard callbacks failed. Successful results accumulated by callers are
   retained.
6. Cancellation originating from the caller is not a shard failure. If the
   caller's context is cancelled, `ForEach` may return `ctx.Err()` after all
   callbacks have exited. A shard callback error must never cause that
   cancellation.
7. A panic is not an availability error and is outside the error-suppression
   contract. The registry must reject nil shard entries during topology
   construction so nil entries cannot cause iteration panics.

## Observability

Errors are handled locally but must remain visible operationally. Logging an
error is required whenever:

- a configured shard cannot initialize;
- a shard-group lease attempt fails for a candidate; or
- a `ForEach` callback returns an error.

Logs should use the existing context logger and structured fields. Repeated
retry logs should use the repository's existing rate-limiting or sampling
conventions if necessary; suppression must not hide the shard name or root
error. Existing shard metrics should continue to be emitted, and a dedicated
availability/error counter may be added separately if needed.

## Acceptance tests

The implementation is complete when tests demonstrate all of the following:

1. With shards A, B, and C configured and B's backend unavailable, registry and
   queue processor construction succeed without probing B.
2. Backend initialization work involving B logs its failure with B's name and
   underlying error while continuing to other shards.
3. `ByName("B")` and `ByGroup` still include B; an operation on B returns its
   backend error at runtime.
4. Direct operations routed to A and C succeed despite B being unavailable.
5. `ForEach` invokes callbacks for A, B, and C when B returns an error.
6. B's callback error is logged, does not cancel A or C, and `ForEach` returns
   `nil` after the callbacks complete.
7. Multiple callback failures are each logged and do not change the successful
   results from other shards.
8. Caller cancellation is distinguishable from a shard callback error and does
   not leak goroutines.
9. Shard-group lease acquisition continues past an unavailable candidate and
   can claim a healthy shard later in the same cycle.
10. When every shard-group candidate is unavailable, lease acquisition retries
    until caller cancellation instead of terminating on the first candidate
    error.
11. Invalid topology, including a nil shard entry, remains a deterministic
    initialization error.

## Non-goals

- Automatically rerouting an operation to a different shard.
- Removing unavailable shards from configured topology.
- Defining a global health-check or circuit-breaker system.
- Hiding configuration errors that prevent the topology from being interpreted
  safely.
