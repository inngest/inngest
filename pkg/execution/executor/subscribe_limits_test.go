package executor

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestValidateSubscribe(t *testing.T) {
	ok := func() *state.SubscribeOpts {
		return &state.SubscribeOpts{Key: "inbox", Mode: state.SubscribeOpen}
	}
	for name, tc := range map[string]struct {
		change func(*state.SubscribeOpts)
		event  string
		bad    bool
	}{
		"valid":                 {change: func(*state.SubscribeOpts) {}},
		"empty key":             {change: func(s *state.SubscribeOpts) { s.Key = "" }, bad: true},
		"long key":              {change: func(s *state.SubscribeOpts) { s.Key = strings.Repeat("k", consts.MaxSubscriptionKeyLength+1) }, bad: true},
		"key at the limit":      {change: func(s *state.SubscribeOpts) { s.Key = strings.Repeat("k", consts.MaxSubscriptionKeyLength) }},
		"long reader":           {change: func(s *state.SubscribeOpts) { s.Reader = strings.Repeat("r", consts.MaxSubscriptionKeyLength+1) }, bad: true},
		"negative stream":       {change: func(s *state.SubscribeOpts) { s.Stream = -1 }, bad: true},
		"too many streams":      {change: func(s *state.SubscribeOpts) { s.Stream = consts.MaxSubscriptionPausesPerRun }, bad: true},
		"last stream":           {change: func(s *state.SubscribeOpts) { s.Stream = consts.MaxSubscriptionPausesPerRun - 1 }},
		"negative attempt":      {change: func(s *state.SubscribeOpts) { s.Attempt = -1 }, bad: true},
		"park seq zero":         {change: func(s *state.SubscribeOpts) { s.Mode = state.SubscribePark }, bad: true},
		"poll seq negative":     {change: func(s *state.SubscribeOpts) { s.Mode = state.SubscribePoll; s.Seq = -3 }, bad: true},
		"park seq one":          {change: func(s *state.SubscribeOpts) { s.Mode = state.SubscribePark; s.Seq = 1 }},
		"reserved event":        {change: func(*state.SubscribeOpts) {}, event: pauses.SubscriptionParkIndexPrefix + ":run:hash", bad: true},
		"ordinary event":        {change: func(*state.SubscribeOpts) {}, event: "agent/message"},
		"reserved name on park": {change: func(s *state.SubscribeOpts) { s.Mode = state.SubscribePark; s.Seq = 1 }, event: pauses.SubscriptionParkIndexPrefix},
	} {
		t.Run(name, func(t *testing.T) {
			sub := ok()
			tc.change(sub)
			err := validateSubscribe(sub, tc.event)
			if !tc.bad {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, state.ErrSubscriptionLimit)
			var std *state.WrappedStandardError
			require.ErrorAs(t, err, &std)
			require.Equal(t, state.InngestErrSubscriptionLimit, std.Name)
		})
	}
}

func messageStack(key string, n int) []string {
	var stack []string
	for i := 0; i < n; i++ {
		stack = append(stack, subscriptionMessageID(key, fmt.Sprintf("evt-%d", i)))
	}
	return stack
}

func TestSubscriptionMessageLimit(t *testing.T) {
	md := func(stack []string, stateSize int) *sv2.Metadata {
		return &sv2.Metadata{Stack: stack, Metrics: sv2.RunMetrics{StateSize: stateSize}}
	}
	step, st := consts.DefaultMaxStepLimit, consts.DefaultMaxStateSizeLimit

	require.Empty(t, subscriptionMessageLimit(md(messageStack("inbox", consts.MaxSubscriptionMessages-1), 0), "inbox", 10, step, st))
	require.NotEmpty(t, subscriptionMessageLimit(md(messageStack("inbox", consts.MaxSubscriptionMessages), 0), "inbox", 10, step, st))

	// Other subscriptions' messages and closed markers do not count against this one.
	other := append(messageStack("other", 150), subscriptionClosedID("inbox"), "step-a")
	require.Empty(t, subscriptionMessageLimit(md(other, 0), "inbox", 10, step, st))

	// The run-wide limit across subscriptions.
	var many []string
	for i := 0; i < consts.MaxSubscriptionMessagesPerRun/100; i++ {
		many = append(many, messageStack(fmt.Sprintf("s%d", i), 100)...)
	}
	require.NotEmpty(t, subscriptionMessageLimit(md(many, 0), "fresh", 10, step, st))

	// Half of a small step limit.
	require.NotEmpty(t, subscriptionMessageLimit(md(messageStack("inbox", 5), 0), "inbox", 10, 10, st))
	require.Empty(t, subscriptionMessageLimit(md(messageStack("inbox", 4), 0), "inbox", 10, 10, st))

	// Three quarters of the state limit.
	require.Empty(t, subscriptionMessageLimit(md(nil, st/4*3-100), "inbox", 100, step, st))
	require.NotEmpty(t, subscriptionMessageLimit(md(nil, st/4*3-100), "inbox", 101, step, st))
}

