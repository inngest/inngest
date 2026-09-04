/**
 * What a run was caused by, and what it caused.
 *
 * `useGetRunLinkage` already walks `defer()` ancestry. Everything else a run is
 * connected to has been reachable all along; it was simply never assembled.
 *
 * WHAT WAS VERIFIED, AND WHAT WAS NOT. Two mechanisms were checked against a
 * live Dev Server before any of this was written, because the brief flagged both
 * as unconfirmed:
 *
 *  - **OTel span links are NOT usable.** `FollowsFrom` really is written, but
 *    only between pause and lifecycle spans *within* a run — never from a
 *    parent's invoke span to a child's root. And it would not reach the client
 *    anyway: `links` is absent from the GraphQL schema, where
 *    `gql.schema.graphql` still carries the comment `# links should be here`.
 *    Exposing it is a server change. So none of this walks span links.
 *
 *  - **`step.sendEvent` ids ARE internal ULIDs.** Confirmed by reading the
 *    output of a real emitting run: `{"ids":["01M1Q00QZAD5YFMY4TP9PH0YZ3", …]}`,
 *    and `GET /v2/events/01M1Q00QZAD5YFMY4TP9PH0YZ3/runs` returns the run those
 *    ids caused, whose own trigger echoes the same id. The loop closes.
 *
 * One correction to the brief while doing so: a `step.sendEvent` is **not**
 * `stepOp == SEND_EVENT`. It is reported as an ordinary `RUN`, and the thing
 * that identifies it is `stepType == 'step.sendEvent'` (mirrored in
 * `stepInfo.type`). Keying on `stepOp` finds nothing at all.
 *
 * WHERE THIS BREAKS, AND THE UI SAYS SO. A bare `inngest.send()` *outside* a
 * step has no span and no output, so nothing ties emitter to emitted — such
 * events are simply invisible here, and their absence is not evidence. Batch
 * fan-in is a join rather than an edge: several events produce one run, and
 * pointing an arrow from one of them would misstate it.
 */
import type { Trace } from './types';

/** A step that sent events. Its ids live in its output, not in the span. */
export type EmittingStep = {
  spanID: string;
  /** The step name the user wrote. */
  label: string;
  /** Needed to fetch the output the ids are inside. */
  outputID: string | null;
};

/** A run this one started directly, via `step.invoke`. */
export type InvokedRun = {
  spanID: string;
  label: string;
  runID: string;
};

/**
 * Steps that sent events, found without fetching anything.
 *
 * This is what makes the feature affordable: the alternative — before noticing
 * `stepType` — was fetching every step's output to see which ones happened to
 * contain an `ids` array, which is one request per step of the run.
 */
export function findEmittingSteps(root: Trace): EmittingStep[] {
  const found: EmittingStep[] = [];

  const walk = (span: Trace) => {
    if (span.stepType === 'step.sendEvent' && span.stepID) {
      found.push({
        spanID: span.spanID,
        label: span.userlandStepID ?? span.name,
        outputID: span.outputID ?? null,
      });
    }
    span.childrenSpans?.forEach(walk);
  };
  walk(root);

  return found;
}

/** Runs started by `step.invoke`, which unlike sent events are labelled. */
export function findInvokedRuns(root: Trace): InvokedRun[] {
  const found: InvokedRun[] = [];

  const walk = (span: Trace) => {
    const info = span.stepInfo as { runID?: string | null } | null | undefined;
    if (info?.runID) {
      found.push({
        spanID: span.spanID,
        label: span.userlandStepID ?? span.name,
        runID: info.runID,
      });
    }
    span.childrenSpans?.forEach(walk);
  };
  walk(root);

  return found;
}

/**
 * The event ids inside a `step.sendEvent`'s output.
 *
 * Returns null rather than an empty array when the payload is not the shape we
 * expect, so a caller can tell "sent nothing" from "could not tell" — the
 * distinction this whole file exists to preserve.
 */
export function parseEmittedEventIDs(output: string | null | undefined): string[] | null {
  if (!output) return null;

  try {
    const parsed: unknown = JSON.parse(output);
    if (!parsed || typeof parsed !== 'object') return null;

    const ids = (parsed as { ids?: unknown }).ids;
    if (!Array.isArray(ids)) return null;
    if (!ids.every((id): id is string => typeof id === 'string')) return null;

    return ids;
  } catch {
    return null;
  }
}
