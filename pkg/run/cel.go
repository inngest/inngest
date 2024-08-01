package run

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/expressions"
	"golang.org/x/sync/errgroup"
)

var (
	eventRegex  = regexp.MustCompile(`^event\..+`)
	outputRegex = regexp.MustCompile(`^output`)
)

type ExprHandlerOpt func(h *ExpressionHandler)

func WithExpressionHandlerExpressions(cel []string) ExprHandlerOpt {
	return func(h *ExpressionHandler) {
		h.add(cel)
	}
}

func WithExpressionHandlerBlob(exp string, delimiter string) ExprHandlerOpt {
	if delimiter == "" {
		delimiter = "\n"
	}
	cel := strings.Split(exp, delimiter)

	return func(h *ExpressionHandler) {
		h.add(cel)
	}
}

type ExpressionHandler struct {
	EventExprList  []string
	OutputExprList []string
}

func NewExpressionHandler(opts ...ExprHandlerOpt) *ExpressionHandler {
	h := &ExpressionHandler{
		EventExprList:  []string{},
		OutputExprList: []string{},
	}

	for _, apply := range opts {
		apply(h)
	}

	return h
}

func (h *ExpressionHandler) add(cel []string) {
	for _, e := range cel {
		switch {
		case eventRegex.MatchString(e):
			h.EventExprList = append(h.EventExprList, e)
		case outputRegex.MatchString(e):
			h.OutputExprList = append(h.OutputExprList, e)
		}
	}
}

func (h *ExpressionHandler) HasFilters() bool {
	return h.HasEventFilters() || h.HasOutputFilters()
}

func (h *ExpressionHandler) HasEventFilters() bool {
	return len(h.EventExprList) > 0
}

func (h *ExpressionHandler) HasOutputFilters() bool {
	return len(h.OutputExprList) > 0
}

func (h *ExpressionHandler) MatchEventExpressions(ctx context.Context, evt event.Event) (bool, error) {
	eg := errgroup.Group{}
	res := make([]bool, len(h.EventExprList))
	data := evt.Map()

	for i, exp := range h.EventExprList {
		idx := i

		eg.Go(func() error {
			eval, err := expressions.NewBooleanEvaluator(ctx, exp)
			if err != nil {
				return fmt.Errorf("error initializing expression evaluator for event: %w", err)
			}

			ok, _, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": data}))
			if err != nil {
				return fmt.Errorf("error evaluating event expression: %w", err)
			}

			res[idx] = ok
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return allMatches(res), nil
}

func allMatches(res []bool) bool {
	for _, v := range res {
		if !v {
			return false
		}
	}
	return true
}
