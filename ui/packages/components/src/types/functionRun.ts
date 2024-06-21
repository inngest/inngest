import type { RawHistoryItem } from '@inngest/components/utils/historyParser';

export const functionRunStatuses = [
  'FAILED',
  'RUNNING',
  'QUEUED',
  'COMPLETED',
  'CANCELLED',
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

export const FunctionRunTimeFields = ['QUEUED_AT', 'STARTED_AT', 'ENDED_AT'] as const;
export type FunctionRunTimeField = (typeof FunctionRunTimeFields)[number];
export function isFunctionTimeField(s: string): s is FunctionRunTimeField {
  return FunctionRunTimeFields.includes(s as FunctionRunTimeField);
}

export type Result = {
  data: string | null;
  error: {
    message: string;
    name: string | null;
    stack: string | null;
  } | null;
};
