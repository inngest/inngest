package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/testharness"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestNewRunMetadata(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]string
		expectedVal   *runMetadata
		expectedError error
	}{
		{
			name: "should return value if data is valid",
			data: map[string]string{
				"status":   "1",
				"pending":  "0",
				"version":  "1",
				"debugger": "false",
			},
			expectedVal: &runMetadata{
				Status:   enums.RunStatusCompleted,
				Pending:  0,
				Version:  1,
				Debugger: false,
			},
			expectedError: nil,
		},
		{
			name:          "should error with missing status",
			data:          map[string]string{},
			expectedError: errors.New("no status stored in metadata"),
		},
		{
			name: "should error with non int status",
			data: map[string]string{
				"status": "hello",
			},
			expectedError: errors.New("invalid function status stored in run metadata: \"hello\""),
		},
		{
			name: "missing version should return 0",
			data: map[string]string{
				"status":  "1",
				"pending": "0",
			},
			expectedVal: &runMetadata{
				Status:  enums.RunStatusCompleted,
				Pending: 0,
				Version: 0,
			},
			expectedError: nil,
		},
		{
			name: "invalid version should return error",
			data: map[string]string{
				"status":  "1",
				"pending": "0",
				"version": "yolo",
			},
			expectedError: errors.New("invalid metadata version detected: \"yolo\""),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runMeta, err := newRunMetadata(test.data)
			require.Equal(t, test.expectedError, err)
			require.Equal(t, test.expectedVal, runMeta)
		})
	}
}

func TestStateHarness(t *testing.T) {
	r := miniredis.RunT(t)
	sm, err := New(
		context.Background(),
		WithKeyPrefix("{test}:"),
		WithFunctionLoader(testharness.FunctionLoader()),
		WithConnectOpts(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		}),
	)
	require.NoError(t, err)

	create := func() (state.Manager, func()) {
		return sm, func() {
			r.FlushAll()
		}
	}

	testharness.CheckState(t, create)
}

func TestScanIter(t *testing.T) {
	ctx := context.Background()
	redis := miniredis.RunT(t)
	r, err := rueidis.NewClient(rueidis.ClientOption{
		// InitAddress: []string{"127.0.0.1:6379"},
		InitAddress:  []string{redis.Addr()},
		Password:     "",
		DisableCache: true,
	})
	require.NoError(t, err)

	entries := 50_000
	key := "test-scan"
	for i := 0; i < entries; i++ {

		cmd := r.B().Hset().Key(key).FieldValue().
			FieldValue(strconv.Itoa(i), strconv.Itoa(i)).
			Build()
		require.NoError(t, r.Do(ctx, cmd).Error())
	}
	fmt.Println("loaded keys")

	// Create a new scanIter, which iterates through all items.
	si := &scanIter{r: r}
	err = si.init(ctx, key, 1000)
	require.NoError(t, err)

	listed := 0

	for n := 0; n < entries*2; n++ {
		if !si.Next(ctx) {
			break
		}
		listed++
	}

	require.Equal(t, context.Canceled, si.Error())
	require.Equal(t, entries, listed)
}

func TestBufIter(t *testing.T) {
	ctx := context.Background()
	redis := miniredis.RunT(t)
	r, err := rueidis.NewClient(rueidis.ClientOption{
		// InitAddress: []string{"127.0.0.1:6379"},
		InitAddress:  []string{redis.Addr()},
		Password:     "",
		DisableCache: true,
	})
	require.NoError(t, err)

	entries := 10_000
	key := "test-bufiter"
	for i := 0; i < entries; i++ {
		cmd := r.B().Hset().Key(key).FieldValue().FieldValue(strconv.Itoa(i), "{}").Build()
		require.NoError(t, r.Do(ctx, cmd).Error())
	}
	fmt.Println("loaded keys")

	// Create a new scanIter, which iterates through all items.
	bi := &bufIter{r: r}
	err = bi.init(ctx, key)
	require.NoError(t, err)

	listed := 0
	for n := 0; n < entries*2; n++ {
		if !bi.Next(ctx) {
			break
		}
		listed++
	}

	require.Equal(t, context.Canceled, bi.Error())
	require.Equal(t, entries, listed)
}

func BenchmarkNew(b *testing.B) {
	r := miniredis.RunT(b)
	sm, err := New(
		context.Background(),
		WithConnectOpts(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		}),
	)
	require.NoError(b, err)

	id := state.Identifier{
		WorkflowID: uuid.New(),
	}
	evt := event.Event{
		Name: "test-event",
		Data: map[string]any{
			"title": "They don't think it be like it is, but it do",
			"data": map[string]any{
				"float": 3.14132,
			},
		},
		User: map[string]any{
			"external_id": "1",
		},
		Version: "1985-01-01",
	}.Map()
	init := state.Input{
		Identifier:     id,
		EventBatchData: []map[string]any{evt},
	}

	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		init.Identifier.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
		_, err := sm.New(ctx, init)
		require.NoError(b, err)
	}

}
