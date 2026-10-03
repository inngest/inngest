# MCP tool annotations

Annotations describe effects, not HTTP methods. In particular, Insights queries
and waiting for a sandbox process use POST but do not mutate application state.
New methods default to destructive and open-world until reviewed.

| Methods | Read only | Destructive | Open world | Reason |
| --- | --- | --- | --- | --- |
| Health, account/environment/app/function/run/event/trace lists and reads, sandbox metadata/files/logs/process output reads, Insights tables/schemas/queries, experiment/session reads | Yes | No | No | Read existing Inngest-managed data without changing application state. |
| QueryInsightsPrompt | Yes | No | Yes | Generates SQL with an external model using the prompt and schema context; does not execute a requested mutation. |
| CreateEnv, CreateWebhook, CreateScore, CreateSandboxSnapshot | No | No | No | Add managed records or snapshots without replacing existing resources. |
| PatchEnv, CancelRun, DestroySandbox, PauseSandbox, DeleteSandboxSnapshot, SignalSandboxProcess, WriteSandboxFile | No | Yes | No | Change, stop, delete, or overwrite existing managed resources. |
| CreateSandbox | No | No | Yes | Creates a sandbox that may fetch an external image and run its entrypoint. |
| ResumeSandbox, ExecSandbox, StartSandboxProcess | No | Yes | Yes | Runs user-selected code that can change files and call external services. |
| SyncApp | No | Yes | Yes | Contacts the supplied application URL and changes registered functions. |
| SendEvent, InvokeFunction, Rerun | No | Yes | Yes | Can run arbitrary application code, including destructive external actions. |

Only reads are marked idempotent. A repeated write may retrigger application
behavior even when the HTTP method is PUT or DELETE. These hints never replace
authorization. The Cloud host publishes OAuth scopes separately.
