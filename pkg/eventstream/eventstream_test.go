package eventstream

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestParseStream_JSON(t *testing.T) {
	// application/json
	contentType := "application/json"

	t.Run("single", func(t *testing.T) {
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
			return ParseStream(context.Background(), r, stream, 512*1024, contentType)
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
	})

	t.Run("multiple", func(t *testing.T) {
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
			return ParseStream(context.Background(), r, stream, 512*1024, contentType)
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
	})

	t.Run("max size", func(t *testing.T) {
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
			return ParseStream(context.Background(), r, stream, 256*1024, contentType)
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
	})
}

func TestParseStream_Multipart(t *testing.T) {
	// multipart/form-data

	t.Run("single form field", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		r.NoError(writer.WriteField("name", "Alice"))
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			// The multipart parser creates a JSON object with form fields as keys
			var result map[string]string
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal("Alice", result["name"])
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("multiple form fields", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		r.NoError(writer.WriteField("name", "Alice"))
		r.NoError(writer.WriteField("age", "25"))
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]string
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]string{
				"name": "Alice",
				"age":  "25",
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("empty form fields", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		r.NoError(writer.WriteField("empty1", ""))
		r.NoError(writer.WriteField("valid", "value"))
		r.NoError(writer.WriteField("empty2", ""))
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]string
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]string{
				"valid": "value",
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("max size exceeded", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		largeData := strings.Repeat("x", 300*1024)
		r.NoError(writer.WriteField("large_field", largeData))
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 256*1024, contentType) // 256KB limit
		})

		n := 0
		for range stream {
			n++
		}
		err := eg.Wait()
		r.Equal(0, n)
		r.Error(err)
		r.Contains(err.Error(), ErrEventTooLarge.Error())
	})

	t.Run("no form fields", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for range stream {
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(0, n)
	})

	t.Run("arbitrary bytes", func(t *testing.T) {
		// We don't support arbitrary bytes in form fields. This test just
		// documents that limitation

		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		arbitraryBytes := []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x42}
		field, err := writer.CreateFormField("arbitrary_bytes")
		r.NoError(err)
		_, err = field.Write(arbitraryBytes)
		r.NoError(err)
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]string
			r.NoError(json.Unmarshal(item.Item, &result))

			// Intentionally gobbledygook because we don't support arbitrary
			// bytes in form fields
			r.Equal("\x00�ޭ��\x00B", result["arbitrary_bytes"])

			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})
}
