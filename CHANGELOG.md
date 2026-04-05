# Changelog

All notable changes to this project will be documented in this file.

## [unreleased]

### 🚀 Features

- Implement the greatest app

### 🐛 Bug Fixes

- Anchor fake clock to minute boundary in TestBacklogsByPartition and TestItemsByPartition (#3936)
- Protect messages slice with mutex and use Eventually in TestRealtime (#3925)
- Fix bug

### ⚙️ Miscellaneous Tasks

- Do something

## [1.17.9] - 2026-04-03

### 🐛 Bug Fixes

- Tolerate expected write errors in TestStreamResponseTooLarge (#3924)

## [1.17.7] - 2026-03-31

### 💼 Other

- Fix editor diagnostic severity (#3908)

## [1.17.6] - 2026-03-30

### 🐛 Bug Fixes

- Userland OTel span parenting w/checkpointing (#3804)
- Make http response_status metadata nilable & omitempty (#3829)

### 💼 Other

- Lots more ux defense around neon/supabase failure scenarios (#3806)
- Fixes partition iterator bug referencing wrong timestamp (#3902)

## [1.17.2] - 2026-03-05

### 🚀 Features

- Add skip reason to cloud ui (#3698)
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

## [1.17.1] - 2026-02-18

### 🚀 Features

- Store skip reason and display in UI (#3538)
- Display HTTP response headers and status code in RunDetailsV4 Headers tab (#3690)

### 🐛 Bug Fixes

- Insights data table rendering for columns with complex names (#3681)

### 💼 Other

- Add a counter metric for shard lease contention (#3678)
- Add a callback for OnShardLeaseAcquired (#3685)
- Update diagnostics banner w/ new design (#3671)
- Use sql-formatter clickhouse dialect for SQL formatting (#3692)
- Add a suffix option to ShardLeaseKeys (#3700)
- Updated data table (#3694)

## [1.17.0] - 2026-02-11

### 💼 Other

- Add query diagnostics (#3593)
- Shard group leasing for dynamic executor-to-shard assignment (#3575)

## [1.16.2] - 2026-02-06

### 💼 Other

- Return new status when first event in a batch is retried (#3590)
- Fix runs list pagination for self-hosted Inngest with postgres (#3626)

## [1.16.1] - 2026-01-21

### 🐛 Bug Fixes

- Add a bit of wait for runID to populate in tests (#3508)
- Prevent Insights tab loading race condition. Wait for saved queries to load before restoring tabs from localStorage (#3530)

### 💼 Other

- Tidy invoke event logic (#3521)
- Update guards with new clauses (#3524)
- [ConstraintAPI] Add conditional high cardinality metrics for leases requested vs granted (#3473)

### 🛡️ Security

- Upgrade tar version to latest (#3571)

## [1.16.0] - 2026-01-07

### 🚀 Features

- Update StepInfo to include loading, No output available and No trace data available messages when applicable (#3365)
- Add Handlebars support to system prompts in Insights agents and make query-writer aware of data schemas (#3502)

### 🐛 Bug Fixes

- Correct dev server event URL path (#3498)

### 🚜 Refactor

- Unified SendEventModal (#3440)

## [1.15.2] - 2025-12-22

### 💼 Other

- Add metrics to track constraint API usage and rollout (#3429)

## [1.15.1] - 2025-12-17

### 🚀 Features

- Send multiple events on the dev server (#3410)

## [1.15.0] - 2025-12-10

### 🐛 Bug Fixes

- Move dev-server Toaster from _dashboard to __root (#3394)
- Extended trace span IDs (#3392)

## [1.14.2] - 2025-12-05

### 🐛 Bug Fixes

- Plumb missing metadata scope in GQL (#3364)

### 💼 Other

- Remove usage from overview - HOTFIX (#3340)
- Improve logging for crons (#3339)

## [1.14.3] - 2025-11-21

### 💼 Other

- Display Shared Queries Separately from Saved Queries (#3283)
- Show Query Authorship History (#3286)
- Add Ability to Share Queries (#3294)
- Show executions usage and limit for Hobby plan (#3279)
- Add shortcuts for saving queries and opening tabs (#3151)
- Add Right-Click Menu on Saved Queries List Items (#3296)
- Fallback to Github for binary (#3318)
- Implement new telemetry (#3329)
- Temporarily remove synced functions count, synced functions and removed functions from App All syncs history until an actual solution can be implemented (#3335)

## [1.13.7] - 2025-11-13

### 💼 Other

- Remove select-none in SchemaViewer ValueRow (#3292)

## [1.13.6] - 2025-11-12

### 🐛 Bug Fixes

- Plumb App ID in state (#3273)

### 💼 Other

- Implement rate limiting in pure Lua (#3265)
- Remove Key Count Pill from Schema Objects (#3274)

### 🚜 Refactor

- Store AppID and FunctionID on userland traces (#3270)

## [1.13.5] - 2025-11-11

### 💼 Other

- Improve UI Rendering of Custom Event Schemas (#3251)
- Fetch Real Event Schemas in Schemas Explorer Widget (#3256)
- Count schemas toward cap even if they are not valid JSONSchema (#3262)
- Various Schema Widget Improvements (#3267)
- Add Schemas in Use Section (#3269)

## [1.13.3] - 2025-11-05

### 🐛 Bug Fixes

- Less comples GraphQL query for nested spans (#3243)

### 💼 Other

- Render Common Event Schema in Schema Explorer (#3228)
- Fix Schema Widget Resizable Overlap Issue (#3240)
- Event emitter for dynamic run list status and endedAt (#3250)
- Update transformJSONSchema for Arbitrary Schemas (#3249)

### 🚜 Refactor

- Improve userland trace view (#3241)

## [1.13.2] - 2025-11-03

### 🐛 Bug Fixes

- Don't zero-out userland trace span EndedAt (#3232)

### 💼 Other

- Enhance VercelIntegrationError component with reconnect option and user guidance (#3235)
- Introduce transformJSONSchema to Simplify UI Schema Rendering (#3226)

### 🧪 Testing

- Add error/retry tests for Go SDK (#3225)

## [1.13.1] - 2025-10-31

### 💼 Other

- Introduce Insights Helper Panel on Right Side (#3220)

## [1.13.0] - 2025-10-30

### 🐛 Bug Fixes

- Show error on function runs page instead of no results on error (#3106)
- Clarify ErrDenied to include timeout as a cause (#3105)
- Collapse steps with a single attempt (#3111)
- Unwrap extra StandardError wrapping (#3120)
- Correct CreateSpan call args (#3202)
- Remove singleton tracer to prevent stale metadata (#3208)

### 💼 Other

- Modify editorSuggestWidget background color (#3086)
- Sync autocomplete function suggestions with BE (#3079)
- Highlight template variables in query editor (#3092)
- Reduce width allotted to resources section (#3118)
- Remove FE feature flag check (#3116)
- Global feature flag toggle (#3121)
- Insights SQL Agent (#3007)
- Increment function version on syncs. (#3114)
- Always add a status when enqueueing root spans (#3140)
- Update event runs API endpoint (#3144)
- Default trace preview feature flag call to true (#3143)
- Fix suggestions widget overflow (#3135)
- Bump pnpm 8 to pnpm 10 and add minimumReleaseAge setting (#3122)
- Confirm pnpm 10 install behavior (#3169)
- Redo cron using queues (#2847)
- Skip docker hub release for beta tag (#3182)
- Cron Health Checks (#3201)

## [1.12.0] - 2025-10-01

### 🚀 Features

- Add OpcodeStepFailed support (#2992)

### 💼 Other

- Refactor GraphQL queries to use latestSyncedConfig (#3017)
- Move query management from LS to BE (#3002)
- Add select * from events query template (#3033)
- Add link to collect feedback (#3016)
- Display results history limit (#3036)
- Remove GQL prefix from visible error messages (#3037)
- Publish on localhost via realtime forwarding (#3023)
- Revise query templates (#3038)
- Update docs links (#3034)
- Make Insights panels resizable, introduce Resizable (#3060)

## [1.11.11] - 2025-09-19

### 🐛 Bug Fixes

- Only trigger 1 onFailure call on parallel step failure (#2907)
- Problem with multiple event trigger expressions (#2951)

### 💼 Other

- Templates tab pane refresh (#2921)
- Improve function configuration table tooltip (#2903)
- Temporarily hide docs side panel (#2926)
- Clean up Insights lint (#2929)
- Update templates (#2862)
- Copy template name to tab name (#2930)
- Temporarily hide query example links (#2935)
- Add permanent home (icon) tab (#2934)
- Indicate row limit on results (#2937)
- Temporarily hide .csv download button (#2941)
- Add missing border between query editor and results section (#2950)
- Check loading state before error state (#2952)
- Remove workspace_id from templates (#2933)
- Remove account_id from templates (#2936)
- Add keyword-based autocomplete (#2867)
- Move insights below events in monitor nav section (#2962)
- Prevent query editor perf issue via memoization (#2973)
- Point Insights feedback link to support page (#2985)
- Tabs]: Prevent active tab switch when closing a tab (#2997)
- Move query history to ephemeral react state (#3000)
- Render prettified JSON data in results table (#2996)
- Change new insight verbiage to new query (#3006)
- Extend query template list (#3005)
- Prevent results table overscroll issue (#3008)
- Mark Insights as a beta feature (#3010)

## [1.11.12] - 2025-09-02

### 💼 Other

- Pass workspaceID (env) to BE when issuing SQL query (#2901)
- Support cmd+enter shortcut to issue an insights query (#2909)

## [1.11.9] - 2025-08-28

### 💼 Other

- Support templates, saved queries, and query history (#2839)
- Integrate query fetching with BE (#2854)

## [1.11.8] - 2025-08-26

### 💼 Other

- Support reusable Tabs component dynamic variant (#2825)

### 📚 Documentation

- Remove `--yes` from `vercel link` (#2853)

### ⚙️ Miscellaneous Tasks

- Sort OtelSpans deterministically in GraphQL (#2859)

## [1.11.6] - 2025-08-21

### 💼 Other

- Add tests for batching events + if triggers (#2817)
- Refactor Insights ahead of multi-tab support (#2807)
- Source paused status from new getter, migration from lock key (#2806)
- Add conditional expression for batching eligibility (#2818)

## [1.11.4] - 2025-08-05

### 💼 Other

- Introduce state context (#2780)
- Add (mocked) results table (#2773)

## [1.11.3] - 2025-08-04

### 💼 Other

- Add (basic) query editor (#2761)

## [1.11.2] - 2025-07-31

### 💼 Other

- Introduce nav item and route (#2757)

## [1.11.0] - 2025-07-25

### 💼 Other

- Increase branch environments table page size 5 -> 10 (#2734)
- Add branch key disclaimer (#2742)
- Update actions UX (#2731)
- Various design updates (#2740)
- Fix table row browser inconsistency (#2747)

## [1.10.0] - 2025-07-21

### 💼 Other

- Allow filtering by status (#2717)
- Add debug run and session ids to scheduler and rerun from step resolver (#2722)

## [1.8.3-beta.1] - 2025-07-15

### 💼 Other

- Fix webhook intent url (#2677)

## [1.7.0] - 2025-06-16

### 💼 Other

- Show FunctionConfiguration in dev server (#2546)

## [1.6.0] - 2025-05-19

### 💼 Other

- Return syscode error when request > 2h (#2430)

## [1.5.12] - 2025-05-09

### 🚀 Features

- Add some indexes to increase performance of self hosted dashboard using postgres (#2318)

### 💼 Other

- Log a warning if waitForEvent.If CEL expression fails validation (#2405)

## [1.5.3] - 2025-03-14

### 💼 Other

- Trace run start/first step alignment fix & finalization (#2242)
- Break out cancel run button (#2244)

## [1.5.0] - 2025-03-06

### 💼 Other

- Swap in split button, input fixes (#2225)

## [1.4.7] - 2025-02-24

### 💼 Other

- New run AI traces (#2151)
- Refer to metrics' "delay" instead of "freshness" (#2193)

## [1.4.6] - 2025-02-05

### 💼 Other

- Add ability to archive events (#1940)

## [1.4.4] - 2025-01-28

### 💼 Other

- Ensure wildcards are matched during execution (#2117)

## [1.4.2] - 2025-01-22

### 💼 Other

- Improve auto discovery to reduce noise and excessive polling (#2094)

## [1.4.0-beta.1] - 2025-01-16

### 💼 Other

- Also set has ai in span attributes on function finished (#2037)
- Add release documentation (#2040)
- Support ai indicators and rerun from step in cloud (#2038)

## [1.3.3] - 2024-12-17

### 🐛 Bug Fixes

- Update url for useFeatureFlags hook to support non default port (#2015)

## [1.3.2] - 2024-12-05

### 💼 Other

- Pass ai gateway step op code up to ui (#1998)
- Add ai indicator for runs and add decoration to runs ui (#2001)

## [1.2.0] - 2024-11-19

### 🐛 Bug Fixes

- Set concurrency in expressions.NewAggregator (#1910)
- Race condition in LoadEventEvaluator (#1911)
- Check for nil persistence interval in single node mode (#1965)

### 💼 Other

- Metrics latest charts (#1838)
- Lift QueueItem into common queue struct and interface (#1872)
- Use skippable queries in metrics (#1901)
- Fix timeouts duration parsing (#1968)

## [1.1.1-beta.1] - 2024-10-03

### 💼 Other

- Always set inner queue name (i.Data.QueueName) during Enqueue() (#1817)

### 🛡️ Security

- Upgrade next-clerk (#1829)

## [1.1.0] - 2024-09-26

### 💼 Other

- Wire up function run and step throughput charts (#1792)
- Rounding, function list fetching, etc (#1809)
- Tooltip fixes (#1810)

## [1.0.1] - 2024-09-22

### 💼 Other

- Metrics dashboard failed function chart and failed function rate list (#1763)

## [1.0.0] - 2024-09-20

### 💼 Other

- Track started batches (#1762)

## [0.29.9] - 2024-09-16

### 💼 Other

- Skip non-default partitions in Scavenge (#1717)
- Add a new user survey link (#1740)
- Metrics dashboard added, function status chart wired up (#1709)

## [0.29.7] - 2024-09-11

### 💼 Other

- Persist payload customizations through tab changes (#1697)

## [0.30.0-beta-2] - 2024-08-19

### 🐛 Bug Fixes

- Worker capacity should be run regularly outside of the lease (#1666)

### 💼 Other

- Graceful pause deletion (#1617)
- Dev server ia nav (#1642)

## [0.29.4] - 2024-08-09

### 💼 Other

- Add circular font to the dashboard (#1544)
- Some vercel fixups (#1561)
- New ia nav wip (#1584)
- The rest of the side navigation widgets (#1605)
- Design qa fixes (#1607)
- Truncate when more than 2 function triggers (#1613)
- New ia top nav (#1620)
- Ia nav functions (#1627)
- A handful of ui/ux fixes (#1662)

## [0.29.2] - 2024-07-03

### 💼 Other

- New integrations (#1421)
- Maze survey cannot be clicked when overlay is shown (#1521)
- Fix up design tokens (#1522)
- Move to proper alert (#1523)
- Some more vercel design cleanup (#1534)
- Implement state size limit (#1543)

## [0.29.1] - 2024-06-07

### 💼 Other

- Trace tweaks (#1371)
- Swap in new date picker for function replay range (#1367)
- Add helper method in Span to store SDK resp (#1403)
- Fix event stream pagination by using ulid as cursor (#1409)
- Date range picker (#1391)

## [0.29.0] - 2024-05-20

### 🐛 Bug Fixes

- Executor.Schedule returns ErrFunctionSkipped when a function is paused (#1337)

### 💼 Other

- Fix consistent sort order for apps (#1324)
- Support function pausing (#1330)
- Make code editor support tailwind/system color schema and overrides (#1357)

## [0.28.0] - 2024-05-03

### 💼 Other

- Minor tweaks for span annotations (#1280)

## [0.26.6] - 2024-04-03

### 💼 Other

- Incorporate tracer into codebase for user traces (#1237)
- Fix empty event payloads in UI due to resolver bug (#1258)

## [0.26.4] - 2024-03-13

### ⚙️ Miscellaneous Tasks

- Remove IGassmann from CODEOWNERS (#1227)

## [0.26.3] - 2024-03-05

### 🚀 Features

- *(dashboard)* Switch to synchronous onboarding flow (#1173)

### 🐛 Bug Fixes

- *(dashboard)* Invite members form is skipped (#1192)

### 💼 Other

- *(dashboard)* Cache revalidate conflicts with clerk (#1190)
- *(dashboard)* Infinite user reload on set-up pages (#1191)
- Ensure `cancelOn` works with `event.ts` in the future (#1206)
- Fix custom event IDs are not idempotent (#1202)

### ⚙️ Miscellaneous Tasks

- *(dashboard)* Enable source maps in production (#1194)
- *(dashboard)* Don't hide source maps in production (#1195)

## [0.26.2] - 2024-02-27

### 🐛 Bug Fixes

- *(dashboard)* Cache isn't invalidated when switching organizations (#1174)
- *(dashboard)* Hover state outline on org and user dropdown (#1180)

### 💼 Other

- Add additional charts to the UI for showing backlog and the number of steps currently running (#1170)

### 🚜 Refactor

- *(dashboard)* Remove legacy password-reset pages (#1181)

### ◀️ Revert

- "Remove buttons for removing org memberships and user accounts tempora…" (#1184)

## [0.26.0] - 2024-02-22

### 🚀 Features

- *(dashboard)* Improve auth for URQL (#1076)
- *(dashboard)* Retry unauthenticated requests (#1079)
- *(dashboard)* Organization list page (#1081)
- *(dashboard)* "Create Organization" page (#1089)
- *(dashboard)* "Organization Settings" page (#1090)
- *(dashboard)* Force users to have an active organization (#1087)
- *(dashboard)* Use accountID of organization (#1088)
- *(dashboard)* Redirect to sign-in with an error after retries of unauthenticated errors (#1096)
- *(dashboard)* OrganizationDropdown (#1116)
- *(dashboard)* UserDropdown (#1117)
- *(dashboard)* OrganizationActiveLayout
- Move anonymous ID to cookies (#488)
- *(auth)* Link to new sign-in and sign-up URLs (#499)
- *(docs)* Add [object, object] type to concurrency (#582)
- *(docs)* Add FAQ for endpoint authentication (#585)
- *(docs)* Replay functions guide (#596)
- *(website)* Enable website workflow file
- *(website)* Add Algolia variables and secrets

### 🐛 Bug Fixes

- *(api)* FunctionRun.events returning array with null item for non-event run (#1097)
- Cron with trailing space fail to parse (#1108)
- *(dashboard)* Don't query GetAccountSupportInfoDocument when signed out
- *(dashboard)* User not redirected after creating org
- *(dashboard)* Invite members form style
- *(dashboard)* Launchdarkly server-side context
- *(dashboard)* Using LD before its initialization completed (#1126)
- *(docs)* Remove plural form for event trigger (#350)
- *(sign-up)* Input text is invisible (#403)
- *(docs)* Deploy link breaks Dev Server (#442)
- Broken links (#475)
- *(patterns)* Incomplete sentence (#541)
- Docs page open graph image for homepage
- *(website)* Website - On Deployment workflow
- *(website)* Next-mdx-remote types
- *(website)* On deployment workflow
- *(dashboard)* Can't invite members during second organization creation (#1149)

### 💼 Other

- Add AppID as a field for state identifier (#1082)
- *(dashboard)* Move /support into (organization-active)
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
- *(dashboard)* Account setup redirect url (#1146)
- Fix account setup redirection (#1147)

### 🚜 Refactor

- *(dashboard)* Move SplitView into (auth)/
- *(dashboard)* Move into (organization-active)/
- *(dashboard)* Remove unused import
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

## [0.25.2] - 2024-01-26

### 🐛 Bug Fixes

- *(dashboard)* Account setup race condition (#1064)

### 💼 Other

- Remove limit line that may be rendered as 1 (#1065)

## [0.25.1] - 2024-01-25

### 🐛 Bug Fixes

- *(dashboard)* Check for accountID on account setup (#1062)

## [0.24.0] - 2024-01-17

### 🚀 Features

- *(dashboard)* Rename View All Logs button to View All Runs (#945)
- *(dashboard)* Send trace headers to API (#950)
- *(dashboard)* Retry when unauthenticated (#954)
- *(dashboard)* Don't sample replays when not erroring (#960)
- *(dashboard)* Sentry user identification (#959)
- *(dashboard)* Send trace headers to API (#970)

### 🐛 Bug Fixes

- *(dashboard)* Don't initialize launchdarkly client multiple times (#966)

### 💼 Other

- *(dashboard)* Occasional 500 errors (#940)
- *(dashboard)* TimeInput's suggestion isn't clickable (#942)
- User cannot scroll when payload is large. (#952)
- Previous period use previous year if Dec (#971)
- Replay time range input issues (#1016)
- Add function to retrieve total number of items in the function (#1027)

### 🚜 Refactor

- *(dashboard)* Encapsulate environment context (#961)

### ⚙️ Miscellaneous Tasks

- *(postinstall.ts)* Use `package.json` version instead of an env variable (#1013)

## [0.23.1] - 2023-12-14

### 🚀 Features

- *(dashboard)* Add links to replay docs (#921)
- *(dashboard)* Poll replays (#920)

### 🐛 Bug Fixes

- *(dashboard)* Webhooks don't save (#928)
- *(dashboard)* Report errors to Sentry (#917)

## [0.23.0] - 2023-12-07

### 🚀 Features

- *(dashboard)* Function replay (#767)
- *(dashboard)* Make selected status option more explicit (#890)
- *(dashboard)* Improve TimeRangeInput (#892)
- *(dashboard)* Set up server-side LaunchDarkly

### 🐛 Bug Fixes

- *(dashboard)* Replay's Total Runs column (#889)
- Revert Node.js and pnpm version upgrade (#898)
- *(dashboard)* Sentry doesn't send logs when unauthenticated (#900)

### 💼 Other

- Use client-side queries to decouple requests (#897)

### 🚜 Refactor

- *(dashboard)* Remove unused GraphQL fields (#876)
- *(dashboard)* Disable launchdarkly streaming

### ⚙️ Miscellaneous Tasks

- *(dashboard)* Run GraphQL Codegen during build (#879)
- Update @clerk/nextjs to latest
- Update @sentry/nextjs to latest
- Update Next.js to latest
- Add @launchdarkly/node-server-sdk

## [0.22.0] - 2023-11-27

### 🚀 Features

- *(dashboard)* Hook up function config (#857)

### 🐛 Bug Fixes

- *(dashboard)* Support form not working (#863)
- *(dashboard)* Don't query nextRun field (#867)

### 🚜 Refactor

- *(dashboard)* Remove feature flag (#870)

## [0.21.1] - 2023-11-15

### 🚀 Features

- *(dashboard)* Function config (#835)

### 🐛 Bug Fixes

- *(dashboard)* Function slugs aren't URL friendly (#841)
- *(dashboard)* Empty page on sign-out (#848)

### 💼 Other

- Add query param as 3rd argument for webhook transform (#839)

## [0.21.0] - 2023-11-13

### 🚀 Features

- *(dashboard)* Remove versions tab (#824)
- *(dashboard)* Display error message on user creation (#830)

### 🐛 Bug Fixes

- *(dashboard)* "Send Event" doesn't display created event
- *(dashboard)* Chart is empty if `queued` is empty list (#816)
- *(dashboard)* Environment links don't work from a function page (#823)
- *(components)* Don't render multiple html and body tags

### 💼 Other

- Add create environment button to /env (#806)
- Display recent tickets. Switch to Plain threads. (#781)
- Add actions for syncing main with next branch (#807)

### ⚙️ Miscellaneous Tasks

- Update Next.js to latest
- Update storybook to latest
- Update @sentry/nextjs to latest
- Update @clerk/nextjs to latest
- Remove deprecated experimental.appDir config
- Remove deprecated viewport option

## [0.20.0] - 2023-11-02

### 🚀 Features

- *(functions)* Remove icon next to chart titles (#694)
- Use Tailwind CSS default font sizes
- Use Tailwind CSS default border radius
- Add fallback mono fonts
- Add base.json file (#706)
- *(dashboard)* Rename function dashboard charts (#710)
- *(dashboard)* Move time selector within main area (#711)
- *(dashboard)* Move metrics to the top (#712)
- *(dashboard)* Use short number format for metrics (#713)
- *(dashboard)* Display chart's loading state when switching time (#727)
- *(dashboard)* Display end states in function runs chart (#728)
- *(dashboard)* Make chart legend colors consistent (#729)
- *(dashboard)* Function dashboard chart legends (#730)
- *(dashboard)* Add beta badge to function dashboard (#741)
- *(dashboard)* Display concurrency limit in chart (#744)
- *(dashboard)* Function replay modal (#763)

### 🐛 Bug Fixes

- *(dev-server-ui)* GraphQL codegen (#701)
- *(dashboard)* Tooltip icon (#740)
- *(dashboard)* Function runs chart when selected 60 minutes (#739)
- *(components)* Ambiguous root paths (#742)

### 💼 Other

- *(tailwind)* Align content lookup
- *(tailwind)* Don't override existing colors
- *(tailwind)* Remove unneeded custom grid templates
- *(tailwind)* Remove unused grid templates
- Type check Tailwind config file
- Add chart for the number of sdk requests being made from the executor (#724)
- Create webhook intent page (#780)

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

### 🛡️ Security

- Add security scan for golang (#760)

## [0.19.0] - 2023-10-11

### 💼 Other

- Add granularity to usage opts and deprecate period (#682)

## [0.18.1] - 2023-10-09

### 💼 Other

- Use time.Duration instead of string for granularity (#676)

## [0.18.0] - 2023-10-06

### 💼 Other

- Replace MaxBatchSize with DefaultBatchSize to make it more configurable (#651)
- Add API structure for timeseries related data for metrics (#668)

## [0.17.0] - 2023-09-26

### 💼 Other

- Add tests and additional check for invalid URIs (#642)

## [0.15.6] - 2023-07-28

### 💼 Other

- Add telemetry setup (#530)
- Add back tracing (#537)

## [0.15.5] - 2023-07-20

### 💼 Other

- Remove tracer for now (#528)

## [0.15.0] - 2023-07-12

### 💼 Other

- Add batch related configs (#465)
- Add actions to be truncated and comment it out for now to (#475)
- *(ui)* Switch to pnpm (#478)
- *(ui)* Migrate from Vite to Next.js (#479)
- *(ui)* Pin dependencies (#480)
- Allow batching to work (#477)
- Add Apps Page (#490)

## [0.14.1] - 2023-06-14

### 💼 Other

- Allow `Event` list in state input for batching (#450)

## [0.13.0] - 2023-05-01

### 💼 Other

- Refactored the UserError function to handle arbitrary response (#435)

## [0.13.1] - 2023-03-31

### 🚀 Features

- Automatically display newly created events (#386)

### 📚 Documentation

- *(readme)* Correct typos (#403)
- *(contributing)* Add contributing guide (#404)

## [0.10.0] - 2023-02-08

### 🐛 Bug Fixes

- Skip saving response if generator res is empty
- Executor re-queuing edges as sleeps after a sleep is found
- `waitForEvent` op erroneously setting stack before returning
- Generator SDK returning 200 fails to save final step data (#383)

## [0.5.4] - 2022-08-16

### 💼 Other

- Matrix for `go test`

## [0.4.0] - 2022-07-05

### 💼 Other

- Add basic Redis state implementation

## [0.3.0] - 2022-06-09

### 💼 Other

- Drag in open source executor

## [0.1.0] - 2022-03-10

### 💼 Other

- Support scopes defined in actions

## [0.0.1] - 2021-07-09

<!-- generated by git-cliff -->
