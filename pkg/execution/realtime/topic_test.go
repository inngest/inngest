package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

var errReceiveInterrupted = errors.New("injected terminal Receive failure")

// Each Receive still uses the real Redis protocol and subscription hook, but
// tests can force it to return an error after confirmation. Dropping a socket
// alone only exercises Rueidis's own retries, not broadcaster recovery.
type controlledReceive struct {
	start chan struct{}
	fail  chan error
	done  chan struct{}
}

type controlledRedis struct {
	rueidis.Client
	calls chan *controlledReceive
	// Optional barrier for testing old-worker cleanup versus replacement.
	unsubscribe func(context.Context)
}

func (c *controlledRedis) Receive(ctx context.Context, cmd rueidis.Completed, fn func(rueidis.PubSubMessage)) error {
	call := &controlledReceive{start: make(chan struct{}), fail: make(chan error, 1), done: make(chan struct{})}
	defer close(call.done)
	select {
	case c.calls <- call:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-call.start:
	case err := <-call.fail:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- c.Client.Receive(ctx, cmd, fn) }()
	select {
	case err := <-call.fail:
		cancel()
		<-result
		return err
	case err := <-result:
		return err
	}
}

func (c *controlledRedis) Do(ctx context.Context, cmd rueidis.Completed) rueidis.RedisResult {
	if cmd.Commands()[0] == "UNSUBSCRIBE" && c.unsubscribe != nil {
		c.unsubscribe(ctx)
	}
	return c.Client.Do(ctx, cmd)
}

func newRecoveryBroadcaster(t *testing.T, timeout time.Duration) (*broadcaster, *controlledRedis) {
	t.Helper()
	r := miniredis.RunT(t)
	pub, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	sub, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	client := &controlledRedis{Client: sub, calls: make(chan *controlledReceive, 32)}
	b := NewRedisBroadcaster(pub, client, BroadcasterOpts{
		ShutdownGracePeriod: time.Millisecond, SubscriptionRecoveryTimeout: timeout,
	}).(*broadcaster)
	t.Cleanup(func() {
		_ = b.Close(context.Background())
		pub.Close()
		sub.Close()
	})
	return b, client
}

func receiveCall(t *testing.T, c *controlledRedis) *controlledReceive {
	t.Helper()
	select {
	case call := <-c.calls:
		return call
	case <-time.After(3 * time.Second):
		t.Fatal("Redis topic worker did not call Receive")
		return nil
	}
}

func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("operation did not finish")
	}
}

func subscribeAsync(b Broadcaster, ctx context.Context, sub Subscription, topics ...Topic) <-chan error {
	result := make(chan error, 1)
	go func() { result <- b.Subscribe(ctx, sub, topics) }()
	return result
}

func awaitSubscribe(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("Subscribe did not return")
		return nil
	}
}

func TestTopicRecoveryResumesDelivery(t *testing.T) {
	for _, kind := range []string{"structured", "raw", "chunk"} {
		t.Run(kind, func(t *testing.T) {
			b, client := newRecoveryBroadcaster(t, 500*time.Millisecond)
			topic := Topic{Kind: streamingtypes.TopicKindRun, EnvID: uuid.New(), Channel: "recovery", Name: "$stream"}
			data := make(chan string, 64)
			sub := NewInmemorySubscription(uuid.New(), func(bytes []byte) error { data <- string(bytes); return nil })
			result := subscribeAsync(b, context.Background(), sub, topic)
			initial := receiveCall(t, client)
			close(initial.start)
			require.NoError(t, awaitSubscribe(t, result))
			publish := func(payload string) {
				msg := Message{Kind: streamingtypes.MessageKindData, EnvID: topic.EnvID, Channel: topic.Channel, Topic: topic.Name, Data: json.RawMessage(`"` + payload + `"`)}
				switch kind {
				case "structured":
					b.Publish(context.Background(), msg)
				case "raw":
					b.Write(context.Background(), topic.EnvID, topic.Channel, []byte(payload))
				case "chunk":
					b.PublishChunk(context.Background(), msg, Chunk{Kind: "chunk", StreamID: "s", Data: payload})
				}
			}
			publish("before")
			select {
			case bytes := <-data:
				require.Contains(t, bytes, "before")
			case <-time.After(time.Second):
				t.Fatal("initial publish was not delivered")
			}
			// Repeat recovery to verify a successful confirmation disarms the
			// outage timeout without disabling recovery of a later failure.
			for i := 0; i < 2; i++ {
				initial.fail <- errReceiveInterrupted
				replacement := receiveCall(t, client)
				if i == 0 {
					// A failed recovery attempt before acknowledgement must retry
					// within the same budget, not leave the topic dead again.
					replacement.fail <- errReceiveInterrupted
					replacement = receiveCall(t, client)
				}
				close(replacement.start)
				require.Eventually(t, func() bool {
					publish("probe")
					select {
					case bytes := <-data:
						return strings.Contains(bytes, "probe")
					default:
						return false
					}
				}, time.Second, 10*time.Millisecond)
				// Once acknowledged, this connection must outlive the recovery
				// budget rather than getting cancelled by a leftover timer.
				select {
				case <-replacement.done:
					t.Fatal("healthy subscription ended after recovery")
				case <-time.After(600 * time.Millisecond):
				}
				initial = replacement
			}
			publish("after")
			deadline := time.After(time.Second)
			for {
				select {
				case bytes := <-data:
					if strings.Contains(bytes, "after") {
						require.Equal(t, 1, subCount(b))
						return
					}
				case <-deadline:
					t.Fatal("publish after recovery was not delivered")
				}
			}
		})
	}
}

