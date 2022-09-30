package inmemory

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/internal/cuedefs"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/logger"
	"golang.org/x/sync/errgroup"
)

func init() {
	registration.RegisterDataStore(func() any { return &Config{} })
}

type Config struct{}

func (c Config) DataStoreName() string {
	return "inmemory"
}

func (c Config) ReadWriter(ctx context.Context) (coredata.ReadWriter, error) {
	return New(ctx)
}

type ReadWriter struct {
	*MemoryAPIReadWriter
	*MemoryExecutionLoader
}

func New(ctx context.Context) (*ReadWriter, error) {
	return &ReadWriter{
		MemoryAPIReadWriter:   NewInMemoryAPIReadWriter(),
		MemoryExecutionLoader: &MemoryExecutionLoader{},
	}, nil
}

// FSLoader is a function and action loader which returns functions and actions
// by reading the given filesystem path recursively, loading functions and actions
// from function definitions.
//
// This should be initialized via the NewFSLoader function.
type FSLoader struct {
	*MemoryExecutionLoader

	// root stores the root path used when searching the filesystem.
	root string
}

// ReadDir recursively reads the root directory, loading all functions
// into the loader.
func (f *FSLoader) ReadDir(ctx context.Context) error {
	logger.From(ctx).
		Debug().
		Str("dir", f.root).
		Msg("scanning directory for functions")

	fns, err := function.LoadRecursive(ctx, f.root)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		f.functions = append(f.functions, *fn)
	}

	return f.MemoryExecutionLoader.SetFunctions(ctx, fns)
}

// NewFSLoader returns an ExecutionLoader which reads functions from the given
// path, recursively.
func NewFSLoader(ctx context.Context, path string) (coredata.ExecutionLoader, error) {
	// XXX: This should probably be a singleton;  this is primarily used
	// for the dev server.  in this case, a single process hosts the
	// runner and the executor together - and we don't want to process
	// the directory multiple times within the same pid.
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	loader := &FSLoader{root: abspath, MemoryExecutionLoader: &MemoryExecutionLoader{}}
	if err := loader.ReadDir(ctx); err != nil {
		return nil, err
	}
	return loader, nil
}

// MemoryExecutionLoader is a function and action loader which returns data from
// in-memory state.
type MemoryExecutionLoader struct {
	// embed the in-memory action loader for querying action versions found within
	// functions.
	*memactionloader

	// functions stores all functions which were found within the given filesystem.
	functions []function.Function

	// actions stores all actions parsed and read from functions within the filesystem.
	actions []inngest.ActionVersion

	l sync.RWMutex
}

func (m *MemoryExecutionLoader) AddFunction(ctx context.Context, fn *function.Function) error {
	m.l.Lock()
	defer m.l.Unlock()

	actions, _, _ := fn.Actions(ctx)

	// Ensure that this action has a version.  In the case of development servers,
	// actions aren't versioned: so we auto-fill a v1.1.
	for n, a := range actions {
		if a.Version == nil {
			actions[n].Version = &inngest.VersionInfo{
				Major: 1,
				Minor: 1,
			}
		}
	}

	// TODO: Is this function and its actions already present?  If so, remove them.

	m.actions = append(m.actions, actions...)
	m.functions = append(m.functions, *fn)

	// recreate the in-memory action loader.
	m.memactionloader = &memactionloader{
		Actions: make(map[string][]inngest.ActionVersion),
		lock:    &sync.RWMutex{},
	}
	for _, a := range m.actions {
		m.memactionloader.Add(a)
	}

	logger.From(ctx).
		Debug().
		Int("len", len(m.functions)).
		Msg("added functions")

	return nil
}

