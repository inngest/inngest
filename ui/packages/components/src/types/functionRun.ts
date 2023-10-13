const functionRunStatuses = ['CANCELLED', 'COMPLETED', 'FAILED', 'RUNNING'] as const;
export type FunctionRunStatus = (typeof functionRunStatuses)[number];
export function isFunctionRunStatus(s: string): s is FunctionRunStatus {
  return functionRunStatuses.includes(s as FunctionRunStatus);
}

export type FunctionRun = {
  id: string;
  name: string;
  output: string | undefined;
  status: FunctionRunStatus;
};