func TestTopicConcurrentStartupFailure(t *testing.T) {
	b, client := newRecoveryBroadcaster(t, time.Second)
	topic := Topic{Name: "startup"}
	first := subscribeAsync(b, context.Background(), NewInmemorySubscription(uuid.New(), nil), topic)
	call := receiveCall(t, client)
	second := subscribeAsync(b, context.Background(), NewInmemorySubscription(uuid.New(), nil), topic)
	require.Eventually(t, func() bool { return subCount(b) == 2 }, time.Second, time.Millisecond)
	select {
	case err := <-second:
		t.Fatalf("second subscriber returned before Redis confirmation: %v", err)
	default:
	}
	call.fail <- errReceiveInterrupted
	require.ErrorIs(t, awaitSubscribe(t, first), errReceiveInterrupted)
	require.ErrorIs(t, awaitSubscribe(t, second), errReceiveInterrupted)
	require.Zero(t, subCount(b))
	select {
	case <-client.calls:
		t.Fatal("initial failure must not retry")
	default:
	}
}

func TestTopicRecoveryCancellation(t *testing.T) {
	for _, stop := range []string{"unsubscribe", "shutdown"} {
		t.Run(stop, func(t *testing.T) {
			b, client := newRecoveryBroadcaster(t, time.Second)
			sub := NewInmemorySubscription(uuid.New(), nil)
			topic := Topic{Name: "cancel"}
			result := subscribeAsync(b, context.Background(), sub, topic)
			first := receiveCall(t, client)
			close(first.start)
			require.NoError(t, awaitSubscribe(t, result))
			first.fail <- errReceiveInterrupted
			retrying := receiveCall(t, client) // Hold before Redis confirmation.
			b.l.RLock()
			worker := b.topics[topic.String()].worker
			b.l.RUnlock()
			if stop == "unsubscribe" {
				require.NoError(t, b.Unsubscribe(context.Background(), sub.ID(), []Topic{topic}))
			} else {
				require.NoError(t, b.Close(context.Background()))
			}
			awaitSignal(t, retrying.done)
			awaitSignal(t, worker.done)
			b.l.RLock()
			require.Empty(t, b.topicWorkers)
			b.l.RUnlock()
		})
	}
}

func TestTopicReplacementWaitsForCleanup(t *testing.T) {
	b, client := newRecoveryBroadcaster(t, time.Second)
	cleanupStarted, releaseCleanup := make(chan struct{}), make(chan struct{})
	var once sync.Once
	client.unsubscribe = func(ctx context.Context) {
		once.Do(func() {
			close(cleanupStarted)
			select {
			case <-releaseCleanup:
			case <-ctx.Done():
			}
		})
	}
	topic := Topic{Name: "replacement"}
	firstSub := NewInmemorySubscription(uuid.New(), nil)
	result := subscribeAsync(b, context.Background(), firstSub, topic)
	close(receiveCall(t, client).start)
	require.NoError(t, awaitSubscribe(t, result))
	require.NoError(t, b.CloseSubscription(context.Background(), firstSub.ID()))
	awaitSignal(t, cleanupStarted)
	ctx, cancel := context.WithCancel(context.Background())
	second := subscribeAsync(b, ctx, NewInmemorySubscription(uuid.New(), nil), topic)
	require.Eventually(t, func() bool { return subCount(b) == 1 }, time.Second, time.Millisecond)
	cancel()
	require.Error(t, awaitSubscribe(t, second))
	// Cancelling a replacement while its predecessor is cleaning up must
	// preserve the barrier for the next generation as well.
	third := subscribeAsync(b, context.Background(), NewInmemorySubscription(uuid.New(), nil), topic)
	require.Eventually(t, func() bool { return subCount(b) == 1 }, time.Second, time.Millisecond)
	select {
	case <-client.calls:
		t.Fatal("replacement subscribed before predecessor's UNSUBSCRIBE completed")
	default:
	}
	close(releaseCleanup)
	close(receiveCall(t, client).start)
	require.NoError(t, awaitSubscribe(t, third))
}