func (m *MemoryExecutionLoader) SetFunctions(ctx context.Context, f []*function.Function) error {
	m.l.Lock()
	defer m.l.Unlock()

	m.functions = []function.Function{}
	m.actions = []inngest.ActionVersion{}

	// Validate all functions.
	eg := &errgroup.Group{}
	for _, fn := range f {
		copied := fn
		eg.Go(func() error {
			return copied.Validate(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for _, fn := range f {
		actions, _, _ := fn.Actions(ctx)
		m.actions = append(m.actions, actions...)
		m.functions = append(m.functions, *fn)
	}

	// recreate the in-memory action loader.
	m.memactionloader = &memactionloader{
		Actions: make(map[string][]inngest.ActionVersion),
		lock:    &sync.RWMutex{},
	}
	for _, a := range m.actions {
		m.memactionloader.Add(a)
	}

	logger.From(ctx).
		Debug().
		Int("len", len(m.functions)).
		Msg("added functions")

	return nil
}

func (m *MemoryExecutionLoader) Functions(ctx context.Context) ([]function.Function, error) {
	return m.functions[:], nil
}

func (m *MemoryExecutionLoader) FunctionsScheduled(ctx context.Context) ([]function.Function, error) {
	m.l.RLock()
	defer m.l.RUnlock()

	fns := []function.Function{}
	for _, fn := range m.functions {
		for _, t := range fn.Triggers {
			if t.CronTrigger != nil {
				fns = append(fns, fn)
				break
			}
		}
	}
	return fns, nil
}

func (m *MemoryExecutionLoader) FunctionsByTrigger(ctx context.Context, eventName string) ([]function.Function, error) {
	m.l.RLock()
	defer m.l.RUnlock()

	fns := []function.Function{}
	for _, fn := range m.functions {
		for _, t := range fn.Triggers {
			if t.EventTrigger != nil && t.Event == eventName {
				fns = append(fns, fn)
				break
			}
		}
	}
	return fns, nil
}

// memactionloader is an in-memory ActionLoader.  This is used within
// the FSLoader to initialize and add actions from functions when loaded.
type memactionloader struct {
	// actions stores all parsed actions, mapped by DSN to a slice representing each
	// action version.
	Actions map[string][]inngest.ActionVersion
	lock    *sync.RWMutex
}

func NewInMemoryActionLoader() *memactionloader {
	return &memactionloader{
		Actions: make(map[string][]inngest.ActionVersion),
		lock:    &sync.RWMutex{},
	}
}

// add adds an action to the in-memory action loader.
func (l *memactionloader) Add(action inngest.ActionVersion) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.Actions[action.DSN]; !ok {
		l.Actions[action.DSN] = []inngest.ActionVersion{action}
		return
	}
	l.Actions[action.DSN] = append(l.Actions[action.DSN], action)
	l.sortActions()
}

// sortActions sorts the actions for easy querying with version constraints.
func (l *memactionloader) sortActions() {
	for dsn, actions := range l.Actions {
		copied := actions
		sort.SliceStable(copied, func(i, j int) bool {
			a, b := copied[i], copied[j]
			return a.Version.Major >= b.Version.Major && a.Version.Minor > b.Version.Minor
		})
		l.Actions[dsn] = copied
	}
}

// Action returns an action given its DSN and optional version constraint.
// This fulfils the Action function within the ActionLoader interface.
func (l memactionloader) Action(ctx context.Context, dsn string, version *inngest.VersionConstraint) (*inngest.ActionVersion, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	actions, ok := l.Actions[dsn]
	if !ok {
		return nil, fmt.Errorf("action not found: %s", dsn)
	}

	if version == nil || version.Major == nil {
		// Always use the latest version and discard minor versions.
		return &actions[0], nil
	}

	for _, a := range actions {
		if a.Version.Major != *version.Major {
			continue
		}
		if version.Minor == nil {
			// Return the latest minor from this major version, which is first
			// as the slice is sorted.
			return &a, nil
		}
		if a.Version.Minor == *version.Minor {
			return &a, nil
		}
	}

	return nil, fmt.Errorf("action not found: %s", dsn)
}

type MemoryAPIFunctionWriter struct {
	*MemoryExecutionLoader
}

func NewInMemoryAPIFunctionWriter() *MemoryAPIFunctionWriter {
	loader := &MemoryAPIFunctionWriter{}
	loader.MemoryExecutionLoader = &MemoryExecutionLoader{}
	return loader
}

func (m *MemoryAPIFunctionWriter) CreateFunctionVersion(ctx context.Context, f function.Function, live bool, env string) (function.FunctionVersion, error) {
	now := time.Now()
	fv := function.FunctionVersion{
		FunctionID: f.ID,
		Version:    uint(1),
		Function:   f,
		ValidFrom:  &now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return fv, nil
}

type MemoryAPIReadWriter struct {
	*MemoryAPIFunctionWriter
	*MemoryAPIActionLoader
}

func NewInMemoryAPIReadWriter() *MemoryAPIReadWriter {
	return &MemoryAPIReadWriter{
		MemoryAPIFunctionWriter: NewInMemoryAPIFunctionWriter(),
		MemoryAPIActionLoader:   NewInMemoryAPIActionLoader(),
	}
}

type MemoryAPIActionLoader struct {
	*memactionloader
}

func NewInMemoryAPIActionLoader() *MemoryAPIActionLoader {
	return &MemoryAPIActionLoader{
		memactionloader: NewInMemoryActionLoader(),
	}
}

func (m *MemoryAPIActionLoader) ActionVersion(ctx context.Context, dsn string, vc *inngest.VersionConstraint) (client.ActionVersion, error) {
	av, err := m.Action(ctx, dsn, vc)
	if err != nil {
		return client.ActionVersion{}, err
	}
	clientActionVersion := client.ActionVersion{
		ActionVersion: *av,
		Name:          av.Name,
		DSN:           av.DSN,
		Config:        "",
	}
	return clientActionVersion, nil
}
func (m *MemoryAPIActionLoader) CreateActionVersion(ctx context.Context, av inngest.ActionVersion) (client.ActionVersion, error) {
	config, err := cuedefs.FormatAction(av)
	if err != nil {
		return client.ActionVersion{}, err
	}
	// Stub out with existing method
	m.Add(av)
	newActionVersion := client.ActionVersion{
		ActionVersion: av,
		Name:          av.Name,
		DSN:           av.DSN,
		Config:        config,
	}
	return newActionVersion, nil
}
func (m *MemoryAPIActionLoader) UpdateActionVersion(ctx context.Context, dsn string, version inngest.VersionInfo, enabled bool) (client.ActionVersion, error) {
	// NOTE - use constraint so we can re-use m.Action for now
	vc := &inngest.VersionConstraint{
		Major: &version.Major,
		Minor: &version.Minor,
	}
	existing, err := m.Action(ctx, dsn, vc)
	if err != nil {
		return client.ActionVersion{}, err
	}

	updatedActionVersion := client.ActionVersion{
		ActionVersion: *existing,
		Name:          existing.Name,
		DSN:           existing.DSN,
		Config:        "",
	}
	if enabled {
		now := time.Now()
		updatedActionVersion.ValidFrom = &now
	}
	// TODO - Add imageSha256 to be sent to function and set

	return updatedActionVersion, nil
}
