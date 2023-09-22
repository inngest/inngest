export type HistoryNode = {
  attempt: number;
  endedAt?: Date;
  groupID: string;
  name?: string;
  scheduledAt?: Date;
  scope?: 'function' | 'step';
  sleepConfig?: {
    until: Date;
  };
  startedAt?: Date;
  status: Status;
  url?: string;
  waitForEventConfig?: {
    eventName: string;
    expression: string | undefined;
    timeout: Date;
  };
  waitForEventResult?: {
    eventID: string | undefined;
    timeout: boolean;
  };
};

export type Status =
  | 'cancelled'
  | 'completed'
  | 'errored'
  | 'failed'
  | 'scheduled'
  | 'sleeping'
  | 'started'
  | 'waiting';

const endStatuses = ['cancelled', 'completed', 'failed'] as const satisfies Readonly<Status[]>;
type EndStatus = (typeof endStatuses)[number];
export function isEndStatus(value: Status): value is EndStatus {
  return endStatuses.includes(value as EndStatus);
}
