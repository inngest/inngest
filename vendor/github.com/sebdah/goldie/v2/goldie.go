// Package goldie provides test assertions based on golden files. It's
// typically used for testing responses with larger data bodies.
//
// The concept is straight forward. Valid response data is stored in a "golden
// file". The actual response data will be byte compared with the golden file,
// and the test will fail if there is a difference.
//
// Updating the golden file can be done by running `go test -update ./...`.
package goldie

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	// defaultFixtureDir is the folder name for where the fixtures are stored.
	// It's relative to the "go test" path.
	defaultFixtureDir = "testdata"

	// defaultFileNameSuffix is the suffix appended to the fixtures. Set to
	// empty string to disable file name suffixes.
	defaultFileNameSuffix = ".golden"

	// defaultFilePerms is used to set the permissions on the golden fixture
	// files.
	defaultFilePerms os.FileMode = 0644

	// defaultDirPerms is used to set the permissions on the golden fixture
	// folder.
	defaultDirPerms os.FileMode = 0755

	// defaultDiffEngine sets which diff engine to use if not defined.
	defaultDiffEngine = ClassicDiff

	// defaultIgnoreTemplateErrors sets the default value for the
	// WithIgnoreTemplateErrors option.
	defaultIgnoreTemplateErrors = false

	// defaultUseTestNameForDir sets the default value for the
	// WithTestNameForDir option.
	defaultUseTestNameForDir = false

	// defaultUseSubTestNameForDir sets the default value for the
	// WithSubTestNameForDir option.
	defaultUseSubTestNameForDir = false
)

var (
	// update determines if the actual received data should be written to the
	// golden files or not. This should be true when you need to update the
	// golden files, but false when actually running the tests.
	update = flag.Bool("update", truthy(os.Getenv("GOLDIE_UPDATE")), "Update golden test file fixture")

	// withTemplate determines if the templating data should be applied to the
	// golden files or not. This should be true when you need to update the
	// golden file with templating data, but false when actually running the
	// tests.
	withTemplate = flag.Bool("template", truthy(os.Getenv("GOLDIE_TEMPLATE")), "Apply template data to golden test file fixture")

	// clean determines if we should remove old golden test files in the output
	// directory or not. This only takes effect if we are updating the golden
	// test files.
	clean = flag.Bool("clean", truthy(os.Getenv("GOLDIE_CLEAN")), "Clean old golden test files before writing new olds")

	// ts saves the timestamp of the test run. We use ts to mark the
	// modification time of golden file dirs for cleaning if required by
	// `-clean` flag.
	ts = time.Now()
)

// Goldie is the root structure for the test runner. It provides test assertions based on golden files. It's
// typically used for testing responses with larger data bodies.
type Goldie struct {
	fixtureDir     string
	fileNameSuffix string
	filePerms      os.FileMode
	dirPerms       os.FileMode

	equalFn              EqualFn
	diffEngine           DiffEngine
	diffFn               DiffFn
	ignoreTemplateErrors bool
	useTestNameForDir    bool
	useSubTestNameForDir bool
}

// === Create new testers ==================================

// New creates a new golden file tester. If there is an issue with applying any
// of the options, an error will be reported and t.FailNow() will be called.
func New(t testing.TB, options ...Option) *Goldie {
	g := Goldie{
		fixtureDir:           defaultFixtureDir,
		fileNameSuffix:       defaultFileNameSuffix,
		filePerms:            defaultFilePerms,
		dirPerms:             defaultDirPerms,
		diffEngine:           defaultDiffEngine,
		ignoreTemplateErrors: defaultIgnoreTemplateErrors,
		useTestNameForDir:    defaultUseTestNameForDir,
		useSubTestNameForDir: defaultUseSubTestNameForDir,
	}

	var err error
	for _, option := range options {
		err = option(&g)
		if err != nil {
			t.Error(fmt.Errorf("could not apply option: %w", err))
			t.FailNow()
		}
	}

	return &g
}

// Diff generates a string that shows the difference between the actual and the
// expected. This method could be called in your own DiffFn in case you want
// to leverage any of the engines defined.
func Diff(engine DiffEngine, actual string, expected string) (diff string) {
	switch engine {
	case Simple:
		diff = fmt.Sprintf("Expected: %s\nGot: %s", expected, actual)

	case ClassicDiff:
		diff, _ = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expected),
			B:        difflib.SplitLines(actual),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  1,
		})

	case ColoredDiff:
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		diff = dmp.DiffPrettyText(diffs)

	default: // Simple
		diff = fmt.Sprintf("Expected: %s\nGot: %s", expected, actual)
	}

	return diff
}

