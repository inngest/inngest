import { runEndGroupID, type HistoryNode, type HistoryType, type RawHistoryItem } from './types';

type Updater = (node: HistoryNode, rawItem: RawHistoryItem) => HistoryNode;

const noop: Updater = (node) => node;

const updaters: {
  [key in HistoryType]: Updater;
} = {
  FunctionCancelled: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      scope: 'function',
      status: 'cancelled',
    } satisfies HistoryNode;
  },
  FunctionCompleted: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      scope: 'function',
      status: 'completed',
    } satisfies HistoryNode;
  },
  FunctionFailed: (node, rawItem) => {
    return {
      ...node,
      attempts: {},
      endedAt: new Date(rawItem.createdAt),
      groupID: runEndGroupID,
      scope: 'function',
      status: 'failed',
    } satisfies HistoryNode;
  },
  FunctionScheduled: noop,
  FunctionStarted: (node, rawItem) => {
    // Treat this as a StepScheduled because the first step doesn't have a
    // dedicated StepScheduled.
    return updaters.StepScheduled(node, rawItem);
  },
  FunctionStatusUpdated: noop,
  None: noop,
  StepCompleted: (node, rawItem) => {
    const name = parseName(rawItem.stepName ?? undefined);

    let waitForEventResult: HistoryNode['waitForEventResult'] | undefined;
    if (rawItem.waitResult) {
      waitForEventResult = {
        eventID: rawItem.waitResult.eventID ?? undefined,
        timeout: rawItem.waitResult.timeout,
      };
    }

    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      name,
      outputItemID: rawItem.id,
      scope: 'step',
      status: 'completed',
      waitForEventResult,
    } satisfies HistoryNode;
  },
  StepErrored: (node, rawItem) => {
    rawItem.attempt;

    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      outputItemID: rawItem.id,
      scope: 'step',
      status: 'errored',
    } satisfies HistoryNode;
  },
  StepFailed: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      name: parseName(rawItem.stepName ?? undefined),
      outputItemID: rawItem.id,
      scope: 'step',
      status: 'failed',
    } satisfies HistoryNode;
  },
  StepScheduled: (node, rawItem) => {
    // When scheduling parallel steps, we know the step name ahead of time.
    const name = parseName(rawItem.stepName ?? undefined);

    return {
      ...node,
      name,
      scheduledAt: new Date(rawItem.createdAt),
      scope: 'step',
      status: 'scheduled',
    } satisfies HistoryNode;
  },
  StepSleeping: (node, rawItem) => {
    // Need to unset endedAt since StepCompleted can precede StepSleeping within
    // the same group.
    const endedAt = undefined;

    let sleepConfig: HistoryNode['sleepConfig'] | undefined;
    if (rawItem.sleep) {
      sleepConfig = {
        until: new Date(rawItem.sleep.until),
      };
    }

    return {
      ...node,
      endedAt,
      scope: 'step',
      sleepConfig,
      startedAt: node.startedAt ?? new Date(rawItem.createdAt),
      status: 'sleeping',
    } satisfies HistoryNode;
  },
  ['StepStarted']: (node, rawItem) => {
    let url: string | undefined;
    if (rawItem.url) {
      url = parseURL(rawItem.url);
    }

    return {
      ...node,
      startedAt: new Date(rawItem.createdAt),
      scope: 'step',
      status: 'started',
      url,
    } satisfies HistoryNode;
  },
  ['StepWaiting']: (node, rawItem) => {
    let waitForEventConfig: HistoryNode['waitForEventConfig'] | undefined;
    if (rawItem.waitForEvent) {
      waitForEventConfig = {
        eventName: rawItem.waitForEvent.eventName,
        expression: rawItem.waitForEvent.expression ?? undefined,
        timeout: new Date(rawItem.waitForEvent.timeout),
      };
    }

    return {
      ...node,
      scope: 'step',
      status: 'waiting',
      waitForEventConfig,
    } satisfies HistoryNode;
  },
} as const;

function parseName(name: string | undefined): string | undefined {
  // This is hacky, but assume that a name of "step" means we're discovering the
  // next step.
  if (name === 'step') {
    return undefined;
  }

  return name;
}

function parseURL(url: string): string {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    return 'Invalid URL';
  }

  parsed.searchParams.delete('fnId');
  parsed.searchParams.delete('stepId');
  return parsed.toString();
}

export function updateNode(node: HistoryNode, rawItem: RawHistoryItem): HistoryNode {
  if (rawItem.attempt > node.attempt) {
    // Since there's a new attempt, the existing group data represents an
    // errored (and therefore ended) attempt. Move that errored attempt to the
    // attempts.
    node = {
      ...node,
      attempt: rawItem.attempt,
      attempts: {
        ...node.attempts,
        [node.attempt]: {
          ...node,
          attempts: {},
        },
      },
    };
  }

  return updaters[rawItem.type](node, rawItem);
}
