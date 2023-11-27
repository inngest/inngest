export const runEndGroupID = 'function-run-end';
export const runStartGroupID = 'function-run-start';

export type HistoryNode = {
  attempt: number;

  // Use a record instead of array in case we're missing an attempt's data.
  attempts: Record<number, HistoryNode>;

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
  invokeFunctionConfig?: {
    eventID: string;
    functionID: string;
    correlationID: string;
    timeout: Date;
  };
  invokeFunctionResult?: {
    eventID: string | undefined;
    timeout: boolean;
    runID: string | undefined;
  };
};

const historyTypes = [
  'FunctionCancelled',
  'FunctionCompleted',
  'FunctionFailed',
  'FunctionScheduled',
  'FunctionStarted',
  'FunctionStatusUpdated',
  'None',
  'StepCompleted',
  'StepErrored',
  'StepFailed',
  'StepScheduled',
  'StepSleeping',
  'StepStarted',
  'StepWaiting',
  'StepInvoking',
] as const;
export type HistoryType = (typeof historyTypes)[number];
export function isHistoryType(value: string): value is HistoryType {
  return historyTypes.includes(value as HistoryType);
}

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
  type: string;
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
  invokeFunction?: {
    eventID: string;
    functionID: string;
    correlationID: string;
    timeout: string;
  } | null;
  invokeFunctionResult?: {
    eventID?: string | null;
    timeout: boolean;
    runID?: string | null;
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
