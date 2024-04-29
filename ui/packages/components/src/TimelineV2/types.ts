export type Trace = {
  attempts: number;
  endedAt: string | null;
  id: string;
  isRoot: boolean;
  name: string;
  outputID: string | null;
  queuedAt: string;
  startedAt: string | null;
  status: string;
  childrenSpans?: Trace[];
  stepInfo?: unknown;
  stepOp?: string | null;
};

export enum StepStatus {
  Cancelled = 'CANCELLED',
  Failed = 'FAILED',
  Queued = 'QUEUED',
  Running = 'RUNNING',
  Succeeded = 'SUCCEEDED',
}

export function isStepStatus(value: string): value is StepStatus {
  return Object.values(StepStatus).includes(value as StepStatus);
}
