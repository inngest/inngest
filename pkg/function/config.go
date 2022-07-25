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
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/inngest/inngest-cli/pkg/cuedefs"
)

var (
	ErrNotFound = fmt.Errorf("No inngest file could be found.")
)

const (
	cueConfigName  = "inngest.cue"
	jsonConfigName = "inngest.json"
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
	cue, byt, err := findFileUp(filepath.Join(abs, cueConfigName))
	if err != nil {
		return nil, err
	}
	if cue != "" {
		return Unmarshal(ctx, byt, cue)
	}

	json, byt, err := findFileUp(filepath.Join(abs, jsonConfigName))
	if err != nil {
		return nil, err
	}
	if json != "" {
		return Unmarshal(ctx, byt, json)
	}

	return nil, ErrNotFound
}

// Finds a file at the given `path``, iterating up through the directory tree
// until it finds the file or reaches the root.
//
// Returns the final path and the file contents.
//
// Will return an error if reading a file errored, but will return empty values
// if no file could be found.
func findFileUp(path string) (string, []byte, error) {
	prevPath := ""
	targetPath := path
	fileName := filepath.Base(path)

	for {
		// This will resolve to the same dir if we're at the root.
		// At this point, we've exhausted all options, so return empty values.
		if targetPath == prevPath {
			return "", nil, nil
		}

		if _, err := os.Stat(targetPath); err != nil {
			// Is the error isn't something other than lack of existence, we should
			// error now.
			if !os.IsNotExist(err) {
				return "", nil, err
			}

			// If we're here, the file doesn't exist, so continue.
			prevPath = targetPath
			targetPath = filepath.Join(filepath.Dir(prevPath), "..", fileName)
			continue
		}

		// Stat succeeded, so let's try reading the file
		byt, err := os.ReadFile(targetPath)
		if err != nil {
			return "", nil, err
		}

		return targetPath, byt, nil
	}
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
		if f.Name() != cueConfigName && f.Name() != jsonConfigName {
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
	cuedefs.Lock()
	defer cuedefs.Unlock()

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

// MarshalCUE formats a function into canonical cue configuration.
func MarshalCUE(f Function) ([]byte, error) {
	return formatCue(f)
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
		if strings.HasPrefix(p, "config") {
			// Config definitions are used to manage services only.
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

func formatCue(fn Function) ([]byte, error) {
	var r cue.Runtime
	codec := gocodec.New(&r, nil)
	v, err := codec.Decode(fn)
	if err != nil {
		return nil, err
	}

	syn := v.Syntax(
		cue.Docs(true),
		cue.Attributes(true),
		cue.Optional(true),
		cue.Definitions(true),
		cue.ResolveReferences(true),
		cue.Final(),
	)
	out, err := format.Node(
		syn,
		format.Simplify(),
		format.TabIndent(false),
		format.UseSpaces(2),
	)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(fnTpl, string(out))), nil
}

const fnTpl = `package main

import (
	defs "inngest.com/defs/v1"
)

function: defs.#Function & %s`
