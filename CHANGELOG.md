# Changelog

## [Unreleased]

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

