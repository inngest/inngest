package publicerr

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	DefaultMessage = "Something went wrong.  Please try again"
	DefaultStatus  = 500
)

// Wrap wraps a root cause error with an HTTP status and a public message.  The status
// is first to conform to Wrapf and Errorf formats.
func Wrap(err error, status int, msg string) error {
	return Error{
		Message: msg,
		Status:  status,
		Err:     err,
	}
}

// Wrapf wraps a root cause in the same way as Wrap, with built-in string formatting
// via fmt.Sprintf.
func Wrapf(err error, status int, msg string, opts ...interface{}) error {
	return Error{
		Message: fmt.Sprintf(msg, opts...),
		Status:  status,
		Err:     err,
	}
}

// WrapDefaults wraps an error with the default message and default status.
func WrapDefaults(err error) error {
	return Error{
		Message: DefaultMessage,
		Status:  DefaultStatus,
		Err:     err,
	}
}

// WrapWithData wraps a root cause error with an HTTP status and a public message.  The status
// is first to conform to Wrapf and Errorf formats.
func WrapWithData(err error, status int, msg string, data map[string]any) error {
	return Error{
		Message: msg,
		Status:  status,
		Err:     err,
	}
}

func WithData(err error, data map[string]any) error {
	d, ok := err.(Error)
	if !ok {
		d = WrapDefaults(err).(Error)
	}
	d.Data = data
	return d
}

// Errorf is much like fmt.Errorf but holds an HTTP status and returns an Error type
// wrapping fmt.Errorf's builtin error.  This is used to create new error structs
// easily if the message is intended for the public.
func Errorf(status int, message string, opts ...interface{}) error {
	err := fmt.Errorf(message, opts...)
	return Error{
		Message: err.Error(), // Use the error message directly.
		Status:  status,
		Err:     err,
	}
}

// Error wraps a root cause error with friendly messages to display to the public.
//
// This package provides the functionality for storing and wrapping errors only. It does NOT
// manage how to marshal and display this to the public;  we have GQL, Rest API endpoints
// and, in the future, other ways in which users can communicate with us.
//
// Each method of communication must check and display these errors correctly.  For example,
// in `gql.go` we create an ErrorPresenter middleware item which checks to see if the error
// is a publicerr.Error and, if so, shows the friendly error directly.
type Error struct {
	Code string `json:"code,omitempty"`
	// Message represents the message to display
	Message string `json:"error"`
	// Data is a KV map of extra error data.
	Data map[string]any `json:"data"`
	// Status represents the HTTP status code to use when responding via HTTP
	Status int `json:"status"`
	// Err is the original err which represents the cause of the issue.  This is
	// the internal error used for debugging.
	Err error `json:"-"`
}

// WriteHTTP renders an error to the response as JSON.
//
// An error that did not come from this package carries no status, and used to
// fall through to WriteHeader's implicit 200 — so a handler passing one along
// answered with what every client reads as success: status 200 and a body of
// `{}`, since a plain error has no exported fields to marshal. A failed query
// was indistinguishable from an empty result. Those now render as 500.
//
// The response says nothing more than the default message. An error that never
// went through this package was not written to be read by the public and may
// name internals; the caller keeps the original for logging.
func WriteHTTP(w http.ResponseWriter, e error) error {
	pe, ok := asError(e)
	if !ok {
		pe = Error{Message: DefaultMessage, Status: DefaultStatus, Err: e}
	}
	if pe.Status == 0 {
		// WriteHeader(0) panics, so an Error built without a status still has to
		// resolve to something. 500 is the honest answer: it reached here as an
		// error.
		pe.Status = DefaultStatus
	}
	w.WriteHeader(pe.Status)
	return json.NewEncoder(w).Encode(pe)
}

// asError returns the public error e carries, if it is one. Deliberately a
// direct type check rather than errors.As: promoting a status out of a wrapped
// error would change what existing handlers answer, which is a wider change than
// this needs to make.
func asError(e error) (Error, bool) {
	switch v := e.(type) {
	case Error:
		return v, true
	case *Error:
		if v != nil {
			return *v, true
		}
	}
	return Error{}, false
}

// Error fulfils the builtin go error interface.  This returns the original root cause
// for debugging via logging - all logging interfaces & tracing packages use Error() to
// capture error information.
func (e Error) Error() string {
	return e.Err.Error()
}

// Unwrap fulfils errors.Unwrap, allowing us to use the builtin error functionality to
// manage the original error, or to chain public errors.
func (e Error) Unwrap() error {
	return e.Err
}

// HTTPErr returns a simple public Error with the given HTTP error code. The message
// is the standard library's text for that code.
func HTTPErr(status int) Error {
	m := http.StatusText(status)
	if m == "" {
		m = http.StatusText(http.StatusInternalServerError)
	}

	return Error{
		Message: m,
		Status:  status,
	}
}
