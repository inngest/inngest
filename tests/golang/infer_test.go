package golang

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

func TestStepInfer(t *testing.T) {
	ctx := context.Background()

	// Create a new mock test server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
		    "id": "chatcmpl-AU1N8cUSBVK5kXQ7Q0zFmBXvKTmQd",
		    "object": "chat.completion",
		    "created": 1731718478,
		    "model": "gpt-4o-2024-08-06",
		    "choices": [
		      {
			"index": 0,
			"message": {
			  "role": "assistant",
			  "content": "Moonlight danced, shadows whispered secrets, love blossomed, time stood still.",
			  "refusal": null
			},
			"logprobs": null,
			"finish_reason": "stop"
		      }
		    ],
		    "usage": {
		      "prompt_tokens": 16,
		      "completion_tokens": 16,
		      "total_tokens": 32,
		      "prompt_tokens_details": {
			"cached_tokens": 0,
			"audio_tokens": 0
		      },
		      "completion_tokens_details": {
			"reasoning_tokens": 0,
			"audio_tokens": 0,
			"accepted_prediction_tokens": 0,
			"rejected_prediction_tokens": 0
		      }
		    },
		    "system_fingerprint": "fp_159d8341cc"
		  }`))
	}))
	defer ts.Close()

	h, server, registerFuncs := NewSDKHandler(t, "infer-test")
	defer server.Close()

	var ctr int32

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test infer"},
		inngestgo.EventTrigger("test/sdk-infer", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			out, err := step.Infer[openai.ChatCompletionRequest, openai.ChatCompletionResponse](ctx, "infer", step.InferOpts[openai.ChatCompletionRequest]{
				Opts: step.InferOpenAIOpts(nil, &ts.URL),
				Body: openai.ChatCompletionRequest{
					Model: "gpt-4o",
					Messages: []openai.ChatCompletionMessage{
						{Role: "system", Content: "Write a story in 10 words or less"},
					},
				},
			})

			if err != nil {
				panic(err)
			}

			fmt.Println("AI Response:")
			fmt.Println(out.Choices[0].Message.Content)

			require.Equal(t, "Moonlight danced, shadows whispered secrets, love blossomed, time stood still.", out.Choices[0].Message.Content)

			atomic.AddInt32(&ctr, 1)

			return out.Choices[0].Message.Content, err
		},
	)
	h.Register(a)
	registerFuncs()

	evt := inngestgo.Event{
		Name: "test/sdk-infer",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	}
	_, err := inngestgo.Send(ctx, evt)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return atomic.LoadInt32(&ctr) == 1 }, 5*time.Second, 50*time.Millisecond)
}
