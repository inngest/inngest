// Package kvredis stores KV entries as redis hashes.  each key is one hash
// with a version field and a data field, so a compare and set reads and
// writes one key inside one script.
package kvredis

import (
	"context"
	"strconv"
	"strings"
	"unsafe"

	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/redis/rueidis"
)

const (
	fieldVersion = "v"
	fieldData    = "d"
)

// setScript writes the data and bumps the version.  ARGV[1] is the data,
// ARGV[2] is the expected version or empty for none, and ARGV[3] is the ttl
// in milliseconds or 0 for none.  it returns the new version, or -1 when
// the expected version does not match.
var setScript = rueidis.NewLuaScript(`
local cur = tonumber(redis.call('HGET', KEYS[1], 'v') or '0')
if ARGV[2] ~= '' and tonumber(ARGV[2]) ~= cur then
	return -1
end
local nv = cur + 1
redis.call('HSET', KEYS[1], 'v', nv, 'd', ARGV[1])
if tonumber(ARGV[3]) > 0 then
	redis.call('PEXPIRE', KEYS[1], ARGV[3])
else
	redis.call('PERSIST', KEYS[1])
end
return nv
`)

// deleteScript removes the key.  ARGV[1] is the expected version or empty
// for none.  it returns 0, or -1 when the expected version does not match.
var deleteScript = rueidis.NewLuaScript(`
local cur = tonumber(redis.call('HGET', KEYS[1], 'v') or '0')
if ARGV[1] ~= '' and tonumber(ARGV[1]) ~= cur then
	return -1
end
redis.call('DEL', KEYS[1])
return 0
`)

// Store is a kv.Store backed by redis
type Store struct {
	rc     rueidis.Client
	prefix string
}

// New returns a Store with keys prefixed by the given string.
func New(rc rueidis.Client, prefix string) *Store {
	return &Store{
		rc:     rc,
		prefix: prefix,
	}
}

// Get reads one value.  the returned bytes alias the redis reply, so the
// caller must treat them as read only.  a missing key returns
// kv.ErrNotFound.
func (s *Store) Get(ctx context.Context, k kv.Key) (kv.Entry, error) {
	if err := k.Validate(); err != nil {
		return kv.Entry{}, err
	}
	e := kv.Entry{Key: k}
	cmd := s.rc.B().Hmget().Key(s.redisKey(k)).Field(fieldVersion, fieldData).Build()
	arr, err := s.rc.Do(ctx, cmd).ToArray()
	if err != nil {
		return e, err
	}
	if len(arr) != 2 || arr[0].IsNil() {
		return e, kv.ErrNotFound
	}
	e.Version, err = arr[0].AsInt64()
	if err != nil {
		return e, err
	}
	e.Value, err = arr[1].AsBytes()
	if err != nil {
		return e, err
	}
	return e, nil
}

// Set writes a value and returns the entry with its new version.
func (s *Store) Set(ctx context.Context, e kv.Entry, opts ...kv.Option) (kv.Entry, error) {
	if err := e.Key.Validate(); err != nil {
		return kv.Entry{}, err
	}
	o := kv.ApplyOptions(opts...)
	// rueidis copies the arguments into its write buffer before Exec
	// returns, so the string view over e.Value does not outlive the call.
	val := unsafe.String(unsafe.SliceData(e.Value), len(e.Value))
	args := []string{
		val,
		expectedArg(o),
		strconv.FormatInt(o.TTL.Milliseconds(), 10),
	}
	v, err := setScript.Exec(ctx, s.rc, []string{s.redisKey(e.Key)}, args).AsInt64()
	if err != nil {
		return kv.Entry{}, err
	}
	if v < 0 {
		return kv.Entry{}, kv.ErrConflict
	}
	e.Version = v
	return e, nil
}

// Delete removes one key.
func (s *Store) Delete(ctx context.Context, k kv.Key, opts ...kv.Option) error {
	if err := k.Validate(); err != nil {
		return err
	}
	o := kv.ApplyOptions(opts...)
	args := []string{expectedArg(o)}
	v, err := deleteScript.Exec(ctx, s.rc, []string{s.redisKey(k)}, args).AsInt64()
	if err != nil {
		return err
	}
	if v < 0 {
		return kv.ErrConflict
	}
	return nil
}

// expectedArg returns the expected version as a script argument.  empty
// means no check.
func expectedArg(o kv.Options) string {
	if o.Expected == nil {
		return ""
	}
	return strconv.FormatInt(*o.Expected, 10)
}

// redisKey returns prefix + key text.  the key text is built in a stack
// buffer, so the only allocation is the returned string.
func (s *Store) redisKey(k kv.Key) string {
	var buf [kv.MaxTextLen]byte
	text, _ := k.AppendText(buf[:0])

	var sb strings.Builder
	sb.Grow(len(s.prefix) + len(text))
	sb.WriteString(s.prefix)
	sb.Write(text)
	return sb.String()
}
