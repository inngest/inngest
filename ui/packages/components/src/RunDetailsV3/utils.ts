import { useCallback, useEffect, useState } from 'react';
import type { Route } from 'next';
import type { Result } from '@inngest/components/types/functionRun';

import type { Trace } from './types';

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

export const maybeBooleanToString = (value: boolean | null): string | null => {
  if (value === null) {
    return null;
  }
  return value ? 'True' : 'False';
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

export type PathCreator = {
  runPopout: (params: { runID: string }) => Route;
};

export type StepInfoType = {
  trace: Trace;
  runID: string;
  result?: Result;
  pathCreator: PathCreator;
};

type Listener = (step: StepInfoType | undefined) => void;

const stepSelectionEmitter = {
  listeners: new Set<Listener>(),

  subscribe(listener: Listener) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  },

  emit(step: StepInfoType | undefined) {
    this.listeners.forEach((listener) => listener(step));
  },
};

export const useStepSelection = () => {
  const [selectedStep, setSelectedStep] = useState<StepInfoType | undefined>(undefined);

  useEffect(() => {
    const cleanup = stepSelectionEmitter.subscribe(setSelectedStep);
    return () => {
      cleanup();
    };
  }, []);

  const selectStep = useCallback((step: StepInfoType | undefined) => {
    stepSelectionEmitter.emit(step);
  }, []);

  return { selectedStep, selectStep };
};
