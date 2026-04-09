# Group defer

Starts a new function run pre-loaded with state from an existing run. Triggered by a `deferred.start` event.

## Event format

```json
{
  "name": "deferred.start",
  "data": {
    "fnSlug": "my-app-my-fn",
    "runId": "01ABC..."
  }
}
```

- `fnSlug` — slug of the function to start.
- `runId` — source run whose step state is copied into the new run.

## Behavior

1. The runner receives the event and looks up the function by `fnSlug`.
2. The executor loads the source run's step outputs and stack, then creates the new run's state with those steps pre-memoized.
3. The source run's original events are embedded into the stored `deferred.start` event as `data.event` and `data.events`, so the SDK can reconstruct the original trigger context.
4. The deferred handler runs synchronously before trigger matching to win the idempotency race against any trigger-matched initialization of the same function.

## Key files

- `pkg/consts/events.go` — `DeferredStartName` constant
- `pkg/execution/runner/runner.go` — `FindDeferredFunction`, `initializeDeferred`
- `pkg/execution/executor/copy_state.go` — `copyRunState`, `embedOriginalEvents`
- `pkg/execution/executor/copy_state_test.go` — integration tests
