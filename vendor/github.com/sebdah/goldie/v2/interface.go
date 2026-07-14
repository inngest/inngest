package goldie

import (
	"os"
	"testing"
)

// Compile time assurance
var _ Tester = (*Goldie)(nil)
var _ OptionProcessor = (*Goldie)(nil)

// Option defines the signature of a functional option method that can apply
// options to an OptionProcessor.
type Option func(OptionProcessor) error

// Tester defines the methods that any golden tester should support.
type Tester interface {
	Assert(t testing.TB, name string, actualData []byte)
	AssertJson(t testing.TB, name string, actualJsonData interface{})
	AssertXml(t testing.TB, name string, actualXmlData interface{})
	AssertWithTemplate(t testing.TB, name string, data interface{}, actualData []byte)
	Update(t testing.TB, name string, actualData []byte) error
	GoldenFileName(t testing.TB, name string) string
	GoldenFileData(t testing.TB, name string) []byte
}

// EqualFn compares if actual and expected are equal.
type EqualFn func(actual []byte, expected []byte) bool

// DiffFn takes in an actual and expected and will return a diff string
// representing the differences between the two.
type DiffFn func(actual string, expected string) string

// DiffEngine is used to enumerate the diff engine processors that are
// available.
type DiffEngine int

//noinspection GoUnusedConst
const (
	// UndefinedDiff represents any undefined diff processor. If a new diff
	// engine is implemented, it should be added to this enumeration and to the
	// `diff` helper function.
	UndefinedDiff DiffEngine = iota

	// ClassicDiff produces a diff similar to what the `diff` tool would
	// produce.
	//		+++ Actual
	//		@@ -1 +1 @@
	//		-Lorem dolor sit amet.
	//		+Lorem ipsum dolor.
	//
	ClassicDiff

	// ColoredDiff produces a diff that will use red and green colors to
	// distinguish the diffs between the two values.
	ColoredDiff

	// Simple is a very plain way of printing the byte comparison. It will look
	// like this:
	//
	// Expected: <data>
	// Got: <data>
	Simple
)

// OptionProcessor defines the functions that can be called to set values for
// a tester.  To expand this list, add a function to this interface and then
// implement the generic option setter below.
type OptionProcessor interface {
	WithFixtureDir(dir string) error
	WithNameSuffix(suffix string) error
	WithFilePerms(mode os.FileMode) error
	WithDirPerms(mode os.FileMode) error

	WithEqualFn(fn EqualFn) error
	WithDiffEngine(engine DiffEngine) error
	WithDiffFn(fn DiffFn) error
	WithIgnoreTemplateErrors(ignoreErrors bool) error
	WithTestNameForDir(use bool) error
	WithSubTestNameForDir(use bool) error
}

// === OptionProcessor ===============================

// WithFixtureDir sets the fixture directory.
//
// Defaults to `testdata`
func WithFixtureDir(dir string) Option {
	return func(o OptionProcessor) error {
		return o.WithFixtureDir(dir)
	}
}

// WithNameSuffix sets the file suffix to be used for the golden file.
//
// Defaults to `.golden`
func WithNameSuffix(suffix string) Option {
	return func(o OptionProcessor) error {
		return o.WithNameSuffix(suffix)
	}
}

// WithFilePerms sets the file permissions on the golden files that are
// created.
//
// Defaults to 0644.
//noinspection GoUnusedExportedFunction
func WithFilePerms(mode os.FileMode) Option {
	return func(o OptionProcessor) error {
		return o.WithFilePerms(mode)
	}
}

// WithDirPerms sets the directory permissions for the directories in which the
// golden files are created.
//
// Defaults to 0755.
//noinspection GoUnusedExportedFunction
func WithDirPerms(mode os.FileMode) Option {
	return func(o OptionProcessor) error {
		return o.WithDirPerms(mode)
	}
}

// WithEqualFn sets the customized equality comapre function that implements
// the EqualFn signature.
//noinspection GoUnusedExportedFunction
func WithEqualFn(fn EqualFn) Option {
	return func(o OptionProcessor) error {
		return o.WithEqualFn(fn)
	}
}

// WithDiffEngine sets the `diff` engine that will be used to generate the
// `diff` text.
//
// Default: ClassicDiff
//noinspection GoUnusedExportedFunction
func WithDiffEngine(engine DiffEngine) Option {
	return func(o OptionProcessor) error {
		return o.WithDiffEngine(engine)
	}
}

// WithDiffFn sets the `diff` engine to be a function that implements the
// DiffFn signature. This allows for any customized diff logic you would like
// to create.
//noinspection GoUnusedExportedFunction
func WithDiffFn(fn DiffFn) Option {
	return func(o OptionProcessor) error {
		return o.WithDiffFn(fn)
	}
}

// WithIgnoreTemplateErrors allows template processing to ignore any variables
// in the template that do not have corresponding data values passed in.
//
// Default value is false.
//noinspection GoUnusedExportedFunction
func WithIgnoreTemplateErrors(ignoreErrors bool) Option {
	return func(o OptionProcessor) error {
		return o.WithIgnoreTemplateErrors(ignoreErrors)
	}
}

// WithTestNameForDir will create a directory with the test's name in the
// fixture directory to store all the golden files.
//
// Default value is false.
//noinspection GoUnusedExportedFunction
func WithTestNameForDir(use bool) Option {
	return func(o OptionProcessor) error {
		return o.WithTestNameForDir(use)
	}
}

// WithSubTestNameForDir will create a directory with the sub test's name to
// store all the golden files. If WithTestNameForDir is enabled, it will be in
// the test name's directory. Otherwise, it will be in the fixture directory.
//
// Default value is false.
//noinspection GoUnusedExportedFunction
func WithSubTestNameForDir(use bool) Option {
	return func(o OptionProcessor) error {
		return o.WithSubTestNameForDir(use)
	}
}
