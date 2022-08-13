package testdsl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/function"
)

var (
	// DefaultDuration represents the default duration between steps.  This
	// is 50ms as localstack, our SQS mock, is very slow.
	//
	// TODO: Change this based off of the config we're using.
	DefaultDuration = 500 * time.Millisecond
)

type Buffer interface {
	io.ReadWriter
	fmt.Stringer
}

type TestData struct {
	Fn          *function.Function
	TriggerData map[string]any
	Out         Buffer

	Config config.Config
}

// registered stores all registered test DSL roots
var registered map[string]Root

// Register registers a new test DSL chain
func Register(r Root) {

	// file should contain the full filepath of the calling file which
	// is registering the test.  We can use this to figure out the directory
	// the test is in.
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("unable to determine directory")
	}
	// Get the directory name of the registering function.
	dir := filepath.Base(filepath.Dir(file))

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
	evt, err := function.GenerateTriggerData(ctx, time.Now().Unix(), td.Fn.Triggers)
	if err != nil {
		return fmt.Errorf("error generating trigger data: %w", err)
	}

	td.TriggerData = evt.Map()

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

// RequireReceiveTrigger asserts that we received the trigger within the default amount of time.
func RequireReceiveTrigger(ctx context.Context, td *TestData) error {
	return RequireOutputWithin("received message", 500*time.Millisecond)(ctx, td)
}

func Wait(t time.Duration) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Waiting %s\n", t.String())
		<-time.After(t)
		return nil
	}
}

func RequireTriggerExecution(ctx context.Context, td *TestData) error {
	return RequireLogFieldsWithin(
		map[string]any{
			"message": "executing step",
			"step":    inngest.TriggerName,
		},
		DefaultDuration,
	)(ctx, td)
}

func RequireLogField(name, value string) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking log field: %s\n", name)
		return requireLogFields(ctx, td, map[string]any{name: value})
	}
}

func RequireLogFields(fields map[string]any) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking log fields: %v\n", fields)
		return requireLogFields(ctx, td, fields)
	}
}

func RequireLogFieldsWithin(fields map[string]any, t time.Duration) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking log fields within %s: %v\n", t, fields)
		return timeout(t, func() error {
			if err := requireLogFields(ctx, td, fields); err != nil {
				return fmt.Errorf("Could not find fields: %v", fields)
			}
			return nil
		})
	}
}

func RequireNoLogFields(fields map[string]any) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking for no log fields: %v\n", fields)
		if err := requireLogFields(ctx, td, fields); err == nil {
			return fmt.Errorf("log fields found: %s", fields)
		}
		return nil
	}
}

// RequireNoLogFieldsWithin ensures that the logs do not contain the specified fields
// within the given amount of time.
func RequireNoLogFieldsWithin(fields map[string]any, t time.Duration) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking for no log fields: %v\n", fields)
		err := timeout(t, func() error {
			if err := requireLogFields(ctx, td, fields); err != nil {
				return fmt.Errorf("Could not find fields: %v", fields)
			}
			return nil
		})
		if err == nil {
			return fmt.Errorf("log fields found: %s", fields)
		}
		return nil
	}
}

func RequireOutput(output string) Proc {
	return func(ctx context.Context, td *TestData) error {
		return requireOutput(ctx, td, output)
	}
}

func RequireNoOutput(output string) Proc {
	return func(ctx context.Context, td *TestData) error {
		fmt.Printf("> Checking for no output containing: %s\n", output)
		if err := requireOutput(ctx, td, output); err == nil {
			return fmt.Errorf("output found: %s", output)
		}
		return nil
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

func RequireStepRetries(step string, count int) Proc {
	return func(ctx context.Context, td *TestData) error {
		var backoffTime uint

		for i := 0; i < count; i++ {
			fmt.Printf("> Checking step %s performs retry %d of %d\n", step, i+1, count)

			backoffTime = uint(10) << i

			fmt.Printf("\t> Checking attempt #%d executes (waiting %d seconds)\n", i+1, backoffTime)
			if err := timeout(time.Second*time.Duration(backoffTime), func() error {
				return requireLogFields(ctx, td, map[string]any{
					"caller":  "executor",
					"step":    step,
					"message": "executing step",
					"attempt": i,
				})
			}); err != nil {
				return err
			}

			if i+1 >= count {
				continue
			}

			fmt.Printf("\t> Checking attempt #%d queues a retry\n", i+1)
			if err := timeout(time.Second*5, func() error {
				return requireLogFields(ctx, td, map[string]any{
					"caller":  "executor",
					"message": "enqueueing retry",
					"edge": map[string]any{
						"errorCount": i + 1,
						"payload": map[string]any{
							"edge": map[string]any{
								"incoming": step,
							},
						},
					},
				})
			}); err != nil {
				return err
			}
		}

		fmt.Printf("> Checking step %s permanently failed after %d retries (waiting %d seconds)\n", step, count, backoffTime)
		if err := timeout(time.Second*time.Duration(backoffTime), func() error {
			return requireLogFields(ctx, td, map[string]any{
				"caller":  "executor",
				"message": "step permanently failed",
				"edge": map[string]any{
					"incoming": step,
				},
			})
		}); err != nil {
			return err
		}

		// Finally, check that the step did not have more retries than it was
		// allowed before it failed.
		if err := timeout(time.Second, func() error {
			return requireLogFields(ctx, td, map[string]any{
				"caller":  "executor",
				"message": "enqueueing retry",
				"edge": map[string]any{
					"errorCount": count + 1,
					"payload": map[string]any{
						"edge": map[string]any{
							"incoming": step,
						},
					},
				},
			})
		}); err == nil {
			return fmt.Errorf("step %s had more retries than allowed", step)
		}

		return nil
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
	// Unfortunately, fields will be marshalled as JSON and values here will likely
	// be ints.  Marshal to/from JSON to ensure types match.
	byt, err := json.Marshal(kv)
	if err != nil {
		return err
	}
	kv = map[string]any{}
	err = json.Unmarshal(byt, &kv)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(td.Out.String(), "\n") {
		data := map[string]any{}
		_ = json.Unmarshal([]byte(line), &data)

		found := cmpPartial(kv, data)

		if found {
			return nil
		}
	}

	return fmt.Errorf("fields not found: %s", kv)
}

func cmpPartial(expectedPartial, actual map[string]interface{}) bool {
	var found int

	for field, val := range actual {
		for searchKey, searchVal := range expectedPartial {
			if field == searchKey {
				if (reflect.ValueOf(searchVal).Kind() == reflect.Map && cmpPartial(searchVal.(map[string]any), val.(map[string]any))) || cmp.Equal(val, searchVal) {
					found++
					break
				}
			}
		}

		if found == len(expectedPartial) {
			return true
		}
	}

	return false
}
