package main

import (
	"bufio"
	"io"
	"reflect"
	"regexp"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

// TestDeps is an implementation of the testing.testDeps interface,
// suitable for passing to testing.MainStart.
type TestDeps struct{}

var matchPat string
var matchRe *regexp.Regexp

func (TestDeps) MatchString(pat, str string) (result bool, err error) {
	if matchRe == nil || matchPat != pat {
		matchPat = pat
		matchRe, err = regexp.Compile(matchPat)
		if err != nil {
			return
		}
	}
	return matchRe.MatchString(str), nil
}

//func (TestDeps) StartTestLog(io.Writer)                                     {}
//func (TestDeps) StopTestLog() error                                         { return nil }
func (TestDeps) StartCPUProfile(w io.Writer) error                          { return pprof.StartCPUProfile(w) }
func (TestDeps) StopCPUProfile()                                            { pprof.StopCPUProfile() }
func (TestDeps) ImportPath() string                                         { return "" }
func (TestDeps) RunFuzzWorker(fn func(corpusEntry corpusEntry) error) error { return nil }
func (TestDeps) ReadCorpus(string, []reflect.Type) ([]corpusEntry, error)   { return nil, nil }
func (TestDeps) SetPanicOnExit0(bool)                                       {}
func (TestDeps) CheckCorpus(vals []any, types []reflect.Type) error         { return nil }
func (TestDeps) ResetCoverage()                                             {}
func (TestDeps) SnapshotCoverage()                                          {}
func (TestDeps) WriteProfileTo(name string, w io.Writer, debug int) error {
	return pprof.Lookup(name).WriteTo(w, debug)
}
func (TestDeps) CoordinateFuzzing(
	timeout time.Duration,
	limit int64,
	minimizeTimeout time.Duration,
	minimizeLimit int64,
	parallel int,
	seed []corpusEntry,
	types []reflect.Type,
	corpusDir,
	cacheDir string) (err error) {
	// nil: fuzz is internal
	return nil
}
func (TestDeps) StartTestLog(w io.Writer) {
	log.mu.Lock()
	log.w = bufio.NewWriter(w)
	if !log.set {
		// Tests that define TestMain and then run m.Run multiple times
		// will call StartTestLog/StopTestLog multiple times.
		// Checking log.set avoids calling testlog.SetLogger multiple times
		// (which will panic) and also avoids writing the header multiple times.
		log.set = true
		//testlog.SetLogger(&log)
		_, _ = log.w.WriteString("# test log\n") // known to cmd/go/internal/test/test.go
	}
	log.mu.Unlock()
}

func (TestDeps) StopTestLog() error {
	log.mu.Lock()
	defer log.mu.Unlock()
	err := log.w.Flush()
	log.w = nil
	return err
}

// corpusEntry is an alias to the same type as internal/fuzz.CorpusEntry.
// We use a type alias because we don't want to export this type, and we can't
// import internal/fuzz from testing.
type corpusEntry = struct {
	Parent     string
	Path       string
	Data       []byte
	Values     []any
	Generation int
	IsSeed     bool
}

// testLog implements testlog.Interface, logging actions by package os.
type testLog struct {
	mu  sync.Mutex
	w   *bufio.Writer
	set bool
}

func (l *testLog) Getenv(key string) {
	l.add("getenv", key)
}

func (l *testLog) Open(name string) {
	l.add("open", name)
}

func (l *testLog) Stat(name string) {
	l.add("stat", name)
}

func (l *testLog) Chdir(name string) {
	l.add("chdir", name)
}

// add adds the (op, name) pair to the test log.
func (l *testLog) add(op, name string) {
	if strings.Contains(name, "\n") || name == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w == nil {
		return
	}
	_, _ = l.w.WriteString(op)
	_ = l.w.WriteByte(' ')
	_, _ = l.w.WriteString(name)
	_ = l.w.WriteByte('\n')
}

var log testLog
