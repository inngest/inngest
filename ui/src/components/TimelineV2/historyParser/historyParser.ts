import { HistoryType, type RunHistoryItem } from '@/store/generated';

export type HistoryNode = {
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

type Status =
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
function isEndStatus(status: Status): status is EndStatus {
  return endStatuses.includes(status as EndStatus);
}

const historyItemToStatus = {
  [HistoryType.FunctionCancelled]: 'cancelled',
  [HistoryType.FunctionCompleted]: 'completed',
  [HistoryType.FunctionFailed]: 'failed',
  [HistoryType.FunctionScheduled]: 'scheduled',
  [HistoryType.FunctionStarted]: 'started',
  [HistoryType.FunctionStatusUpdated]: null,
  [HistoryType.None]: null,
  [HistoryType.StepCompleted]: 'completed',
  [HistoryType.StepErrored]: 'errored',
  [HistoryType.StepFailed]: 'failed',
  [HistoryType.StepScheduled]: 'scheduled',
  [HistoryType.StepSleeping]: 'sleeping',
  [HistoryType.StepStarted]: 'started',
  [HistoryType.StepWaiting]: 'waiting',
} as const satisfies Readonly<{ [key in HistoryType]: Status | null }>;

function getStatus(rawItem: RunHistoryItem): Status | null {
  if (rawItem.sleep) {
    return 'sleeping';
  }

  if (rawItem.waitForEvent) {
    return 'waiting';
  }

  return historyItemToStatus[rawItem.type];
}

/**
 * Parses and groups raw history. Each history node has enough data to display a
 * history node as our users envision it. For example, if a user calls
 * `step.waitForEvent` then they expect a single history node, rather than the
 * multiple history items we actually store in our DB. Unlike raw history,
 * history nodes are mutable.
 */
export class HistoryParser {
  history: Record<string, HistoryNode> = {};

  constructor(rawHistory?: RunHistoryItem[]) {
    if (rawHistory) {
      for (const item of rawHistory) {
        this.append(item);
      }
    }
  }

  append(rawItem: RunHistoryItem) {
    const status = getStatus(rawItem);
    if (status === null) {
      // Ignore this item.
      return;
    }

    let node: HistoryNode;
    if (this.history[rawItem.groupID]) {
      node = { ...this.history[rawItem.groupID] };
    } else {
      node = {
        groupID: rawItem.groupID,
        status,
      };
    }

    if (rawItem.sleep) {
      node.sleepConfig = {
        until: new Date(rawItem.sleep.until),
      };
    }

    if (rawItem.waitForEvent) {
      node.waitForEventConfig = {
        eventName: rawItem.waitForEvent.eventName,
        expression: rawItem.waitForEvent.expression ?? undefined,
        timeout: new Date(rawItem.waitForEvent.timeout),
      };
    }

    if (rawItem.waitResult) {
      node.waitForEventResult = {
        eventID: rawItem.waitResult.eventID ?? undefined,
        timeout: rawItem.waitResult.timeout,
      };
    }

    node.status = status;

    const itemTime = new Date(rawItem.createdAt);
    if (status === 'scheduled') {
      node.scheduledAt = itemTime;
    } else if (status === 'started') {
      node.startedAt = itemTime;
    } else if (isEndStatus(status)) {
      node.endedAt = itemTime;

      if (rawItem.type.includes('Function')) {
        node.name = undefined;
        node.scope = 'function';
      } else if (rawItem.type.includes('Step')) {
        // We'll know the real step name after it ends.
        node.name = rawItem.stepName ?? undefined;

        node.scope = 'step';
      }
    }

    this.history = {
      ...this.history,
      [node.groupID]: node,
    };
  }
}
