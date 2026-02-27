package expr

import (
	"os"
	"sync/atomic"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/google/uuid"
)

// KV represents temporary evaluable storage for mapping evalubles by ID.  This allows
// aggregate evaluators to store Evaluables on disk mapped by ID using Pebble.
type KV[T Evaluable] interface {
	Get(evalID uuid.UUID) (T, error)
	Set(eval T) error
	Remove(evalID uuid.UUID) error
	Len() int32
}

// KVOpts
type KVOpts[T Evaluable] struct {
	Unmarshal func(bytes []byte) (T, error)
	Marshal   func(T) ([]byte, error)

	// FS is the pebble FS to use.  If nil, this uses the OS filesystem.
	FS vfs.FS
	// Dir is the direcorty to store the KV data within.
	Dir string
}

// New returns a new temporary EvalKV KV store written to disk.
func NewKV[T Evaluable](o KVOpts[T]) (KV[T], error) {
	if o.Dir == "" {
		o.Dir = os.TempDir()
	}
	if o.FS == nil {
		o.FS = vfs.Default
	}

	db, err := pebble.Open(o.Dir, &pebble.Options{
		FS: o.FS,
		// cockroachdb defaults that should slightly help with faster writes
		// https://github.com/cockroachdb/cockroach/blob/5a1f5da5bb3b2d962d8737848a4fca69f915dacb/pkg/storage/pebble.go#L668-L673
		L0CompactionThreshold:       2,
		L0StopWritesThreshold:       1000,
		MemTableSize:                64 << 20, // 64 MB
		MemTableStopWritesThreshold: 4,
	})
	if err != nil {
		return nil, err
	}

	return &EvalKV[T]{db: db, opts: o}, nil
}

// EvalKV is a small Pebble wrapper which stores evals in Pebble.
type EvalKV[T Evaluable] struct {
	opts KVOpts[T]
	db   *pebble.DB
	len  int32
}

func (p *EvalKV[T]) Len() int32 {
	return p.len
}

// Get returns an Evaluable.
func (p *EvalKV[T]) Get(evalID uuid.UUID) (T, error) {
	var response T

	byt, closer, err := p.db.Get(evalID[:])
	if err != nil {
		return response, err
	}
	defer func() {
		_ = closer.Close()
	}()
	return p.opts.Unmarshal(byt)
}

// Set stores an Evalauble.
func (p *EvalKV[T]) Set(eval T) error {
	byt, err := p.opts.Marshal(eval)
	if err != nil {
		return err
	}
	id := eval.GetID()
	err = p.db.Set(id[:], byt, &pebble.WriteOptions{
		Sync: false,
	})
	if err != nil {
		return err
	}
	atomic.AddInt32(&p.len, 1)
	return nil
}

// Remove removes an Evaluable.
func (p *EvalKV[T]) Remove(evalID uuid.UUID) error {
	err := p.db.Delete(evalID[:], &pebble.WriteOptions{
		Sync: false,
	})
	if err != nil {
		return err
	}
	atomic.AddInt32(&p.len, -1)
	return nil
}
