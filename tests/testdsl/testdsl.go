package testdsl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/function"
)

type TestData struct {
	Fn          *function.Function
	TriggerData map[string]any
	Out         *bytes.Buffer

	Config config.Config
}

// registered stores all registered test DSL roots
var registered map[string]Root

// Register registers a new test DSL chain
func Register(dir string, r Root) {
	if registered == nil {
		registered = map[string]Root{}
	}
	registered[dir] = r
}

// ForDir returns the rest DSL chain for the given dir.
func ForDir(dir string) Root {
	return registered[dir]
}

// Root is the root test DSL for running tests over a function.
type Root func(context.Context) Chain

// Proc represents a procedure within the test DSL chain.
type Proc func(context.Context, *TestData) error

// Chain represents a step of procedures that must pass in order
// to complete the function.
type Chain []Proc

// SendTrigger sends a trigger event to the API.
func SendTrigger(ctx context.Context, td *TestData) error {
	fmt.Println("> Sending trigger")

	var err error
	td.TriggerData, err = function.GenerateTriggerData(ctx, time.Now().Unix(), td.Fn.Triggers)
	if err != nil {
		return fmt.Errorf("error generating trigger data: %w", err)
	}
	byt, _ := json.Marshal(td.TriggerData)
	resp, err := http.Post(
		fmt.Sprintf("http://%s:%d/e/key", td.Config.EventAPI.Addr, td.Config.EventAPI.Port),
		"application/json",
		bytes.NewBuffer(byt),
	)
	if err != nil {
		return fmt.Errorf("error sending event: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		byt, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid status code sending event: %d\nResponse: %s", resp.StatusCode, string(byt))
	}
	return nil
}

func Wait(t time.Duration) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Waiting %s\n", t.String())
		<-time.After(t)
		return nil
	}
}

func RequireLogField(name, value string) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking log field: %s\n", name)
		return requireLogFields(ctx, td, map[string]any{name: value})
	}
}

func RequireLogFields(fields map[string]any) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking log fields: %s\n", fields)
		return requireLogFields(ctx, td, fields)
	}
}

func RequireLogFieldsWithin(fields map[string]any, t time.Duration) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking log fields within %s: %s\n", t, fields)
		return timeout(t, func() error {
			if err := requireLogFields(ctx, td, fields); err != nil {
				return fmt.Errorf("Could not find fields: %s", fields)
			}
			return nil
		})
	}
}

func RequireOutput(output string) Proc {
	return func(ctx context.Context, td *TestData) error {
		return requireOutput(ctx, td, output)
	}
}

func RequireOutputWithin(output string, within time.Duration) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking output within %s: %s\n", within, output)
		// Require output within timeout
		return timeout(within, func() error {
			if err := requireOutput(ctx, td, output); err != nil {
				return fmt.Errorf("Could not find output: %s", output)
			}
			return nil
		})
	}
}

// timeout is a helper for timeout funcs.
func timeout(t time.Duration, f func() error) error {
	timeout := time.After(t)
	for {
		select {
		case <-timeout:
			return f()
		case <-time.After(50 * time.Millisecond):
			if err := f(); err == nil {
				return nil
			}
		}

	}
}

func requireOutput(ctx context.Context, td *TestData, output string) error {
	if strings.Contains(td.Out.String(), output) {
		return nil
	}
	return fmt.Errorf("output not found")
}

func requireLogFields(ctx context.Context, td *TestData, kv map[string]any) error {
	for _, line := range strings.Split(td.Out.String(), "\n") {
		data := map[string]any{}
		_ = json.Unmarshal([]byte(line), &data)

		// Iterate through each line, and search for each key/value
		present := true
		for field, val := range data {
			// Assume that this is found, then if any key isn't found we set
			// this to false, which will be retained after this loop.
			present = true

			// Ensure that each KV is found.  If not, break.
			found := false
			for searchKey, searchVal := range kv {
				if field == searchKey && reflect.DeepEqual(val, searchVal) {
					found = true
					break
				}
			}
			if !found {
				present = false
				continue
			}
		}
		if present {
			return nil
		}
	}
	return fmt.Errorf("fields not found: %s", kv)
}
