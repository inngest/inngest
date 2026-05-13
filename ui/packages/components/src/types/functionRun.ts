import type { RawHistoryItem } from '@inngest/components/utils/historyParser';

export const functionRunStatuses = [
  'FAILED',
  'RUNNING',
  'PAUSED',
  'QUEUED',
  'COMPLETED',
  'CANCELLED',
  'SKIPPED',
  'WAITING',
  'UNKNOWN',
] as const;
const FunctionRunEndedStatuses = ['CANCELLED', 'COMPLETED', 'FAILED'] as const;
export type FunctionRunStatus = (typeof functionRunStatuses)[number];
export type FunctionRunEndStatus = (typeof FunctionRunEndedStatuses)[number];
export function isFunctionRunStatus(s: string): s is FunctionRunStatus {
  return functionRunStatuses.includes(s as FunctionRunStatus);
}

export const replayRunStatuses = ['COMPLETED', 'FAILED', 'CANCELLED', 'SKIPPED_PAUSED'] as const;
export type ReplayRunStatus = (typeof replayRunStatuses)[number];
export function isReplayRunStatus(s: string): s is ReplayRunStatus {
  return replayRunStatuses.includes(s as ReplayRunStatus);
}

// Defer statuses describe the lifecycle of a deferred run scheduling row,
// not the run itself. Mirrors enums.DeferStatus.
export const deferStatuses = ['ABORTED', 'SCHEDULED'] as const;
export type DeferStatus = (typeof deferStatuses)[number];
export function isDeferStatus(s: string): s is DeferStatus {
  return deferStatuses.includes(s as DeferStatus);
}

export const runTypes = ['PRIMARY', 'DEFER'] as const;
export type RunType = (typeof runTypes)[number];
export function isRunType(s: string): s is RunType {
  return runTypes.includes(s as RunType);
}

export type FunctionRun = {
  batchCreatedAt: Date | null;
  batchID: string | null;
  canRerun: boolean | null;
  endedAt: Date | null;
  functionID: string;
  history: RawHistoryItem[];
  id: string;
  name: string;
  output: string | null;
  startedAt: Date | null;
  status: FunctionRunStatus;
};

export const FunctionRunTimeField = {
  QueuedAt: 'QUEUED_AT',
  StartedAt: 'STARTED_AT',
  EndedAt: 'ENDED_AT',
} as const;
export type FunctionRunTimeField = (typeof FunctionRunTimeField)[keyof typeof FunctionRunTimeField];

export function isFunctionTimeField(s: string): s is FunctionRunTimeField {
  for (const value of Object.values(FunctionRunTimeField)) {
    if (value === s) {
      return true;
    }
  }

  return false;
}
