<div align="center">
  <a href="https://www.inngest.com"><img src="https://github.com/inngest/.github/raw/main/profile/github-readme-banner-2025-06-20.png"/></a>
  <br/>
  <br/>
  <p>
    Write durable functions in Go via the <a href="https://www.inngest.com">Inngest</a> SDK.<br />
    Read the <a href="https://www.inngest.com/docs?ref=github-inngest-js-readme">documentation</a> and get started in minutes.
  </p>
  <p>

[![GoDoc](https://godoc.org/github.com/inngest/inngestgo?status.svg)](http://godoc.org/github.com/inngest/inngestgo)
[![discord](https://img.shields.io/discord/842170679536517141?label=discord)](https://www.inngest.com/discord)
[![twitter](https://img.shields.io/twitter/follow/inngest?style=social)](https://twitter.com/inngest)

  </p>
</div>

<hr />

# `inngestgo`: Durable execution in Go

`inngestgo` allows you to create durable functions in your existing HTTP handlers or via outbound TCP connections,
without managing orchestrators, state, scheduling, or new infrastructure.

It's useful if you want to build reliable software without worrying about queues, events, subscribers, workers, or other
complex primitives such as concurrency, parallelism, event batching, or distributed debounce. These are all built in.

- [Godoc docs](http://godoc.org/github.com/inngest/inngestgo)
- [Inngest docs](https://www.inngest.com/docs)

# Features

- Type safe functions, durable workflows, and steps using generics
- Event stream sampling built in
- Declarative flow control (concurrency, prioritization, batching, debounce, rate limiting)
- Zero-infrastructure. Inngest handles orchestration and calls your functions.

# Examples

The following is the bare minimum setup for a fully distributed durable workflow:

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
)

func main() {
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID: "core",
	})
	if err != nil {
		panic(err)
	}

	_, err = inngestgo.CreateFunction(
		client,
		inngestgo.FunctionOpts{
			ID: "account-created",
		},
		// Run on every api/account.created event.
		inngestgo.EventTrigger("api/account.created", nil),
		AccountCreated,
	)
	if err != nil {
		panic(err)
	}

	http.ListenAndServe(":8080", client.Serve())
}

// AccountCreated is a durable function which runs any time the "api/account.created"
// event is received by Inngest.
//
// It is invoked by Inngest, with each step being backed by Inngest's orchestrator.
// Function state is automatically managed, and persists across server restarts,
// cloud migrations, and language changes.
func AccountCreated(
	ctx context.Context,
	input inngestgo.Input[AccountCreatedEventData],
) (any, error) {
	// Sleep for a second, minute, hour, week across server restarts.
	step.Sleep(ctx, "initial-delay", time.Second)

	// Run a step which emails the user.  This automatically retries on error.
	// This returns the fully typed result of the lambda.
	result, err := step.Run(ctx, "on-user-created", func(ctx context.Context) (bool, error) {
		// Run any code inside a step.
		result, err := emails.Send(emails.Opts{})
		return result, err
	})
	if err != nil {
		// This step retried 5 times by default and permanently failed.
		return nil, err
	}
	// `result` is  fully typed from the lambda
	_ = result

	// Sample from the event stream for new events.  The function will stop
	// running and automatially resume when a matching event is found, or if
	// the timeout is reached.
	fn, err := step.WaitForEvent[FunctionCreatedEvent](
		ctx,
		"wait-for-activity",
		step.WaitForEventOpts{
			Name:    "Wait for a function to be created",
			Event:   "api/function.created",
			Timeout: time.Hour * 72,
			// Match events where the user_id is the same in the async sampled event.
			If: inngestgo.StrPtr("event.data.user_id == async.data.user_id"),
		},
	)
	if err == step.ErrEventNotReceived {
		// A function wasn't created within 3 days.  Send a follow-up email.
		step.Run(ctx, "follow-up-email", func(ctx context.Context) (any, error) {
			// ...
			return true, nil
		})
		return nil, nil
	}

	// The event returned from `step.WaitForEvent` is fully typed.
	fmt.Println(fn.Data.FunctionID)

	return nil, nil
}

// AccountCreatedEvent represents the fully defined event received when an account is created.
//
// This is shorthand for defining a new Inngest-conforming struct:
//
//	type AccountCreatedEvent struct {
//		Name      string                  `json:"name"`
//		Data      AccountCreatedEventData `json:"data"`
//		User      map[string]any          `json:"user"`
//		Timestamp int64                   `json:"ts,omitempty"`
//		Version   string                  `json:"v,omitempty"`
//	}
type AccountCreatedEvent inngestgo.GenericEvent[AccountCreatedEventData]
type AccountCreatedEventData struct {
	AccountID string
}

type FunctionCreatedEvent inngestgo.GenericEvent[FunctionCreatedEventData]
type FunctionCreatedEventData struct {
	FunctionID string
}
```
