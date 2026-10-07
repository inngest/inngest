package loader

import (
	"fmt"
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
		for i, el := range child.SpanPath {
			id := fmt.Sprintf("span:%q", pathIDs(child.SpanPath[:i+1]))
			group, ok := groups[id]
			if !ok {
				group = &models.RunTraceSpan{
					AppID:        run.AppID,
					FunctionID:   run.FunctionID,
					RunID:        run.RunID,
					TraceID:      run.TraceID,
					SpanID:       id,
					ParentSpanID: &parent.SpanID,
					Name:         el.Name,
					StepType:     SpanGroupStepType,
				}
				if el.Kind != "" {
					group.GroupKind = &el.Kind
				}
				if el.Origin != "" {
					group.Origin = &el.Origin
				}
				groups[id] = group
				parent.ChildrenSpans = append(parent.ChildrenSpans, group)
			}
			parent = group
		}

		if parent != run {
			child.ParentSpanID = &parent.SpanID
		}
		parent.ChildrenSpans = append(parent.ChildrenSpans, child)
	}

	for _, child := range run.ChildrenSpans {
		if child.StepType == SpanGroupStepType {
			finishSpanGroup(child)
		}
	}
}

func pathIDs(path []meta.SpanPathElement) []string {
	ids := make([]string, len(path))
	for i, el := range path {
		ids[i] = el.ID
	}
	return ids
}

// finishSpanGroup orders a group's children, subgroups first finished, and
// derives its timing and status from them: it runs from its first child's
// start to its last child's end, and takes the status of the child that ended
// last, or is running while any child is.
func finishSpanGroup(group *models.RunTraceSpan) {
	for _, child := range group.ChildrenSpans {
		if child.StepType == SpanGroupStepType {
			finishSpanGroup(child)
		}
	}

	slices.SortStableFunc(group.ChildrenSpans, func(a, b *models.RunTraceSpan) int {
		return a.QueuedAt.Compare(b.QueuedAt)
	})

	first := group.ChildrenSpans[0]
	group.QueuedAt = first.QueuedAt
	group.StartedAt = first.StartedAt

	var last *models.RunTraceSpan
	for _, child := range group.ChildrenSpans {
		if !models.RunTraceEnded(child.Status) {
			group.Status = models.RunTraceSpanStatusRunning
			return
		}

		if last == nil || (child.EndedAt != nil && (last.EndedAt == nil || child.EndedAt.After(*last.EndedAt))) {
			last = child
		}
	}

	group.Status = last.Status
	group.EndedAt = last.EndedAt
	if group.StartedAt != nil && group.EndedAt != nil {
		dur := int(group.EndedAt.Sub(*group.StartedAt).Milliseconds())
		group.Duration = &dur
	}
}
