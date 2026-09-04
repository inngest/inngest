# Run canvas — running log

Terse, append-only, newest at the bottom. One entry per item from `PROMPT.md`: what changed, how it
was verified, what is uncertain, what was deferred.

`HANDOVER.md` is the cold-start doc and `todo.md` is the decision log D1–D65 from the lineage work.
This file is the log of the visualisation work that followed.

---

## Item 0 — Orientation

**Dev server.** `pnpm run --filter @inngest/dev-server-ui dev:vite` running in the background for
the whole session. Ports 5173 and 5174 were already taken, so it is on **:5175**. `dev:vite` rather
than `dev` deliberately: `dev` also spawns a graphql-codegen watcher that needs the Go package tree.

**Read first**, before writing anything: `graph.ts` + `graph.types.ts`, `Timeline.tsx` /
`TimelineBar.tsx` / `TimelineBar.types.ts`, `TimeBrush.tsx` / `TimelineHeader.tsx`,
`RunDetailsV4.tsx`, `runDetailsUtils.ts`, `utils/timing.ts`, `__fixtures__/README.md`,
`HANDOVER.md`, `lessons.md`.

### Findings that changed the plan

- **`calculateBarPosition` and `calculateDuration` are already separate** (`utils/timing.ts:37,64`),
  and `calculateBarPosition` has exactly one caller (`Timeline.tsx:499`). So item B's invariant —
  only the axis compresses, every reported duration stays wall-clock — can be made structural rather
  than a rule someone has to remember: only the position function ever learns about the scale.

- **Item C is half-built already.** `TimeBrush.tsx` (364 lines, 32 assertions) is a general
  drag-range control, and `TimelineHeader.tsx:199` renders a flat status-coloured bar inside its
  `children` slot. The density strip is a histogram dropped into that slot, not a new control, and
  drag-to-set-viewport comes free.

- **Item E's span-links shortcut is dead.** Checked because the brief flagged it as unconfirmed and
  cheap. `FollowsFrom` *is* written — `executor.go:3419`, `:3601`, `:4029` and ~10 more — but only
  for pause spans and lifecycle queue items, i.e. **within** a single run. Nothing writes a link
  between a parent's invoke span and a child's root span. And it would not reach the client anyway:
  `links` is not in the GraphQL schema at all, and `gql.schema.graphql:641` carries the literal
  comment `# links should be here`. Exposing it is a Go change, which is a shared surface and a
  stop-and-ask, so item E uses only the read-only route the brief lists as its fallback
  (`childRunID`, `trigger` event ids, `sendEvent` `output.ids`, `/v2/events/{id}/runs`).

- **The gallery needs no platform work.** dev-server-ui aliases `@inngest/components` straight to
  source (`vite.config.ts`) and JSON imports already work in the canvas tests, so a route can render
  the real `Canvas` and `Timeline` from a committed fixture with no GraphQL. `Canvas` takes only
  `{trace, runID, getTrigger?}`. Being in dev-server-ui it is Cloud-excluded by construction —
  Cloud is a separate app (`ui/apps/dashboard`).

- **No new dependency is needed** for any item. Sparklines and the density strip are inline SVG.

### Smaller things noticed, not yet acted on

- `traceRollup` runs **twice** per render — `RunDetailsV4.tsx:84` and `:136` — each
  `structuredClone`ing the whole trace. Hoist when item C adds a third consumer.
- `TimelineBar.tsx:871` sets row height twice, as `h-7` and as an inline
  `height: ${ROW_HEIGHT_PX}px`. They agree today; the class is the stale one. Fix in item B2.
- `__fixtures__/README.md` documents 19 of the 38 fixtures. `v4deep3.json` has no importer.
- `useStepSelection` / `stepSelectionEmitter` have no test coverage. Item D adds the first.
- The Dev Server's `function_runs` table has no primary key and no indexes (`schema.sql:38-51`), so
  event→runs is a scan locally. Recorded as a follow-up; not changed, because Go stays read-only.

### Decisions taken up front

Recorded here rather than asked, per the brief's §5.

- **Batched runs: one start node, not N.** The event node carries `batchSize` and draws as the
  existing offset-card stack; it becomes expandable to the N events. One run is one beginning.
- **Cancelled is a third state.** `status.cancelled` tokens, distinct from both success and failure.
  A step still running when the run was cancelled reports `(interrupted)` where a duration would
  be — Terraform's `(known after apply)` rule: a token in place of the value, never a silent gap
  that reads as "nothing happened".
- **Restyle the timeline in place**, with no Cloud-gated second presentation. The branch has no
  upstream and is never pushed, so there is no production exposure, and two presentations of the
  same component is real ongoing cost.
- **Elastic axis on by default**, but only for gaps clearing both a relative and an absolute floor,
  and always drawn as a visible break.

Nothing deferred. Nothing uncertain beyond the two checks item E still owes (whether `sendEvent`
`ids` are internal ULIDs).
