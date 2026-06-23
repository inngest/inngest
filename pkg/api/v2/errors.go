package apiv2

import "errors"

var (
	ErrFunctionNotFound      = errors.New("function not found")
	ErrAppNotFound           = errors.New("app not found")
	ErrRunNotFound           = errors.New("run not found")
	ErrCronRerunNotSupported = errors.New("cron rerun is not supported")

	// ErrScoresNotEnabled is returned by ScoreProvider implementations when
	// score submission is not enabled for the authenticated account.
	ErrScoresNotEnabled = errors.New("scores are not enabled")

	// ErrScoreTargetNotFound is returned by ScoreProvider implementations when
	// the run or step a score targets cannot be found.
	ErrScoreTargetNotFound = errors.New("score target not found")
)
