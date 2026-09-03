## [v0.16.1] - 2026-08-28

### 🐛 Bug Fixes

- Send environment on signing-key API requests (#252)
- Hydrate API-backed invocation state (#251)

### ⚙️ Miscellaneous Tasks

- *(connect)* Adding tests for lease nack (#248)
## [v0.16.0] - 2026-07-20

### 🚀 Features

- [**breaking**] Always enable authenticated syncs (#242)
- [**breaking**] Strip details in unauthed response (#243)
- [**breaking**] Disable unauthed syncs by default (#244)

### 🐛 Bug Fixes

- Signing key not required in cloud mode (#245)

### ⚙️ Miscellaneous Tasks

- Change 0-major semver logic (#239)
- *(release)* V0.16.0 (#240)

### 🛡️ Security

- Bump golang.org/x/net from 0.52.0 to 0.55.0 (#241)
## [v0.15.3] - 2026-06-18

### 🐛 Bug Fixes

- Update github.com/inngest/inngest to v1.19.3 (#238)

### ⚙️ Miscellaneous Tasks

- *(release)* V0.15.3 (#224)
## [v0.15.2] - 2026-06-17

### 🚀 Features

- *(connect)* Configure heartbeat tolerance (#234)

### 🐛 Bug Fixes

- Add nil guard to Trigger.MarshalJSON to prevent panic (#212)
- Ignore stale connect lease acks (#216)
- *(connect)* Stale Connect websocket writes (#220)
- *(connect)* Retire generation on ACK failure (#226)
- *(connect)* Model drain and closing lifecycles (#231)
- Mark event parse errors no-retry (#230)
- *(connect)* Harden websocket lifecycle boundaries (#233)
- Use range assertion for requestCount (#232)
- Buffer initialConnectionDone and notifyConnectDoneChan to prevent deadlock (#221)
- Assert we lock ops when checkpointing (#236)

### 💼 Other

- Improve SDK logging (#217)
- Fix reporting synchronous durable endpoints runs (#219)

### 🚜 Refactor

- *(connect)* Add websocket generation lifecycle (#228)
- *(connect)* Gate writes by lifecycle phase (#229)

### ⚙️ Miscellaneous Tasks

- *(release)* Adopt git-cliff release flow (#223)
- *(middleware)* Export middleware package (#225)
## [v0.14.1] - 2025-10-27

### 🚀 Features

- Step failed opcode (#183)
## [v0.8.0] - 2025-03-06

### 💼 Other

- Convert timeout duration types to strings for (#74)
## [v0.5.0] - 2023-11-02

### 💼 Other

- Add batch config for function opts (#10)
## [v0.1.1] - 2021-05-11
