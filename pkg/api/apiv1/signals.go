package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/util"
)

// maxScopedSignalBodyBytes bounds the body of a token-authenticated signal. The
// token holder is not a trusted API client, so the body is capped well below
// the general request limits. Two 8 KiB output tails can grow to about 96 KiB
// once JSON escapes control characters, so the cap leaves room for that.
const maxScopedSignalBodyBytes = 256 * 1024

type ReceiveSignalRequest struct {
	Signal string          `json:"signal"`
	Data   json.RawMessage `json:"data"`
}

type ReceiveSignalResponse struct {
	RunID string `json:"run_id"`
}

func (a router) receiveSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	data := ReceiveSignalRequest{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid signal data"))
		return
	}

	a.resumeSignal(w, r, auth.WorkspaceID(), data)
}

// receiveScopedSignal sends a signal authenticated by a signal token instead of
// the environment's API key. The token names the one signal it may send, so a
// holder (for example a sandbox) can only wake the wait it was given.
func (a router) receiveScopedSignal(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(nil, 401, "Missing signal token"))
		return
	}

	claims, err := apiv1auth.VerifySignalJWT(a.opts.SignalJWTSecret, token)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Invalid signal token"))
		return
	}

	data := ReceiveSignalRequest{}
	body := http.MaxBytesReader(w, r.Body, maxScopedSignalBodyBytes)
	decoder := json.NewDecoder(body)
	err = decoder.Decode(&data)
	if err == nil {
		// Reject trailing data after the one JSON value.
		if _, extra := decoder.Token(); extra != io.EOF {
			err = extra
			if err == nil {
				err = errors.New("unexpected data after signal body")
			}
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, http.StatusRequestEntityTooLarge, "Signal body is too large"))
			return
		}
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid signal data"))
		return
	}

	if data.Signal != claims.SignalID {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(nil, 403, "Token cannot send this signal"))
		return
	}

	a.resumeSignal(w, r, claims.Env, data)
}

func (a router) resumeSignal(w http.ResponseWriter, r *http.Request, workspaceID uuid.UUID, data ReceiveSignalRequest) {
	ctx := r.Context()

	signalRes, err := util.WithRetry(
		ctx,
		"apiv1.receiveSignal",
		func(ctx context.Context) (*execution.ResumeSignalResult, error) {
			return a.opts.Executor.ResumeSignal(ctx, workspaceID, data.Signal, data.Data)
		},
		util.NewRetryConf(),
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Failed to receive signal"))
		return
	}

	if signalRes == nil || !signalRes.MatchedSignal {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "No signal found"))
		return
	}

	res := ReceiveSignalResponse{
		RunID: signalRes.RunID.String(),
	}

	_ = WriteResponse(w, res)
}
