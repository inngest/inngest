import type { RawHistoryItem } from '@inngest/components/utils/historyParser';

const functionRunStatuses = ['CANCELLED', 'COMPLETED', 'FAILED', 'RUNNING', 'QUEUED'] as const;
const FunctionRunEndedStatuses = ['CANCELLED', 'COMPLETED', 'FAILED'] as const;
export type FunctionRunStatus = (typeof functionRunStatuses)[number];
export type FunctionRunEndStatus = (typeof FunctionRunEndedStatuses)[number];
export function isFunctionRunStatus(s: string): s is FunctionRunStatus {
  return functionRunStatuses.includes(s as FunctionRunStatus);
}

export type FunctionRun = {
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
