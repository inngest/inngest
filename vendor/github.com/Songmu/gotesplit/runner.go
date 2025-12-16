package gotesplit

import (
	"context"
	"io"
)

var dispatch = map[string]runner{
	"regexp": &cmdRegexp{},
}

type runner interface {
	run(context.Context, []string, io.Writer, io.Writer) error
}
