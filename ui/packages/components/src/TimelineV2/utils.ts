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

  const timelineWidth = maxTime.getTime() - minTime.getTime();

  if (beforeWidth > 0) {
    beforeWidth = Math.max(Math.floor((beforeWidth / timelineWidth) * 1000), 1);
  }
  if (queuedWidth > 0) {
    queuedWidth = Math.max(Math.floor((queuedWidth / timelineWidth) * 1000), 1);
  }
  if (runningWidth > 0) {
    runningWidth = Math.max(Math.floor((runningWidth / timelineWidth) * 1000), 1);
  }
  if (afterWidth > 0) {
    afterWidth = Math.max(Math.floor((afterWidth / timelineWidth) * 1000), 1);
  }

  return {
    after: afterWidth,
    before: beforeWidth,
    queued: queuedWidth,
    running: runningWidth,
  };
}

export function toMaybeDate<T extends string | null | undefined>(value: T): Date | null {
  if (!value) {
    return null;
  }

  return new Date(value);
}
