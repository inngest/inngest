import durationPlugin from 'dayjs/plugin/duration';

const replayStatuses = ['CREATED', 'ENDED'] as const;
export type ReplayStatus = (typeof replayStatuses)[number];

export type Replay = {
  name: string;
  status: ReplayStatus;
  createdAt: Date;
  endedAt?: Date;
  duration?: durationPlugin.Duration;
  runsCount: number;
};
