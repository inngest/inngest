package redis_state

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestLuaGCRA(t *testing.T) {
	runScript := func(t *testing.T, rc rueidis.Client, key string, now time.Time, period time.Duration, limit, burst, capacity int) (int, time.Time) {
		nowMS := now.UnixMilli()
		args, err := StrSlice([]any{
			key,
			nowMS,
			limit,
			burst,
			period.Milliseconds(),
			capacity,
		})
		require.NoError(t, err)

		res, err := scripts["test/gcra_capacity"].Exec(t.Context(), rc, []string{}, args).ToAny()
		require.NoError(t, err)

		capacityAndRetry, ok := res.([]any)
		require.True(t, ok)

		statusOrCapacity, ok := capacityAndRetry[0].(int64)
		require.True(t, ok)

		var retryAt time.Time
		retryAtMillis, ok := capacityAndRetry[1].(int64)
		require.True(t, ok)

		if retryAtMillis > nowMS {
			retryAt = time.UnixMilli(retryAtMillis)
		}

		switch statusOrCapacity {
		case -1:
			return 0, retryAt
		default:
			return int(statusOrCapacity), retryAt
		}
	}

	t.Run("should reduce throttle capacity", func(t *testing.T) {
		clock := clockwork.NewFakeClock()

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Hour
		limit := 100
		burst := 10

		// Read initial capacity
		capa, _ := runScript(t, rc, key, clock.Now(), period, limit, burst, 0)
		require.Equal(t, 110, capa)
		require.Len(t, r.Keys(), 0)

		// "Start" one run
		capa, _ = runScript(t, rc, key, clock.Now(), period, limit, burst, 1)
		require.Equal(t, 0, capa)
		require.Len(t, r.Keys(), 1)
		require.True(t, r.Exists(key))
		capa, _ = runScript(t, rc, key, clock.Now(), period, limit, burst, 0)
		require.Equal(t, 109, capa)

		clock.Advance(2 * time.Hour)
		r.FastForward(2 * time.Hour)

		// Should match initial capacity
		capa, _ = runScript(t, rc, key, clock.Now(), period, limit, burst, 0)
		require.Equal(t, 110, capa)
		require.Len(t, r.Keys(), 0)
	})

	t.Run("should prevent overflowing", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Hour
		limit := 5
		burst := 0

		// Read initial capacity
		capa, _ := runScript(t, rc, key, clock.Now(), period, limit, burst, 0)
		require.Equal(t, 5, capa)
		require.Len(t, r.Keys(), 0)

		// "Start" 5 runs
		capa, _ = runScript(t, rc, key, clock.Now(), period, limit, burst, 5)
		require.Equal(t, 0, capa)
		require.Len(t, r.Keys(), 1)
		require.True(t, r.Exists(key))

		now := clock.Now().Add(time.Second)
		capa, retryAt := runScript(t, rc, key, now, period, limit, burst, 0)
		require.Equal(t, 0, capa)
		require.False(t, retryAt.IsZero())
		// for a gcra period of 60min and 5 items, we expect to "refill" one item every 12 minutes,
		// thus the earliest next request should arrive in 12min
		expectedRetry := now.Add(12 * time.Minute)
		require.WithinDuration(t, expectedRetry, retryAt, 10*time.Second)

		capa, _ = runScript(t, rc, key, clock.Now(), period, limit, burst, 2)
		require.Equal(t, 0, capa)

		clock.Advance(2 * time.Hour)
		r.FastForward(2 * time.Hour)

		// Should match initial capacity
		capa, _ = runScript(t, rc, key, clock.Now(), period, limit, burst, 0)
		require.Equal(t, 5, capa)
		require.Len(t, r.Keys(), 0)
	})

	type action struct {
		delay time.Duration

		consumeCapacity int

		capacityBefore int
		capacityAfter  int

		retryAt time.Duration
	}

	type tableTest struct {
		name string

		actions []action
		period  time.Duration
		limit   int
		burst   int
	}

	tests := []tableTest{
		{
			name:   "basic limits without passing time",
			period: time.Hour,
			limit:  10,
			burst:  0,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  10,
					consumeCapacity: 0,
					capacityAfter:   10,
					retryAt:         time.Minute * (60 / 10),
				},
				{
					delay:           0,
					capacityBefore:  10,
					consumeCapacity: 5,
					capacityAfter:   5,
					retryAt:         time.Minute * (60 / 10),
				},
				{
					delay:           0,
					capacityBefore:  5,
					consumeCapacity: 5,
					capacityAfter:   0,
					retryAt:         time.Minute * (60 / 10),
				},
				{
					delay:           1 * time.Hour,
					capacityBefore:  10,
					consumeCapacity: 0,
					capacityAfter:   10,
					retryAt:         time.Minute * (60 / 10),
				},
			},
		},
		{
			name:   "basic limits with passing time",
			period: 10 * time.Hour,
			limit:  100,
			// refill 10 every hour
			burst: 0,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  100,
					consumeCapacity: 0,
					capacityAfter:   100,
					retryAt:         time.Minute * (600 / 100),
				},
				{
					delay:           0,
					capacityBefore:  100,
					consumeCapacity: 20,
					capacityAfter:   80,
					retryAt:         time.Minute * (600 / 100),
				},
				{
					delay:           1 * time.Hour,
					capacityBefore:  90, // assume 10 items got refilled since 1 hour passed
					consumeCapacity: 90,
					capacityAfter:   0,
					retryAt:         6 * time.Minute,
				},
				{
					delay:           1 * time.Hour,
					capacityBefore:  10, // assume 10 items got refilled again
					consumeCapacity: 0,
					capacityAfter:   10,
					retryAt:         6 * time.Minute,
				},
			},
		},
		{
			name:   "with burst",
			period: 1 * time.Hour,
			limit:  10,
			// 10 are refilled per hour, or 1 every 10 minutes
			burst: 2,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  12,
					consumeCapacity: 0,
					capacityAfter:   12,
					retryAt:         6 * time.Minute,
				},
				{
					delay:           0,
					capacityBefore:  12,
					consumeCapacity: 5,
					capacityAfter:   7,
					retryAt:         6 * time.Minute,
				},
				{
					delay:           10 * time.Minute,
					capacityBefore:  8, // assume 1 item got refilled
					consumeCapacity: 0,
					capacityAfter:   8,
					retryAt:         2 * time.Minute,
				},
				{
					delay:           5 * time.Minute,
					capacityBefore:  9, // assume 1 item got refilled
					consumeCapacity: 0,
					capacityAfter:   9,
					retryAt:         3 * time.Minute,
				},
				{
					delay:           0,
					capacityBefore:  9,
					consumeCapacity: 9,
					capacityAfter:   0,
					// we are 15mins in (10 + 5 above) without consuming, so the next
					// refill is expected in (6 + 6 + 6) - (10 + 5) = 3 mins
					retryAt: 3 * time.Minute,
				},
				{
					delay:           60 * time.Minute,
					capacityBefore:  10,
					consumeCapacity: 0,
					capacityAfter:   10,
					retryAt:         3 * time.Minute,
				},
			},
		},
		{
			name:   "with short limit",
			period: 5 * time.Second,
			limit:  1,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  1,
					consumeCapacity: 0,
					capacityAfter:   1,
					retryAt:         5 * time.Second,
				},
				{
					delay:           time.Second,
					capacityBefore:  1,
					consumeCapacity: 1,
					capacityAfter:   0,
					retryAt:         5 * time.Second,
				},
				{
					delay:           5 * time.Second,
					capacityBefore:  1,
					consumeCapacity: 0,
					capacityAfter:   1,
					retryAt:         5 * time.Second,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clock := clockwork.NewFakeClock()
			_, rc := initRedis(t)
			defer rc.Close()

			key := "test"

			current := clock.Now()
			for i, a := range test.actions {
				current = current.Add(a.delay)

				if a.capacityBefore > 0 {
					capa, _ := runScript(t, rc, key, current, test.period, test.limit, test.burst, 0)
					require.Equal(t, a.capacityBefore, capa, "capacity before in action %d failed", i)
				}

				if a.consumeCapacity > 0 {
					capa, _ := runScript(t, rc, key, current, test.period, test.limit, test.burst, a.consumeCapacity)
					require.Equal(t, 0, capa, "gcra update in action %d failed", i)
				}

				capa, retryAt := runScript(t, rc, key, current, test.period, test.limit, test.burst, 0)
				require.Equal(t, a.capacityAfter, capa, "capacity after in action %d failed", i)
				if a.retryAt > 0 {
					require.False(t, retryAt.IsZero())
					require.WithinDuration(t, current.Add(a.retryAt), retryAt, 10*time.Second, "retry after in action %d did not match expectation", i)
				} else {
					require.True(t, retryAt.IsZero(), "retry after in action %d failed with unexpected retry in %s", i, retryAt.Sub(current).String())
				}
			}
		})
	}
}

