# Runs AI assistance

A primary outlined AI icon button sits at the end of the Cloud Runs filters, after Run type, with a tooltip and accessible label. Both discovery and failed-run debugging use shared underline tabs for ChatGPT / Codex, Claude, and Other tools, with the first tab selected by default. Production options use an outlined Connect Inngest button linking to the official Inngest plugin or Claude connector listing. Optional CLI setup uses compact, borderless design-system accordions. Existing connections can be reused. Staging and custom endpoints retain endpoint-specific manual setup and do not recommend production-only plugins. Codex add and login commands copy together; setup guides open in a new tab. The “Available MCP tools” link opens the tools section of the MCP documentation in a new tab. The local Dev Server is unchanged. There is no feature flag.

A small muted line below the prompt summarizes the validated Runs filters: account/environment, statuses, apps, optional function/time field, and time range. It is hidden when the prompt is blocked or context is unavailable.

The preview uses the shared read-only text area with normal app typography and a compact scrollable height. It shows exactly what Copy prompt copies: API filter IDs, UTC boundaries frozen when the popover opens, and full investigation instructions. Reopening refreshes relative time bounds. Unsupported filters block copying. Clipboard failures direct users to select and copy from the same text area. Both copy actions share one handler and the design-system CopyButton, including outlined secondary styling and copied-state feedback.

## Tracking

Use the standard Segment registry in `utils/analyticsEvents.ts`. Event: `Assistant Action Performed`, with `feature: runs-ai-assist`, `action`, and an explicit `account_id`; user identity comes from the existing Segment identify call. Each event includes `surface` (`environment_runs`, `function_runs`, or `run_details`) and `template_version` (`basic_v1` or `run_debug_v1`). Debugging also includes `detail_view` (`inline` or `standalone`). This replaces the legacy Inngest event `app/runs.ai_assist.action`; it does not dual-send or backfill historical usage.

| Action | Meaning | Additional data |
| --- | --- | --- |
| viewed | Entry rendered, once per mount | None |
| opened | Popover opened | None |
| setup_clicked | User switches the selected tool tab (default selection does not emit this event) | client |
| setup_opened | Connection listing opened | client: chatgpt_codex or claude |
| setup_copied | Setup text copied | client |
| setup_copy_failed | Setup copy failed | client |
| prompt_copy_clicked | Copy prompt activated | None |
| prompt_copied | Prompt copied | None |
| prompt_copy_failed | Prompt copy failed | failure_category |
| guide_clicked | Guide opened | guide: connection |

Current client values: `chatgpt_codex` and `claude` for production tabs and listing clicks; `codex_cli`, `claude_code`, and `other` for manual setup and command copies. The `chatgpt` value remains accepted in the event type for compatibility but is not emitted by the current UI. A setup_opened event measures a link click, not installation or authorization. Failure categories: permission_denied, unavailable, other. No prompt contents, filter values, URL visit IDs, or session storage are used in tracking. Count total actions and distinct users/accounts over the same reporting period. Setup clicks and copies do not prove a successful connection.

## Validation

Run `pnpm exec vitest run src/components/Runs src/components/RunDetails/RunAIDebug.test.tsx` and `pnpm exec tsc --noEmit` from `ui/apps/dashboard`. Run the MCP setup and RunsPage tests from `ui/packages/components`. In staging, verify a known run-finding task, clipboard fallback, and analytics delivery. In Segment Source Debugger, open each popover and copy its prompt; verify `opened` and `prompt_copied` for each surface, with the expected account and user identity. Confirm downstream warehouse delivery before using Hex counts. Segment requires the existing `VITE_SEGMENT_WRITE_KEY` deployment configuration; these events will no longer appear in Inngest Insights.

## Live verification

Authenticated Cloud MCP smoke testing found that a dashboard alias such as `production` can differ from the MCP environment identifier. Resolve the current environment by its ID against the existing environment metadata query, use its API-facing slug in the prompt, and keep the dashboard label in the readable summary. Missing metadata blocks copying.

After correction, the generated prompt returned the requested 10 newest runs and a readable trace. A separate app/status-filtered request returned only matching failed runs and a failure trace. Returned timestamps and ordering were checked; pagination remained explicitly partial. Only read operations were called. Regression tests cover environment resolution, missing environments, and display/API identity separation.

## Plugin verification (2026-10-23)

The live [Inngest listing](https://chatgpt.com/plugins/plugin_asdk_app_6abe99ad4764819183b795f0d74e5216), version 0.4.1, declares OAuth Cloud MCP for apps, functions, events, runs, traces, and Insights, with bundled coding skills. Its MCP server is `inngest-cloud` at `https://api.inngest.com/mcp`. The existing prompts use the same Cloud server's documented `fetch_account`, `list_runs`, `get_run`, and `get_run_trace` capabilities. No new server tools or prompt changes are needed.

On 2026-10-26, discovery and debugging prompts were also exercised in ChatGPT against the connected Inngest account: discovery returned matching failed runs and a trace; debugging retrieved the selected run and trace and produced an evidence-based diagnosis. A Demo-account mismatch stopped rather than querying another account. Claude’s direct [Inngest connector listing](https://claude.ai/directory/inngest) identifies Inngest, Inc. as publisher and uses the same production OAuth MCP endpoint; authenticated Claude execution was not tested. The [Claude setup guide](https://www.inngest.com/docs/ai-dev-tools/agent-skills#claude-code-plugin) documents the plugin install command and `plugin:inngest:inngest-cloud` sign-in. Manual Codex CLI add/login commands remain available under its accordion.
