package apiv1

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/publicerr"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

// DeleteCancellation is implementation which finds and deletes a cancellation given the ID.
func (a API) DeleteCancellation(ctx context.Context, cancellationID ulid.ULID) error {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return publicerr.Wrap(err, 401, "No auth found")
	}

	all, err := a.opts.CancellationReadWriter.Cancellations(ctx, auth.WorkspaceID())
	if err != nil {
		return publicerr.Wrap(err, 500, "Error deleting cancellation")
	}

	for _, c := range all {
		if bytes.Equal(c.ID[:], cancellationID[:]) {
			err := a.opts.CancellationReadWriter.DeleteCancellation(ctx, c)
			if err != nil {
				return publicerr.Wrap(err, 500, "Error deleting cancellation")
			}
			if err == nil {
				return nil
			}
		}
	}

	return publicerr.Wrap(err, 404, "Cancellation not found")
}

// DeleteCancellation is the HTTP handler implementation.
func (a router) deleteCancellation(w http.ResponseWriter, r *http.Request) {
	id, err := ulid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid cancellation ID"))
		return
	}
	ctx := r.Context()
	if err := a.API.DeleteCancellation(ctx, id); err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}
	w.WriteHeader(200)
	_ = WriteResponse(w, map[string]any{"ok": true})
}

func (a API) GetCancellations(ctx context.Context) ([]cqrs.Cancellation, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}

	all, err := a.opts.CancellationReadWriter.Cancellations(ctx, auth.WorkspaceID())
	if err != nil {
		return nil, publicerr.Wrap(err, 500, "Error listing cancellations")
	}
	return all, nil
}

func (a router) getCancellations(w http.ResponseWriter, r *http.Request) {
	c, err := a.API.GetCancellations(r.Context())
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}
	_ = WriteResponse(w, c)
}

type CreateCancellationBody struct {
	// AppID is the client ID specified via the SDK in the app that defines the function.
	AppID string `json:"app_id"`
	// FunctionID is the function ID string specified in configuration via the SDK.
	FunctionID    string     `json:"function_id"`
	StartedAfter  *time.Time `json:"started_after"`
	StartedBefore time.Time  `json:"started_before"`
	If            *string    `json:"if,omitempty"`
}

func (c CreateCancellationBody) Validate() error {
	var err error
	if c.AppID == "" {
		err = errors.Join(err, errors.New("app_id is required"))
	}
	if c.FunctionID == "" {
		err = errors.Join(err, errors.New("function_id is required"))
	}
	if c.StartedBefore.IsZero() {
		err = errors.Join(err, errors.New("started_before is required"))
	}
	if c.StartedBefore.After(time.Now().Add(5 * time.Second)) {
		err = errors.Join(err, errors.New("started_before must be in the past"))
	}
	if c.StartedAfter != nil && c.StartedAfter.After(c.StartedBefore) {
		err = errors.Join(err, errors.New("started_after must be before started_before"))
	}
	return err
}

func (a API) CreateCancellation(ctx context.Context, opts CreateCancellationBody) (*cqrs.Cancellation, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}
	fn, err := a.opts.FunctionReader.GetFunctionByExternalID(
		ctx,
		auth.WorkspaceID(),
		opts.AppID,
		opts.FunctionID,
	)
	if err != nil {
		return nil, publicerr.Wrap(err, 404, "function not found")
	}
	// Create a new cancellation for the given function ID
	cancel := cqrs.Cancellation{
		CreatedAt:     time.Now(),
		ID:            ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:     auth.AccountID(),
		WorkspaceID:   auth.WorkspaceID(),
		AppID:         fn.AppID,
		FunctionID:    fn.ID,
		FunctionSlug:  fn.Slug,
		StartedAfter:  opts.StartedAfter,
		StartedBefore: opts.StartedBefore,
		If:            opts.If,
		Kind:          enums.CancellationKindBulkRun,
		Type:          enums.CancellationTypeAPI,
	}
	if err := a.opts.CancellationReadWriter.CreateCancellation(ctx, cancel); err != nil {
		var compileError *expressions.CompileError
		if errors.As(err, &compileError) {
			return nil, publicerr.Wrap(err, 400, fmt.Sprintf("invalid expression: %s", compileError.Message()))
		}
		return nil, publicerr.Wrap(err, 500, "Error creating cancellation")
	}
	return &cancel, nil
}

func (a router) createCancellation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	opts := CreateCancellationBody{}
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid cancellation request"))
		return
	}
	if err := opts.Validate(); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, err.Error()))
		return
	}

	cancel, err := a.API.CreateCancellation(ctx, opts)
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	_ = WriteResponse(w, cancel)
}
