package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/inngest/inngest/pkg/cuedefs"
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

	// Then attempt to find inngest.cue|json, the canonical reference.
	configPath, byt, err := findConfigFileUp(abs)
	if err != nil {
		return nil, err
	}
	if configPath != "" {
		return Unmarshal(ctx, byt, configPath)
	}

	return nil, ErrNotFound
}

// Finds an Inngest config file at the given `path“, iterating up through the
// directory tree until it finds a file or reaches the root.
//
// Returns the final path and the file contents.
//
// Will return an error if reading a file errored, but will return empty values
// if no file could be found.
func findConfigFileUp(path string) (string, []byte, error) {
	prevDir := ""
	targetDir := path
	foundPath := ""

	for {
		// This will resolve to the same dir if we're at the root.
		// At this point, we've exhausted all options, so return empty values.
		if targetDir == prevDir {
			return "", nil, nil
		}

		cueConfigPath := filepath.Join(targetDir, cueConfigName)
		_, err := os.Stat(cueConfigPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", nil, err
			}
		} else {
			foundPath = cueConfigPath
			break
		}

		jsonConfigPath := filepath.Join(targetDir, jsonConfigName)
		_, err = os.Stat(jsonConfigPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", nil, err
			}
		} else {
			foundPath = jsonConfigPath
			break
		}

		// If we're here, no config files could be found in this dir, so move up.
		prevDir = targetDir
		targetDir = filepath.Join(targetDir, "..")
	}

	// If we're here, we've found a file that looks correct; let's try to read it.
	byt, err := os.ReadFile(foundPath)
	if err != nil {
		return "", nil, err
	}

	return foundPath, byt, nil
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
	path = filepath.FromSlash(path)

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
	return cuedefs.Prepare(input)
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
