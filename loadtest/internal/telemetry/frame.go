// Package telemetry defines the worker → harness wire protocol and both ends
// of the unix-socket transport.
//
// Wire: length-prefixed (uint32 big-endian) JSON frames.
package telemetry

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

// Phase names the lifecycle event a frame describes.
type Phase string

const (
	PhaseReady            Phase = "ready" // worker ready signal
	PhaseSDKRequestRecv   Phase = "sdk_request_recv"
	PhaseFnStart          Phase = "fn_start"
	PhaseStepStart        Phase = "step_start"
	PhaseStepEnd          Phase = "step_end"
	PhaseFnEnd            Phase = "fn_end"
	PhaseSDKResponseSent  Phase = "sdk_response_sent"
)

// Frame is one telemetry record.
type Frame struct {
	WorkerID      string `json:"workerId"`
	Seq           uint64 `json:"seq"`
	InngestRunID  string `json:"runId,omitempty"`
	CorrelationID string `json:"corr,omitempty"` // event-level correlation ID; present on fn_start
	FunctionSlug  string `json:"fn,omitempty"`
	StepID        string `json:"step,omitempty"`
	Attempt       int    `json:"attempt,omitempty"`
	Phase         Phase  `json:"phase"`
	TSNanos       int64  `json:"ts"`
}

// maxFrameBytes is a sanity cap so a misbehaving peer cannot allocate
// unbounded memory. Frames are small JSON objects — 64KiB is plenty.
const maxFrameBytes = 64 * 1024

// ErrFrameTooLarge indicates the peer sent a frame exceeding maxFrameBytes.
var ErrFrameTooLarge = errors.New("telemetry: frame exceeds maximum size")

// WriteFrame serializes f to w using the length-prefixed JSON wire format.
func WriteFrame(w io.Writer, f Frame) error {
	body, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if len(body) > maxFrameBytes {
		return ErrFrameTooLarge
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(body)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// ReadFrame decodes one frame from r. Returns io.EOF cleanly when the peer
// closes the connection between frames.
func ReadFrame(r io.Reader) (Frame, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return Frame{}, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > maxFrameBytes {
		return Frame{}, ErrFrameTooLarge
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Frame{}, err
	}
	var f Frame
	if err := json.Unmarshal(buf, &f); err != nil {
		return Frame{}, err
	}
	return f, nil
}
