package cuedefs

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/pkg/errors"
)

//go:embed pkg/**/*.cue
var FS embed.FS

const (
	packageName = "inngest.com"
)

var (
	ErrInvalid      = errors.New("definition is invalid")
	ErrNoDefinition = errors.New("no definition provided in config")
)

// internalPackageLoader reads all packages from FS, adding them to a cue loader
// so that they can be referenced by cue files.
func internalPackageLoader() (*load.Config, error) {
	cfg := &load.Config{
		Dir:        string(filepath.Separator),
		ModuleRoot: string(filepath.Separator),
		Overlay:    map[string]load.Source{},
	}

	err := fs.WalkDir(FS, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		if filepath.Ext(entry.Name()) != ".cue" {
			return nil
		}

		contents, err := FS.ReadFile(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}

		// Ensure we push all of our schemas into a "cue.mod/pkg/inngest.com" directory -
		// by default, Cue looks up packages in this directory.
		//
		// This ensures that we can reference any of or packages via "inngest.com/workflows",
		// for example.
		p = strings.ReplaceAll(p, "pkg/", "") // "inngest.com/workflows", vs "inngest.com/pkg/workflows"
		root := filepath.Join(string(filepath.Separator), "cue.mod", "pkg", packageName, p)

		// Add this file to the cue loader so that it can be read.
		cfg.Overlay[root] = load.FromBytes(contents)

		// And, Cue on windows is odd, and requires a C:\ prefix to our files _in addition to_
		// root slashes.
		if runtime.GOOS == "windows" {
			path, err := os.Executable()
			if err != nil {
				return err
			}

			// Get the current disk, then use this as a prefix.
			disk := path[0:3] + root
			cfg.Overlay[disk] = load.FromBytes(contents)

			// This is the logic which Cue uses, which may
			// result in a different root disk than the OS
			// executable.  This defers to syscall.FullPath
			// to return a disk prefix.
			abs, _ := filepath.Abs(string(filepath.Separator))
			cleaned := filepath.Clean(abs + root)
			cfg.Overlay[cleaned] = load.FromBytes(contents)
		}

		return nil
	})

	if err != nil {
		return cfg, fmt.Errorf("erorr initializing packages: %w", err)
	}

	return cfg, err
}

func parseDef(input, lookup, suffix string) (*cue.Value, error) {
	cfg, err := internalPackageLoader()
	if err != nil {
		return nil, err
	}

	// XXX: We can't (to my knowledge) look up the #Workflow definition within
	// inngest.com/workflows.
	//
	// It would be nice to be able to make shre that the workflow definition is
	// of type workflow via:
	//
	// 	wf := inst.LookupDef("#Workflow")
	// 	if wf.Err() != nil {
	// 		return nil, fmt.Errorf("error finding definition: %w", wf.Err())
	// 	}
	//
	// 	if !wval.Subsumes(wf) {
	// 		return nil, errors.New("workflow is not an instance of workflows.#Workflow")
	// 	}
	//
	// Howevwer, as we can't look up that def this doens't work.
	//
	// This is why we have a hack:  we suffix the file with a string ensuring workflow is of
	// the correct type.

	// Add the input as Stdin and load the instances using the "-" filename, which means "read from
	// stdin".  The presence of stdin is not enough here.
	cfg.Stdin = strings.NewReader(fmt.Sprintf("%s\n%s", input, suffix))
	instances := load.Instances([]string{"-"}, cfg)

	if len(instances) != 1 {
		return nil, fmt.Errorf("unsupported number of packages: %d", len(instances))
	}

	runtime := cuecontext.New()
	inst := runtime.BuildInstance(instances[0])
	if err := inst.Err(); err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, err, nil)
		return nil, fmt.Errorf("error parsing config: %s", buf.String())
	}

	// Find the variable defined as "workflow":  this is a constant used to reference
	// the workflow in our cue configuration.
	wval := inst.Lookup(lookup)
	if err := wval.Err(); err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, err, nil)
		return nil, fmt.Errorf("error validating config: %s", buf.String())
	}

	if err := wval.Validate(cue.Final(), cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("config is invalid: %w:", err)
	}

	return &wval, nil
}

func FormatValue(input *cue.Value) (string, error) {
	syn := input.Syntax(
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
	return string(out), err
}

// FormatDef takes an input and formats the go value according to cue conventions.
func FormatDef(input interface{}) (string, error) {
	var r cue.Runtime
	codec := gocodec.New(&r, nil)
	v, err := codec.Decode(input)
	if err != nil {
		return "", err
	}
	return FormatValue(&v)
}
