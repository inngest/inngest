// cuedefs provides cue definitions for configuring functions and events within Inngest.
//
// It also provides an embed.FS which contains the cue definitions for use within Go at
// runtime.
package cuedefs

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
)

// FS embeds the cue module and definitions.
//
//go:embed cue.mod v1 config
var FS embed.FS

var (
	lock *sync.Mutex
)

func init() {
	lock = &sync.Mutex{}
}

// Unfortunately, cue is not thread safe.  We only parse cue when reading and validating
// configuration;  parsed functions and workflows are cached.  We add a mutex here
// to prevent concurrent access to Cue right now.
//
// Lock claims the mutex.
func Lock() {
	lock.Lock()
}

// Unlock releases the mutex.
func Unlock() {
	lock.Unlock()
}

// Prepare generates a cue instance for the configuration.
func Prepare(input []byte) (*cue.Instance, error) {
	cfg := &load.Config{
		Overlay:    map[string]load.Source{},
		Dir:        string(filepath.Separator),
		ModuleRoot: string(filepath.Separator),
		Package:    "inngest.com/defs",
		Stdin:      bytes.NewBuffer(input),
	}

	// Add each of the embedded cue files from our definitions to our config.
	err := fs.WalkDir(FS, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := FS.ReadFile(p)
		if err != nil {
			return err
		}

		// We'll join this using the OS-specific path separator;  for some reason fs.WalkDir
		// doesn't respect os-specific separators and always uses forward slashes.
		p = filepath.Join(strings.Split(p, "/")...)

		// Always add a root slash prior to the overlat.
		cfg.Overlay[string(filepath.Separator)+p] = load.FromBytes(contents)

		// And, Cue on windows is odd, and requires a C:\ prefix to our files _in addition to_
		// root slashes.
		if runtime.GOOS == "windows" {
			cfg.Overlay[`C:\`+p] = load.FromBytes(contents)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Add the input.
	builds := load.Instances([]string{"-"}, cfg)
	if len(builds) != 1 {
		return nil, fmt.Errorf("unexpected cue build instances generated: %d", len(builds))
	}

	if builds[0].Err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, builds[0].Err, nil)
		return nil, fmt.Errorf("error loading instance: %s", buf.String())
	}

	r := &cue.Runtime{}
	inst, err := r.Build(builds[0])
	if err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, err, nil)
		return nil, fmt.Errorf("error building instance: %s", buf.String())
	}

	return inst, nil
}
