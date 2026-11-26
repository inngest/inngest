package kvredis

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/inngest/inngest/pkg/execution/state/kv"
	"github.com/redis/rueidis"
)

// a list is one redis hash with one field per item.  the field name is the
// item id.  the value is the arrival number of the item, then ':', then the
// item bytes.  two more fields contain the counters.  an item id cannot
// contain '#', so it never equals a counter field.
const (
	// listSuffix ends the redis key of every list.  a kv key part cannot
	// contain '/', so a list never shares a redis key with a value.
	listSuffix = "/list"

	// fieldSeq is the arrival number of the last item that was pushed.
	fieldSeq = "#seq"
	// fieldSize is the total byte length of the item values in the hash.
	fieldSize = "#size"
)

var _ kv.ListStore = (*Store)(nil)

// pushScript adds one item.  ARGV[1] is the item id, ARGV[2] is the item
// bytes, ARGV[3] is the most items or 0 for no limit, ARGV[4] is the most
// value bytes or 0 for no limit, and ARGV[5] is the ttl in milliseconds or
// 0 for none.  it returns 1 when the item was added, 0 when the id is
// already in the list, and -1 when the list is full.
var pushScript = rueidis.NewLuaScript(`
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 1 then
	return 0
end
local n = redis.call('HLEN', KEYS[1])
	- redis.call('HEXISTS', KEYS[1], '` + fieldSeq + `')
	- redis.call('HEXISTS', KEYS[1], '` + fieldSize + `')
local size = tonumber(redis.call('HGET', KEYS[1], '` + fieldSize + `') or '0')
local maxItems = tonumber(ARGV[3])
local maxBytes = tonumber(ARGV[4])
if maxItems > 0 and n >= maxItems then
	return -1
end
if maxBytes > 0 and size + #ARGV[2] > maxBytes then
	return -1
end
local seq = redis.call('HINCRBY', KEYS[1], '` + fieldSeq + `', 1)
redis.call('HSET', KEYS[1], ARGV[1], string.format('%d', seq) .. ':' .. ARGV[2])
redis.call('HINCRBY', KEYS[1], '` + fieldSize + `', #ARGV[2])
if tonumber(ARGV[5]) > 0 then
	redis.call('PEXPIRE', KEYS[1], ARGV[5])
else
	redis.call('PERSIST', KEYS[1])
end
return 1
`)

// removeScript removes items.  ARGV[1] is the ttl in milliseconds or 0 for
// none, and every later argument is one item id.  it returns the number of
// items that were removed.  the counter fields stay in the hash when the
// last item is removed.
var removeScript = rueidis.NewLuaScript(`
local removed = 0
local freed = 0
for i = 2, #ARGV do
	local v = redis.call('HGET', KEYS[1], ARGV[i])
	if v then
		redis.call('HDEL', KEYS[1], ARGV[i])
		freed = freed + #v - string.find(v, ':', 1, true)
		removed = removed + 1
	end
end
if removed > 0 then
	redis.call('HINCRBY', KEYS[1], '` + fieldSize + `', -freed)
	if tonumber(ARGV[1]) > 0 then
		redis.call('PEXPIRE', KEYS[1], ARGV[1])
	else
		redis.call('PERSIST', KEYS[1])
	end
end
return removed
`)

// Push adds one item to the end of a list.
func (s *Store) Push(ctx context.Context, k kv.Key, it kv.Item, opts ...kv.Option) (bool, error) {
	if err := k.Validate(); err != nil {
		return false, err
	}
	if err := kv.ValidateItemID(it.ID); err != nil {
		return false, err
	}
	o := kv.ApplyOptions(opts...)
	args := []string{
		it.ID,
		rueidis.BinaryString(it.Value),
		strconv.Itoa(o.MaxItems),
		strconv.FormatInt(o.MaxBytes, 10),
		strconv.FormatInt(o.TTL.Milliseconds(), 10),
	}
	v, err := pushScript.Exec(ctx, s.rc, []string{s.listKey(k)}, args).AsInt64()
	if err != nil {
		return false, err
	}
	if v < 0 {
		return false, kv.ErrFull
	}
	return v > 0, nil
}

// Peek returns the first limit items of a list.  it reads the whole hash
// and sorts by arrival number, because a hash has no order.
func (s *Store) Peek(ctx context.Context, k kv.Key, limit int) ([]kv.Item, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	fields, err := s.rc.Do(ctx, s.rc.B().Hgetall().Key(s.listKey(k)).Build()).AsStrMap()
	if err != nil {
		return nil, err
	}

	type entry struct {
		seq  int64
		item kv.Item
	}
	entries := make([]entry, 0, len(fields))
	for field, val := range fields {
		if field == fieldSeq || field == fieldSize {
			continue
		}
		num, data, ok := strings.Cut(val, ":")
		if !ok {
			return nil, fmt.Errorf("kv: list item %q has no arrival number", field)
		}
		seq, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("kv: list item %q: %w", field, err)
		}
		entries = append(entries, entry{
			seq: seq,
			item: kv.Item{
				ID:    field,
				Value: []byte(data),
			},
		})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Compare(a.seq, b.seq)
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	items := make([]kv.Item, len(entries))
	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

// Remove removes items from a list by id.
func (s *Store) Remove(ctx context.Context, k kv.Key, ids []string, opts ...kv.Option) error {
	if err := k.Validate(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := kv.ValidateItemID(id); err != nil {
			return err
		}
	}
	if len(ids) == 0 {
		return nil
	}
	o := kv.ApplyOptions(opts...)
	args := make([]string, 0, len(ids)+1)
	args = append(args, strconv.FormatInt(o.TTL.Milliseconds(), 10))
	args = append(args, ids...)
	return removeScript.Exec(ctx, s.rc, []string{s.listKey(k)}, args).Error()
}

// Clear deletes one list.
func (s *Store) Clear(ctx context.Context, k kv.Key) error {
	if err := k.Validate(); err != nil {
		return err
	}
	return s.rc.Do(ctx, s.rc.B().Del().Key(s.listKey(k)).Build()).Error()
}

// listKey returns the redis key of the list for k.
func (s *Store) listKey(k kv.Key) string {
	return s.redisKey(k) + listSuffix
}
