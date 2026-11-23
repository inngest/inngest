package libduckdb

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// DB wraps a duckdb_database handle opened via libduckdb.
type DB struct {
	lib    *Lib
	mu     sync.RWMutex
	handle uintptr
	closed bool
}

// Open opens a DuckDB database at path (use "" or ":memory:" for in-memory)
// with optional configuration key/value pairs.
func Open(lib *Lib, path string, cfg map[string]string) (*DB, error) {
	if lib == nil {
		return nil, errors.New("libduckdb: nil library")
	}
	if path == "" {
		path = ":memory:"
	}

	var cfgHandle uintptr
	if len(cfg) > 0 {
		if State(lib.createConfig(&cfgHandle)) != StateSuccess {
			return nil, errors.New("libduckdb: duckdb_create_config failed")
		}
		defer lib.destroyConfig(&cfgHandle)

		for k, v := range cfg {
			if State(lib.setConfig(cfgHandle, k, v)) != StateSuccess {
				return nil, fmt.Errorf("libduckdb: duckdb_set_config(%q) failed", k)
			}
		}
	}

	var dbHandle uintptr
	var errPtr *byte
	state := State(lib.openExt(path, &dbHandle, cfgHandle, &errPtr))
	if errPtr != nil {
		errMsg := cString(errPtr)
		lib.free(unsafe.Pointer(errPtr))
		if state != StateSuccess {
			return nil, fmt.Errorf("libduckdb: open %q: %s", path, errMsg)
		}
	}
	if state != StateSuccess {
		return nil, fmt.Errorf("libduckdb: open %q failed", path)
	}

	return &DB{
		lib:    lib,
		handle: dbHandle,
	}, nil
}

// Open opens a DuckDB database using this loaded library.
func (l *Lib) Open(path string, cfg map[string]string) (*DB, error) {
	return Open(l, path, cfg)
}

// Close closes the database handle. Safe to call multiple times.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return nil
	}
	db.closed = true
	if db.handle != 0 {
		db.lib.closeDB(&db.handle)
		db.handle = 0
	}
	return nil
}

// Connect creates a new independent connection to the database. Multiple
// connections may run queries concurrently.
func (db *DB) Connect() (*Conn, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed || db.handle == 0 {
		return nil, errors.New("libduckdb: database is closed")
	}

	var connHandle uintptr
	if State(db.lib.connect(db.handle, &connHandle)) != StateSuccess {
		return nil, errors.New("libduckdb: duckdb_connect failed")
	}

	return &Conn{
		lib:    db.lib,
		handle: connHandle,
	}, nil
}

// Conn wraps a duckdb_connection handle.
type Conn struct {
	lib *Lib

	// handleMu protects handle lifecycle (RLock for Interrupt, Lock for Close).
	handleMu sync.RWMutex
	handle   uintptr
	closed   bool

	// execMu serializes Query/Exec on a single connection while allowing
	// Interrupt to run concurrently from another goroutine.
	execMu sync.Mutex
}

// Close disconnects the connection. Safe to call multiple times.
func (c *Conn) Close() error {
	c.execMu.Lock()
	defer c.execMu.Unlock()

	c.handleMu.Lock()
	defer c.handleMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.handle != 0 {
		c.lib.disconnect(&c.handle)
		c.handle = 0
	}
	return nil
}

// Interrupt interrupts any currently executing query on this connection.
// Safe to call concurrently from another goroutine while Query or Exec is
// blocked inside DuckDB.
func (c *Conn) Interrupt() {
	c.handleMu.RLock()
	defer c.handleMu.RUnlock()
	if c.closed || c.handle == 0 {
		return
	}
	c.lib.interrupt(c.handle)
}

// Query executes sql and returns a streaming Result. The caller must call
// (*Result).Close() to release C resources.
func (c *Conn) Query(sql string) (*Result, error) {
	c.execMu.Lock()
	defer c.execMu.Unlock()

	c.handleMu.RLock()
	handle := c.handle
	closed := c.closed
	c.handleMu.RUnlock()
	if closed || handle == 0 {
		return nil, errors.New("libduckdb: connection is closed")
	}

	var res cResult
	state := State(c.lib.query(handle, sql, &res))
	if state != StateSuccess {
		errMsg := cString(c.lib.resultError(&res))
		c.lib.destroyResult(&res)
		if errMsg == "" {
			errMsg = "query failed"
		}
		return nil, fmt.Errorf("libduckdb: %s", errMsg)
	}

	colCount := c.lib.columnCount(&res)
	cols := make([]ColumnInfo, colCount)
	for i := uint64(0); i < colCount; i++ {
		cols[i] = ColumnInfo{
			Name: cString(c.lib.columnName(&res, i)),
			Type: Type(c.lib.columnType(&res, i)),
		}
	}

	return &Result{
		lib:  c.lib,
		res:  res,
		cols: cols,
	}, nil
}

// Exec executes sql and discards any result rows, freeing C resources before
// returning.
func (c *Conn) Exec(sql string) error {
	res, err := c.Query(sql)
	if err != nil {
		return err
	}
	res.Close()
	return nil
}
