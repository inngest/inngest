# Changelog

## [v0.27.0] - 2024-04-16

### Added
- Added support for throttling function runs
- Added support for function timeouts
- Added support for new metrics

### Changed
- Improved support for tracing
- Improved SDK registration handling

### Fixed
- Various dev server fixes


## [v0.26.0] - 2024-02-22

### Added
- Added support for batching in the dev server

### Fixed
- Various dev server fixes, battery life improvements


## [v0.25.0] - 2024-01-25

### Added
- Added support for per-step errors
- Added support for StepRun opcode

## [v0.24.0] - 2024-01-17

### Added
- Added debounce timeouts
- Added multiple concurrency keys
- Added step invocation
- Added improvements to apps, registration

### Fixed
- Removed goroutine handling of jobs, leading to FIFO guarantees within the job queue itself

## [v0.17.0] - 2023-09-26

### Added
- A new stream page, allowing you to see all events and function runs in an easier view
- Added a new code block to improve the UI for function outputs, step outputs, and events
- Added support for V3 of the TS executor
- Added a new execution lifecycle backend, and new history implementations

### Fixed
- Fixed dev server payload sizes, now conforming to cloud usage
- Improved auto-scanning and local development flows

## [v0.16.7] - 2023-08-19

### Added
- Added lifecycle handlers
- Added history lifecycle manager
- Added note on disabling auto-discovery on boot

### Changed
- Changed max event size dev server can ingest

## [v0.16.6] - 2023-08-19

### Added
- Added initial lifecycle interface to track execution lifecycles of steps

### Fixed
- Fixed tracking of function completions from earlier regresion due to lifecycle changes

## [v0.16.0] - 2023-07-31

### Added
- Added support for custom step backoff stragies specified within function code
- Added support for scheduling functions based off of a future `ts` unix-ms timestamp within events
- Added support for testing AWS Lambda functions locally without an AWS Gateway

## [v0.15.3] - 2023-07-14

### Added
- Added the ability to prevent polling of SDKs for updates.  Any function updates must
  be triggered by re-adding your app via the UI.

### Fixed
- Ensures new functions appear within apps automatically

## [v0.15.0] - 2023-07-12

### Added
- Added apps to the dev server.  You can now submit multiple app endpoints via the UI,
  and see apps and their statuses/errors from a tab.

### Fixed
- Fixed open-source rate limit keys
- Fixed expression concatenation with undefined event data 


## [v0.14.5] - 2023-06-22

### Fixed
- Fixed open-source rate limit keys
- Fixed expression concatenation with undefined event data 

## [v0.14.0] - 2023-06-09

### Added
- Added rate limiting to OSS dev server
- Updated function definitions for V2 function config

## [v0.13.3] - 2023-05-19

### Changed
- Fixed local development concurrency when functions had no IDs specified

## [v0.13.0] - 2023-05-03

### Added
- Moved to the same queue and state store as the cloud, implementing:
  - Concurrency
  - State management
  - Partitioned queues
- Also added improved support for history, functions local testing, etc

## [v0.7.0] - 2022-11-10

### Added

- Executor and state support for generator functions!  This allows you to build
  multi-step functions using the SDK.  Each step is retried individually, and you
  can sleep, wait for events, etc. within code without writing config.
- Function statuses to function metadata in state
- Cancellations to the executor and fn definitions, allowing you to cancel long-
  running step functions automatically via events.
- Added OnComplete/OnError callbacks to redis state interface
- Added HTTP signatures to externally called functions (eg. via SDK)

### Changed

- Modified the state store interface to support generator steps
- Allowed consuming pauses to store data within a function run's state

## [v0.4.0] - 2022-07-01

### Added

- Added a simple queueing interface to the `execution` package
- Updated the `inmemory` state package to implement the new queue package
- Added expression and cron validation when validating a function
- Added distributed waitgroups to the `state.Manager` interface
- Added the ability for the  `state.Manager` interface to record driver
  responses directly

### Changed (breaking)

- Removed storing output and errors directly from the `state.Manager` interface

### Changed (non-breaking)

- Changed the mechanisms for the dev server and `inngest run` to use distributed
  waitgroups when running functions, and to use the new queue interface.

