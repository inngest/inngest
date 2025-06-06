package pauses

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// NewRedisBlockLeaser creates a new lease manager across Redis, using the given
// prefix for any keys stored.
func NewRedisBlockLeaser(rc rueidis.Client, prefix string, duration time.Duration) BlockLeaser {
	return &redisBlockLeaser{rc: rc, prefix: prefix, duration: duration}
}

type redisBlockLeaser struct {
	rc       rueidis.Client
	prefix   string
	duration time.Duration
}

func (r redisBlockLeaser) indexKey(index Index) string {
	return fmt.Sprintf("%s:%s:%s", r.prefix, index.WorkspaceID, index.EventName)
}

// Lease leases a given index, ensuring that only one worker can
// flush an index at a time.
func (r redisBlockLeaser) Lease(ctx context.Context, index Index) (leaseID ulid.ULID, err error) {
	expires := time.Now().Add(r.duration)
	leaseID, err = ulid.New(uint64(expires.UnixMilli()), rand.Reader)
	if err != nil {
		return leaseID, err
	}

	status, err := lease.Exec(
		ctx,
		r.rc,
		[]string{r.indexKey(index)},
		[]string{
			strconv.Itoa(int(time.Now().UnixMilli())),
			leaseID.String(),
			"",
			strconv.Itoa(int(r.duration.Seconds())),
		},
	).ToInt64()

	switch status {
	case -1:
		return leaseID, err
	default:
		return leaseID, fmt.Errorf("unable to lease: already leased")
	}
}

// Renew renews a lease while we are flushing an index.
func (r redisBlockLeaser) Renew(ctx context.Context, index Index, existingLeaseID ulid.ULID) (newLeaseID ulid.ULID, err error) {
	expires := time.Now().Add(r.duration)
	newLeaseID, err = ulid.New(uint64(expires.UnixMilli()), rand.Reader)
	if err != nil {
		return newLeaseID, err
	}

	status, err := lease.Exec(
		ctx,
		r.rc,
		[]string{r.indexKey(index)},
		[]string{
			strconv.Itoa(int(time.Now().UnixMilli())),
			newLeaseID.String(),
			existingLeaseID.String(),
			strconv.Itoa(int(r.duration.Seconds())),
		},
	).AsInt64()
	if err != nil {
		return newLeaseID, fmt.Errorf("error renewing block lease: %w", err)
	}
	switch status {
	case -1:
		return newLeaseID, nil
	default:
		return newLeaseID, fmt.Errorf("unable to renew lease")
	}
}

// Revoke drops a lease, allowing any other worker to flush an index.
func (r redisBlockLeaser) Revoke(ctx context.Context, index Index, leaseID ulid.ULID) (err error) {
	return r.rc.Do(ctx, r.rc.B().Del().Key(r.indexKey(index)).Build()).Error()
}

var (
	lease = rueidis.NewLuaScript(`
--[[

Output:
  -1: Successfully leased key
  -2: Lease mismatch / already leased

]]

local keyLease        = KEYS[1]

local currentTime     = tonumber(ARGV[1]) -- Current time, in ms, to check if existing lease expired.
local newLeaseID      = ARGV[2] -- New lease ID
local existingLeaseID = ARGV[3] -- existing lease ID
local expirySeconds   = tonumber(ARGV[4])

-- This table is used when decoding ulid timestamps.
local ulidMap = { ["0"] = 0, ["1"] = 1, ["2"] = 2, ["3"] = 3, ["4"] = 4, ["5"] = 5, ["6"] = 6, ["7"] = 7, ["8"] = 8, ["9"] = 9, ["A"] = 10, ["B"] = 11, ["C"] = 12, ["D"] = 13, ["E"] = 14, ["F"] = 15, ["G"] = 16, ["H"] = 17, ["J"] = 18, ["K"] = 19, ["M"] = 20, ["N"] = 21, ["P"] = 22, ["Q"] = 23, ["R"] = 24, ["S"] = 25, ["T"] = 26, ["V"] = 27, ["W"] = 28, ["X"] = 29, ["Y"] = 30, ["Z"] = 31 }

--- decode_ulid_time decodes a ULID into a ms epoch
local function decode_ulid_time(s)
	if #s < 10 then
		return 0
	end

	-- Take first 10 characters of the ULID, which is the time portion.
	s = string.sub(s, 1, 10)
	local rev = tostring(s.reverse(s))
	local time = 0
	for i = 1, #rev do
		time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i-1))
	end
	return time
end

local fetched = redis.call("GET", keyLease)

if existingLeaseID ~= "" and fetched == false then
	return -2
end

if fetched == false or decode_ulid_time(fetched) < currentTime or fetched == existingLeaseID then
	-- Either nil, an expired key, or a release, so we're okay.
	redis.call("SET", keyLease, newLeaseID, "EX", expirySeconds)
	return -1
end

return -2
	`)
)
