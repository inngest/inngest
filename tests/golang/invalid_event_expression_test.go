package golang

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
)

func TestInvalidEventExpression(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t)
	defer server.Close()

	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{Name: "test sdk"},
		inngestgo.EventTrigger(
			"test/sdk",
			// NOTE: This is not a valid expression.
			inngestgo.StrPtr("event.data.what == len(lol).find('test') -"),
		),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) { return nil, nil },
	)
	h.Register(a)
	registerFuncs()

	<-time.After(20 * time.Second)
}
