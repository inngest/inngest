# Runs AI assistance

Use AI sits beside Show search in Cloud Runs. Client buttons expand a copyable command block and short sign-in note, without numbered steps. Codex add and login commands copy together; the setup guide opens the Cloud connection documentation in a new tab. The MCP documentation link also opens a new tab. The local Dev Server is unchanged. There is no feature flag.

The scrollable preview shows exactly what Copy prompt copies: API filter IDs, UTC boundaries frozen when the popover opens, and full investigation instructions. Reopening refreshes relative time bounds. Unsupported filters block copying. Clipboard failures expose selectable full text. Both copy actions share one handler.

## Tracking

Reuse the dashboard transport and existing user/account identity. Event: `app/runs.ai_assist.action`, version `1`. Each event includes `surface` (environment_runs or function_runs) and `template_version` (basic_v1).

| Action | Meaning | Additional data |
| --- | --- | --- |
| viewed | Entry rendered, once per mount | None |
| opened | Popover opened | None |
| setup_clicked | Client instructions expanded | client |
| setup_copied | Setup text copied | client |
| setup_copy_failed | Setup copy failed | client |
| prompt_copy_clicked | Copy prompt activated | None |
| prompt_copied | Prompt copied | None |
| prompt_copy_failed | Prompt copy failed | failure_category |
| guide_clicked | Guide opened | guide: connection |

Client values: claude_code, codex_cli, other. Failure categories: permission_denied, unavailable, other. No prompt contents, filter values, URL visit IDs, or session storage are used in tracking. Count total actions and distinct users/accounts over the same reporting period. Setup clicks and copies do not prove a successful connection.

## Validation

Run `pnpm exec vitest run src/components/Runs` and `pnpm exec tsc --noEmit` from `ui/apps/dashboard`. Run the MCP setup and RunsPage tests from `ui/packages/components`. In staging, verify a known run-finding task, clipboard fallback, and analytics delivery.
