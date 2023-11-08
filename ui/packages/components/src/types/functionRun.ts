const functionRunStatuses = ['CANCELLED', 'COMPLETED', 'FAILED', 'RUNNING'] as const;
export type FunctionRunStatus = (typeof functionRunStatuses)[number];
export function isFunctionRunStatus(s: string): s is FunctionRunStatus {
  return functionRunStatuses.includes(s as FunctionRunStatus);
}

export type FunctionRun = {
  canRerun: boolean | null;

  // TODO: Change to Date
  endedAt: string | null;

  id: string;
  name: string;
  output: string | null;

  // TODO: Change to Date
  startedAt: string | null;

  status: FunctionRunStatus;
};
