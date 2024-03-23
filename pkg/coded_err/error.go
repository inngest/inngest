package coded_err

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

func fromError(err error) Error {
	ce := &Error{}
	if !errors.As(err, ce) && !errors.As(err, &ce) {
		ce = &Error{
			Code:    CodeUnknown,
			Message: err.Error(),
		}
	}

	return *ce
}

type Error struct {
	Code    string         `json:"code"`
	Data    map[string]any `json:"data"`
	Message string         `json:"message"`
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
	me := &MultiErrData{}
	err := json.Unmarshal(data, me)
	if err != nil {
		return "", fmt.Errorf("not MultiErrData: %v", err)
	}
	if len(me.Errors) == 0 {
		return "", errors.New("not MultiErrData")
	}

	// Build a string that mimics what multierror.Error does. This is for
	// backcompat, since we used the multierror.Error message before we added
	// the coded_err package
	out := fmt.Sprintf("%d errors occurred:", len(me.Errors))
	for _, e := range me.Errors {
		out += " * " + e.Error()
	}
	return out, nil
}

// Used to structure Error.Data when there are multiple errors (e.g.
// synchronization validation)
type MultiErrData struct {
	Errors []Error `json:"errors"`
}

func (e *MultiErrData) Append(err error) {
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

func (e *MultiErrData) ToMap() map[string]any {
	return map[string]any{"errors": e.Errors}
}
