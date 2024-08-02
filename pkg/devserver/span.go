package devserver

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

type spanIngestionHandler struct {
	mu sync.Mutex

	dedup map[string]*cqrs.Span
	runs  map[string]*cqrs.TraceRun
	data  cqrs.Manager
}

func newSpanIngestionHandler(data cqrs.Manager) *spanIngestionHandler {
	handler := &spanIngestionHandler{
		dedup: map[string]*cqrs.Span{},
		runs:  map[string]*cqrs.TraceRun{},
		data:  data,
	}

	return handler
}

// Add adds the span and dedup it, taking the latest one needed
func (sh *spanIngestionHandler) Add(ctx context.Context, span *cqrs.Span) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// marked as delete, so don't ingest in the first place
	todelete := spanAttr(span.SpanAttributes, consts.OtelSysStepDelete)
	if len(todelete) > 0 {
		return
	}

	acctID := parseUUID(spanAttr(span.SpanAttributes, consts.OtelSysAccountID))
	wsID := parseUUID(spanAttr(span.SpanAttributes, consts.OtelSysWorkspaceID))
	appID := parseUUID(spanAttr(span.SpanAttributes, consts.OtelSysAppID))
	fnID := parseUUID(spanAttr(span.SpanAttributes, consts.OtelSysFunctionID))

	id := fmt.Sprintf("%s:%s:%s:%s:%s:%s", acctID, wsID, appID, fnID, span.TraceID, span.SpanID)
	h := sha1.New()
	_, _ = h.Write([]byte(id))
	key := base64.StdEncoding.EncodeToString(h.Sum(nil))

	{
		s, ok := sh.dedup[key]
		switch ok {
		case false:
			sh.dedup[key] = span
		case true:
			currentCode, _ := strconv.Atoi(spanAttr(s.SpanAttributes, consts.OtelSysFunctionStatusCode))
			newCode, _ := strconv.Atoi(spanAttr(span.SpanAttributes, consts.OtelSysFunctionStatusCode))

			// HACK:
			// if a function has no steps and the function finishes quickly,
			// there's a possibility of a race where the function start hook finishes
			// after the function end hook.
			// so check if the code is larger, use the larger one.
			// this should not be an issue on prod
			if newCode > currentCode {
				sh.dedup[key] = span
				break
			}

			if span.Duration > s.Duration {
				sh.dedup[key] = span
			}
		}
	}

	// TODO: find if there's already an entry in the DB and retrieve that instead
	if span.RunID != nil {
		// construct the run
		var run *cqrs.TraceRun
		if r, ok := sh.runs[span.RunID.String()]; ok {
			run = r
		} else {
			var err error
			run, err = sh.data.FindOrBuildTraceRun(ctx, cqrs.FindOrCreateTraceRunOpt{
				AccountID:   acctID,
				WorkspaceID: wsID,
				AppID:       appID,
				FunctionID:  fnID,
				TraceID:     span.TraceID,
				RunID:       *span.RunID,
			})
			if err != nil {
				return
			}
		}

		// construct triggerIDs
		if len(run.TriggerIDs) == 0 {
			evtIDs := spanAttr(span.SpanAttributes, consts.OtelSysEventIDs)
			if evtIDs != "" {
				run.TriggerIDs = strings.Split(evtIDs, ",")
			}
		}

		// assign output
		if run.Output == nil || len(run.Output) == 0 {
			for _, e := range span.Events {
				if spanAttr(e.Attributes, consts.OtelSysFunctionOutput) != "" {
					run.Output = []byte(e.Name)
				}
			}
		}

		// Update status
		status, _ := strconv.ParseInt(spanAttr(span.SpanAttributes, consts.OtelSysFunctionStatusCode), 10, 64)
		if status > run.Status.ToCode() {
			run.Status = enums.RunCodeToStatus(status)
		}

		// Update timestamps
		if span.ScopeName == consts.OtelScopeFunction {
			if span.Timestamp.UnixMilli() > run.StartedAt.UnixMilli() {
				run.StartedAt = span.Timestamp
			}

			if span.Duration > run.Duration {
				run.Duration = span.Duration
				run.EndedAt = run.StartedAt.Add(span.Duration)
			}
		}

		// Annotate if run is batch or debounce
		batchID := spanAttr(span.SpanAttributes, consts.OtelSysBatchID)
		if batchID != "" {
			if bid, err := ulid.Parse(batchID); err == nil {
				run.BatchID = &bid
			}
			run.IsBatch = true
		}
		if spanAttr(span.SpanAttributes, consts.OtelSysDebounceTimeout) != "" {
			run.IsDebounce = true
		}
		cron := spanAttr(span.SpanAttributes, consts.OtelSysCronExpr)
		if cron != "" {
			run.CronSchedule = &cron
		}

		// assign it back
		sh.runs[span.RunID.String()] = run
	}
}

func (sh *spanIngestionHandler) Spans() []*cqrs.Span {
	res := []*cqrs.Span{}
	for _, v := range sh.dedup {
		res = append(res, v)
	}
	return res
}

func (sh *spanIngestionHandler) TraceRuns() []*cqrs.TraceRun {
	res := []*cqrs.TraceRun{}
	for _, v := range sh.runs {
		res = append(res, v)
	}
	return res
}

func spanAttr(sattr map[string]string, key string) string {
	if v, ok := sattr[key]; ok {
		return v
	}
	return ""
}

func parseUUID(str string) uuid.UUID {
	if id, err := uuid.Parse(str); err == nil {
		return id
	}
	return uuid.UUID{}
}
