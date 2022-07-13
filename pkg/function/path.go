package function

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	FilePrefix = "file://"
)

var (
	ErrNoPath = fmt.Errorf("the provided path could not be parsed")

	DefaultStepPath = fmt.Sprintf("%s./steps/%s", FilePrefix, DefaultStepName)
)

// PathName returns the path as defined with the "file://" prefix removed.
func PathName(ctx context.Context, path string) (string, error) {
	var prefix string
	if ctx.Value(pathCtxKey) != nil {
		prefix = ctx.Value(pathCtxKey).(string)
	}
	if !strings.HasPrefix(path, FilePrefix) {
		return "", ErrNoPath
	}
	return filepath.Join(prefix, strings.Replace(path, FilePrefix, "", 1)), nil
}
