package syscode

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

func fromError(err error) Error {
	e := &Error{}
	if !errors.As(err, e) && !errors.As(err, &e) {
		e = &Error{
			Code:    CodeUnknown,
			Message: err.Error(),
		}
	}

	return *e
}

type Error struct {
	Code    string `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}

	byt, err := json.Marshal(e.Data)
	if err != nil {
		return e.Code
	}

	msg, err := messageFromMultiErrData(byt)
	if err == nil {
		return msg
	}

	return e.Code
}

// Create an single error message from error data that contains multiple errors.
// Returns an error if the data isn't valid MultiErrData
func messageFromMultiErrData(data []byte) (string, error) {
	me := &DataMultiErr{}
	err := json.Unmarshal(data, me)
	if err != nil {
		return "", fmt.Errorf("not MultiErrData: %v", err)
	}
	if len(me.Errors) == 0 {
		return "", errors.New("not MultiErrData")
	}

	// Build a string that mimics what multierror.Error does. This is for
	// backcompat, since we used the multierror.Error message before we added
	// the syscode package
	out := fmt.Sprintf("%d errors occurred:", len(me.Errors))
	for _, e := range me.Errors {
		out += " * " + e.Error()
	}
	return out, nil
}

// Used to structure Error.Data when there was an HTTP-related error
type DataHTTPErr struct {
	Headers    map[string][]string `json:"headers"`
	StatusCode int                 `json:"status_code"`
}

func (d DataHTTPErr) ToMap() map[string]any {
	return map[string]any{
		"headers":     d.Headers,
		"status_code": d.StatusCode,
	}
}

// Used to structure Error.Data when there are multiple errors (e.g.
// synchronization validation)
type DataMultiErr struct {
	Errors []Error `json:"errors"`
}

func (e *DataMultiErr) Append(err error) {
	if err == nil {
		return
	}

	if me, ok := err.(*multierror.Error); ok {
		for i := range me.Errors {
			e.Append(fromError(me.Errors[i]))
		}
		return
	}

	e.Errors = append(e.Errors, fromError(err))
}

func (e *DataMultiErr) ToMap() map[string]any {
	return map[string]any{"errors": e.Errors}
}
