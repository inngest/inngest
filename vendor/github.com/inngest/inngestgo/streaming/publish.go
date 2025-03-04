package streaming

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/step"
)

var (
	DefaultPublishURL = "https://api.inngest.com/v1/realtime/publish"
)

// Publish sends data on a given channel and topic, allowing any subscribers
// to read this data in real time. This allows you to stream data or updates
// from functions directly to your user's UI.
//
// A channel is an environment-level identifier, which can contain one or more
// topics for sending data: "user:123", or "run_id:01JNGVNB7BJVRCZWTBF8G73BPJ"
//
// Topics are individual topics within a channel to differentiate between
// types of data sent.
func Publish(ctx context.Context, channel, topic string, data []byte) error {
	return PublishWithURL(ctx, DefaultPublishURL, channel, topic, data)
}

// PublishWithURL is the same as Publish, but allows overriding the API endpoint
// which is used for publishing.
func PublishWithURL(ctx context.Context, apiUrl, channel, topic string, data []byte) error {
	// Get the step context.  Right now, publishing is only available
	// within Inngest functions.
	mgr, ok := sdkrequest.Manager(ctx)
	if !ok {
		return step.ErrNotInFunction
	}

	// Is this being sent in between steps?  What's the last opcode
	// we've ran, and what are the opcodes in manager state.  If this is
	// in between, do not publish - this gives us "step-like" semantics
	// for code within steps;  we must do this as publish should be used
	// within steps and outside of steps.
	if !step.IsWithinStep(ctx) && len(mgr.Request().CallCtx.Stack.Stack) > 0 {
		// Only publish on the first attempt.
		if mgr.Request().CallCtx.Attempt > 0 {
			return nil
		}

		// If we're outside of a step context, only publish if we've replayed
		// the last step.
		last := mgr.Request().CallCtx.Stack.Stack[len(mgr.Request().CallCtx.Stack.Stack)-1]
		if !mgr.ReplayedStep(last) {
			return nil
		}
	}

	qp := url.Values{}
	qp.Add("channel", channel)
	qp.Add("topic", topic)
	qp.Encode()
	u := fmt.Sprintf("%s?%s", apiUrl, qp.Encode())

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", mgr.SigningKey()))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		byt, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error publishing (status %d): %s", resp.StatusCode, byt)
	}

	return nil
}
