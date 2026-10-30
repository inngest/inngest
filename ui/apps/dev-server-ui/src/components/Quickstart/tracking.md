# Quickstart telemetry

Uses the existing `useTracking` → `POST /v0/telemetry` → CLI telemetry pipeline.
All events are named `cli/onboarding.action` with `flow: "quickstart"` and
`version: 1`. Filter on these fields to distinguish this flow from legacy
onboarding. No additional analytics SDK, identity, backend endpoint, or UI is added.
Existing CLI telemetry metadata and `DO_NOT_TRACK` behavior remain in effect.

| `action`                      | When                                                                                                       | Additional properties                                                 |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `viewed`                      | Quickstart has loaded apps, once per browser session/server                                                | `entry_stage`: `setup`, `run`, or `complete`                          |
| `agent_prompt_copied`         | Prompt successfully written to clipboard                                                                   | None                                                                  |
| `manual_install_copy_clicked` | User clicks SDK command copy                                                                               | `language`, `package_manager`                                         |
| `docs_clicked`                | User opens a guide from Quickstart                                                                         | `guide`: `overview`, `mcp`, or `manual`; `language` for manual guides |
| `app_ready`                   | Quickstart observes a connected app with an invokable function, before completion, once per session/server | None                                                                  |
| `invoke_clicked`              | Valid payload accepted by the UI and invocation submitted, protected against duplicate submission          | `path`: `manual`                                                      |
| `first_run_completed`         | Provider verifies the exact run for an active onboarding attempt as Completed, once per session/server     | `path`: `agent` or `manual`                                           |

## Reading the numbers

- Reach: `viewed`, split by `entry_stage`. Exclude `complete` for onboarding conversion; `run` users arrived with a usable app.
- Interest: prompt copies versus manual install-command clicks and manual guide clicks. Both paths are visible and a user can try both; these are not mutually exclusive cohorts.
- Progress: `app_ready` and `invoke_clicked`; completion: `first_run_completed`. Use the existing backend `device_id` for device-level comparisons where present. No new browser/session identifier is introduced.
- Prompt copying measures intent, not editor connection, prompt pasting, or agent execution. Completion `path` reflects the active UI handoff/attempt, not independent proof of which tool invoked the function. Editor identity is unknown.
- Manual copy measures a click, not successful clipboard writing or SDK installation. App ready measures usable functions, not a functionless app sync. Invoke clicks measure attempts, not acceptance or success.
- Historical runs can restore returning-user state but do not emit new completion events. Completion requires an active agent handoff or manual UI invocation attempt. Invocations started elsewhere without either are not attributed as new conversions.
- Events are best effort. Session storage deduplicates view/milestone effects across polls, remounts, and reloads. If storage is unavailable, deduplication lasts only for the current component mount. Telemetry failures never gate onboarding.

Only categorical action metadata is added. Prompt text, code, payloads, outputs,
server URLs, app/function names, run IDs, and exception messages are not sent.
