package inmemory

/*

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

// TODO - Remove FS Loader
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
	return f.MemoryExecutionLoader.SetFunctions(ctx, nil)
}

// NewFSLoader returns an ExecutionLoader which reads functions from the given
// path, recursively.
func NewFSLoader(ctx context.Context, path string) (coredata.ExecutionLoader, error) {
	// XXX: This should probably be a singleton;  this is primarily used
	// for the dev server.  in this case, a single process hosts the
	// runner and the executor together - and we don't want to process
	// the directory multiple times within the same pid.
	loader, err := NewEmptyFSLoader(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := loader.ReadDir(ctx); err != nil {
		return loader, err
	}
	return loader, nil
}

func NewEmptyFSLoader(ctx context.Context, path string) (*FSLoader, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &FSLoader{root: abspath, MemoryExecutionLoader: &MemoryExecutionLoader{}}, nil
}

// MemoryExecutionLoader is a function and action loader which returns data from
// in-memory state.
type MemoryExecutionLoader struct {
	// embed the in-memory action loader for querying action versions found within
	// functions.
	*memactionloader

	// functions stores all functions which were found within the given filesystem.
	functions []inngest.Function

	l sync.RWMutex
}

func (m *MemoryExecutionLoader) AddFunction(ctx context.Context, fn *inngest.Function) error {
	m.l.Lock()
	defer m.l.Unlock()

	// Is this function and its actions already present?  If so, remove them.
	for n, f := range m.functions {
		if f.ID == fn.ID {
			m.functions = append(m.functions[:n], m.functions[n+1:]...)
			break
		}
	}
	m.functions = append(m.functions, *fn)

	// recreate the in-memory action loader.
	m.memactionloader = &memactionloader{
		Actions: make(map[string][]inngest.ActionVersion),
		lock:    &sync.RWMutex{},
	}

	logger.From(ctx).
		Debug().
		Int("len", len(m.functions)).
		Msg("added functions")

	return nil
}

func (m *MemoryExecutionLoader) LoadFunction(ctx context.Context, id state.Identifier) (*inngest.Function, error) {
	for _, fn := range m.functions {
		if fn.ID == id.WorkflowID {
			return &fn, nil
		}
	}
	return nil, fmt.Errorf("Function ID '%s' not found for run ID '%s'", id.WorkflowID, id.RunID)
}

func (m *MemoryExecutionLoader) SetFunctions(ctx context.Context, f []*inngest.Function) error {
	m.l.Lock()
	defer m.l.Unlock()

	m.functions = []inngest.Function{}

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
		m.functions = append(m.functions, *fn)
	}

	// recreate the in-memory action loader.
	m.memactionloader = &memactionloader{
		Actions: make(map[string][]inngest.ActionVersion),
		lock:    &sync.RWMutex{},
	}

	logger.From(ctx).
		Debug().
		Int("len", len(m.functions)).
		Msg("added functions")

	return nil
}

func (m *MemoryExecutionLoader) Functions(ctx context.Context) ([]inngest.Function, error) {
	return m.functions[:], nil
}

func (m *MemoryExecutionLoader) FunctionsScheduled(ctx context.Context) ([]inngest.Function, error) {
	m.l.RLock()
	defer m.l.RUnlock()

	fns := []inngest.Function{}
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

func (m *MemoryExecutionLoader) FunctionsByTrigger(ctx context.Context, eventName string) ([]inngest.Function, error) {
	m.l.RLock()
	defer m.l.RUnlock()

	fns := []inngest.Function{}
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
*/
