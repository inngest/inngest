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
  | 'StepWaiting'
  | 'StepInvokingFunction';

export type RawHistoryItem = {
  attempt: number;
  cancel?: {
    eventID?: string | null;
    expression?: string | null;
    userID?: string | null;
  } | null;
  createdAt: string;
  functionVersion: number;
  groupID?: string | null;
  id: string;
  result?: {
    errorCode?: string | null;
  } | null;
  sleep?: {
    until: string;
  } | null;
  stepName?: string | null;
  stepType?: 'Run' | 'Send' | 'Sleep' | 'Wait' | null;
  type: HistoryType;
  url?: string | null;
  waitForEvent?: {
    eventName: string;
    expression?: string | null;
    timeout: string;
  } | null;
  waitResult?: {
    eventID?: string | null;
    timeout: boolean;
  } | null;
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
