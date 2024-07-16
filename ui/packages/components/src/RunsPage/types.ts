import { type FunctionRunStatus } from '@inngest/components/types/functionRun';

// Whether the view is at the environment or function level
export type ViewScope = 'env' | 'fn';

export type Run = {
  app: {
    externalID: string;
    name: string;
  };
  function: {
    name: string;
    slug: string;
  };
  status: FunctionRunStatus;
  durationMS: number | null;
  id: string;
  queuedAt: string;
  endedAt: string | null;
  startedAt: string | null;
};
