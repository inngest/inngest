package actionsdk

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var (
	// args represents args that have been unmarshalled for the given action.
	// This only happens once and is read-only, therefore it's safe to keep this
	// in a single package-level variable.
	//
	// If args is nil, args has not yet been initialized.
	args *Args
)

// Args is the function context, showing:
// - The triggering event
// - Data from previous steps
// - Any function-specific config for this step.
type Args struct {
	Event  Event                             `json:"event"`
	Steps  map[string]map[string]interface{} `json:"steps"`
	Ctx    map[string]interface{}            `json:"ctx"`
	Config json.RawMessage                   `json:"config"`
}

// Event is the triggering event for this function.
type Event struct {
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data"`
	User      map[string]interface{} `json:"user,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Timestamp int64                  `json:"ts,omitempty"`
	Version   string                 `json:"v,omitempty"`
}

// Result is the data returned from this step.
type Result struct {
	Body   interface{} `json:"body"`
	Status int         `json:"status"`
}

// WriteError writes an error to stdout with a standard format.  The error is
// added to a json object with an "error" key: {"error": err.Error()}.
//
// This does _not_ stop the action or workflow.
//
// To stop the action and prevent the workflow branch from continuing, exit
// with a non-zero status code (ie. `os.Exit(1)`).
//
// To stop the action but allow workflows to continue, exit with a zero status
// code (ie. `os.Exit(0)`)
func WriteError(err error, retryable bool) {
	// 4xx errors are not retryable;  it indicates that the request, or input
	// data, is wrong and simply re-running this step will not fix.
	status := 400
	if retryable {
		// 5xx errors are retryable
		status = 500
	}
	byt, err := json.Marshal(map[string]interface{}{
		"error":  err.Error(),
		"status": status,
	})
	if err != nil {
		log.Fatal(fmt.Errorf("unable to marshal error: %w", err))
	}

	_, err = fmt.Println(string(byt))
	if err != nil {
		log.Fatal(fmt.Errorf("unable to write error: %w", err))
	}
}

// WriteResult writes the output as a JSON-encoded string to stdout.  Any data written
// here is captured as action output, which is added to the workflow context and can be
// used by future actions in the workflow.
//
// Note that this does _not_ stop the action.  To stop the action, call `os.Exit(0)` or
// return from your main function.
func WriteResult(i *Result) error {
	if i == nil {
		_, err := fmt.Fprint(os.Stdout, `{"body": null, "status": 201}`)
		return err
	}

	byt, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}

	_, err = fmt.Println(string(byt))
	return err
}

// GetConfig returns the config for the action as configured within this specific workflow.
// The type for this struct must match the definitions within the action config (action.cue).
func GetConfig(dest interface{}) error {
	args, err := GetArgs()
	if err != nil {
		return err
	}
	return json.Unmarshal(args.Config, dest)
}

// GetSecret returns the secret stored within the current workspace.  If no secret is found
// this returns an error.
func GetSecret(str string) (string, error) {
	if secret := os.Getenv(str); secret != "" {
		return secret, nil
	}
	return "", fmt.Errorf("secret not found: %s", str)
}

// GetArgs returns the arguments provided to the step, returning an error
// if invalid
func GetArgs() (*Args, error) {
	if args != nil {
		return args, nil
	}

	// We pass in a JSON string as the first arugment.  This payload contains the action metadata,
	// workflow context, etc.
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("no arguments present")
	}

	args = &Args{}
	if err := json.Unmarshal([]byte(os.Args[1]), args); err != nil {
		return nil, fmt.Errorf("unable to parse arguments: %s", err)
	}

	return args, nil
}

// MustGetArgs returns the arguments provided to the step.
func MustGetArgs() *Args {
	args, err := GetArgs()
	if err != nil {
		WriteError(err, false)
		os.Exit(1)
	}
	return args
}
