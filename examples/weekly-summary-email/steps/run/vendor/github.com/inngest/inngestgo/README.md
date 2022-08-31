[![GoDoc](https://godoc.org/github.com/inngest/inngestgo?status.svg)](http://godoc.org/github.com/inngest/inngestgo)

# Inngest Go SDK

A simple SDK for sending well-formed events to [Inngest](https://www.inngest.com).

## Example usage:

Using a dedicated client as a dependency:

```go
import (
	"context"
	"os"

	"github.com/inngest/inngestgo"
)

func sendEvent(ctx context.Context) {
	// Create a new client
	client := inngestgo.NewClient(os.Getenv("INGEST_KEY"))

	// Send an event
	client.Send(ctx, inngestgo.Event{
		Name: "user.created",
		Data: map[string]interface{}{
			"plan": account.PlanType,
			"ip":   req.RemoteAddr,
		},
		User: map[string]interface{}{
			// Use the external_id field within User so that we can identify
			// this event as authored by the given user.
			inngestgo.ExternalID: user.ID,
			inngestgo.Email:      ou.Email,
		},
		Version:   "2021-07-01.01",
		Timestamp: inngestgo.Now(),
	})
}
```

Using the default client, so that any package can call `inngestgo.Send`:

```go
import (
	"context"
	"os"

	"github.com/inngest/inngestgo"
)

func init() {
	// Set the default client.
	inngestgo.DefaultClient = inngestgo.NewClient(os.Getenv("INGEST_KEY"))
}

func do() {
	// Now, we can use inngestgo.Send(ctx, evt) to send using the
	// default client.  This reduces the need for you to pass the
	// inngest client down as a dependency for each package.
	inngestgo.Send(ctx, inngestgo.Event{
		Name: "user.created",
		Data: map[string]interface{}{
			"plan": account.PlanType,
			"ip":   req.RemoteAddr,
		},
		User: map[string]interface{}{
			// Use the external_id field within User so that we can identify
			// this event as authored by the given user.
			inngestgo.ExternalID: user.ID,
			inngestgo.Email:      ou.Email,
		},
		Version:   "2021-07-01.01",
		Timestamp: inngestgo.Now(),
	})
}
```
