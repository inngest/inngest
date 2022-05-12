package actionloader

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/inngest/inngestctl/internal/cuedefs"
)

// FSLoader returns a new action loader which recursively scans the given directory for
// actions, parsing them and storing them in a MemoryLoader.
func FSLoader(path string) (ActionLoader, error) {
	loader := NewMemoryLoader()

	var walk fs.WalkDirFunc
	walk = func(path string, d fs.DirEntry, err error) error {
		file, _ := filepath.Abs(path)
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".cue") {
			return nil
		}
		byt, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("error reading file '%s': %w", file, err)
		}
		// Attempt to parse the action.
		action, err := cuedefs.ParseAction(string(byt))
		if err != nil {
			// Ignore ill-defined actions for now.
			return nil
		}
		loader.Add(*action)
		return nil
	}

	if err := filepath.WalkDir(path, walk); err != nil {
		return nil, err
	}

	return loader, nil
}
