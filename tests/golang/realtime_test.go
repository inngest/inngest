package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtime(t *testing.T) {
	ctx := context.Background()
	h, server, registerFuncs := NewSDKHandler(t, "realtime")
	defer server.Close()

	var (
		started, finished int32
		runID             string
		l                 sync.Mutex
	)

	fun := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "realtime"},
		inngestgo.EventTrigger("test/realtime", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			runID = input.InputCtx.RunID
			atomic.AddInt32(&started, 1)

			_, _ = step.Run(ctx, "step-1", func(ctx context.Context) (string, error) {

				// Wait for the lock so that we respond on demand.
				l.Lock()
				defer l.Unlock()

				return "step 1 data", nil
			})

			defer atomic.AddInt32(&finished, 1)
			return map[string]any{"output": "fn result", "done": true}, nil
		},
	)
	h.Register(fun)
	registerFuncs()

	t.Run("It shows step results", func(t *testing.T) {

		// Lock the mutex so that the step doesn't finish until we let it.
		l.Lock()

		_, err := inngestgo.Send(ctx, inngestgo.Event{
			Name: "test/realtime",
			Data: map[string]any{"number": 10},
		})
		require.NoError(t, err)
		require.Eventually(t, func() bool { return atomic.LoadInt32(&started) > 0 }, 5*time.Second, 5*time.Millisecond)

		jwt, err := NewToken(t, realtime.Topic{
			Kind:  realtime.TopicKindRun,
			RunID: ulid.MustParse(runID),
			Name:  realtime.TopicNameStep, // all step outputs
		})
		require.NoError(t, err)

		url := strings.Replace(os.Getenv("API_URL")+"/v1/realtime/connect", "http://", "ws://", 1)
		c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Authorization": []string{"Bearer " + jwt},
			},
		})
		assert.NoError(t, err)
		if c != nil {
			assert.NoError(t, err)
		}

		messages := []realtime.Message{}

		go func() {
			for {
				_, resp, err := c.Read(ctx)
				if isWebsocketClosed(err) {
					return
				}
				require.NoError(t, err)
				msg := realtime.Message{}
				err = json.Unmarshal(resp, &msg)
				require.NoError(t, err)
				messages = append(messages, msg)
			}
		}()

		l.Unlock()

		require.Eventually(t, func() bool { return atomic.LoadInt32(&finished) == 1 }, 5*time.Second, 5*time.Millisecond)
		require.NoError(t, c.CloseNow())

		require.Equal(t, 1, len(messages))
		require.Equal(t, realtime.MessageKindStep, messages[0].Kind)
		require.Equal(t, json.RawMessage(`"step 1 data"`), messages[0].Data)
		require.Equal(t, runID, messages[0].RunID.String())
	})

}

func NewToken(t *testing.T, topics ...realtime.Topic) (string, error) {
	resp, err := http.Post(os.Getenv("API_URL")+"/v1/realtime/token", "application/json", topicBuffer(topics))
	require.NoError(t, err)

	r := struct {
		JWT string `json:"jwt"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	require.NoError(t, err)

	return r.JWT, nil
}

func topicBuffer(topics []realtime.Topic) *bytes.Buffer {
	byt, _ := json.Marshal(topics)
	return bytes.NewBuffer(byt)
}

func isWebsocketClosed(err error) bool {
	if err == nil {
		return false
	}
	if websocket.CloseStatus(err) != -1 {
		return true
	}
	if err.Error() == "failed to get reader: use of closed network connection" {
		return true
	}
	return false
}
