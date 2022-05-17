package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/inngest/inngestctl/pkg/cuedefs"
)

var (
	ErrNotFound = fmt.Errorf("No inngest file could be found.")
)

// Load loads the inngest function from the given directory.  It searches for both inngest.cue
// and inngest.json as both are supported.  If neither exist, this returns ErrNotFound.
func Load(ctx context.Context, dir string) (*Function, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// First attempt to read the specific file given to us.
	stat, err := os.Stat(abs)
	if err == nil {
		if !stat.IsDir() {
			// The cue file exists.
			byt, err := os.ReadFile(abs)
			if err != nil {
				return nil, err
			}
			return Unmarshal(ctx, byt, abs)
		}
	}

	// Then attempt to find inngest.cue, the canonical reference.
	cue := filepath.Join(abs, "inngest.cue")
	if _, err := os.Stat(cue); err == nil {
		// The cue file exists.
		byt, err := os.ReadFile(cue)
		if err != nil {
			return nil, err
		}
		return Unmarshal(ctx, byt, cue)
	}

	// Finally, use inngest.json in the given dir.
	json := filepath.Join(abs, "inngest.json")
	if _, err := os.Stat(json); err != nil {
		// This doesn't exist.  Return ErrNotFound.
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	byt, err := os.ReadFile(json)
	if err != nil {
		return nil, err
	}
	return Unmarshal(ctx, byt, json)
}

func LoadRecursive(ctx context.Context, dir string) ([]*Function, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	var functions []*Function
	err = filepath.WalkDir(abs, func(path string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if f.Name() != "inngest.cue" && f.Name() != "inngest.json" {
			return nil
		}
		function, err := Load(ctx, path)
		if err != nil {
			return err
		}
		functions = append(functions, function)
		return nil
	})
	if err != nil {
		return []*Function{}, err
	}
	return functions, nil
}

// Unmarshal parses the input data and returns a function definition or an error.  The input
// data may be either a cue definition of a function or a JSON object containing the function
// definition.  Our canonical reference and format is Cue, although we allow JSON to be passed
// for ease of use.
//
// This validates the function after parsing, returning any validation errors.
func Unmarshal(ctx context.Context, input []byte, path string) (*Function, error) {
	// Note that cue is a superset of JSON;  we can parse the input using our cue definition
	// for both a JSON and Cue input.
	instance, err := prepare(input)
	if err != nil {
		return nil, err
	}
	fn, err := parse(instance)
	if err != nil {
		return nil, err
	}

	// Note that some of the fields are optional for a quick-start experience.  For example,
	// it's not necessary to include a "step" array if you have a single step function which
	// runs custom code.
	//
	// Here we want to ensure that the struct fields are all filled out in a canonical
	// format.
	if err := fn.canonicalize(ctx, path); err != nil {
		return nil, err
	}

	if err := fn.Validate(ctx); err != nil {
		return nil, fmt.Errorf("The function is not valid: %w", err)
	}

	// Store the directory for future reference.
	fn.dir = filepath.Dir(path)

	return fn, nil
}

// MarshalJSON marshals a function to pretty JSON.  It's a plain wrapper
// around json.MarshalIndent with defaults.
func MarshalJSON(f Function) ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

// prepare generates a cue instance for the configuration.
func prepare(input []byte) (*cue.Instance, error) {
	cfg := &load.Config{
		Overlay:    map[string]load.Source{},
		Dir:        "/",
		ModuleRoot: "/",
		Package:    "inngest.com/defs",
		Stdin:      bytes.NewBuffer(input),
	}

	// Add each of the embedded cue files from our definitions to our config.
	err := fs.WalkDir(cuedefs.FS, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := cuedefs.FS.ReadFile(p)
		if err != nil {
			return err
		}

		cfg.Overlay[path.Join("/", p)] = load.FromBytes(contents)
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
		return nil, fmt.Errorf("error loading instance: %w", builds[0].Err)
	}

	r := &cue.Runtime{}
	inst, err := r.Build(builds[0])
	if err != nil {
		return nil, fmt.Errorf("error building instance: %w", err)
	}

	return inst, nil
}

// parse attempts to parse the input within a cue instance.
func parse(i *cue.Instance) (*Function, error) {
	// Initialize our definition as the root value of the cue instance.  This is
	// the root, top-level object.
	def := i.Value()

	field, err := i.LookupField("function")
	if err == nil {
		// This is a cue definition which contains a function definition.  Cue
		// definitions always have a root level object, and we define the function
		// using the "function" identifier.
		def = field.Value
	}

	// XXX: When we can, pull out the definition of "v1.#Functions" and ensure
	// that the value "Subsumes" the definition.
	//
	// See https://github.com/cue-lang/cue/discussions/1571 for more info.

	if err := def.Validate(cue.Final(), cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("function is not valid: %w", err)
	}

	f := &Function{}
	if err := def.Decode(f); err != nil {
		return nil, fmt.Errorf("error decoding function: %w", err)
	}

	return f, nil
}
