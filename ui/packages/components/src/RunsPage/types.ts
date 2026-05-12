import { type FunctionRunStatus, type RunType } from '@inngest/components/types/functionRun';

// Whether the view is at the environment or function level
export type ViewScope = 'env' | 'fn';

export type Run = {
  app: {
    externalID: string;
    name: string;
  };
  cronSchedule: string | null;
  eventName: string | null;
  function: {
    name: string;
    slug: string;
  };
  status: FunctionRunStatus;
  durationMS: number | null;
  id: string;
  isBatch: boolean;
  queuedAt: string;
  runType: RunType;
  endedAt: string | null;
  startedAt: string | null;
  hasAI?: boolean;
  deferredFrom?: {
    parentRun: {
      function: {
        name: string;
        slug: string;
      };
    } | null;
  } | null;
};