func TestLuaEndsWith(t *testing.T) {
	runScript := func(t *testing.T, rc rueidis.Client, key string) bool {
		val, err := scripts["test/ends_with"].Exec(
			t.Context(),
			rc,
			[]string{key},
			[]string{},
		).AsInt64()
		require.NoError(t, err)

		switch val {
		case 1:
			return true
		default:
			return false
		}
	}

	_, rc := initRedis(t)
	defer rc.Close()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.kg

	t.Run("with empty string", func(t *testing.T) {
		key := kg.BacklogSet("")
		require.Contains(t, key, ":-")
		require.False(t, runScript(t, rc, key))
	})

	t.Run("with non empty string", func(t *testing.T) {
		key := kg.BacklogSet("hello")
		require.NotContains(t, key, ":-")
		require.True(t, runScript(t, rc, key))
	})
}

func TestLuaScriptSnapshots(t *testing.T) {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}

	scripts := make(map[string]string)

	var readRedisScripts func(path string, entries []fs.DirEntry)

	readRedisScripts = func(path string, entries []fs.DirEntry) {
		for _, e := range entries {
			// NOTE: When using embed go always uses forward slashes as a path
			// prefix. filepath.Join uses OS-specific prefixes which fails on
			// windows, so we construct the path using Sprintf for all platforms
			if e.IsDir() {
				entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
				readRedisScripts(path+"/"+e.Name(), entries)
				continue
			}

			byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
			if err != nil {
				panic(fmt.Errorf("error reading redis lua script: %w", err))
			}

			name := path + "/" + e.Name()
			name = strings.TrimPrefix(name, "lua/")
			name = strings.TrimSuffix(name, ".lua")
			val := string(byt)

			// Add any includes.
			items := include.FindAllStringSubmatch(val, -1)
			if len(items) > 0 {
				// Replace each include
				for _, include := range items {
					byt, err = embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
					if err != nil {
						panic(fmt.Errorf("error reading redis lua include: %w", err))
					}
					val = strings.ReplaceAll(val, include[0], string(byt))
				}
			}

			scripts[name] = val
		}
	}

	readRedisScripts("lua", entries)

	// Test each script
	for scriptName, rawContent := range scripts {
		t.Run(scriptName, func(t *testing.T) {
			// Process the script

			// Read expected snapshot from fixture file
			snapshotPath := filepath.Join("testdata", "snapshots", scriptName+".lua")
			// Generate snapshot file if it doesn't exist
			err := os.MkdirAll(filepath.Dir(snapshotPath), 0755)
			require.NoError(t, err)

			err = os.WriteFile(snapshotPath, []byte(rawContent), 0644)
			require.NoError(t, err)

			t.Logf("Generated snapshot for %s at %s", scriptName, snapshotPath)
		})
	}
}
