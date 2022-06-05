package function

import (
	"fmt"
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
func PathName(path string) (string, error) {
	if !strings.HasPrefix(path, FilePrefix) {
		return "", ErrNoPath
	}
	return strings.Replace(path, FilePrefix, "", 1), nil
}
