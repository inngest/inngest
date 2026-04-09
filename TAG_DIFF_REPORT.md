# Tag Diff Report: v1.16.x – v1.17.x

Generated: 2026-04-09

Tags covered: v1.16.0 → v1.17.9

| From | To | Date Range | Commits | Files Changed | Insertions | Deletions |
|------|-----|------------|---------|---------------|------------|-----------|
| v1.16.0 | v1.16.1 | 2026-01-07 → 2026-01-21 | 46 | 228 | +22185 | -14225 |
| v1.16.1 | v1.16.2 | 2026-01-21 → 2026-02-06 | 48 | 462 | +30916 | -21526 |
| v1.16.2 | v1.16.3 | 2026-02-06 → 2026-02-09 | 7 | 25 | +942 | -76 |
| v1.16.3 | v1.17.0 | 2026-02-09 → 2026-02-11 | 9 | 95 | +10408 | -3199 |
| v1.17.0 | v1.17.1 | 2026-02-11 → 2026-02-18 | 25 | 181 | +10325 | -8521 |
| v1.17.1 | v1.17.2 | 2026-02-18 → 2026-03-04 | 65 | 183 | +5846 | -1090 |
| v1.17.2 | v1.17.2-beta.1 | 2026-03-04 → 2026-02-24 | 1 | 143 | +779 | -4139 |
| v1.17.2-beta.1 | v1.17.3 | 2026-02-24 → 2026-03-09 | 52 | 184 | +6388 | -1276 |
| v1.17.3 | v1.17.4 | 2026-03-09 → 2026-03-09 | 2 | 5 | +150 | -10 |
| v1.17.4 | v1.17.5 | 2026-03-09 → 2026-03-10 | 2 | 562 | +145901 | -50626 |
| v1.17.5 | v1.17.6 | 2026-03-10 → 2026-03-30 | 7 | 219 | +15058 | -2076 |
| v1.17.6 | v1.17.7 | 2026-03-30 → 2026-03-31 | 9 | 76 | +4175 | -1870 |
| v1.17.7 | v1.17.8 | 2026-03-31 → 2026-04-02 | 6 | 112 | +3382 | -9331 |
| v1.17.8 | v1.17.9 | 2026-04-02 → 2026-04-03 | 6 | 7 | +37 | -30 |

---

## v1.16.0 → v1.16.1

**Date range:** 2026-01-07 → 2026-01-21  
**Commits:** 46  
**Changes:**  228 files changed, 22185 insertions(+), 14225 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 2420 | `ui/apps/dev-server-ui/src/store/generated-types.ts` |
| 2310 | `pkg/execution/state/redis_state/queue.go` |
| 2159 | `pkg/execution/state/redis_state/queue_processor.go` |
| 2158 | `ui/apps/dashboard/src/routeTree.gen.ts` |
| 1301 | `pkg/execution/state/redis_state/queue_test.go` |
| 1077 | `pkg/execution/state/redis_state/shadow_queue.go` |
| 1017 | `pkg/execution/state/redis_state/queue_lease_test.go` |
| 1000 | `pkg/constraintapi/cache_test.go` |
| 875 | `pkg/execution/state/redis_state/backlog.go` |
| 784 | `pkg/execution/queue/option.go` |
| 754 | `pkg/execution/state/redis_state/shadow_queue_test.go` |
| 722 | `pkg/execution/state/redis_state/constraints_test.go` |
| 705 | `pkg/execution/queue/shadow_process.go` |
| 692 | `tests/execution/queue/queue_semaphore_test.go` |
| 677 | `pkg/constraintapi/constraints_test.go` |

### Commits

