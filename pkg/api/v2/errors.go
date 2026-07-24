package apiv2

import "errors"

var (
	ErrFunctionNotFound      = errors.New("function not found")
	ErrAppNotFound           = errors.New("app not found")
	ErrRunNotFound           = errors.New("run not found")
	ErrCronRerunNotSupported = errors.New("cron rerun is not supported")
	ErrRerunStepNotFound     = errors.New("rerun step not found")
	ErrRerunStepAmbiguous    = errors.New("rerun step name is ambiguous")

	// ErrScoresNotEnabled is returned by ScoreProvider implementations when
	// score submission is not enabled for the authenticated account.
	ErrScoresNotEnabled = errors.New("scores are not enabled")
)
