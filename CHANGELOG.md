# Changelog

All notable changes to this project will be documented in this file.

## [v1.19.3] - 2026-05-14

### 🐛 Bug Fixes

- *(connect)* Keep persisted SDK replies from closing Connect workers (#4152)
- *(grpc)* Handle unimplemented error for `LoadDefersMeta` for backwards compatibility (#4159)
- Gracefully handle checkpoint dupes (#4130)
- Opcode Gateway -> StepTypeFetch (#4180)

### 💼 Other

- Add phase 1 agentic api contract (#3984)
- Entirely Synchronous Durable endpoints move to Completed Status (#4119)
- Test Durable Endpoints against JS SDK (#4146)
- Remove release/next branch trigger

### 🚜 Refactor

- Refactor debouncer ops to expose via ShardOperations (SYS-824) (#4127)

### ⚡ Performance

- *(pauses)* Evaluations containing string comparisons (#4154)

### ⚙️ Miscellaneous Tasks

- Fix PR release job (#4163)
- Run full CI for release PRs
- Link release PRs to compare view
- Update semaphore manager to return idempotency info (#4186)
- Add "docs" to allowed list (#4175)

## [v1.19.2] - 2026-05-07

### 🚀 Features

- Add SDK conformance CLI serve runner (#4027)

### 🐛 Bug Fixes

- *(registration)* Nil-pointer panic on empty-function sync (/fn/register) (#4129)

## [v1.9.2-beta.1] - 2026-05-05

### 🐛 Bug Fixes

- *(tests)* Increase TestWorkerConcurrency timeouts for CI stability (#4100)
- *(queue)* Pass RequestTime in queue Acquire to bypass stale constraint cache (#4104)
- Add rueidis goleak ignore for singleflight goroutine (#4116)

### 💼 Other

- Drop 5-candidate clamp in pickConnection (#4112)
- API Docs (#3968)
- Log step name for duplicate/failed step saves (#4077)
- Create a ShardRegisty interface that owns all known shards and shard selector logic (#4082)
- Make Singleton Manager blissfully unaware of underlying queue stores. (#4081)
- Rename NewRedisCronManager to NewManager (#4109)

### ⚙️ Miscellaneous Tasks

- *(build)* Bump MCP and go-jwt package versions for CVE fixes (#4099)

## [v1.19.1] - 2026-04-30

### 🐛 Bug Fixes

- Broaden hasPlanOp check (#4079)

### 💼 Other

- Add tables to schema explorer (#4072)

## [v1.19.0] - 2026-04-29

### 🐛 Bug Fixes

- *(cqrs/postgres)* COALESCE finish columns in function-run LEFT JOINs (#4043)

### 💼 Other

- Flexible table column widths (#4033)
- Accept date in WaitForEventOpts.Expires (#4048)
- Add per-function Prometheus metrics (#4070)

## [v1.18.0] - 2026-04-22

### 🐛 Bug Fixes

- Anchor fake clock to minute boundary in TestBacklogsByPartition and TestItemsByPartition (#3936)
- Protect messages slice with mutex and use Eventually in TestRealtime (#3925)
- Stop clock advances after Release() to prevent flaky early-release test (#3937)
- Future-date cancellation spans (#3995)
- *(deps)* Upgrade pgx to v5.9.1 to patch CVE-2026-33816 (#4014)

### 💼 Other

- Reset queue item from status key sets when calling RemoveQueueItem (#3982)
- Gracefully handle long userland attributes (#4035)

### 🚜 Refactor

- *(base_cqrs)* Switch migrations to goose baselines (#3945)

### ⚙️ Miscellaneous Tasks

- Run go generate and go mod tidy after protobuf regen to keep generated files in sync (#3939)
- Clean up base cqrs migrations (#4024)

## [v1.17.9] - 2026-04-03

### 🐛 Bug Fixes

- Tolerate expected write errors in TestStreamResponseTooLarge (#3924)

## [v1.17.6] - 2026-03-30

### 🐛 Bug Fixes

- Userland OTel span parenting w/checkpointing (#3804)
- Make http response_status metadata nilable & omitempty (#3829)

### 💼 Other

- Lots more ux defense around neon/supabase failure scenarios (#3806)
- Fixes partition iterator bug referencing wrong timestamp (#3902)

## [v1.17.2] - 2026-03-05

### 🚀 Features

- Implement useTripleEscapeToggle hook and integrate into RunDetails components to toggle between old and new views (#3711)
- Reset `die` after parallelism ends (#3717)

### 🐛 Bug Fixes

- Propagate SaveStep errors in checkpoint API (#3758)

### 💼 Other

- Moar logging (#3709)
- Release shard leases when done (#3745)
- New code block api cleanup (#3743)
- Retry transient shard renewal failures (#3752)
- Add test for duplicate apps (#3777)

## [v1.17.1] - 2026-02-18

### 🚀 Features

- Store skip reason and display in UI (#3538)
- Display HTTP response headers and status code in RunDetailsV4 Headers tab (#3690)

### 💼 Other

- Add a counter metric for shard lease contention (#3678)
- Add a callback for OnShardLeaseAcquired (#3685)
- Update diagnostics banner w/ new design (#3671)
- Use sql-formatter clickhouse dialect for SQL formatting (#3692)
- Add a suffix option to ShardLeaseKeys (#3700)
- Updated data table (#3694)

## [v1.17.0] - 2026-02-11

### 💼 Other

- Shard group leasing for dynamic executor-to-shard assignment (#3575)

## [v1.16.2] - 2026-02-06

### 💼 Other

- Return new status when first event in a batch is retried (#3590)
- Fix runs list pagination for self-hosted Inngest with postgres (#3626)

## [v1.16.1] - 2026-01-21

### 🐛 Bug Fixes

- Add a bit of wait for runID to populate in tests (#3508)

### 💼 Other

- Tidy invoke event logic (#3521)
- Update guards with new clauses (#3524)
- [ConstraintAPI] Add conditional high cardinality metrics for leases requested vs granted (#3473)

### 🛡️ Security

- Upgrade tar version to latest (#3571)

## [v1.16.0] - 2026-01-07

### 🚀 Features

- Update StepInfo to include loading, No output available and No trace data available messages when applicable (#3365)
- Add Handlebars support to system prompts in Insights agents and make query-writer aware of data schemas (#3502)

### 🐛 Bug Fixes

- Correct dev server event URL path (#3498)

### 🚜 Refactor

- Unified SendEventModal (#3440)

## [v1.15.2] - 2025-12-22

### 💼 Other

- Add metrics to track constraint API usage and rollout (#3429)

## [v1.15.1] - 2025-12-17

### 🚀 Features

- Send multiple events on the dev server (#3410)

## [v1.15.0] - 2025-12-10

### 🐛 Bug Fixes

- Move dev-server Toaster from _dashboard to __root (#3394)
- Extended trace span IDs (#3392)

## [v1.14.2] - 2025-12-05

### 🐛 Bug Fixes

- Plumb missing metadata scope in GQL (#3364)

### 💼 Other

- Improve logging for crons (#3339)

## [v1.14.3] - 2025-11-21

### 💼 Other

- Fallback to Github for binary (#3318)
- Implement new telemetry (#3329)

## [v1.13.7] - 2025-11-13

### 💼 Other

- Remove select-none in SchemaViewer ValueRow (#3292)

## [v1.13.6] - 2025-11-12

### 🐛 Bug Fixes

- Plumb App ID in state (#3273)

### 💼 Other

- Implement rate limiting in pure Lua (#3265)
- Remove Key Count Pill from Schema Objects (#3274)

### 🚜 Refactor

- Store AppID and FunctionID on userland traces (#3270)

## [v1.13.5] - 2025-11-11

### 💼 Other

- Improve UI Rendering of Custom Event Schemas (#3251)

## [v1.13.3] - 2025-11-05

### 💼 Other

- Render Common Event Schema in Schema Explorer (#3228)
- Fix Schema Widget Resizable Overlap Issue (#3240)
- Event emitter for dynamic run list status and endedAt (#3250)
- Update transformJSONSchema for Arbitrary Schemas (#3249)

### 🚜 Refactor

- Improve userland trace view (#3241)

## [v1.13.2] - 2025-11-03

### 🐛 Bug Fixes

- Don't zero-out userland trace span EndedAt (#3232)

### 💼 Other

- Introduce transformJSONSchema to Simplify UI Schema Rendering (#3226)

### 🧪 Testing

- Add error/retry tests for Go SDK (#3225)

## [v1.13.0] - 2025-10-30

### 🐛 Bug Fixes

- Show error on function runs page instead of no results on error (#3106)
- Clarify ErrDenied to include timeout as a cause (#3105)
- Collapse steps with a single attempt (#3111)
- Unwrap extra StandardError wrapping (#3120)
- Correct CreateSpan call args (#3202)
- Remove singleton tracer to prevent stale metadata (#3208)

### 💼 Other

- Modify editorSuggestWidget background color (#3086)
- Global feature flag toggle (#3121)
- Insights SQL Agent (#3007)
- Increment function version on syncs. (#3114)
- Always add a status when enqueueing root spans (#3140)
- Update event runs API endpoint (#3144)
- Default trace preview feature flag call to true (#3143)
- Bump pnpm 8 to pnpm 10 and add minimumReleaseAge setting (#3122)
- Redo cron using queues (#2847)
- Skip docker hub release for beta tag (#3182)
- Cron Health Checks (#3201)

## [v1.12.0] - 2025-10-01

### 🚀 Features

- Add OpcodeStepFailed support (#2992)

### 💼 Other

- Publish on localhost via realtime forwarding (#3023)
- Make Insights panels resizable, introduce Resizable (#3060)

## [v1.11.13] - 2025-09-19

### 🐛 Bug Fixes

- Only trigger 1 onFailure call on parallel step failure (#2907)
- Problem with multiple event trigger expressions (#2951)

### 💼 Other

- Improve function configuration table tooltip (#2903)
- Add permanent home (icon) tab (#2934)
- Point Insights feedback link to support page (#2985)
- Tabs]: Prevent active tab switch when closing a tab (#2997)

## [v1.11.9] - 2025-08-28

### 💼 Other

- Support reusable Tabs component dynamic variant (#2825)

### ⚙️ Miscellaneous Tasks

- Sort OtelSpans deterministically in GraphQL (#2859)

## [v1.11.7] - 2025-08-21

### 💼 Other

- Add tests for batching events + if triggers (#2817)
- Source paused status from new getter, migration from lock key (#2806)
- Add conditional expression for batching eligibility (#2818)

## [v1.11.4] - 2025-08-05

### 💼 Other

- Add (mocked) results table (#2773)

## [v1.11.2] - 2025-07-31

### 💼 Other

- Introduce nav item and route (#2757)

## [v1.11.0] - 2025-07-25

### 💼 Other

- Update actions UX (#2731)

## [v1.10.0] - 2025-07-21

### 💼 Other

- Add debug run and session ids to scheduler and rerun from step resolver (#2722)

## [v1.7.0] - 2025-06-16

### 💼 Other

- Show FunctionConfiguration in dev server (#2546)

## [v1.6.0] - 2025-05-19

### 💼 Other

- Return syscode error when request > 2h (#2430)

## [v1.5.12] - 2025-05-09

### 🚀 Features

- Add some indexes to increase performance of self hosted dashboard using postgres (#2318)

### 💼 Other

- Log a warning if waitForEvent.If CEL expression fails validation (#2405)

## [v1.5.3] - 2025-03-14

### 💼 Other

- Trace run start/first step alignment fix & finalization (#2242)
- Break out cancel run button (#2244)

## [v1.5.0] - 2025-03-06

### 💼 Other

- Swap in split button, input fixes (#2225)

## [v1.4.7] - 2025-02-24

### 💼 Other

- New run AI traces (#2151)

## [v1.4.4] - 2025-01-28

### 💼 Other

- Ensure wildcards are matched during execution (#2117)

## [v1.4.2] - 2025-01-22

### 💼 Other

- Improve auto discovery to reduce noise and excessive polling (#2094)

## [v1.4.0-beta.1] - 2025-01-16

### 💼 Other

- Also set has ai in span attributes on function finished (#2037)
- Add release documentation (#2040)
- Support ai indicators and rerun from step in cloud (#2038)

## [v1.3.3] - 2024-12-17

### 🐛 Bug Fixes

- Update url for useFeatureFlags hook to support non default port (#2015)

## [v1.3.2] - 2024-12-05

### 💼 Other

- Pass ai gateway step op code up to ui (#1998)
- Add ai indicator for runs and add decoration to runs ui (#2001)

## [v1.2.0] - 2024-11-19

### 🐛 Bug Fixes

- Set concurrency in expressions.NewAggregator (#1910)
- Race condition in LoadEventEvaluator (#1911)
- Check for nil persistence interval in single node mode (#1965)

### 💼 Other

- Metrics latest charts (#1838)
- Lift QueueItem into common queue struct and interface (#1872)
- Fix timeouts duration parsing (#1968)

## [v1.1.1-beta.1] - 2024-10-03

### 💼 Other

- Always set inner queue name (i.Data.QueueName) during Enqueue() (#1817)

### 🛡️ Security

- Upgrade next-clerk (#1829)

## [v1.1.0] - 2024-09-26

### 💼 Other

- Wire up function run and step throughput charts (#1792)
- Rounding, function list fetching, etc (#1809)

## [v1.0.1] - 2024-09-22

### 💼 Other

- Metrics dashboard failed function chart and failed function rate list (#1763)

## [v1.0.0] - 2024-09-20

### 💼 Other

- Track started batches (#1762)

## [v0.29.9] - 2024-09-16

### 💼 Other

- Skip non-default partitions in Scavenge (#1717)
- Add a new user survey link (#1740)
- Metrics dashboard added, function status chart wired up (#1709)

## [v0.30.0-beta-2] - 2024-08-19

### 🐛 Bug Fixes

- Worker capacity should be run regularly outside of the lease (#1666)

### 💼 Other

- Graceful pause deletion (#1617)
- Dev server ia nav (#1642)

## [v0.29.4] - 2024-08-09

### 💼 Other

- Add circular font to the dashboard (#1544)
- New ia nav wip (#1584)
- The rest of the side navigation widgets (#1605)
- Ia nav functions (#1627)
- A handful of ui/ux fixes (#1662)

## [v0.29.2] - 2024-07-03

### 💼 Other

- New integrations (#1421)
- Maze survey cannot be clicked when overlay is shown (#1521)
- Move to proper alert (#1523)
- Implement state size limit (#1543)

## [v0.29.1] - 2024-06-07

### 🐛 Bug Fixes

- Executor.Schedule returns ErrFunctionSkipped when a function is paused (#1337)

### 💼 Other

- Support function pausing (#1330)
- Make code editor support tailwind/system color schema and overrides (#1357)
- Trace tweaks (#1371)
- Swap in new date picker for function replay range (#1367)
- Add helper method in Span to store SDK resp (#1403)
- Fix event stream pagination by using ulid as cursor (#1409)
- Date range picker (#1391)

## [v0.28.0] - 2024-05-03

### 💼 Other

- Minor tweaks for span annotations (#1280)

## [v0.26.6] - 2024-04-03

### 💼 Other

- Incorporate tracer into codebase for user traces (#1237)
- Fix empty event payloads in UI due to resolver bug (#1258)

## [v0.26.4] - 2024-03-13

### ⚙️ Miscellaneous Tasks

- Remove IGassmann from CODEOWNERS (#1227)

## [v0.26.3] - 2024-03-05

### 💼 Other

- Ensure `cancelOn` works with `event.ts` in the future (#1206)
- Fix custom event IDs are not idempotent (#1202)

## [v0.26.2] - 2024-02-27

### 💼 Other

- Add additional charts to the UI for showing backlog and the number of steps currently running (#1170)

## [v0.26.0] - 2024-02-22

### 🚀 Features

- *(dashboard)* Improve auth for URQL (#1076)
- *(dashboard)* Retry unauthenticated requests (#1079)
- *(dashboard)* Redirect to sign-in with an error after retries of unauthenticated errors (#1096)
- *(dashboard)* OrganizationDropdown (#1116)
- *(dashboard)* UserDropdown (#1117)
- Move anonymous ID to cookies (#488)
- *(auth)* Link to new sign-in and sign-up URLs (#499)
- *(docs)* Add [object, object] type to concurrency (#582)
- *(docs)* Add FAQ for endpoint authentication (#585)
- *(docs)* Replay functions guide (#596)
- *(website)* Enable website workflow file
- *(website)* Add Algolia variables and secrets

### 🐛 Bug Fixes

- Cron with trailing space fail to parse (#1108)
- *(docs)* Remove plural form for event trigger (#350)
- *(sign-up)* Input text is invisible (#403)
- *(docs)* Deploy link breaks Dev Server (#442)
- Broken links (#475)
- *(patterns)* Incomplete sentence (#541)
- Docs page open graph image for homepage
- *(website)* Website - On Deployment workflow
- *(website)* Next-mdx-remote types
- *(website)* On deployment workflow

### 💼 Other

- Add AppID as a field for state identifier (#1082)
- New site
- Workflow CLI
- Add new high level arch doc
- Add Next.js HTTP function post
- Add basic http functions page
- Add Managing Secrets
- Inngest Dev Server
- Open source announcement
- Make posts live for launch day
- A/B test homepage hero text and graphic
- Trigger your code directly from Retool
- No workers necessary Node/Express
- Cron page
- Completing the Jamstack in 2022
- Patterns: Run code on a schedule
- Build reliable webhooks
- Patterns: Reliably run critical workflows
- Patterns: Reliable scheduling systems
- Blog: Build more reliable workflows with events
- New local development doc page with dev server
- Add deduplication id info
- Remove outdated framework docs. Redirect to serve framework docs
- Blog post: E-commerce API imports
- Docs search cannot re-focus search
- Blog: Long running background functions on Vercel
- Customer quote on mobile
- Hide AI code generator banner
- Blog: April announcement blog post (#363)
- *(blog)* Is the Next.js 13 App Router Stable Enough for Production? (#367)
- *(blog)* Adapt Next.js blog post (#376)
- Cloud Functions, more consistent serve docs (#377)
- Docs: Working With Environments (#385)
- Create reference section (#386)
- Meta tags: description & use case pages (#391)
- Blog: Branch Environments (#364)
- Add handling failures reference (#401)
- Add docs regarding our current limitations (#400)
- Add docs related for logging (#411)
- Pricing Page Updates (#412)
- Clarify event key security further (#425)
- Create user-defined workflows guide (#406)
- Improvements in Quick Start Tutorial (#430)
- *(docs)* Make http://localhost:8288 string a link (#438)
- *(docs)* Add missing types for req and res (#439)
- *(docs)* Add comment to indicate placement of previous code (#440)
- *(docs)* Typo (#441)
- Blog: Seed fundraising announcement (#437)
- Add guide for batching and update references (#435)
- Add blog post for event batching feature (#446)
- Blog: Migrating from Vite to Next.js (#463)
- *(blog)* Style callout (#464)
- *(blog)* Set `moduleResolution` to `bundler` (#467)
- Discord link due to expired link (#474)
- Switch to pnpm (#471)
- Pin dependencies (#472)
- Docs: Add Vercel bypass protection docs (#486)
- Add concurrency key to docs (#500)
- Create rateLimit reference (#509)
- Add if option to event trigger docs (#518)
- Document idempotency key (#521)
- Add blog post for fn metrics release (#550)
- Add custom environments to the docs (#559)
- Create new webhooks platform guide (#580)
- Building metrics with TimescaleDB (#584)
- Show free plan has Discord support (#606)
- Write guide on expressions (#613)
- Add handling idempotency guide (#610)
- Add launch week landing page and banner (#622)
- Create new cancelation guide
- Updated fan-out guide (#637)
- Move event api info to send events guide (#635)
- Fix broken links on blog (#663)
- Add a16z funding blog post (#668)

### 🚜 Refactor

- *(website)* Move files to ui/apps/website/
- *(website)* Merge website repository
- *(website)* Remove pnpm-lock.yaml
- Align package.json metadata
- *(website)* Align .gitignore
- *(website)* Remove duplicated shell.nix

### 📚 Documentation

- *(package)* Clean up package.json (#473)
- *(basics)* Replace serve landing page with dev server (#566)

### ⚙️ Miscellaneous Tasks

- Run pnpm install
- *(website)* Update next-mdx-remote to latest
- Format files
- *(website)* Align postcss version
- *(ui)* Align zod version
- Remove website app (#1151)

## [v0.24.0] - 2024-01-17

### 💼 Other

- *(dashboard)* Occasional 500 errors (#940)
- Add function to retrieve total number of items in the function (#1027)

### ⚙️ Miscellaneous Tasks

- *(postinstall.ts)* Use `package.json` version instead of an env variable (#1013)

## [v0.23.0] - 2023-12-07

### 🚀 Features

- *(dashboard)* Function replay (#767)

### 🐛 Bug Fixes

- Revert Node.js and pnpm version upgrade (#898)

### ⚙️ Miscellaneous Tasks

- *(dashboard)* Run GraphQL Codegen during build (#879)
- Update @clerk/nextjs to latest
- Update @sentry/nextjs to latest
- Update Next.js to latest
- Add @launchdarkly/node-server-sdk

## [v0.22.0] - 2023-11-27

### 🚀 Features

- *(dashboard)* Hook up function config (#857)

## [v0.21.1] - 2023-11-15

### 🚀 Features

- Use Tailwind CSS default font sizes
- Use Tailwind CSS default border radius
- Add fallback mono fonts
- Add base.json file (#706)
- *(dashboard)* Make chart legend colors consistent (#729)
- *(dashboard)* Function replay modal (#763)
- *(dashboard)* Display error message on user creation (#830)
- *(dashboard)* Function config (#835)

### 🐛 Bug Fixes

- *(dev-server-ui)* GraphQL codegen (#701)
- *(components)* Ambiguous root paths (#742)
- *(components)* Don't render multiple html and body tags

### 💼 Other

- *(tailwind)* Align content lookup
- *(tailwind)* Don't override existing colors
- *(tailwind)* Remove unneeded custom grid templates
- *(tailwind)* Remove unused grid templates
- Type check Tailwind config file
- Add chart for the number of sdk requests being made from the executor (#724)
- Display recent tickets. Switch to Plain threads. (#781)
- Add actions for syncing main with next branch (#807)

### 🚜 Refactor

- Remove unused ui/.github/ dir (#693)
- Remove unneeded display config
- Use camelCase for variable
- Remove unused font
- Use complete class names
- Remove custom plugin for icon sizes
- Move dev-server-ui/ to ui/apps/
- Clean up package.json files
- Change paths for runners
- Align Tailwind CSS configuration (#700)
- Set up shared tsconfig (#704)

### ⚙️ Miscellaneous Tasks

- Set up env vars with Vercel (#695)
- Update pnpm lock file
- Hoist prettier
- Clean up package.json
- Clean up .gitignore
- Align TypeScript configurations (#702)
- Update Next.js to latest
- Update storybook to latest
- Update @sentry/nextjs to latest
- Update @clerk/nextjs to latest

### 🛡️ Security

- Add security scan for golang (#760)

## [v0.19.0] - 2023-10-11

### 💼 Other

- Add granularity to usage opts and deprecate period (#682)

## [v0.18.1] - 2023-10-09

### 💼 Other

- Use time.Duration instead of string for granularity (#676)

## [v0.18.0] - 2023-10-06

### 💼 Other

- Replace MaxBatchSize with DefaultBatchSize to make it more configurable (#651)
- Add API structure for timeseries related data for metrics (#668)

## [v0.17.0] - 2023-09-26

### 💼 Other

- Add tests and additional check for invalid URIs (#642)

## [v0.15.6] - 2023-07-28

### 💼 Other

- Add telemetry setup (#530)
- Add back tracing (#537)

## [v0.15.5] - 2023-07-20

### 💼 Other

- Remove tracer for now (#528)

## [v0.15.0] - 2023-07-12

### 💼 Other

- Add batch related configs (#465)
- Add actions to be truncated and comment it out for now to (#475)
- *(ui)* Switch to pnpm (#478)
- *(ui)* Migrate from Vite to Next.js (#479)
- *(ui)* Pin dependencies (#480)
- Allow batching to work (#477)
- Add Apps Page (#490)

## [v0.14.1] - 2023-06-14

### 💼 Other

- Allow `Event` list in state input for batching (#450)

## [v0.13.0] - 2023-05-01

### 💼 Other

- Refactored the UserError function to handle arbitrary response (#435)

## [v0.13.1] - 2023-03-31

### 🚀 Features

- Automatically display newly created events (#386)

### 📚 Documentation

- *(readme)* Correct typos (#403)
- *(contributing)* Add contributing guide (#404)

## [v0.10.0] - 2023-02-08

### 🐛 Bug Fixes

- Skip saving response if generator res is empty
- Executor re-queuing edges as sleeps after a sleep is found
- `waitForEvent` op erroneously setting stack before returning
- Generator SDK returning 200 fails to save final step data (#383)

## [v0.5.4] - 2022-08-16

### 💼 Other

- Matrix for `go test`

## [v0.4.0] - 2022-07-05

### 💼 Other

- Add basic Redis state implementation

## [v0.3.0] - 2022-06-09

### 💼 Other

- Drag in open source executor

## [v0.1.0] - 2022-03-10

### 💼 Other

- Support scopes defined in actions

## [v0.0.1] - 2021-07-09

<!-- generated by git-cliff -->
