export type Trace = {
  attempts: number | null;
  endedAt: string | null;
  isRoot: boolean;
  name: string;
  outputID: string | null;
  queuedAt: string;
  spanID: string;
  startedAt: string | null;
  status: string;
  childrenSpans?: Trace[];
  stepInfo?: unknown;
  stepOp?: string | null;
};

export enum StepStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Queued = 'QUEUED',
  Running = 'RUNNING',
}

export function isStepStatus(value: string): value is StepStatus {
  return Object.values(StepStatus).includes(value as StepStatus);
}
