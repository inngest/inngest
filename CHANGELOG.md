# Changelog

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

