// TODO: Use `| undefined` instead of this. It's confusing to use both null and
// undefined all over the place, but right now our GraphQL APIs use both.
type Nullish<T> = T | null | undefined;

export type HistoryNode = {
  attempt: number;
  endedAt?: Date;
  groupID: string;
  name?: string;
  outputItemID?: string;
  scheduledAt: Date;
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

export type HistoryType =
  | 'FunctionCancelled'
  | 'FunctionCompleted'
  | 'FunctionFailed'
  | 'FunctionScheduled'
  | 'FunctionStarted'
  | 'FunctionStatusUpdated'
  | 'None'
  | 'StepCompleted'
  | 'StepErrored'
  | 'StepFailed'
  | 'StepScheduled'
  | 'StepSleeping'
  | 'StepStarted'
  | 'StepWaiting';

export type RawHistoryItem = {
  attempt: number;
  cancel?: Nullish<{
    eventID?: Nullish<string>;
    expression?: Nullish<string>;
    userID?: Nullish<string>;
  }>;
  createdAt: string;
  functionVersion: number;
  groupID?: Nullish<string>;
  id: string;
  result?: Nullish<{
    errorCode?: Nullish<string>;
  }>;
  sleep?: Nullish<{
    until: string;
  }>;
  stepName?: Nullish<string>;
  stepType?: Nullish<'Run' | 'Send' | 'Sleep' | 'Wait'>;
  type: HistoryType;
  url?: Nullish<string>;
  waitForEvent?: Nullish<{
    eventName: string;
    expression?: Nullish<string>;
    timeout: string;
  }>;
  waitResult?: Nullish<{
    eventID?: Nullish<string>;
    timeout: boolean;
  }>;
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
