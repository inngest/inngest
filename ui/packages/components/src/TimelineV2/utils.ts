type Span = {
  endedAt: Date | null;
  queuedAt: Date;
  startedAt: Date | null;
};

export type SpanWidths = {
  after: number;
  before: number;
  queued: number;
  running: number;
};

export function createSpanWidths({
  maxTime,
  minTime,
  trace,
}: {
  maxTime: Date;
  minTime: Date;
  trace: Span;
}) {
  let beforeWidth = trace.queuedAt.getTime() - minTime.getTime();
  let queuedWidth = (trace.startedAt ?? maxTime).getTime() - trace.queuedAt.getTime();
  let runningWidth = 0;
  let afterWidth = 0;

  if (trace.startedAt) {
    runningWidth = (trace.endedAt ?? maxTime).getTime() - trace.startedAt.getTime();
  }

  afterWidth = maxTime.getTime() - (trace.endedAt ?? maxTime).getTime();

  const totalWidth = maxTime.getTime() - minTime.getTime();

  return {
    after: normalizeWidth({ width: afterWidth, totalWidth }),
    before: normalizeWidth({ width: beforeWidth, totalWidth }),
    queued: normalizeWidth({ width: queuedWidth, totalWidth }),
    running: normalizeWidth({ width: runningWidth, totalWidth }),
  };
}

/**
 * Turn the width into an integer and scale it down to ensure it isn't a massive
 * number
 */
function normalizeWidth({ totalWidth, width }: { totalWidth: number; width: number }): number {
  if (width === 0) {
    return 0;
  }

  // Ensure the width is between the min and max
  const minWidth = 1;
  const maxWidth = 1000;
  return Math.max(Math.floor((width / totalWidth) * maxWidth), minWidth);
}

export function toMaybeDate<T extends string | null | undefined>(value: T): Date | null {
  if (!value) {
    return null;
  }

  return new Date(value);
}
