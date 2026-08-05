package publicerr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) Error {
	t.Helper()
	var got Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

// The bug: an error that never went through this package has no status, so
// WriteHTTP fell through to WriteHeader's implicit 200 and encoded a value with
// no exported fields. Handlers passing one along — a failed trace-reader query,
// a marshalling failure — answered 200 with `{}`, which every client reads as a
// successful empty result.
func TestWriteHTTPFailsClosedOnABareError(t *testing.T) {
	rec := httptest.NewRecorder()
	err := WriteHTTP(rec, errors.New("connection refused: 10.0.0.7:5432"))
	require.NoError(t, err)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	got := decode(t, rec)
	require.Equal(t, DefaultMessage, got.Message)
	require.Equal(t, DefaultStatus, got.Status)

	// The original is for logs, not for the caller: it may name internals.
	require.NotContains(t, rec.Body.String(), "10.0.0.7")
}

func TestWriteHTTPUsesTheStatusAPublicErrorCarries(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		rec := httptest.NewRecorder()
		require.NoError(t, WriteHTTP(rec, Wrap(errors.New("boom"), 404, "Not found")))

		require.Equal(t, http.StatusNotFound, rec.Code)
		got := decode(t, rec)
		require.Equal(t, "Not found", got.Message)
		require.Equal(t, 404, got.Status)
	})

	t.Run("pointer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		e := Wrap(errors.New("boom"), 400, "Bad request").(Error)
		require.NoError(t, WriteHTTP(rec, &e))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Bad request", decode(t, rec).Message)
	})

	t.Run("data survives", func(t *testing.T) {
		rec := httptest.NewRecorder()
		e := WithData(Wrap(errors.New("boom"), 422, "Invalid"), map[string]any{"field": "name"})
		require.NoError(t, WriteHTTP(rec, e))

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		require.Equal(t, map[string]any{"field": "name"}, decode(t, rec).Data)
	})
}

// WriteHeader(0) panics, so an Error built by hand without a status still has to
// resolve to something rather than taking the process down.
func TestWriteHTTPHandlesAStatuslessError(t *testing.T) {
	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		_ = WriteHTTP(rec, Error{Message: "no status set"})
	})
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "no status set", decode(t, rec).Message,
		"only the status is defaulted; the message the caller chose is kept")
}

// A nil *Error and a nil error both reach the default branch rather than
// dereferencing.
func TestWriteHTTPHandlesNil(t *testing.T) {
	for name, e := range map[string]error{
		"nil error":  nil,
		"nil *Error": (*Error)(nil),
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			require.NotPanics(t, func() { _ = WriteHTTP(rec, e) })
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}