type limitRunService struct {
	sv2.RunService
	md    sv2.Metadata
	saved map[string][]byte
	stack []string
}

func (s *limitRunService) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	return s.md, nil
}
func (s *limitRunService) LoadStack(context.Context, sv2.ID) ([]string, error) { return s.stack, nil }
func (s *limitRunService) SaveStep(_ context.Context, _ sv2.ID, stepID string, data []byte) (bool, error) {
	if s.saved == nil {
		s.saved = map[string][]byte{}
	}
	s.saved[stepID] = data
	s.stack = append(s.stack, stepID)
	return false, nil
}

type emptyIter struct{}

func (emptyIter) Count() int                       { return 0 }
func (emptyIter) Index() int64                     { return 0 }
func (emptyIter) Error() error                     { return nil }
func (emptyIter) Next(context.Context) bool        { return false }
func (emptyIter) Val(context.Context) *state.Pause { return nil }

type limitPauseManager struct {
	pauses.Manager
	runPauses int
	wrote     []*state.Pause
	deleted   []uuid.UUID
	markers   []*state.Pause
}

func (m *limitPauseManager) Write(_ context.Context, _ pauses.Index, p ...*state.Pause) (int, error) {
	m.wrote = append(m.wrote, p...)
	return len(p), nil
}
func (m *limitPauseManager) PauseIDsForRun(context.Context, ulid.ULID) ([]uuid.UUID, error) {
	return make([]uuid.UUID, m.runPauses), nil
}
func (m *limitPauseManager) Delete(_ context.Context, _ pauses.Index, p state.Pause, _ ...state.DeletePauseOpt) error {
	m.deleted = append(m.deleted, p.ID)
	return nil
}
func (m *limitPauseManager) PausesSince(context.Context, pauses.Index, time.Time) (state.PauseIterator, error) {
	return &markerIter{markers: m.markers}, nil
}

type markerIter struct {
	markers []*state.Pause
	i       int
}

func (m *markerIter) Count() int                { return len(m.markers) }
func (m *markerIter) Index() int64              { return int64(m.i) }
func (m *markerIter) Error() error              { return nil }
func (m *markerIter) Next(context.Context) bool { m.i++; return m.i <= len(m.markers) }
func (m *markerIter) Val(context.Context) *state.Pause {
	return m.markers[m.i-1]
}

func limitMetadata() sv2.Metadata {
	return sv2.Metadata{
		ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New(), Tenant: sv2.Tenant{EnvID: uuid.New(), AccountID: uuid.New()}},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}
}

// A message that breaks a limit is dropped, the stream's pause is removed so it
// stops matching, and the subscription is closed with the limit named.
func TestDeliverSubscriptionLimitClosesTheSubscription(t *testing.T) {
	md := limitMetadata()
	md.Stack = messageStack("inbox", consts.MaxSubscriptionMessages)
	rs := &limitRunService{md: md, stack: md.Stack}
	pm := &limitPauseManager{}
	e := &executor{log: logger.VoidLogger(), smv2: rs, pm: pm}

	name := "agent/message"
	pause := &state.Pause{
		ID: uuid.New(), WorkspaceID: md.ID.Tenant.EnvID, Event: &name,
		Identifier: sv2.NewPauseIdentifier(md.ID),
		Subscribe:  &state.PauseSubscribe{Key: "inbox"},
	}
	evt := event.InternalEvent{ID: ulid.MustNew(ulid.Now(), nil), Event: event.Event{Name: name}}

	require.NoError(t, e.deliverSubscription(context.Background(), evt, pause, logger.VoidLogger()))

	require.Equal(t, []uuid.UUID{pause.ID}, pm.deleted)
	require.NotContains(t, rs.saved, subscriptionMessageID("inbox", evt.GetInternalID().String()))
	require.Contains(t, string(rs.saved[subscriptionClosedID("inbox")]), "limit")

	// A redelivery of a message that was saved before the limit is not new.
	rs2 := &limitRunService{md: md, stack: md.Stack}
	pm2 := &limitPauseManager{}
	e2 := &executor{log: logger.VoidLogger(), smv2: rs2, pm: pm2}
	md2 := md
	md2.Stack = append(append([]string{}, md.Stack...), subscriptionMessageID("inbox", evt.GetInternalID().String()))
	rs2.md = md2
	require.NoError(t, e2.deliverSubscription(context.Background(), evt, pause, logger.VoidLogger()))
	require.Empty(t, pm2.deleted)
	require.NotContains(t, rs2.saved, subscriptionClosedID("inbox"))
}

func openCtx(md sv2.Metadata) *mockRunContext {
	return &mockRunContext{md: md}
}

func newLimitExecutor(rs *limitRunService, pm *limitPauseManager) *executor {
	return &executor{
		log:            logger.VoidLogger(),
		smv2:           rs,
		pm:             pm,
		queue:          &stubQueue{},
		tracerProvider: tracing.NewNoopTracerProvider(),
	}
}

