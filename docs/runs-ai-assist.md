# Runs AI assistance

The Cloud Runs toolbar includes a Use AI popover. Its client buttons expand short MCP connection instructions in place, with copyable commands or a server URL and a full-setup link. It copies a prompt using account, environment, app, function scope, status, and time filters. It makes no MCP calls and grants no access. The local Dev Server has no AI toolbar entry. There is no feature flag; review the change in staging before deployment.

CEL search, restrictive Run type selections, PAUSED, and unsupported statuses block copying with an explanation. App identities resolve from dashboard metadata; function IDs follow the existing Runs REST adapter. The preview uses a short natural-language summary. Copy prompt includes the resolved API filter IDs, exact UTC boundaries frozen at copy time, and full investigation instructions. If clipboard access fails, selectable full text is shown separately from the short preview. Prompt contents and filter values are excluded from feature telemetry.

## Tracking contract

Use the existing dashboard event transport with event name `app/runs.ai_assist.action`, event version `1`, and template version `basic_v1`.

| Action | Meaning | Additional data |
| --- | --- | --- |
| viewed | Entry became visible | None |
| opened | Popover opened | None |
| setup_clicked | Inline client instructions expanded | client: claude_code, codex_cli, other |
| setup_copied | Setup command or server URL copied | client: claude_code, codex_cli, other |
| setup_copy_failed | Setup copy failed | client: claude_code, codex_cli, other |
| prompt_copy_clicked | Enabled copy button activated | None |
| prompt_copied | Clipboard write completed | None |
| prompt_copy_failed | Clipboard write failed | failure_category: permission_denied, unavailable, other |
| guide_clicked | Documentation link activated | guide: connection, troubleshooting, more_ai_resources |

Each event includes `surface` (environment_runs or function_runs) and `visit_id`. Existing user/account identity comes from `useTrackingUser`. Exposure is deduplicated during rerenders and, when session storage is available, across setup-and-return navigation. The internal `aiVisit` parameter correlates that journey; it is never added to external documentation URLs. Analytics failures do not interrupt the feature.

For a seven-day report, count total actions by action and distinct authenticated user/account IDs for adoption. Exposure-to-open rate is distinct openers / distinct exposed users. Open-to-copy rate is distinct users who open and copy in the same visit / distinct openers. Break setup clicks down by client and failures down by category. Copy failure rate is failures / attempts. Exclude internal/test traffic using existing analytics segmentation. Do not treat setup clicks or copies as confirmed MCP connections or completed investigations.

## Review checklist

- Confirm setup opens the selected client and Back to Runs restores filters.
- Verify one known run-finding task against the deployed Cloud MCP schemas and inspect its trace.
- Verify blocked filters, metadata loading/failure, clipboard denial, and keyboard focus.
- Confirm events arrive in the staging analytics pipeline with the fields above and no prompt/filter values.

## Local checks

From `ui/apps/dashboard`:

```sh
pnpm exec vitest run src/components/Runs
pnpm exec eslint src/components/Runs/AIAssist src/components/Runs/Runs.tsx src/routes/_authed/mcp/setup/index.tsx
pnpm exec tsc --noEmit
```

From `ui/packages/components`:

```sh
pnpm exec vitest run src/MCP/Setup.test.tsx src/RunsPage/RunsPage.test.tsx
```
