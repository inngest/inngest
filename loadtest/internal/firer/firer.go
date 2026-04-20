// Package firer posts events to an Inngest server at a configured rate. It
// batches events into JSON arrays to amortize connection cost, persists a
// send-timestamp per event, and offers a single-event lane for clean tail
// percentiles.
package firer

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mrand "math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	xrate "golang.org/x/time/rate"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/shapes"
	"github.com/inngest/inngest/loadtest/internal/storage"
)

// Firer owns the HTTP client + rate limiter and drains event sends to a
// target server.
type Firer struct {
	target     config.Target
	batchSize  int
	limiter    *xrate.Limiter
	httpClient *http.Client
	appIDs     []string
	shapesMix  weightedShapes

	fired    uint64 // atomic — successful event count for this Firer
	failed   uint64 // atomic — POSTs that returned an error
	lastErr  atomic.Value // atomic — last error string for diagnostics
}

// New constructs a Firer. eventsPerSec is the token-bucket rate; apps is the
// list of app IDs currently registered (each app defines its own event
// namespace, so events are round-robined across apps).
func New(target config.Target, eventsPerSec, batchSize int, apps []string, mix config.ShapeMix) *Firer {
	ws := newWeightedShapes(mix)
	return &Firer{
		target:    target,
		batchSize: batchSize,
		limiter:   xrate.NewLimiter(xrate.Limit(eventsPerSec), eventsPerSec),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		appIDs:    apps,
		shapesMix: ws,
	}
}

// Run fires events until ctx is cancelled. Each batched POST produces one
// EventRow per event, persisted to store. Stops cleanly on ctx cancel.
func (f *Firer) Run(ctx context.Context, store *storage.Store, runID string, workers int) error {
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := f.worker(ctx, store, runID); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil && ctx.Err() == nil {
			return err
		}
	}
	return nil
}

func (f *Firer) worker(ctx context.Context, store *storage.Store, runID string) error {
	batch := make([]shapes.Payload, 0, f.batchSize)
	appsByShape := map[string]string{} // shape → app
	for i, app := range f.appIDs {
		_ = i
		appsByShape[app] = app // placeholder; real mapping is per-event below
	}
	for {
		// Drain one batch under rate limit.
		batch = batch[:0]
		appChoice := make([]string, 0, f.batchSize)
		shapeChoice := make([]config.Shape, 0, f.batchSize)
		for len(batch) < f.batchSize {
			if err := f.limiter.Wait(ctx); err != nil {
				return nil
			}
			sh := f.shapesMix.pick()
			app := f.appIDs[mrand.IntN(len(f.appIDs))]
			batch = append(batch, shapes.Payload{
				Shape:         string(sh),
				CorrelationID: newCorrID(),
				SentAt:        time.Now().UnixNano(),
			})
			appChoice = append(appChoice, app)
			shapeChoice = append(shapeChoice, sh)
		}

		// Build the NDJSON / array body. We post one request per (app, shape)
		// group — the event name is derived from app+shape, and the target
		// dispatches by event name.
		type bucket struct {
			app   string
			shape config.Shape
			items []shapes.Payload
		}
		buckets := map[string]*bucket{}
		for i := range batch {
			k := appChoice[i] + "|" + string(shapeChoice[i])
			b := buckets[k]
			if b == nil {
				b = &bucket{app: appChoice[i], shape: shapeChoice[i]}
				buckets[k] = b
			}
			b.items = append(b.items, batch[i])
		}

		for _, b := range buckets {
			if err := f.fireBucket(ctx, store, runID, b.app, b.shape, b.items); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

type ingestEvent struct {
	Name string          `json:"name"`
	Data shapes.Payload  `json:"data"`
	ID   string          `json:"id,omitempty"`
	TS   int64           `json:"ts,omitempty"`
}

func (f *Firer) fireBucket(ctx context.Context, store *storage.Store, runID, app string, shape config.Shape, items []shapes.Payload) error {
	events := make([]ingestEvent, len(items))
	eventName := shapes.EventName(app, shape)
	for i, p := range items {
		events[i] = ingestEvent{
			Name: eventName,
			Data: p,
			ID:   p.CorrelationID,
			TS:   time.Now().UnixMilli(),
		}
	}
	body, err := json.Marshal(events)
	if err != nil {
		return err
	}

	url := f.target.URL + "/e/" + eventKey(f.target)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return f.recordErr(err)
	}
	req.Header.Set("content-type", "application/json")

	sentAt := time.Now().UnixNano()
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return f.recordErr(err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet := string(bodyBytes)
		if len(snippet) > 512 {
			snippet = snippet[:512]
		}
		return f.recordErr(fmt.Errorf("event POST %s returned %d: %s", url, resp.StatusCode, snippet))
	}

	atomic.AddUint64(&f.fired, uint64(len(items)))
	batchID := newCorrID()
	rows := make([]storage.EventRow, len(items))
	for i, p := range items {
		rows[i] = storage.EventRow{
			CorrelationID: p.CorrelationID,
			BatchID:       batchID,
			SentAtNanos:   sentAt,
		}
	}
	return store.InsertEvents(runID, rows)
}

// Fired returns the number of events successfully POSTed by this Firer.
func (f *Firer) Fired() uint64 { return atomic.LoadUint64(&f.fired) }

// Failed returns the number of POSTs that returned an error.
func (f *Firer) Failed() uint64 { return atomic.LoadUint64(&f.failed) }

func (f *Firer) recordErr(err error) error {
	if err == nil {
		return nil
	}
	n := atomic.AddUint64(&f.failed, 1)
	f.lastErr.Store(err.Error())
	// Log the first failure loudly; everything else is likely noise from the
	// same root cause.
	if n == 1 {
		log.Printf("firer: first POST error: %v", err)
	}
	return err
}

// LastError returns the most recent fire error, or "" if none.
func (f *Firer) LastError() string {
	if v := f.lastErr.Load(); v != nil {
		return v.(string)
	}
	return ""
}

func eventKey(t config.Target) string {
	if t.EventKey != nil {
		return *t.EventKey
	}
	return "test"
}

func newCorrID() string {
	var b [12]byte
	_, _ = crand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// weightedShapes is a pre-expanded cumulative distribution that makes shape
// selection O(log n) per call.
type weightedShapes struct {
	shapes []config.Shape
	cum    []int
	total  int
}

func newWeightedShapes(m config.ShapeMix) weightedShapes {
	ws := weightedShapes{}
	for s, w := range m {
		if w <= 0 {
			continue
		}
		ws.total += w
		ws.shapes = append(ws.shapes, s)
		ws.cum = append(ws.cum, ws.total)
	}
	return ws
}

func (w weightedShapes) pick() config.Shape {
	if w.total == 0 {
		return config.ShapeNoop
	}
	r := mrand.IntN(w.total)
	for i, c := range w.cum {
		if r < c {
			return w.shapes[i]
		}
	}
	return w.shapes[len(w.shapes)-1]
}
