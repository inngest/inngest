import { type FunctionRunStatus } from './functionRun';

export enum ReplayStatus {
  Created = 'CREATED',
  Ended = 'ENDED',
}

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
  return Object.values(ReplayStatus).includes(s as ReplayStatus);
}
