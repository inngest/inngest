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
  ended,
  max,
  min,
  queued,
  started,
}: {
  ended: number | null;
  max: number;
  min: number;
  queued: number;
  started: number | null;
}) {
  let beforeWidth = queued - min;
  let queuedWidth = (started ?? max) - queued;
  let runningWidth = 0;
  let afterWidth = 0;

  if (started) {
    runningWidth = (ended ?? max) - started;
  }

  afterWidth = max - (ended ?? max);

  const totalWidth = max - min;

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
