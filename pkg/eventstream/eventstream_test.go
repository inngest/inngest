package eventstream

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestParseStream_Single(t *testing.T) {
	// A large string just under the event size
	data := make([]byte, 1024*200)
	_, err := rand.Read(data)
	require.NoError(t, err)
	str := hex.EncodeToString(data)

	actual := event.Event{
		Name: "1",
		Data: map[string]any{
			"order":  float64(1),
			"string": str,
		},
	}

	byt, err := json.Marshal(actual)
	require.NoError(t, err)
	r := bytes.NewReader(byt)
	stream := make(chan StreamItem)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return ParseStream(context.Background(), r, stream, 512*1024)
	})

	n := 0
	for item := range stream {
		evt := event.Event{}
		err := json.Unmarshal(item.Item, &evt)
		require.NoError(t, err)
		require.EqualValues(t, actual, evt)
		n++
	}

	require.NoError(t, eg.Wait())
	require.EqualValues(t, n, 1)
}

func TestParseStream_Multiple(t *testing.T) {
	evts := []event.Event{
		{
			Name: "1",
			Data: map[string]any{
				"order": float64(1),
			},
		},
		{
			Name: "2",
			Data: map[string]any{
				"order": float64(2),
			},
		},
		{
			Name: "3",
			Data: map[string]any{
				"order": float64(3),
			},
		},
	}

	byt, err := json.Marshal(evts)
	require.NoError(t, err)

	r := bytes.NewReader(byt)
	stream := make(chan StreamItem)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return ParseStream(context.Background(), r, stream, 512*1024)
	})

	n := 0
	for item := range stream {
		evt := event.Event{}
		err := json.Unmarshal(item.Item, &evt)
		require.NoError(t, err)
		require.EqualValues(t, evts[n], evt)
		n++
	}

	require.NoError(t, eg.Wait())
	require.EqualValues(t, n, 3)
}

func TestParseStream_MaxSize(t *testing.T) {
	data := make([]byte, 1024*512)
	_, err := rand.Read(data)
	require.NoError(t, err)

	str := hex.EncodeToString(data)

	evts := []event.Event{
		{
			Name: "large",
			Data: map[string]any{
				"order": float64(1),
				"large": str,
			},
		},
	}

	byt, err := json.Marshal(evts)
	require.NoError(t, err)

	r := bytes.NewReader(byt)
	stream := make(chan StreamItem)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return ParseStream(context.Background(), r, stream, 256*1024)
	})

	n := 0
	for range stream {
		n++
	}

	<-time.After(10 * time.Millisecond)

	err = eg.Wait()
	require.EqualValues(t, n, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrEventTooLarge.Error())
}
