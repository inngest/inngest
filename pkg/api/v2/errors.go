package apiv2

import "errors"

var (
	ErrFunctionNotFound      = errors.New("function not found")
	ErrAppNotFound           = errors.New("app not found")
	ErrRunNotFound           = errors.New("run not found")
	ErrCronRerunNotSupported = errors.New("cron rerun is not supported")
)
