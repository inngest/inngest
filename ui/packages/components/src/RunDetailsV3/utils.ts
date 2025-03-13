import { useCallback, useEffect, useState } from 'react';
import type { Route } from 'next';
import type { Result } from '@inngest/components/types/functionRun';

import type { Trace } from './types';

export const FINAL_SPAN_DISPLAY = 'Finalization';
export const FINAL_SPAN_NAME = 'function success';

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

type Listener = {
  callback: (step: StepInfoType | undefined) => void;
  runID?: string;
};

const stepSelectionEmitter = {
  listeners: new Set<Listener>(),

  subscribe(callback: (step: StepInfoType | undefined) => void, runID?: string) {
    const listener = { callback, runID };
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  },

  emit(step: StepInfoType | undefined) {
    this.listeners.forEach((listener) => {
      if (!listener.runID || !step || listener.runID === step.runID) {
        listener.callback(step);
      }
    });
  },
};

export const useStepSelection = (runID?: string) => {
  const [selectedStep, setSelectedStep] = useState<StepInfoType | undefined>(undefined);

  useEffect(() => {
    const cleanup = stepSelectionEmitter.subscribe(setSelectedStep, runID);
    return () => {
      cleanup();
    };
  }, [runID]);

  const selectStep = useCallback((step: StepInfoType | undefined) => {
    stepSelectionEmitter.emit(step);
  }, []);

  return { selectedStep, selectStep };
};

export const formatDuration = (ms: number): string => {
  const units = [
    { label: 'd', value: 86400000 }, // 24 * 60 * 60 * 1000
    { label: 'h', value: 3600000 }, // 60 * 60 * 1000
    { label: 'm', value: 60000 }, // 60 * 1000
    { label: 's', value: 1000 }, // 1000
    { label: 'ms', value: 1 },
  ];

  for (const { label, value } of units) {
    if (ms >= value) {
      const amount = ms / value;
      const rounded = Math.round(amount * 10) / 10;
      const display = rounded % 1 === 0 ? rounded.toFixed(0) : rounded.toFixed(1);
      return `${display}${label}`;
    }
  }

  return '0ms';
};

export const getSpanName = (name: string) => {
  return name === FINAL_SPAN_NAME ? FINAL_SPAN_DISPLAY : name;
};
