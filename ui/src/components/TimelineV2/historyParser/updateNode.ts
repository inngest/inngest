import { HistoryType, type RunHistoryItem } from '@/store/generated';
import type { HistoryNode, Status } from './types';

type Updater = (node: HistoryNode, rawItem: RunHistoryItem) => HistoryNode;

const noop: Updater = (node) => node;

const updaters: {
  [key in HistoryType]: Updater;
} = {
  [HistoryType.FunctionCancelled]: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      scope: 'function',
      status: 'cancelled',
    } satisfies HistoryNode;
  },
  [HistoryType.FunctionCompleted]: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      scope: 'function',
      status: 'completed',
    } satisfies HistoryNode;
  },
  [HistoryType.FunctionFailed]: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      scope: 'function',
      status: 'failed',
    } satisfies HistoryNode;
  },
  [HistoryType.FunctionScheduled]: (node, rawItem) => {
    return {
      ...node,
      scheduledAt: new Date(rawItem.createdAt),
      status: 'scheduled',
    } satisfies HistoryNode;
  },
  [HistoryType.FunctionStarted]: (node, rawItem) => {
    return {
      ...node,
      scheduledAt: new Date(rawItem.createdAt),
      status: 'scheduled',
    } satisfies HistoryNode;
  },
  [HistoryType.FunctionStatusUpdated]: noop,
  [HistoryType.None]: noop,
  [HistoryType.StepCompleted]: (node, rawItem) => {
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
  [HistoryType.StepErrored]: (node, rawItem) => {
    rawItem.attempt;

    return {
      ...node,
      scope: 'step',
      status: 'errored',
    } satisfies HistoryNode;
  },
  [HistoryType.StepFailed]: (node, rawItem) => {
    return {
      ...node,
      endedAt: new Date(rawItem.createdAt),
      name: parseName(rawItem.stepName ?? undefined),
      outputItemID: rawItem.id,
      scope: 'step',
      status: 'failed',
    } satisfies HistoryNode;
  },
  [HistoryType.StepScheduled]: (node, rawItem) => {
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
  [HistoryType.StepSleeping]: (node, rawItem) => {
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
      status: 'sleeping',
    } satisfies HistoryNode;
  },
  [HistoryType.StepStarted]: (node, rawItem) => {
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
  [HistoryType.StepWaiting]: (node, rawItem) => {
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

/**
 * Updates that should happen on all history types.
 */
const commonUpdater: Updater = (node, rawItem) => {
  return {
    ...node,
    attempt: rawItem.attempt,
  } satisfies HistoryNode;
};

export function updateNode(node: HistoryNode, rawItem: RunHistoryItem): HistoryNode {
  node = updaters[rawItem.type](node, rawItem);
  return commonUpdater(node, rawItem);
}
