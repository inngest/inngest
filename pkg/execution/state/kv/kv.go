package kv

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/domain"
)

const (
	// KeySep joins the tenant, namespace, and key in the text form.
	KeySep = '/'

	// MaxPartLen caps the namespace and the key.  the cap bounds the text form
	// so stores can size key buffers up front.
	MaxPartLen = 32

	// MaxTextLen is the longest text form a valid Key can have.  stores use it
	// to size a stack buffer when building a backend key.
	MaxTextLen = domain.TenantTextLen + 1 + MaxPartLen + 1 + MaxPartLen
)

var (
	// ErrNotFound is returned by Get when the key has no value.
	ErrNotFound = errors.New("kv: key not found")
	// ErrConflict is returned by Set and Delete when WithExpected names a
	// version other than the one in the store.  nothing is written.
	ErrConflict = errors.New("kv: version conflict")

	ErrEmptyNamespace   = errors.New("kv: namespace must not be empty")
	ErrEmptyKey         = errors.New("kv: key must not be empty")
	ErrNamespaceTooLong = errors.New("kv: namespace must be at most 32 bytes")
	ErrKeyTooLong       = errors.New("kv: key must be at most 32 bytes")
	ErrInvalidNamespace = errors.New("kv: namespace must contain only letters, digits, '-', and '_'")
	ErrInvalidKey       = errors.New("kv: key must contain only letters, digits, '-', and '_'")
)

// Store represents a KV store for state.  every write bumps the entry's
// version, and WithExpected turns a write into a compare and set on it.
type Store interface {
	Get(context.Context, Key) (Entry, error)
	// Set writes a value and returns the entry with its new version.  without
	// WithTTL the key does not expire, and an earlier TTL is removed.
	Set(context.Context, Entry, ...Option) (Entry, error)
	// Delete removes one key.  without WithExpected, a missing key is not an
	// error.
	Delete(context.Context, Key, ...Option) error
}

// Entry represents one value in a KV store.
type Entry struct {
	Key   Key
	Value json.RawMessage
	// Version is set by the store.  it starts at 1 on the first Set and
	// goes up by one on every later Set.  a missing key has version 0.
	Version int64
}

// Options controls settings for Set or Delete.  stores read it through
// ApplyOptions.
type Options struct {
	// Expected is the version the key must have for the write to happen.
	// zero means the key must not exist.  nil means no check.
	Expected *int64
	// TTL is how long the key lives after the write.  zero means no expiry.
	// Delete ignores it.
	TTL time.Duration
}

// Option changes one setting on a Set or Delete.
type Option func(*Options)

// WithExpected makes the write a compare and set.  the write happens only
// when the key is at version v, and v of zero means the key must not
// exist.  on a mismatch the store returns ErrConflict.
func WithExpected(v int64) Option {
	return func(o *Options) {
		o.Expected = &v
	}
}

// WithTTL makes the key expire d after the write.
func WithTTL(d time.Duration) Option {
	return func(o *Options) {
		o.TTL = d
	}
}

// ApplyOptions folds opts into an Options value.
func ApplyOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Key represents one key in a KV store.
type Key struct {
	// Tenant represents the owner of the key.
	Tenant domain.Tenant
	// Namespace is the heirarchy for the KV space.  There can be
	// many namespaces within a single tenant.
	Namespace string
	// Key is the actual key in the tenant's namespace.
	Key string
}

// Validate checks that the namespace and key are non-empty, at most
// MaxPartLen bytes, and contain only ASCII letters, digits, '-', and '_'.
//
// this keeps the text form unambiguous, since neither part can contain
// the separator.
func (k Key) Validate() error {
	if k.Namespace == "" {
		return ErrEmptyNamespace
	}
	if len(k.Namespace) > MaxPartLen {
		return ErrNamespaceTooLong
	}
	if !isIdent(k.Namespace) {
		return ErrInvalidNamespace
	}
	if k.Key == "" {
		return ErrEmptyKey
	}
	if len(k.Key) > MaxPartLen {
		return ErrKeyTooLong
	}
	if !isIdent(k.Key) {
		return ErrInvalidKey
	}
	return nil
}

// isIdent reports whether s contains only ASCII letters, digits, '-', and '_'.
func isIdent(s string) bool {
	for i := 0; i < len(s); i++ {
		// can optimize via swar
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

// textLen is the exact byte length of the text form, so each writer
// allocates once.
func (k Key) textLen() int {
	return domain.TenantTextLen + 1 + len(k.Namespace) + 1 + len(k.Key)
}

// String returns "tenant/namespace/key".  the tenant is written through a
// stack buffer, so the only allocation is the returned string.
func (k Key) String() string {
	var tenant [domain.TenantTextLen]byte
	tb, _ := k.Tenant.AppendText(tenant[:0])

	var sb strings.Builder
	sb.Grow(k.textLen())
	sb.Write(tb)
	sb.WriteByte(KeySep)
	sb.WriteString(k.Namespace)
	sb.WriteByte(KeySep)
	sb.WriteString(k.Key)
	return sb.String()
}

// AppendText implements encoding.TextAppender.  it does not allocate when
// b has room for the text form.
func (k Key) AppendText(b []byte) ([]byte, error) {
	b, _ = k.Tenant.AppendText(b)
	b = append(b, KeySep)
	b = append(b, k.Namespace...)
	b = append(b, KeySep)
	b = append(b, k.Key...)
	return b, nil
}

// MarshalText implements encoding.TextMarshaler.
func (k Key) MarshalText() ([]byte, error) {
	return k.AppendText(make([]byte, 0, k.textLen()))
}
