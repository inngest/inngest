package gotesplit

import (
	"context"
	"io"
)

type runner interface {
	run(context.Context, []string, io.Writer, io.Writer) error
}

func dispatch(mode listMode) map[string]runner {
	return map[string]runner{
		"regexp": &cmdRegexp{mode: mode},
	}
}
