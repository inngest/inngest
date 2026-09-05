package util

import (
	"os"
	"path/filepath"

	"github.com/inngest/inngest/pkg/consts"
)

// ResolveStateDir resolves the on-disk directory that persistent local state
// (SQLite databases, the DuckDB catalog, ...) lives in.
//
// An empty dir means "use the default", consts.DefaultInngestConfigDir. A
// non-empty relative override is made absolute against the process's working
// directory, so callers never accidentally scatter state relative to whatever
// cwd the process happens to be started from later.
func ResolveStateDir(dir string) (string, error) {
	if dir == "" {
		return consts.DefaultInngestConfigDir, nil
	}
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, dir), nil
}