- 6f53b6ad4 EXE-1179: Upgrade tar version to latest (#3571)
- e99b06251 Add 2s retryAfter for concurrency limits in Acquire, make cache TTLs configurable (#3570)
- ca227e7b3 Add range picker option corresponding to account history limit (#3568)
- 13243a453 Improve histogram bucketing for Constraint API durations (#3564)
- 6a82d70c9 Tag source and limiting constraint identifier in metrics (#3561)
- 251443164 Tag source in ConstraintAPI Lua script duration metrics (#3560)
- 583fc6dd2 Improve queue semaphore (#3559)
- 45e29c657 Extract metrics middleware (#3553)
- a93890d6b Fix potential processor iterator race condition (#3550)
- 1f0de152d Add feature flag to limiting constraint cache (#3558)
- 6ee06a8a7 Add cache for limiting constraints (#3546)
- 4ab5cee37 expand events code editor to use more screen height (#3548)
- 18280af9d favicon fixes (#3547)
- 96e896571 Report ConstraintAPI request latency during Acquire (#3545)
- 04711023d Report Constraint Scavenger metrics by shard (#3544)
- 73bd450b9 Adding Tests For UpdateMetaData (#3543)
- b462ac6ce Revert "[SYS-566] State Store - updateMetadata - Lua 5.4 Support (#3528)" (#3541)
- 7b7784ca3 Constraint API pre-release fixes (#3531)
- a66cd8e4b making templates easier to add and remove (#3540)
- 040311a3d Remove support button from right sidebar (#3527)
- 0d7612ba4 Replace simpleJSONExtractString() with data.function_id in Insights (#3536)
- dab4dbae3 fix constraint api not releasing semaphore when item is constrained (#3537)
- c02e4cadf Update pause idempotency error handling (#3534)
- 1a55af2f7 return an error from naive pause handling so that it gets retried (#3533)
- acd9d9e8f fix: Prevent Insights tab loading race condition. Wait for saved queries to load before restoring tabs from localStorage (#3530)
- b344d0e40 refactor out doneChan in favour of jobCtx (#3532)
- 11f5ca55d Queue interface refactor: Prepare for FDB (#3480)
- bf458c15f SYS-382: [ConstraintAPI] Add conditional high cardinality metrics for leases requested vs granted (#3473)
- 0a2b41d31 [SYS-566] State Store - updateMetadata - Lua 5.4 Support (#3528)
- ecd57528a Insights AI agent updates (#3516)
- 8051402d4 Handle "Invalid Date" that is an instance of Date (#3526)
- 9416c948e debug -> trace
- 76c37d2d0 nit on ctx cancel (#3525)
- ed479953e add context cancel check
- 53306f9d0 fixed credit card flow (#3504)
- 8eba49a54 fix expired pauses not being cleaned up because the context is done (#3523)
- 29f90991d nit:  update guards with new clauses (#3524)
- adc8c81b2 fix hanging runs when waiting for event fails to create timeout jobs (#3522)
- e02c189f6 Handle array types. Bump schema version (#3518)
- efc693887 update fields within invoke event
- aa80b4a17 kill cross app dep error in browser console (#3515)
- 9155c2b7c qol:  tidy invoke event logic (#3521)
- 8ee923872 Ensures that we tie invoking event ID to step metadata (#3520)
- e3df006fc Add basic metadata tab to trace view (#3409)
- e8b0d1c81 updated toast styles (#3517)
- dc8458921 fix: add a bit of wait for runID to populate in tests (#3508)

---

## v1.16.1 → v1.16.2

**Date range:** 2026-01-21 → 2026-02-06  
**Commits:** 48  
**Changes:**  462 files changed, 30916 insertions(+), 21526 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 2624 | `vendor/github.com/RoaringBitmap/roaring/runcontainer.go` |
| 2383 | `vendor/github.com/google/cel-go/parser/gen/cel_parser.go` |
| 1918 | `vendor/github.com/RoaringBitmap/roaring/roaring.go` |
| 1236 | `vendor/github.com/RoaringBitmap/roaring/bitmapcontainer.go` |
| 1137 | `vendor/github.com/bits-and-blooms/bitset/bitset.go` |
| 1101 | `vendor/github.com/RoaringBitmap/roaring/arraycontainer.go` |
| 976 | `tests/execution/state_store/lua_compatibility_test.go` |
| 962 | `pkg/execution/batch/buffer_test.go` |
| 887 | `vendor/github.com/google/cel-go/common/env/env.go` |
| 816 | `pkg/constraintapi/constraints_test.go` |
| 766 | `pkg/debugapi/debugapi_test.go` |
| 761 | `vendor/github.com/RoaringBitmap/roaring/roaringarray.go` |
| 702 | `vendor/github.com/google/cel-go/common/stdlib/standard.go` |
| 675 | `pkg/execution/state/redis_state/redis_state.go` |
| 662 | `vendor/github.com/RoaringBitmap/roaring/serialization_littleendian.go` |

### Commits

- 498156cf9 More postgres regression testing (#3631)
- 398585e26 Support retry counts and parallel failures in Durable Endpoints (#3624)
- 8b1a34456 Reapply "Send parallel step IDs when re-executing (#3565)" (#3581) (#3588)
- 490041001 [SYS-566] State Store - Update Metatdata Lua Tests (#3535)
- 6ab8fcdb7 Direct users to new support center (support.inngest.com) (#3625)
- dc4625e8c EXE-1239: Fix runs list pagination for self-hosted Inngest with postgres (#3626)
- 5702a906d Add postgres matrix tests for integration tests (#3619)
- 6024f23d8 EXE-1216 Add tracking of Durable Endpoint runs to span attributes (#3620)
- 42f58aeec EXE-806 Unpack Durable Endpoint responses (#3621)
- 968833ff5 Fix rate limit edge case (#3622)
- 4b6cc24cd Cache concurrency capacity in Acquire (#3623)
- 640261085 Optimize Acquire performance + caching (#3613)
- 8098cb513 Stub out Insights GQL scalars (#3618)
- e479d8246 Fix run list count after run list refresh (#3492)
- f97285c9e Add batching to batching (#3616)
- a5db97933 Add postgres testing for devserver/self hosted (#3614)
- d2ba41d02 ignore route gen formatting (#3615)
- 83fbf2568 Update Insights AI with new syntax and optimized prompts (#3607)
- 05dc3a428 Benchmark Acquire + add source to metrics (#3610)
- 72e7bbb62 Cleanup orphaned pauses in indexes (#3609)
- 2eb28b0eb Added makefile targets for tygo (#3580)
- acad3b4ac [SYS-602] Run Lua Compatibility Tests in Non Cluster Mode (#3605)
- e32965716 Include source in lease meta (#3606)
- eda6b9c78 Remove claude code + gemini cli from nix (#3584)
- 1a8d5cbd8 [SYS-601] Add Lua Tests For Pauses And Update Garnet Version To 1.0.94 (#3594)
- c5664a92b Simplify debug API to only require function_id (#3603)
- f5c32a64b optimize fast expressions mem usage in pauses aggregator (#3539)
- db07f9780 Add debug API endpoints for batch, singleton, and debounce insights (#3599)
- c815866c5 don't load page view tracker locally (#3598)
- 5fadb4682 Add Figma connect config and connect available components (#3597)
- e7f48cbd3 Optimistically release expired leases (#3596)
- ee02a36f1 fix block compaction failing when only 1 pause is left undeleted (#3595)
- 8753d7549 Batching: Return new status when first event in a batch is retried (#3590)
- 30b0c5347 Conditionally log capacity lease acquires (#3591)
- 021457954 Debug logs for batch scheduling (#3587)
- 7a09d0d8f Improve capacity lease extension + log in constraint scavenger (#3586)
- 167053927 Only check feature flag when capacity lease is set (#3585)
- 07c55e25a Add go struct to ts type generator (#3577)
- afede06bc Add utility to recover in crit (#3582)
- 559c84beb Improve Constraint API lease instrumentation (#3583)
- 7d289b7e0 Revert "Send parallel step IDs when re-executing (#3565)" (#3581)
- 489b3c5df Fix SDK told to use request version 0 (#3579)
- 89f7234c3 scaffolding directory for generated types (#3572)
- 284c2d8d0 Add more Constraint API metrics (#3576)
- 1115ebf98 add help command to makefile (#3573)
- 91c670089 Send parallel step IDs when re-executing (#3565)
- c395a97f5 decouple state store & pauses redis impls (#3562)
- 220287569 Update realtime endpoints (#3358)

---

## v1.16.2 → v1.16.3

**Date range:** 2026-02-06 → 2026-02-09  
**Commits:** 7  
**Changes:**  25 files changed, 942 insertions(+), 76 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 633 | `ui/apps/dev-server-ui/src/components/MCP/MCPPage.tsx` |
| 106 | `ui/apps/dev-server-ui/src/store/generated-types.ts` |
| 52 | `pkg/execution/checkpoint/checkpoint.go` |
| 50 | `pkg/execution/batch/buffer.go` |
| 24 | `pkg/execution/driver/httpv2/httpv2.go` |
| 21 | `ui/apps/dev-server-ui/src/routeTree.gen.ts` |
| 20 | `ui/packages/components/src/icons/sections/AI.tsx` |
| 15 | `pkg/connect/gateway.go` |
| 14 | `ui/packages/components/src/RunDetailsV3/RunDetailsV3.tsx` |
| 12 | `pkg/connect/routing/router.go` |
| 10 | `ui/apps/dev-server-ui/src/routes/_dashboard/mcp/index.tsx` |
| 9 | `pkg/connect/grpc/proxy.go` |
| 8 | `ui/apps/dev-server-ui/src/components/Layout/SideBar.tsx` |
| 7 | `pkg/devserver/devserver.go` |
| 5 | `tests/client/function_run.go` |

### Commits

- 834efbb78 Devserver mcp page (#3604)
- a8e6049cc Fix trace-related checkpointing issues (#3635)
- ecb42eef8 fix lint
- 2c45d44d4 tidy logs
- 63705d9ff remove at capacity log
- 98e358e68 Update log levels (#3634)
- 01ebf141d update in-memory buffer logging

---

## v1.16.3 → v1.17.0

**Date range:** 2026-02-09 → 2026-02-11  
**Commits:** 9  
**Changes:**  95 files changed, 10408 insertions(+), 3199 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 2254 | `ui/pnpm-lock.yaml` |
| 1511 | `ui/apps/dev-server-ui/src/store/generated-types.ts` |
| 662 | `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx` |
| 529 | `ui/packages/components/src/RunDetailsV4/TimeBrush.test.tsx` |
| 444 | `pkg/cqrs/base_cqrs/cqrs.go` |
| 378 | `ui/packages/components/src/RunDetailsV4/Timeline.tsx` |
| 375 | `tests/golang/function_runs_test.go` |
| 364 | `ui/packages/components/src/RunDetailsV4/TimeBrush.tsx` |
| 356 | `ui/packages/components/src/RunDetailsV4/RunDetailsV4.tsx` |
| 346 | `ui/packages/components/src/RunDetailsV4/StepInfo.tsx` |
| 345 | `ui/packages/components/src/RunDetailsV4/TimelineHeader.test.tsx` |
| 332 | `ui/packages/components/src/RunDetailsV4/utils/traceConversion.test.ts` |
| 320 | `ui/packages/components/src/RunDetailsV4/TopInfo.tsx` |
| 316 | `ui/packages/components/src/RunDetailsV4/README.md` |
| 269 | `pkg/run/cel_sql.go` |

### Commits

- 605e4d90a add CEL filtering support for runs using trace refactor (i.e. spans table) (#3640)
- b91f5ef78 trace ui design tweaks (#3670)
- c4c208bf4 EXE-1217 New Timing Bar (#3611)
- 2706177cc update our tanstack deps  (#3636)
- f551945e3 SYS-585: Shard group leasing for dynamic executor-to-shard assignment (#3575)
- 2be89b61b Fix some Durable Endpoints redirect issues (#3641)
- 80f82f42e Remove events page survey popover. (#3639)
- f466b262d Fix flaky cron test (#3638)
- 82bdde3a6 Insights: Add query diagnostics (#3593)

---

## v1.17.0 → v1.17.1

**Date range:** 2026-02-11 → 2026-02-18  
**Commits:** 25  
**Changes:**  181 files changed, 10325 insertions(+), 8521 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 1593 | `tests/execution/constraintapi/constraintapi_test.go` |
| 1520 | `ui/apps/dev-server-ui/src/store/generated-types.ts` |
| 1158 | `pkg/constraintapi/scavenge_test.go` |
| 1101 | `pkg/execution/state/redis_state/constraints_test.go` |
| 914 | `pkg/constraintapi/constraints_test.go` |
| 888 | `pkg/execution/queue/constraints_test.go` |
| 860 | `pkg/constraintapi/validation_test.go` |
| 703 | `proto/gen/debug/v1/queue.pb.go` |
| 381 | `pkg/execution/state/redis_state/constraints.go` |
| 353 | `ui/packages/components/src/RunDetailsV4/StepInfo.test.tsx` |
| 334 | `pkg/execution/queue/backlog_test.go` |
| 324 | `ui/apps/dashboard/src/gql/graphql.ts` |
| 317 | `pkg/execution/queue/constraints.go` |
| 307 | `pkg/tracing/metadata/extractors/response_headers_test.go` |
| 297 | `proto/gen/constraintapi/v1/service.pb.go` |

### Commits

- 0afddb3e9 Default feature-flag for run-details-v4 to on (#3704)
- 1feadb30a don't ref clerk in catch boundary (#3696)
- 105162321 Add shard name to the lease key (#3702)
- be81caa37 Insights: updated data table (#3694)
- d4e2362d6 SYS-585: Add a suffix option to ShardLeaseKeys (#3700)
- 447ad2773 Add debug API for shadow partitions and backlogs (#3688)
- 05d799457 Insights: Use sql-formatter clickhouse dialect for SQL formatting (#3692)
- b8b654e88 Add DeleteDebounceByID API for direct debounce deletion by ID (#3691)
- 90223766a Always delete debounce item (#3687)
- 28f4a97dc feat: display HTTP response headers and status code in RunDetailsV4 Headers tab (#3690)
- c0cafb0cc fix delay timings in traces (#3677)
- daf43940e Adds HTTP timings to traces (#3684)
- e8dcbb376 Insights: Update diagnostics banner w/ new design (#3671)
- bf309bc96 Speed traces styling update (#3689)
- 974008f95 Low hanging new traces fruit (#3682)
- ab2a2364c Set autoClosingBrackets and autoClosingBrackets to beforeWhitespace (#3686)
- 616164668 SYS-630: Add a callback for OnShardLeaseAcquired (#3685)
- feea246ea feat: store skip reason and display in UI (#3538)
- e24264b1b fix: Insights data table rendering for columns with complex names (#3681)
- 9adb06ed8 Fix 500 when loading insights page by using Uri off lazy loaded monaco instance (#3680)
- 379e589ca Prepare Capacity Manager for full cutover rollout (#3673)
- e40945dbc add empty check for next batch id (#3679)
- 7dd603095 SYS-631: Add a counter metric for shard lease contention (#3678)
- 3ae3c754f Fix unscheduled batches in buffered bulk append (#3675)
- 44a19b408 Fix key queues feature flag (#3676)

---

## v1.17.1 → v1.17.2

**Date range:** 2026-02-18 → 2026-03-04  
**Commits:** 65  
**Changes:**  183 files changed, 5846 insertions(+), 1090 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 345 | `ui/apps/dashboard/src/gql/graphql.ts` |
| 326 | `pkg/execution/queue/latency_test.go` |
| 294 | `pkg/util/rueidisconn/rueidisconn_test.go` |
| 288 | `pkg/tracing/metadata/extractors/ai.go` |
| 255 | `pkg/util/rueidisconn/rueidisconn.go` |
| 242 | `pkg/coreapi/generated/generated.go` |
| 238 | `pkg/devserver/api_test.go` |
| 191 | `pkg/execution/queue/constraints_test.go` |
| 171 | `pkg/tracing/metadata/extractors/ai_test.go` |
| 142 | `pkg/execution/executor/executor.go` |
| 137 | `pkg/execution/batch/buffer_test.go` |
| 135 | `pkg/execution/state/redis_state/shard_lease_test.go` |
| 115 | `pkg/constraintapi/lua_test.go` |
| 110 | `ui/apps/support/src/data/plain.ts` |
| 104 | `ui/packages/components/src/NewCodeBlock/NewCodeBlock.tsx` |

### Commits

- 62dd6c73c Use `_inngest.response.headers` attribute rather than `inngest.response_headers` metadata (#3797)
- 16d23ca96 Store lease issue timestamps and provide to Extend/Release (#3800)
- 9aa9aa584 Track scavenge count for requeued queue items (#3799)
- 805d60944 Fix idempotency key validation for long IDs (#3795)
- 8387bde13 Add tooltips for Inngest and server timing (#3789)
- 7b48d6a29 EXE-1382: Add test for duplicate apps (#3777)
- 025f14ae7 Make latency tracker a system job (#3793)
- 9a2b283d7 Fix infrastructure error output for trace (#3772)
- 2c3c38217 ensure we only save pending steps with opcode planned (#3790)
- 96a2efdf1 Add specific constructor for latency partition job tracking (#3787)
- f52a00755 fix pointer mismatch + empty error message (#3778)
- 4acc82b87 Add tracer opts for QueueTracer (#3785)
- 1787e3fbc Add weighted distribution for peek sizes (#3784)
- a072a475e Feat/ewma cache (#3776)
- 5fca510c5 move hook above error return on events page (#3774)
- ca88a3d84 Use goroutines for processing partitions (#3775)
- 9200d1d8d Update batch jobs (#3773)
- 5c0f68382 Update partition processing (#3770)
- 1e152494d Fix flaky wait run tests (#3771)
- 8e83b01e4 add TTFB tracking to rueidis (#3769)
- 300140db0 Try using NewCodeBlock for Insights CellDetailContext (#3726)
- 2dbd84d7b For CellDetailView - display dates in ISO 8601, UTC, local, and unix milliseconds (#3729)
- e395e9a07 fix: propagate SaveStep errors in checkpoint API (#3758)
- f6dbeb065 Reset active shard lease gauge on all shutdown paths (#3763)
- 4826ff469 Add cache for tls tickets (#3762)
- 9c9cc7019 Fix timeline visualization: update queue segment styling and add delay segment generation (#3756)
- 8c939b12e Add ticket filters to support center (#3753)
- cb62c5180 Add max buffer size for batching (#3747)
- 2d6849dc9 adds skip cache search param to post webhook delete nav (#3755)
- 3a62f3bda add reference to tygo in notion (#3713)
- d3c0a272c SYS-585: Retry transient shard renewal failures (#3752)
- 62eb1963b Update constraint API to handle combination of debounce + ratelimti (#3749)
- 02d0c54c5 added render order (#3699)
- cf353c48c fix expand/contract (#3742)
- 9b70bff75 Fix GCRA in Constraint API (#3748)
- 884d3b1a0 INN-5538: new code block api cleanup (#3743)
- 97fc9c9e7 Fix capacity lease extension error log (#3746)
- f9876a6a5 SYS-585: Release shard leases when done (#3745)
- f7978433f [NO-TKT] Adding Connect Section To Metrics UI (#3739)
- 15533f746 retry pause writes for waitForSignal registration (#3732)
- 58fabb710 remove excessive logging (#3734)
- 6dc2c4fad Fix throttle constraints (#3740)
- 36fc5f105 Disable capacity lease acquire on backlog refill (#3736)
- 77dada18b Properly handle Redis network errors in Constraint API (#3735)
- b10f7e3af Fix extend lease (#3733)
- 5a0c59528 don't print requeue messages for connect worker capacity error (#3730)
- a82196d46 Fix peeked item metrics: successive count logic, division by zero, clock consistency (#3728)
- 1dd637ffd Start collecting peeked item metrics (#3727)
- 253882cbc insights table keyboard shortcut experiment (#3697)
- 9569b0a64 feat: reset `die` after parallelism ends (#3717)
- 2bb3ece73 Follow up on insights data table (#3723)
- a2d9cdf54 [NO-TKT] Remove Connect Workers Feature Flag (#3724)
- 089e6be0e prevent compaction from being context cancelled (#3722)
- a282d5b31 verify shard lease is set before exiting shard lease loop (#3721)
- f51ac3b47 trigger compaction on delete by id (#3720)
- e81b33a3d [SYS-585] Attempt to grab a shard lease before kicking off the ticker (#3719)
- bb19e3926 add inngest status to default catch boundary (#3714)
- a94e21af4 SYS-655: moar logging (#3709)
- 77cfea33a Add more AI metadata (#3629)
- abd8a0125 feat: implement useTripleEscapeToggle hook and integrate into RunDetails components to toggle between old and new views (#3711)
- d997ed7cd no longer debug log every pause deletion (#3708)
- abf2ca4f8 Add delete-by-id command for debounces (#3710)
- 51b231355 feat: add skip reason to cloud ui (#3698)
- de7b13f91 Exclude custom concurrency keys missing in config (#3706)
- 286101451 Add correct constraint check source (#3705)

---

## v1.17.2 → v1.17.2-beta.1

**Date range:** 2026-03-04 → 2026-02-24  
**Commits:** 1  
**Changes:**  143 files changed, 779 insertions(+), 4139 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 326 | `pkg/execution/queue/latency_test.go` |
| 294 | `pkg/util/rueidisconn/rueidisconn_test.go` |
| 255 | `pkg/util/rueidisconn/rueidisconn.go` |
| 242 | `pkg/coreapi/generated/generated.go` |
| 238 | `pkg/devserver/api_test.go` |
| 137 | `pkg/execution/batch/buffer_test.go` |
| 135 | `pkg/execution/state/redis_state/shard_lease_test.go` |
| 110 | `ui/apps/support/src/data/plain.ts` |
| 104 | `ui/packages/components/src/NewCodeBlock/NewCodeBlock.tsx` |
| 104 | `pkg/util/rueidisconn/rueidisconn_miniredis_test.go` |
| 96 | `ui/packages/components/src/RunDetailsV4/TimelineBar.test.tsx` |
| 95 | `pkg/execution/state/driver_response_test.go` |
| 90 | `ui/apps/dashboard/src/components/Insights/InsightsTabManager/InsightsHelperPanel/features/CellDetail/CellDetailView.tsx` |
| 86 | `pkg/execution/executor/executor.go` |
| 84 | `ui/packages/components/src/Select/Select.stories.tsx` |

### Commits

- 6b3ce5420 test postgres perf fixes

---

## v1.17.2-beta.1 → v1.17.3

**Date range:** 2026-02-24 → 2026-03-09  
**Commits:** 52  
**Changes:**  184 files changed, 6388 insertions(+), 1276 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 373 | `proto/gen/api/v2/service.pb.go` |
| 326 | `pkg/execution/queue/latency_test.go` |
| 294 | `pkg/util/rueidisconn/rueidisconn_test.go` |
| 272 | `ui/apps/support/src/routeTree.gen.ts` |
| 255 | `pkg/util/rueidisconn/rueidisconn.go` |
| 242 | `pkg/coreapi/generated/generated.go` |
| 238 | `pkg/devserver/api_test.go` |
| 231 | `pkg/execution/queue/process.go` |
| 209 | `tests/api/v2_invoke_test.go` |
| 209 | `proto/api/v2/service.proto` |
| 191 | `pkg/api/v2/endpoints_function.go` |
| 189 | `pkg/execution/executor/executor.go` |
| 150 | `pkg/constraintapi/constraints_test.go` |
| 137 | `pkg/execution/batch/buffer_test.go` |
| 135 | `pkg/execution/state/redis_state/shard_lease_test.go` |

### Commits

- 35d7fc918 wait for in-progress runs to finish and gracefully exit (#3819)
- 39feadc7b Fix Connect worker drain not immediate (#3812)
- 8716b3d9c Include key expression hash in rate limit and throttle state keys (#3816)
- dcd79bec7 Only extend capacity leases if provided (#3815)
- a4c877cce Fix release of expired capacity lease (#3814)
- 9b0f06d07 Use background context for lease extensions in schedule path (#3813)
- 75082930c Add idempotency in V2 invoke API (#3811)
- b31e2e1d1 Plumb span response attribute field in GQL and stop sending response_header metadata (#3801)
- df7b9e8d4 Add ratelimits to V2 API (#3808)
- a87822772 Add customer.io snippet for in-app messaging (#3807)
- 816aaa937 Replace traces tooltip by hover card (#3803)
- 3a12189f6 V2 Invoke API (#3731)
- 62dd6c73c Use `_inngest.response.headers` attribute rather than `inngest.response_headers` metadata (#3797)
- 16d23ca96 Store lease issue timestamps and provide to Extend/Release (#3800)
- 9aa9aa584 Track scavenge count for requeued queue items (#3799)
- 805d60944 Fix idempotency key validation for long IDs (#3795)
- 8387bde13 Add tooltips for Inngest and server timing (#3789)
- 7b48d6a29 EXE-1382: Add test for duplicate apps (#3777)
- 025f14ae7 Make latency tracker a system job (#3793)
- 9a2b283d7 Fix infrastructure error output for trace (#3772)
- 2c3c38217 ensure we only save pending steps with opcode planned (#3790)
- 96a2efdf1 Add specific constructor for latency partition job tracking (#3787)
- f52a00755 fix pointer mismatch + empty error message (#3778)
- 4acc82b87 Add tracer opts for QueueTracer (#3785)
- 1787e3fbc Add weighted distribution for peek sizes (#3784)
- a072a475e Feat/ewma cache (#3776)
- 5fca510c5 move hook above error return on events page (#3774)
- ca88a3d84 Use goroutines for processing partitions (#3775)
- 9200d1d8d Update batch jobs (#3773)
- 5c0f68382 Update partition processing (#3770)
- 1e152494d Fix flaky wait run tests (#3771)
- 8e83b01e4 add TTFB tracking to rueidis (#3769)
- 300140db0 Try using NewCodeBlock for Insights CellDetailContext (#3726)
- 2dbd84d7b For CellDetailView - display dates in ISO 8601, UTC, local, and unix milliseconds (#3729)
- e395e9a07 fix: propagate SaveStep errors in checkpoint API (#3758)
- f6dbeb065 Reset active shard lease gauge on all shutdown paths (#3763)
- 4826ff469 Add cache for tls tickets (#3762)
- 9c9cc7019 Fix timeline visualization: update queue segment styling and add delay segment generation (#3756)
- 8c939b12e Add ticket filters to support center (#3753)
- cb62c5180 Add max buffer size for batching (#3747)
- 2d6849dc9 adds skip cache search param to post webhook delete nav (#3755)
- 3a62f3bda add reference to tygo in notion (#3713)
- d3c0a272c SYS-585: Retry transient shard renewal failures (#3752)
- 62eb1963b Update constraint API to handle combination of debounce + ratelimti (#3749)
- 02d0c54c5 added render order (#3699)
- cf353c48c fix expand/contract (#3742)
- 9b70bff75 Fix GCRA in Constraint API (#3748)
- 884d3b1a0 INN-5538: new code block api cleanup (#3743)
- 97fc9c9e7 Fix capacity lease extension error log (#3746)
- f9876a6a5 SYS-585: Release shard leases when done (#3745)
- f7978433f [NO-TKT] Adding Connect Section To Metrics UI (#3739)
- 15533f746 retry pause writes for waitForSignal registration (#3732)

---

## v1.17.3 → v1.17.4

**Date range:** 2026-03-09 → 2026-03-09  
**Commits:** 2  
**Changes:**  5 files changed, 150 insertions(+), 10 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 78 | `pkg/execution/executor/constraints_test.go` |
| 46 | `pkg/connect/gateway_test.go` |
| 30 | `pkg/connect/gateway.go` |
| 4 | `pkg/execution/queue/item.go` |
| 2 | `pkg/execution/queue/item_test.go` |

### Commits

- ab570c85d Fix Connect heartbeat undo drain (#3821)
- 7445d7bb8 Fix config retrieval in schedule (#3820)

---

## v1.17.4 → v1.17.5

**Date range:** 2026-03-09 → 2026-03-10  
**Commits:** 2  
**Changes:**  562 files changed, 145901 insertions(+), 50626 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 20791 | `vendor/github.com/minio/minlz/asm_amd64.s` |
| 7230 | `vendor/github.com/DataDog/zstd/xxhash.h` |
| 6315 | `vendor/github.com/DataDog/zstd/zstd_compress.c` |
| 3942 | `vendor/github.com/cockroachdb/pebble/compaction.go` |
| 3676 | `vendor/github.com/cockroachdb/pebble/v2/compaction.go` |
| 2381 | `vendor/github.com/cockroachdb/pebble/ingest.go` |
| 2328 | `vendor/github.com/cockroachdb/pebble/v2/ingest.go` |
| 2223 | `vendor/github.com/cockroachdb/pebble/v2/compaction_picker.go` |
| 2068 | `vendor/github.com/cockroachdb/pebble/compaction_picker.go` |
| 1976 | `vendor/github.com/cockroachdb/pebble/v2/sstable/rowblk/rowblk_iter.go` |
| 1916 | `vendor/github.com/DataDog/zstd/zstd.h` |
| 1879 | `vendor/github.com/DataDog/zstd/huf_decompress.c` |
| 1860 | `vendor/github.com/cockroachdb/pebble/sstable/block.go` |
| 1797 | `vendor/github.com/cockroachdb/pebble/{sstable/writer.go` |
| 1711 | `vendor/github.com/cockroachdb/swiss/map.go` |

### Commits

- db8cff1a6 Fix not setting last heartbeat when draining (#3822)
- f09b17876 support caching ast for pause expressions with numeric literals (#3751)

---

## v1.17.5 → v1.17.6

**Date range:** 2026-03-10 → 2026-03-30  
**Commits:** 7  
**Changes:**  219 files changed, 15058 insertions(+), 2076 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 1130 | `tests/execution/state_store/state_migration_test.go` |
| 926 | `pkg/constraintapi/acquire_test.go` |
| 908 | `pkg/execution/state/redis_state/reader_test.go` |
| 680 | `pkg/db/postgres/querier.go` |
| 659 | `pkg/execution/batch/migrating_test.go` |
| 635 | `pkg/db/sqlite/querier.go` |
| 628 | `ui/apps/support/src/data/plain.ts` |
| 581 | `pkg/execution/realtime/broadcaster.go` |
| 554 | `pkg/connect/gateway_drain_test.go` |
| 397 | `ui/apps/support/src/routes/_authed/case.$ticketId.tsx` |
| 344 | `ui/apps/support/src/components/Support/AttachmentUploadField.tsx` |
| 303 | `pkg/db/params.go` |
| 299 | `pkg/execution/realtime/broadcaster_redis.go` |
| 272 | `ui/apps/support/src/routeTree.gen.ts` |
| 267 | `pkg/db/models.go` |

### Commits

- bd1f81d28 Pause processing considers event receive time (#3856)
- 5c2290bcd Add Durable Endpoints streaming support (#3863)
- b3b4ce49c Add expand/contract all buttons (#3741)
- 4c9302460 Always enable Constraint API for devserver tests (#3903)
- 682d7118d Add warn log when cancel finds metadata but events are missing (#3906)
- 6e28fd613 Fix connect graceful shutdown draining (#3893)
- 22c6056f2 Allow releasing capacity lease early (#3857)

---

## v1.17.6 → v1.17.7

**Date range:** 2026-03-30 → 2026-03-31  
**Commits:** 9  
**Changes:**  76 files changed, 4175 insertions(+), 1870 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 660 | `ui/packages/components/src/RunDetailsV4/Timeline.tsx` |
| 581 | `pkg/execution/realtime/broadcaster.go` |
| 299 | `pkg/execution/realtime/broadcaster_redis.go` |
| 289 | `pkg/execution/checkpoint/checkpoint_sync_test.go` |
| 286 | `pkg/execution/executor/create_metadata_span_test.go` |
| 225 | `pkg/execution/realtime/broadcaster_test.go` |
| 223 | `proto/gen/connect/v1/connect.pb.go` |
| 209 | `pkg/coreapi/graph/loaders/trace_test.go` |
| 205 | `pkg/execution/realtime/broadcaster_redis_test.go` |
| 179 | `ui/packages/components/src/RunDetailsV4/TimelineBar.tsx` |
| 179 | `pkg/tracing/metadata_test.go` |
| 174 | `pkg/execution/realtime/api_publish_test.go` |
| 163 | `ui/packages/components/src/RunDetailsV4/utils/traceConversion.test.ts` |
| 156 | `ui/packages/components/src/RunDetailsV4/utils/traceConversion.ts` |
| 128 | `pkg/tracing/metadata/extractors/timing_test.go` |

### Commits

- b63b20d9e Timings Delays - Add Timing Metadata (#3897)
- aa93fde8a Insights: fix editor diagnostic severity (#3908)
- 8c740e63b Revert "Add Durable Endpoints streaming support (#3863)" (#3916)
- c0eff85b1 Remove shard for now, this might be nil (#3915)
- a9d1817de Add missing pkgName (#3914)
- f611ade74 Track handled opcodes (#3913)
- 1b565fac4 Add worker status messages (#3904)
- ba6c383b1 Fix state proxy not terminating fast enough (#3910)
- a4a9dea99 Add metadata span size limits and enable step metadata by default (#3840)

---

## v1.17.7 → v1.17.8

**Date range:** 2026-03-31 → 2026-04-02  
**Commits:** 6  
**Changes:**  112 files changed, 3382 insertions(+), 9331 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 1107 | `pkg/execution/state/redis_state/shadow_queue_test.go` |
| 1097 | `pkg/execution/state/redis_state/queue_scavenge_test.go` |
| 1092 | `pkg/execution/state/redis_state/lua_test.go` |
| 861 | `pkg/execution/state/redis_state/queue_test.go` |
| 771 | `pkg/execution/state/redis_state/active_checker.go` |
| 581 | `pkg/execution/realtime/broadcaster.go` |
| 525 | `pkg/execution/state/redis_state/queue_dequeue_test.go` |
| 323 | `pkg/execution/state/redis_state/lua/queue/backlogRefill.lua` |
| 317 | `pkg/execution/state/redis_state/queue_lease_test.go` |
| 299 | `pkg/execution/realtime/broadcaster_redis.go` |
| 265 | `pkg/execution/state/redis_state/queue.go` |
| 225 | `pkg/execution/realtime/broadcaster_test.go` |
| 222 | `tests/execution/queue/queue_operation_test.go` |
| 205 | `pkg/execution/realtime/broadcaster_redis_test.go` |
| 204 | `pkg/execution/state/redis_state/backlog.go` |

### Commits

- 3bca99f16 Add Durable Endpoints streaming support (#3926)
- c9d9aa28b Remove constraint enforcement logic in queue (#3765)
- aa71e1b3c Fix connect workers at capacity does not retry if at max attempts for other errors (#3927)
- d965b6e93 Fix Connect Gateway dropping requests on termination (#3923)
- 353e293ac Extend lease lifetime to 2 minutes and track counter if expired (#3922)
- 11be11e56 Fix finalization bar rendering (#3921)

---

## v1.17.8 → v1.17.9

**Date range:** 2026-04-02 → 2026-04-03  
**Commits:** 6  
**Changes:**  7 files changed, 37 insertions(+), 30 deletions(-)

### Top changed files

| Lines changed | File |
|--------------|------|
| 26 | `ui/apps/dashboard/src/routes/(auth)/agent-deep-link.tsx` |
| 19 | `pkg/connect/gateway.go` |
| 6 | `ui/apps/dashboard/src/components/Support/Status.tsx` |
| 5 | `ui/apps/dashboard/src/components/Billing/Plans/CheckoutModal.tsx` |
| 5 | `pkg/execution/driver/httpdriver/httpdriver_test.go` |
| 4 | `ui/apps/dashboard/src/components/Billing/Plans/ConfirmPlanChangeModal.tsx` |
| 2 | `ui/apps/dashboard/package.json` |

### Commits

- ce0021b68 Fix Connect ack race (#3934)
- 2fd431578 fix: tolerate expected write errors in TestStreamResponseTooLarge (#3924)
- 1f4a3f1c9 Add undocumented status to enum (#3931)
- 373f86575 prevent duplicate token consumption everywhere and strip all auth params on redirect (#3930)
- c8750f66f Hotfix update to node 22
- 4560a5861 Adjust billing copy to reflect billing accurately (#3928)

---

