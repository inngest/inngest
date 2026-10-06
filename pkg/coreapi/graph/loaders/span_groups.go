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
	var top []*models.RunTraceSpan

	for _, child := range run.ChildrenSpans {
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
				groups[id] = group
				created = append(created, group)

				if parent == run {
					top = append(top, group)
				} else {
					parent.ChildrenSpans = append(parent.ChildrenSpans, group)
				}
			}
			parent = group
		}

		if parent == run {
			top = append(top, child)
		} else {
			child.ParentSpanID = &parent.SpanID
			parent.ChildrenSpans = append(parent.ChildrenSpans, child)
		}
	}

	// Groups are created before their subgroups, so finishing them in reverse
	// sees every subgroup's final timing first.
	for _, group := range slices.Backward(created) {
		finishSpanGroup(group)
	}

	run.ChildrenSpans = top
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
