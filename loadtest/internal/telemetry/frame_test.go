package telemetry

import (
	"bytes"
	"io"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	want := Frame{
		WorkerID:      "w-1",
		Seq:           42,
		InngestRunID:  "run-abc",
		CorrelationID: "corr-xyz",
		FunctionSlug:  "steps-3",
		StepID:        "s2",
		Attempt:       1,
		Phase:         PhaseStepEnd,
		TSNanos:       1_700_000_000_000_000_000,
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if got != want {
		t.Errorf("roundtrip mismatch:\n  got  %+v\n  want %+v", got, want)
	}
}

func TestReadFrameEOF(t *testing.T) {
	_, err := ReadFrame(&bytes.Buffer{})
	if err != io.EOF && err != io.ErrUnexpectedEOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestReadFrameTooLarge(t *testing.T) {
	// header declaring size 10MB, which exceeds maxFrameBytes.
	hdr := []byte{0x00, 0xA0, 0x00, 0x00}
	_, err := ReadFrame(bytes.NewReader(hdr))
	if err != ErrFrameTooLarge {
		t.Errorf("expected ErrFrameTooLarge, got %v", err)
	}
}
