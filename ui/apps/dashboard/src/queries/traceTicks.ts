/**
 * Joins the ticks of a run's trace back onto its spans.
 *
 * Ticks are the small events on a span's row, such as messages arriving at an
 * inbox. Cloud caps query complexity, and the run trace query has no room left
 * for them, so they come from their own lighter query (see `useGetRun`) and are
 * merged in here by span ID.
 *
 * @module
 */

export type SpanTick = {
  id: string;
  sources: string[];
  at: string;
  readAt: string | null;
};

/** A span from the ticks query. */
type TickedSpan = {
  spanID: string;
  ticks: SpanTick[];
  childrenSpans?: TickedSpan[];
};

/** A span from the run trace query. */
type TraceSpan = {
  spanID: string;
  childrenSpans?: TraceSpan[];
};

/**
 * Lists the ticks of every span in a ticks tree, by span ID. Spans without
 * ticks are left out.
 */
export function indexTicks(
  /** The root of the ticks query result, if the query returned one */
  root: TickedSpan | null | undefined,
): Map<string, SpanTick[]> {
  const bySpanID = new Map<string, SpanTick[]>();

  if (!root) {
    return bySpanID;
  }

  const pending = [root];
  for (let span = pending.pop(); span; span = pending.pop()) {
    if (span.ticks.length > 0) {
      bySpanID.set(span.spanID, span.ticks);
    }

    pending.push(...(span.childrenSpans ?? []));
  }

  return bySpanID;
}

/**
 * Copies a trace with a `ticks` list on every span. Spans the ticks tree
 * doesn't know, for example because the run moved on between the two queries,
 * get an empty list.
 */
export function mergeTicks<T extends TraceSpan>(
  /** The root span of the run trace query result */
  span: T,
  /** The ticks of each span, from `indexTicks` */
  ticksBySpanID: ReadonlyMap<string, SpanTick[]>,
): T & { ticks: SpanTick[] } {
  return {
    ...span,
    ticks: ticksBySpanID.get(span.spanID) ?? [],
    ...(span.childrenSpans && {
      childrenSpans: span.childrenSpans.map((child) => {
        return mergeTicks(child, ticksBySpanID);
      }),
    }),
  };
}
