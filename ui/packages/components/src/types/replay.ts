import { type FunctionRunStatus } from './functionRun';

const replayStatuses = ['CREATED', 'ENDED'] as const;
export type ReplayStatus = (typeof replayStatuses)[number];

export type Replay = {
  id: string;
  name: string;
  status: ReplayStatus;
  createdAt: Date;
  endedAt?: Date;
  duration?: number;
  runsCount: number;
  runsSkippedCount?: number;
  fromRange?: Date;
  toRange?: Date;
  filters?: { statuses: FunctionRunStatus[] } | null;
};

export function isReplayStatus(s: string): s is ReplayStatus {
  return replayStatuses.includes(s as ReplayStatus);
}