func subscribeGen(sub state.SubscribeOpts, id string) (state.GeneratorOpcode, *state.WaitForEventOpts) {
	return state.GeneratorOpcode{Op: enums.OpcodeWaitForEvent, ID: id}, &state.WaitForEventOpts{Event: "agent/message", Subscribe: &sub}
}

// Opening a stream when the run already holds the most pauses fails the run and
// leaves no pause behind; one under the limit opens.
func TestOpenSubscriptionPauseLimit(t *testing.T) {
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}

	md := limitMetadata()
	rs := &limitRunService{md: md}
	pm := &limitPauseManager{runPauses: consts.MaxSubscriptionPausesPerRun}
	e := newLimitExecutor(rs, pm)
	gen, opts := subscribeGen(state.SubscribeOpts{Key: "inbox", Mode: state.SubscribeOpen}, "open")
	require.NoError(t, e.handleGeneratorSubscribe(context.Background(), openCtx(md), gen, edge, OpcodeGroup{}, opts, nil))
	require.Len(t, pm.wrote, 1)

	pm = &limitPauseManager{runPauses: consts.MaxSubscriptionPausesPerRun + 1}
	e = newLimitExecutor(rs, pm)
	err := e.handleGeneratorSubscribe(context.Background(), openCtx(md), gen, edge, OpcodeGroup{}, opts, nil)
	require.ErrorIs(t, err, state.ErrSubscriptionLimit)
	require.Equal(t, []uuid.UUID{pm.wrote[0].ID}, pm.deleted)
}

// A subscription with too many reads waiting at once fails the run on the park
// that is over, and removes that marker.
func TestParkLimit(t *testing.T) {
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}
	md := limitMetadata()
	rs := &limitRunService{md: md}

	var markers []*state.Pause
	for i := 0; i < consts.MaxSubscriptionParks; i++ {
		markers = append(markers, &state.Pause{ID: uuid.New(), Outgoing: fmt.Sprintf("park-%d", i)})
	}
	pm := &limitPauseManager{markers: markers}
	e := newLimitExecutor(rs, pm)
	stepID := subscriptionStepID("inbox", "", 1, 0)
	gen, opts := subscribeGen(state.SubscribeOpts{Key: "inbox", Mode: state.SubscribePark, Seq: 1}, stepID)

	// At the limit: allowed.
	require.NoError(t, e.handleGeneratorSubscribe(context.Background(), openCtx(md), gen, edge, OpcodeGroup{}, opts, nil))

	// One more: refused.
	pm = &limitPauseManager{markers: append(markers, &state.Pause{ID: uuid.New(), Outgoing: "park-extra"})}
	e = newLimitExecutor(rs, pm)
	err := e.handleGeneratorSubscribe(context.Background(), openCtx(md), gen, edge, OpcodeGroup{}, opts, nil)
	require.ErrorIs(t, err, state.ErrSubscriptionLimit)
	require.Len(t, pm.deleted, 1)

	// Markers whose park is answered are not waiting.
	answered := append([]*state.Pause{}, markers...)
	answered = append(answered, &state.Pause{ID: uuid.New(), Outgoing: "done"})
	rs.stack = []string{"done"}
	pm = &limitPauseManager{markers: answered}
	e = newLimitExecutor(rs, pm)
	require.NoError(t, e.handleGeneratorSubscribe(context.Background(), openCtx(md), gen, edge, OpcodeGroup{}, opts, nil))
}

// A run at its step limit takes no message, whatever messages it holds.
func TestSubscriptionMessageLimitStepCount(t *testing.T) {
	md := &sv2.Metadata{Metrics: sv2.RunMetrics{StepCount: 10}}
	require.NotEmpty(t, subscriptionMessageLimit(md, "inbox", 1, 10, consts.DefaultMaxStateSizeLimit))
	md.Metrics.StepCount = 9
	require.Empty(t, subscriptionMessageLimit(md, "inbox", 1, 10, consts.DefaultMaxStateSizeLimit))
}

// Another stream of a closed subscription saves nothing and removes its pause.
func TestDeliverToClosedSubscriptionRemovesThePause(t *testing.T) {
	md := limitMetadata()
	md.Stack = []string{subscriptionClosedID("inbox")}
	rs := &limitRunService{md: md, stack: md.Stack}
	pm := &limitPauseManager{}
	e := &executor{log: logger.VoidLogger(), smv2: rs, pm: pm}

	name := "agent/message"
	pause := &state.Pause{
		ID: uuid.New(), WorkspaceID: md.ID.Tenant.EnvID, Event: &name,
		Identifier: sv2.NewPauseIdentifier(md.ID),
		Subscribe:  &state.PauseSubscribe{Key: "inbox", Stream: 1},
	}
	evt := event.InternalEvent{ID: ulid.MustNew(ulid.Now(), nil), Event: event.Event{Name: name}}

	require.NoError(t, e.deliverSubscription(context.Background(), evt, pause, logger.VoidLogger()))
	require.Equal(t, []uuid.UUID{pause.ID}, pm.deleted)
	require.Empty(t, rs.saved)
}
