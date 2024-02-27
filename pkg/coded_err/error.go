package coded_err

import (
	"encoding/json"
	"errors"
	"fmt"
)

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

func messageFromMultiErrData(data []byte) (string, error) {
	me := &MultiErrData{}
	err := json.Unmarshal(data, me)
	if err != nil {
		return "", fmt.Errorf("not MultiErrData: %v", err)
	}
	if len(me.Errors) == 0 {
		return "", errors.New("not MultiErrData")
	}

	// Build a string that mimics what multierror.Error does
	out := fmt.Sprintf("%d errors occurred: ", len(me.Errors))
	for _, e := range me.Errors {
		out += "* " + e.Error()
	}
	return out, nil
}

type MultiErrData struct {
	Errors []Error `json:"errors"`
}

func (e *MultiErrData) Add(err error) {
	if err == nil {
		return
	}

	ce := &Error{}
	if !errors.As(err, ce) && !errors.As(err, &ce) {
		ce = &Error{
			Code:    CodeUnknown,
			Message: err.Error(),
		}
	}

	e.Errors = append(e.Errors, *ce)
}