// Closing a failed client must not detach a healthy client on another topic.
type closingSubscription struct {
	Subscription
	closed chan struct{}
	once   sync.Once
}

func (s *closingSubscription) Close() error {
	s.once.Do(func() { close(s.closed) })
	return nil
}

func TestTopicRecoveryExhaustionIsolated(t *testing.T) {
	b, client := newRecoveryBroadcaster(t, 150*time.Millisecond)
	bad := &closingSubscription{Subscription: NewInmemorySubscription(uuid.New(), nil), closed: make(chan struct{})}
	goodData := make(chan string, 1)
	good := &closingSubscription{Subscription: NewInmemorySubscription(uuid.New(), func(data []byte) error { goodData <- string(data); return nil }), closed: make(chan struct{})}
	topic := Topic{Name: "bad"}
	result := subscribeAsync(b, context.Background(), bad, topic)
	first := receiveCall(t, client)
	close(first.start)
	require.NoError(t, awaitSubscribe(t, result))
	goodTopic := Topic{Kind: streamingtypes.TopicKindRun, Name: "good"}
	result = subscribeAsync(b, context.Background(), good, goodTopic)
	close(receiveCall(t, client).start)
	require.NoError(t, awaitSubscribe(t, result))
	first.fail <- errReceiveInterrupted
	retrying := receiveCall(t, client) // Never confirm: timeout must interrupt Receive.
	awaitSignal(t, retrying.done)
	awaitSignal(t, bad.closed)
	require.Equal(t, 1, subCount(b))
	select {
	case <-good.closed:
		t.Fatal("unrelated subscription was closed")
	default:
	}
	b.Publish(context.Background(), Message{Kind: streamingtypes.MessageKindData, Topic: goodTopic.Name, Data: json.RawMessage(`"healthy"`)})
	select {
	case data := <-goodData:
		require.Contains(t, data, "healthy")
	case <-time.After(time.Second):
		t.Fatal("unrelated topic stopped delivering")
	}
}

func TestTopicRecoveryExhaustionDisconnectsClients(t *testing.T) {
	for _, protocol := range []string{"ws", "sse-http1", "sse-http2"} {
		t.Run(protocol, func(t *testing.T) {
			b, client := newRecoveryBroadcaster(t, 150*time.Millisecond)
			topic := Topic{Kind: streamingtypes.TopicKindRun, Name: "disconnect", EnvID: uuid.New()}
			server := httptest.NewUnstartedServer(NewAPI(APIOpts{JWTSecret: []byte("secret"), Broadcaster: b}))
			server.EnableHTTP2 = protocol == "sse-http2"
			server.StartTLS()
			defer server.Close()
			token, err := NewJWT(context.Background(), []byte("secret"), uuid.New(), topic.EnvID, []Topic{topic})
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			ended := make(chan struct{})
			connected := make(chan struct{})
			var clientErr error
			go func() {
				defer close(ended)
				if protocol == "ws" {
					conn, _, err := websocket.Dial(ctx, "wss"+strings.TrimPrefix(server.URL, "https")+"/realtime/connect", &websocket.DialOptions{
						HTTPClient: server.Client(), HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
					})
					if err != nil {
						clientErr = err
						return
					}
					defer conn.CloseNow()
					close(connected)
					_, _, clientErr = conn.Read(ctx)
					return
				}
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/realtime/sse", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				resp, err := server.Client().Do(req)
				if err != nil {
					clientErr = err
					return
				}
				defer resp.Body.Close()
				if protocol == "sse-http2" && resp.ProtoMajor != 2 {
					clientErr = errors.New("HTTP/2 was not negotiated")
					return
				}
				close(connected)
				_, clientErr = io.Copy(io.Discard, resp.Body)
			}()
			first := receiveCall(t, client)
			close(first.start)
			awaitSignal(t, connected)
			// WebSocket upgrades precede Redis confirmation. Wait for the
			// subscription hook, not just the successful HTTP upgrade.
			require.Eventually(t, func() bool {
				b.l.RLock()
				defer b.l.RUnlock()
				for _, worker := range b.topicWorkers {
					if worker.initialized.Load() {
						return true
					}
				}
				return false
			}, time.Second, time.Millisecond)
			first.fail <- errReceiveInterrupted
			receiveCall(t, client) // Leave recovery unconfirmed.
			awaitSignal(t, ended)
			require.NoError(t, ctx.Err(), "server must disconnect before the client's deadline")
			if protocol == "ws" {
				require.Equal(t, websocket.StatusGoingAway, websocket.CloseStatus(clientErr))
			} else {
				require.NoError(t, clientErr)
			}
			require.Zero(t, subCount(b))
		})
	}
}

