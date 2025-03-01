package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
			Kind:       streamingtypes.MessageKindRun,
			Data:       json.RawMessage(`"foo"`),
			CreatedAt:  time.Now().Truncate(time.Millisecond).UTC(),
			Channel:    "user:123",
			TopicNames: []string{"ai"},
			EnvID:      consts.DevServerEnvID,
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

	// Attempt to publish stream data to the websocket
	data := "test please"

	resp, err := http.Post(
		s.URL+"/realtime/publish?channel=user:123&topic=ai",
		"text/stream",
		strings.NewReader(data),
	)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, 200)

	//
	// Track all messages received by the websocket.  This is a streaming
	// request, so we should receive 3 messages: stream start, the stream chunk,
	// and stream end.
	//

	contents := [][]byte{}
	var counter int32
	go func() {
		for {
			_, resp, err := c.Read(context.TODO())
			require.NoError(t, err)
			contents = append(contents, resp)
			atomic.AddInt32(&counter, 1)
		}
	}()

	require.Eventually(
		t,
		func() bool { return atomic.LoadInt32(&counter) == 3 },
		3*time.Second,
		time.Millisecond,
	)

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
