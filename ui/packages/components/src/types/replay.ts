const replayStatuses = ['CREATED', 'ENDED'] as const;
export type ReplayStatus = (typeof replayStatuses)[number];

export type Replay = {
  name: string;
  status: ReplayStatus;
  createdAt: Date;
  endedAt?: Date;
  duration?: number;
  runsCount: number;
  runsSkippedCount?: number;
};

export function isReplayStatus(s: string): s is ReplayStatus {
  return replayStatuses.includes(s as ReplayStatus);
}