func TestTopicShutdownBeforeConfirmation(t *testing.T) {
	b, client := newRecoveryBroadcaster(t, time.Second)
	server := httptest.NewServer(NewAPI(APIOpts{JWTSecret: []byte("secret"), Broadcaster: b}))
	defer server.Close()
	topic := Topic{Kind: streamingtypes.TopicKindRun, Name: "pending", EnvID: uuid.New()}
	token, err := NewJWT(context.Background(), []byte("secret"), uuid.New(), topic.EnvID, []Topic{topic})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/realtime/sse", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := server.Client().Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()
	first := receiveCall(t, client) // Leave initial Receive waiting for confirmation.
	closed := make(chan struct{})
	go func() { _ = b.Close(context.Background()); close(closed) }()
	awaitSignal(t, closed)
	awaitSignal(t, first.done)
	awaitSignal(t, requestDone)
	require.NoError(t, ctx.Err())
}

func TestTopicFailedCleanupIsBounded(t *testing.T) {
	b, client := newRecoveryBroadcaster(t, time.Second)
	cleanupStarted := make(chan struct{})
	var once sync.Once
	client.unsubscribe = func(ctx context.Context) {
		once.Do(func() { close(cleanupStarted); <-ctx.Done() })
	}
	topic := Topic{Name: "cleanup-timeout"}
	sub := NewInmemorySubscription(uuid.New(), nil)
	result := subscribeAsync(b, context.Background(), sub, topic)
	close(receiveCall(t, client).start)
	require.NoError(t, awaitSubscribe(t, result))
	require.NoError(t, b.CloseSubscription(context.Background(), sub.ID()))
	awaitSignal(t, cleanupStarted)
	result = subscribeAsync(b, context.Background(), NewInmemorySubscription(uuid.New(), nil), topic)
	close(receiveCall(t, client).start)
	require.NoError(t, awaitSubscribe(t, result))
}

type blockedCloseSubscription struct {
	Subscription
	started chan struct{}
	release chan struct{}
}

func (s *blockedCloseSubscription) Close() error {
	close(s.started)
	<-s.release
	return nil
}

func TestTopicRecoveryClosesClientsIndependently(t *testing.T) {
	b, client := newRecoveryBroadcaster(t, 150*time.Millisecond)
	blocked := &blockedCloseSubscription{
		Subscription: NewInmemorySubscription(uuid.New(), nil),
		started:      make(chan struct{}), release: make(chan struct{}),
	}
	defer close(blocked.release)
	other := &closingSubscription{Subscription: NewInmemorySubscription(uuid.New(), nil), closed: make(chan struct{})}
	topic := Topic{Name: "blocked-close"}
	result := subscribeAsync(b, context.Background(), blocked, topic)
	first := receiveCall(t, client)
	close(first.start)
	require.NoError(t, awaitSubscribe(t, result))
	require.NoError(t, b.Subscribe(context.Background(), other, []Topic{topic}))
	first.fail <- errReceiveInterrupted
	receiveCall(t, client) // Hold recovery until its deadline.
	awaitSignal(t, blocked.started)
	awaitSignal(t, other.closed)
	require.Zero(t, subCount(b))
}

func TestTopicInitialFailureReturnsHTTPError(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		name := "HTTP1"
		if http2 {
			name = "HTTP2"
		}
		t.Run(name, func(t *testing.T) {
			b, client := newRecoveryBroadcaster(t, time.Second)
			server := httptest.NewUnstartedServer(NewAPI(APIOpts{JWTSecret: []byte("secret"), Broadcaster: b}))
			server.EnableHTTP2 = http2
			server.StartTLS()
			defer server.Close()
			token, err := NewJWT(context.Background(), []byte("secret"), uuid.New(), uuid.New(), []Topic{{Name: "startup-failure"}})
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			type response struct {
				res *http.Response
				err error
			}
			result := make(chan response, 1)
			go func() {
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/realtime/sse", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				res, err := server.Client().Do(req)
				result <- response{res, err}
			}()
			receiveCall(t, client).fail <- errReceiveInterrupted
			select {
			case got := <-result:
				require.NoError(t, got.err)
				defer got.res.Body.Close()
				require.Equal(t, http.StatusInternalServerError, got.res.StatusCode)
				if http2 {
					require.Equal(t, 2, got.res.ProtoMajor)
				}
			case <-ctx.Done():
				t.Fatal("startup failure did not return an HTTP error")
			}
			require.Zero(t, subCount(b))
		})
	}
}
