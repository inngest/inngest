package eventstream

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"mime/multipart"
	"net/url"
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
			evt.ClearSize()
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
			evt.ClearSize()
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

	t.Parallel()

	t.Run("single form field", func(t *testing.T) {
		t.Parallel()
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
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{
				"name": []any{"Alice"},
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("multiple form fields", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		r.NoError(writer.WriteField("name", "Alice"))
		r.NoError(writer.WriteField("messages", "hello"))
		r.NoError(writer.WriteField("messages", "world"))
		r.NoError(writer.Close())

		contentType := writer.FormDataContentType()
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{
				"name":     []any{"Alice"},
				"messages": []any{"hello", "world"},
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("empty form fields", func(t *testing.T) {
		t.Parallel()
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
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{
				"empty1": []any{""},
				"empty2": []any{""},
				"valid":  []any{"value"},
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("max size exceeded", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		for item := range stream {
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("arbitrary bytes", func(t *testing.T) {
		// We don't support arbitrary bytes in form fields. This test just
		// documents that limitation

		t.Parallel()
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
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))

			// Intentionally gobbledygook because we don't support arbitrary
			// bytes in form fields
			r.Equal(map[string]any{
				"arbitrary_bytes": []any{"\x00�ޭ��\x00B"},
			}, result)

			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})
}

func TestParseStream_FormUrlencoded(t *testing.T) {
	// application/x-www-form-urlencoded

	t.Parallel()
	contentType := "application/x-www-form-urlencoded"

	t.Run("single form field", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		// Pick a long string to ensure that we don't truncate the body. This is
		// a good check because of the way ParseStream buffers the body
		longString := strings.Repeat("x", 10*1024)

		formData := url.Values{}
		formData.Set("msg", longString)
		body := strings.NewReader(formData.Encode())

		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			// The parser creates a JSON object with form fields as keys
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{
				"msg": []any{longString},
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("multiple form fields", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		formData := url.Values{}
		formData.Set("name", "Alice")
		formData.Add("messages", "hello")
		formData.Add("messages", "world")
		body := strings.NewReader(formData.Encode())

		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{
				"name":     []any{"Alice"},
				"messages": []any{"hello", "world"},
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("empty form fields", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		formData := url.Values{}
		formData.Set("empty1", "")
		formData.Set("valid", "value")
		formData.Set("empty2", "")
		body := strings.NewReader(formData.Encode())

		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{
				"empty1": []any{""},
				"empty2": []any{""},
				"valid":  []any{"value"},
			}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})

	t.Run("max size exceeded", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		formData := url.Values{}
		formData.Set("large_field", strings.Repeat("x", 300*1024))
		body := strings.NewReader(formData.Encode())

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
		// Reject requests with no form fields

		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		formData := url.Values{}
		body := strings.NewReader(formData.Encode())

		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		n := 0
		for item := range stream {
			var result map[string]any
			r.NoError(json.Unmarshal(item.Item, &result))
			r.Equal(map[string]any{}, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(0, n)
	})

	t.Run("JSON", func(t *testing.T) {
		// Replicate what curl does when sending a JSON body without specifying
		// the content-type:
		// curl -v localhost:8288/e/test -d '{"name": "my-event"}'
		//
		// It'll be application/x-www-form-urlencoded even though the body is a
		// JSON object

		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		body := strings.NewReader(`{"name": "my-event", "data": {"name": "Alice"}}`)
		stream := make(chan StreamItem)
		eg := errgroup.Group{}
		eg.Go(func() error {
			return ParseStream(ctx, body, stream, 512*1024, contentType)
		})

		expected := event.Event{
			Name: "my-event",
			Data: map[string]any{
				"name": "Alice",
			},
		}

		n := 0
		for item := range stream {
			var result event.Event
			r.NoError(json.Unmarshal(item.Item, &result))
			result.ClearSize()
			r.Equal(expected, result)
			n++
		}
		r.NoError(eg.Wait())
		r.Equal(1, n)
	})
}
