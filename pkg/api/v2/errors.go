package apiv2

import (
	"errors"

	"github.com/inngest/inngest/pkg/enums"
)

var (
	ErrFunctionNotFound      = errors.New("function not found")
	ErrAppNotFound           = errors.New("app not found")
	ErrRunNotFound           = errors.New("run not found")
	ErrCronRerunNotSupported = errors.New("cron rerun is not supported")
	ErrRerunStepNotFound     = errors.New("rerun step not found")
	ErrRerunStepAmbiguous    = errors.New("rerun step name is ambiguous")
	ErrRunAlreadyCancelled   = errors.New("run is already cancelled")
	ErrRunEnded              = errors.New("run has already ended")

	// ErrScoresNotEnabled is returned by ScoreProvider implementations when
	// score submission is not enabled for the authenticated account.
	ErrScoresNotEnabled = errors.New("scores are not enabled")
)

// RunCancellability returns an error when a run's status prevents cancellation.
func RunCancellability(status enums.RunStatus) error {
	//
	// RunStatusEnded includes cancelled runs, so preserve the more specific error.
	if status == enums.RunStatusCancelled {
		return ErrRunAlreadyCancelled
	}
	if enums.RunStatusEnded(status) {
		return ErrRunEnded
	}
	return nil
}