// meta takes any data structure and returns a map of the data structure's
// values to their paths. This allows us to replace values in the golden file
// with template variables that match the values.
func meta(a interface{}) map[string]string {
	meta := map[string]string{}
	v := reflect.ValueOf(a)
	var recurseValuePath func(v reflect.Value, path string)
	recurseValuePath = func(v reflect.Value, path string) {
		switch v.Kind() {
		case reflect.Ptr:
			recurseValuePath(v.Elem(), path)
		case reflect.Interface:
			recurseValuePath(reflect.ValueOf(v.Interface()), path)
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				key := v.Type().Field(i).Name
				recurseValuePath(v.Field(i), joinPath(path, key))
			}
		case reflect.Map:
			iter := v.MapRange()
			for iter.Next() {
				key := iter.Key().Interface()
				recurseValuePath(iter.Value(), joinPath(path, key))
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				recurseValuePath(v.Index(i), fmt.Sprintf("index (%s) %d", path, i))
			}
		default:
			meta[fmt.Sprintf("%v", v)] = path
		}
	}
	recurseValuePath(v, ".")
	return meta
}

// Update will update the golden fixtures with the received actual data.
//
// This method does not need to be called from code, but it's exposed so that
// it can be explicitly called if needed. The more common approach would be to
// update using `go test -update ./...` or `GOLDIE_UPDATE=true go test ./...`.
func (g *Goldie) Update(t testing.TB, name string, actualData []byte) error {
	goldenFile := g.GoldenFileName(t, name)
	goldenFileDir := filepath.Dir(goldenFile)
	if err := g.ensureDir(goldenFileDir); err != nil {
		return err
	}

	if err := os.WriteFile(goldenFile, actualData, g.filePerms); err != nil {
		return err
	}

	if err := os.Chtimes(goldenFileDir, ts, ts); err != nil {
		return err
	}

	return nil
}

// UpdateWithTemplate will update the golden fixtures with the received actual
// data, replacing any values in the actual data with template variables that
// match the values in the data structure.
//
// This method does not need to be called from code, but it's exposed so that
// it can be explicitly called if needed. The more common approach would be to
// update using `go test -update ./...` or `GOLDIE_UPDATE=true go test ./...`.
func (g *Goldie) UpdateWithTemplate(t testing.TB, name string, data interface{}, actualData []byte) error {
	meta := meta(data)

	// get a reverse-sorted list of map keys so that when we loop over them,
	// we replace the most specific keys first.
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	// loop over the map and replace any instances of the map key with
	// the map value (which contains the path reference to the value in the
	// data structure).
	for _, key := range keys {
		ref := fmt.Sprintf("{{%s}}", meta[key])
		actualData = bytes.ReplaceAll(actualData, []byte(key), []byte(ref))
	}

	return g.Update(t, name, actualData)
}

// joinPath will join the path with the key, ensuring that the path is
// formatted correctly. If the path is ".", it will simply append the key to
// the path. Otherwise, it will append the key with a dot separator.
func joinPath(path string, key interface{}) string {
	if path == "." {
		return fmt.Sprintf("%s%v", path, key)
	}
	return fmt.Sprintf("%s.%v", path, key)
}

// ensureDir will create the fixture folder if it does not already exist.
func (g *Goldie) ensureDir(loc string) error {
	s, err := os.Stat(loc)

	switch {
	case err != nil && os.IsNotExist(err):
		// the location does not exist, so make directories to there
		return os.MkdirAll(loc, g.dirPerms)

	case err == nil && s.IsDir() && *clean && s.ModTime().UnixNano() < ts.UnixNano():
		if err := os.RemoveAll(loc); err != nil {
			return err
		}
		return os.MkdirAll(loc, g.dirPerms)

	case err == nil && !s.IsDir():
		return newErrFixtureDirectoryIsFile(loc)
	}

	return err
}

// GoldenFileName simply returns the file name of the golden file fixture.
func (g *Goldie) GoldenFileName(t testing.TB, name string) string {
	dir := g.fixtureDir

	if g.useTestNameForDir {
		dir = filepath.Join(dir, strings.Split(t.Name(), "/")[0])
	}

	if g.useSubTestNameForDir {
		n := strings.Split(t.Name(), "/")
		if len(n) > 1 {
			dir = filepath.Join(append([]string{dir}, n[1:]...)...)
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s%s", name, g.fileNameSuffix))
}

// GoldenFileData returns the data from the requested golden fixture file.
// `name` refers to the name of the test and it should typically be unique within the package.
// Also it should be a valid file name (so keeping to `a-z0-9\-\_` is a good
// idea).
func (g *Goldie) GoldenFileData(t testing.TB, name string) []byte {
	expectedData, err := g.goldenFileData(t, name)
	if err != nil {
		t.Fatal(err)
	}
	return expectedData
}

// goldenFileData returns the data from the requested golden fixture file.
// `name` refers to the name of the test and it should typically be unique within the package.
// Also it should be a valid file name (so keeping to `a-z0-9\-\_` is a good
// idea).
func (g *Goldie) goldenFileData(t testing.TB, name string) ([]byte, error) {
	expectedData, err := os.ReadFile(g.GoldenFileName(t, name))

	if err != nil {
		if os.IsNotExist(err) {
			return nil, newErrFixtureNotFound()
		}

		return nil, fmt.Errorf("expected %s to be nil", err.Error())
	}

	return expectedData, nil
}

func truthy(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true", "t":
		return true
	default:
		return false
	}
}
