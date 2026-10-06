package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

// SpanGroupStepType is the step type of a virtual span that groups the steps
// an SDK called inside a span group.
const SpanGroupStepType = "SPAN_GROUP"

// groupBySpanPath nests the run's step spans under virtual span groups, one
// per prefix of each step's span path, so the same path always lands in the
// same group. Steps without a path stay where they are, and a top-level group
// takes the place of its first step.
func groupBySpanPath(run *models.RunTraceSpan) {
	groups := map[string]*models.RunTraceSpan{}
	var created []*models.RunTraceSpan
	children := run.ChildrenSpans
	run.ChildrenSpans = nil

	// An attempt that failed before the SDK answered has no step ID or path,
	// only the group ID it shares with the later attempts of its step, so it
	// takes the path of the next step with that group ID.
	nextStepPath := map[string][]meta.SpanPathElement{}
	for _, child := range slices.Backward(children) {
		switch {
		case child.GroupID == nil:
		case child.StepID != nil && *child.StepID != "":
			nextStepPath[*child.GroupID] = child.SpanPath
		case child.SpanPath == nil:
			child.SpanPath = nextStepPath[*child.GroupID]
		}
	}

	for _, child := range children {
		parent := run
		for i := range child.SpanPath {
			id := spanGroupID(child.SpanPath[:i+1])
			group, ok := groups[id]
			if !ok {
				group = &models.RunTraceSpan{
					AppID:        run.AppID,
					FunctionID:   run.FunctionID,
					RunID:        run.RunID,
					TraceID:      run.TraceID,
					SpanID:       id,
					ParentSpanID: &parent.SpanID,
					Name:         child.SpanPath[i].Name,
					StepType:     SpanGroupStepType,
				}
				if kind := child.SpanPath[i].Kind; kind != "" {
					group.SpanKind = &kind
				}
				groups[id] = group
				created = append(created, group)
				parent.ChildrenSpans = append(parent.ChildrenSpans, group)
			}
			parent = group
		}

		if parent != run {
			child.ParentSpanID = &parent.SpanID
		}
		parent.ChildrenSpans = append(parent.ChildrenSpans, child)
	}

	// Groups are created before their subgroups, so finishing them in reverse
	// sees every subgroup's final timing first.
	for _, group := range slices.Backward(created) {
		finishSpanGroup(group)
	}
}

// finishSpanGroup orders a group's children and derives its timing and status
// from them: it runs from its first child's start to its last child's end, and
// takes the status of the child that ended last.
func finishSpanGroup(group *models.RunTraceSpan) {
	slices.SortStableFunc(group.ChildrenSpans, func(a, b *models.RunTraceSpan) int {
		return a.QueuedAt.Compare(b.QueuedAt)
	})

	group.QueuedAt = group.ChildrenSpans[0].QueuedAt

	running := false
	var last *models.RunTraceSpan
	for _, child := range group.ChildrenSpans {
		if child.StartedAt != nil && (group.StartedAt == nil || child.StartedAt.Before(*group.StartedAt)) {
			group.StartedAt = child.StartedAt
		}

		if !models.RunTraceEnded(child.Status) {
			running = true
			continue
		}

		if last == nil || (child.EndedAt != nil && (last.EndedAt == nil || child.EndedAt.After(*last.EndedAt))) {
			last = child
		}
	}

	if running {
		group.Status = models.RunTraceSpanStatusRunning
		return
	}

	group.Status = last.Status
	group.EndedAt = last.EndedAt
	if group.StartedAt != nil && group.EndedAt != nil {
		dur := int(group.EndedAt.Sub(*group.StartedAt).Milliseconds())
		group.Duration = &dur
	}
}

// spanGroupID is a stable span ID for the group at the end of path.
func spanGroupID(path []meta.SpanPathElement) string {
	h := sha256.New()
	for _, el := range path {
		h.Write([]byte(el.ID))
		h.Write([]byte{0})
	}
	return "span:" + hex.EncodeToString(h.Sum(nil))[:16]
}
