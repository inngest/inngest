import { type FunctionRunStatus } from '@inngest/components/types/functionRun';

// Whether the view is at the environment, app, or function level
export type ViewScope = 'env' | 'app' | 'fn';

export type Run = {
  app: {
    externalID: string;
  };
  function: {
    name: string;
  };
  status: FunctionRunStatus;
  durationMS: number | null;
  id: string;
  queuedAt: string;
  endedAt: string | null;
  startedAt: string | null;
};
