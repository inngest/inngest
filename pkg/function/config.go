package function

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/inngest/inngestctl/pkg/cuedefs"
)

// Unmarshal parses the input data and returns a function definition or an error.  The input
// data may be either a cue definition of a function or a JSON object containing the function
// definition.  Our canonical reference and format is Cue, although we allow JSON to be passed
// for ease of use.
//
// This validates the function after parsing, returning any validation errors.
func Unmarshal(input []byte) (*Function, error) {
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
	if err := fn.Validate(); err != nil {
		return nil, err
	}
	return fn, nil
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
