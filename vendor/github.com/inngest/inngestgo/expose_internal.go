package inngestgo

import (
	"github.com/inngest/inngestgo/internal/fn"
)

type ServableFunction = fn.ServableFunction
type FunctionOpts = fn.FunctionOpts
type Debounce = fn.Debounce
type Throttle = fn.Throttle
type RateLimit = fn.RateLimit
type Timeouts = fn.Timeouts

type Input[T any] = fn.Input[T]
type InputCtx = fn.InputCtx
