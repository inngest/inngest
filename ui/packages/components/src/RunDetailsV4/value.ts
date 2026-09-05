/**
 * What Inngest did that the user would otherwise have built or lost.
 *
 * This page is the best surface in the product for showing that, and it
 * currently shows none of it. Two failed attempts followed by a success is a
 * saved order. Seven days asleep is a cron, a queue and a scheduler the user
 * did not write. Being held at a concurrency limit is a rate limiter they did
 * not write either.
 *
 * DISCIPLINE, and it is the whole design of this file:
 *
 *  - **Factual and specific, never celebratory.** "Recovered after 2 failed
 *    attempts", never "🎉". A number, or nothing.
 *  - **Only what is actually derivable.** Anything that would need a guess is
 *    not reported at all. There is no partial credit and no rounding up: a
 *    claim the trace cannot support is worse than silence, because the one
 *    thing this line has to be is trustworthy.
 *  - **Quantified where the number is real**, and omitted where it is not.
 *
 * DELIBERATELY NOT REPORTED, because this trace cannot support them:
 *
 *  - *Memoized steps on resume.* The most under-sold thing the platform does —
 *    "14 steps served from memory, 0 re-run" — and there is nothing in the
 *    payload that distinguishes a step served from state from one that ran.
 *    Every step has spans and timings either way. It needs a signal the SDK or
 *    executor does not currently emit.
 *  - *Survived a deploy.* Nothing in a run's trace records the deploy that
 *    happened during it.
 *  - *Deduplicated / debounced N events into 1.* The run that was kept has no
 *    record of the ones that were not.
 */
import type { Trace } from './types';

export type RunValue = {
  /** Steps that failed at least once and then succeeded. */
  recoveredSteps: number;
  /** Failed attempts across those steps — the work that was retried for free. */
  recoveredAttempts: number;
  /** Time suspended in sleeps and waits, holding nothing and costing nothing. */
  suspendedMs: number;
  /** Time the run spent held by flow control before it could start. */
  heldMs: number;
  /** Events this run was triggered by, when it was a batch. */
  batchedFrom: number;
  /** Events this run sent onward from inside a step. */
  emittedSteps: number;
  /**
   * Time actually spent executing on the user's own app.
   *
   * The sum of what each step ran for, and nothing else: not the queueing, not
   * the discovery requests, not Inngest's latency, and emphatically not the
   * sleeps and waits. This is the number a durable-execution platform exists to
   * keep small, and the run's own wall-clock says nothing about it — a run can
   * be seven days long and cost four seconds of compute.
   *
   * Summed rather than measured as elapsed, deliberately. Two steps running in
   * parallel for 100ms each consume 200ms of the user's compute even though the
   * run only spent 100ms doing it, and consumption is the question.
   */
  computeMs: number;
  /** The run's wall-clock, so the two can be read against each other. */
  elapsedMs: number;
};

/** A single factual sentence per thing that is true of this run. */
export function valueLines(value: RunValue): string[] {
  const lines: string[] = [];

  if (value.recoveredSteps > 0) {
    lines.push(
      value.recoveredSteps === 1
        ? `Recovered after ${plural(value.recoveredAttempts, 'failed attempt')}`
        : `Recovered ${plural(value.recoveredSteps, 'step')} after ${plural(
            value.recoveredAttempts,
            'failed attempt'
          )}`
    );
  }

  if (value.heldMs > 0) {
    lines.push(`Held ${duration(value.heldMs)} by flow control, then run`);
  }

  if (value.suspendedMs > 0) {
    lines.push(`Suspended ${duration(value.suspendedMs)} holding nothing`);
  }

  if (value.batchedFrom > 1) {
    lines.push(`Batched ${plural(value.batchedFrom, 'event')} into one run`);
  }

  // Only where the two numbers differ enough to be worth the comparison. On a
  // run that was almost entirely execution this says nothing, and a line that
  // says nothing is worse than no line.
  if (value.computeMs > 0 && value.elapsedMs > value.computeMs * 1.5) {
    lines.push(
      `${duration(value.computeMs)} of compute across ${duration(value.elapsedMs)} elapsed`
    );
  }

  return lines;
}

function plural(n: number, noun: string): string {
  return `${n} ${noun}${n === 1 ? '' : 's'}`;
}

/** Coarser than the timeline's, because a value line is prose, not a reading. */
function duration(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3_600_000) return `${Math.round(ms / 60_000)}m`;
  if (ms < 86_400_000) return `${Math.round(ms / 3_600_000)}h`;
  return `${Math.round(ms / 86_400_000)}d`;
}

const WAIT_OPS = new Set(['SLEEP', 'WAIT_FOR_EVENT', 'WAIT_FOR_SIGNAL']);

/**
 * Takes the ROLLED-UP trace, and counts only its direct children.
 *
 * Both halves of that matter. The raw payload lists a retried step three times —
 * once as an attempt group with `Attempt 0` / `Attempt 1` beneath it, and twice
 * more flat — so walking it recursively reports one recovered step as three.
 * `traceRollup` is what collapses those into the one step that happened, and
 * its direct children are exactly the steps of the run.
 */
export function runValue(rolledUp: Trace, batchSize = 1): RunValue {
  let recoveredSteps = 0;
  let recoveredAttempts = 0;
  let suspendedMs = 0;
  let emittedSteps = 0;
  let computeMs = 0;

  for (const span of rolledUp.childrenSpans ?? []) {
    // A retry only counts as a recovery when it actually recovered. A step that
    // failed twice and stayed failed is not a saved order, and calling it one
    // would be the exact kind of overclaiming this file exists to avoid.
    if ((span.attempts ?? 0) > 0 && span.status === 'COMPLETED' && span.stepID) {
      recoveredSteps += 1;
      recoveredAttempts += span.attempts ?? 0;
    }

    if (span.stepOp && WAIT_OPS.has(span.stepOp) && span.startedAt && span.endedAt) {
      suspendedMs += Math.max(0, Date.parse(span.endedAt) - Date.parse(span.startedAt));
    }

    // What the user's app actually ran for. A wait or a sleep is the run
    // suspended, holding nothing — counting it as compute would invert the
    // meaning of the line entirely.
    const isWait = span.stepOp ? WAIT_OPS.has(span.stepOp) : false;
    if (!isWait && span.startedAt && span.endedAt) {
      computeMs += Math.max(0, Date.parse(span.endedAt) - Date.parse(span.startedAt));
    }

    if (span.stepType === 'step.sendEvent') emittedSteps += 1;
  }

  // Flow control shows up as the run itself waiting before its first step —
  // queued, not executing. Measured against the run's own start rather than any
  // step's, because a concurrency hold happens before any step is scheduled.
  const queuedAt = rolledUp.queuedAt ? Date.parse(rolledUp.queuedAt) : null;
  const startedAt = rolledUp.startedAt ? Date.parse(rolledUp.startedAt) : null;
  const held = queuedAt !== null && startedAt !== null ? startedAt - queuedAt : 0;

  const endedAt = rolledUp.endedAt ? Date.parse(rolledUp.endedAt) : null;
  const elapsedMs = queuedAt !== null && endedAt !== null ? Math.max(0, endedAt - queuedAt) : 0;

  return {
    computeMs,
    elapsedMs,
    recoveredSteps,
    recoveredAttempts,
    suspendedMs,
    // A few hundred milliseconds is ordinary scheduling, not flow control doing
    // something for you. Only claim it when it is unambiguous.
    heldMs: held > 1000 ? held : 0,
    batchedFrom: batchSize,
    emittedSteps,
  };
}
