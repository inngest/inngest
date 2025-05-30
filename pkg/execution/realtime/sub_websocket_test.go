package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/stretchr/testify/require"
)

func TestWebsocketMessage(t *testing.T) {
	ctx := context.Background()

	b := NewInProcessBroadcaster()
	s := httptest.NewServer(NewAPI(APIOpts{
		JWTSecret:   []byte("foo"),
		Broadcaster: b,
	}))

	c := wsConnect(t, s, Topic{
		Kind:    streamingtypes.TopicKindRun,
		Channel: "user:123",
		Name:    "ai",
	})

	<-time.After(time.Second)

	// Broadcasting should publish.
	t.Run("broadcasting publishes to websocket", func(t *testing.T) {
		send := Message{
			Kind:      streamingtypes.MessageKindRun,
			Data:      json.RawMessage(`"foo"`),
			CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
			Channel:   "user:123",
			Topic:     "ai",
			EnvID:     consts.DevServerEnvID,
		}
		b.Publish(ctx, send)

		msg := readMessageWithin(t, c, time.Second)
		require.EqualValues(t, send, *msg)
	})
}

func TestWebsocketPostRealtimeMessage(t *testing.T) {
	b := NewInProcessBroadcaster()
	s := httptest.NewServer(NewAPI(APIOpts{
		JWTSecret:   []byte("foo"),
		Broadcaster: b,
	}))

	c := wsConnect(t, s, Topic{
		Kind:    streamingtypes.TopicKindRun,
		Channel: "user:123",
		Name:    "ai",
	})

	<-time.After(time.Second)

	// Attempt to publish stream data to the websocket

	resp, err := http.Post(
		s.URL+"/realtime/publish?channel=user:123&topic=ai",
		"",
		strings.NewReader("test"),
	)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, 200)

	msg := readMessageWithin(t, c, time.Second)
	require.EqualValues(t, `"test"`, string(msg.Data))
}

func TestWebsocketPostStreamingMessage(t *testing.T) {
	b := NewInProcessBroadcaster()
	s := httptest.NewServer(NewAPI(APIOpts{
		JWTSecret:   []byte("foo"),
		Broadcaster: b,
	}))

	c := wsConnect(t, s, Topic{
		Kind:    streamingtypes.TopicKindRun,
		Channel: "user:123",
		Name:    "ai",
	})

	<-time.After(time.Second)

	//
	// Track all messages received by the websocket.  This is a streaming
	// request, so we should receive 3 messages: stream start, the stream chunk,
	// and stream end.
	//

	var (
		counter  int32
		contents = [][]byte{}
		l        sync.Mutex
	)

	go func() {
		for {
			_, resp, err := c.Read(context.TODO())
			require.NoError(t, err)
			atomic.AddInt32(&counter, 1)
			l.Lock()
			contents = append(contents, resp)
			l.Unlock()
			fmt.Println("received", string(resp))
		}
	}()

	t.Run("manual publish", func(t *testing.T) {
		// Attempt to publish stream data to the websocket
		data := "test please"

		resp, err := http.Post(
			s.URL+"/realtime/publish?channel=user:123&topic=ai",
			"text/stream",
			strings.NewReader(data),
		)
		require.NoError(t, err)
		require.Equal(t, resp.StatusCode, 200)

		require.Eventually(
			t,
			func() bool { return atomic.LoadInt32(&counter) == 3 },
			3*time.Second,
			time.Millisecond,
		)

		l.Lock()
		defer l.Unlock()

		// Assert that the first msg is a stream start, the last is a stream end,
		// and we have our stream content in the middle.
		var start, end Message
		err = json.Unmarshal(contents[0], &start)
		require.NoError(t, err)
		err = json.Unmarshal(contents[2], &end)
		require.NoError(t, err)

		var streamID string
		err = json.Unmarshal(start.Data, &streamID)
		require.NoError(t, err)

		require.EqualValues(t, streamingtypes.MessageKindDataStreamStart, start.Kind)
		require.EqualValues(t, streamingtypes.MessageKindDataStreamEnd, end.Kind)

		byt, _ := json.Marshal(Chunk{
			Kind:     string(streamingtypes.MessageKindDataStreamChunk),
			StreamID: streamID,
			Data:     data,
		})

		require.EqualValues(t, byt, contents[1])
	})

	t.Run("TeeStreamReaderToAPI", func(t *testing.T) {
		counter = 0
		l.Lock()
		contents = [][]byte{}
		l.Unlock()

		// Create a single server which will respond with content, eg. server-sent-events.
		og := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Simulate sending events (you can replace this with real data)
			for i := range 5 {
				fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf("Event %d", i))
				time.Sleep(2 * time.Second)
				w.(http.Flusher).Flush()
			}
		}))

		// Create a new request which hits our test server, then tees this into
		// the stream.
		resp, err := http.Get(og.URL)
		require.NoError(t, err)
		read, err := TeeStreamReaderToAPI(resp.Body, s.URL+"/realtime/publish", TeeStreamOptions{
			Channel: "user:123",
			Topic:   "ai",
			Token:   "test-token",
			Metadata: map[string]any{
				"content-type": "text/event-stream",
			},
		},
		)
		require.NoError(t, err)
		byt, err := io.ReadAll(read)
		require.NoError(t, err)
		require.NotEmpty(t, byt)
		resp.Body.Close()

		require.Eventually(
			t,
			func() bool { return atomic.LoadInt32(&counter) == 7 },
			3*time.Second,
			time.Millisecond,
		)

		l.Lock()
		defer l.Unlock()
		// Assert that we had 7 messages:  stream start, 5 events, and stream end.
		require.EqualValues(t, 7, len(contents))

		// SSE
		require.Contains(t, string(contents[1]), "data: Event 0")
		require.Contains(t, string(contents[2]), "data: Event 1")
		require.Contains(t, string(contents[3]), "data: Event 2")
		require.Contains(t, string(contents[4]), "data: Event 3")
		require.Contains(t, string(contents[5]), "data: Event 4")
	})
}

func readMessageWithin(t *testing.T, c *websocket.Conn, dur time.Duration) *Message {
	mc := make(chan Message)

	go func() {
		_, resp, err := c.Read(context.Background())
		require.NoError(t, err)

		msg := Message{}
		err = json.Unmarshal(resp, &msg)
		require.NoError(t, err)
		mc <- msg
	}()

	select {
	case <-time.After(dur):
		t.Fatalf("didnt receive message within timeout")
	case msg := <-mc:
		return &msg
	}
	return nil
}

func wsConnect(t *testing.T, s *httptest.Server, topic Topic) *websocket.Conn {
	jwt, err := newToken(t, s.URL, topic)
	require.NoError(t, err)

	url := s.URL + "/realtime/connect"
	c, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{"Bearer " + jwt},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

func newToken(t *testing.T, url string, topics ...Topic) (string, error) {
	resp, err := http.Post(url+"/realtime/token", "application/json", topicBuffer(topics))
	require.NoError(t, err)

	byt, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	r := struct {
		JWT string `json:"jwt"`
	}{}
	err = json.Unmarshal(byt, &r)
	require.NoError(t, err, "%s", byt)

	return r.JWT, nil
}

func topicBuffer(topics []Topic) *bytes.Buffer {
	byt, _ := json.Marshal(topics)
	return bytes.NewBuffer(byt)
}
