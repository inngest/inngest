# Inngest API Functions

This package enables using Inngest's step tooling within synchronous API handlers, providing full observability and tracing for your HTTP endpoints.

## Features

- **Full step observability**: Every `step.Run()` call is automatically traced and logged
- **Background checkpointing**: Step data is sent to Inngest in the background without blocking your API
- **Seamless integration**: Use existing `step.Run()` functions in API handlers
- **APM out of the box**: Get metrics, traces, and monitoring for every API endpoint
- **Event triggering**: Send events from API handlers that trigger async workflows
