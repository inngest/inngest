import {
  isHistoryType,
  runEndGroupID,
  type HistoryNode,
  type HistoryType,
  type RawHistoryItem,
} from './types';

type Updater = (node: HistoryNode, rawItem: RawHistoryItem) => HistoryNode;

const noop: Updater = (node) => node;

const updaters: Record<HistoryType, Updater> = {
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
        eventID: rawItem.waitResult.eventID || undefined,
        timeout: rawItem.waitResult.timeout,
      };
    }

    let invokeFunctionResult: HistoryNode['invokeFunctionResult'] | undefined;
    if (rawItem.invokeFunctionResult) {
      invokeFunctionResult = {
        eventID: rawItem.invokeFunctionResult.eventID || undefined,
        timeout: rawItem.invokeFunctionResult.timeout,
        runID: rawItem.invokeFunctionResult.runID || undefined,
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
      invokeFunctionResult,
    } satisfies HistoryNode;
  },
  StepErrored: (node, rawItem) => {
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
      startedAt: new Date(rawItem.createdAt),
      status: 'waiting',
      waitForEventConfig,
    } satisfies HistoryNode;
  },
  StepInvoking: (node, rawItem) => {
    let invokeFunctionConfig: HistoryNode['invokeFunctionConfig'] | undefined;
    if (rawItem.invokeFunction) {
      invokeFunctionConfig = {
        eventID: rawItem.invokeFunction.eventID,
        functionID: rawItem.invokeFunction.functionID,
        correlationID: rawItem.invokeFunction.correlationID,
        timeout: new Date(rawItem.invokeFunction.timeout),
      };
    }

    return {
      ...node,
      scope: 'step',
      status: 'waiting',
      invokeFunctionConfig,
    };
  },
} satisfies {
  [key in HistoryType]: Updater;
};

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
  const historyType = rawItem.type;
  if (!isHistoryType(historyType)) {
    // Return the node unchanged if the history type is unexpected.
    return node;
  }

  const update = updaters[historyType];
  node = update(node, rawItem);
  node.attempt = rawItem.attempt;

  const attemptNode = node.attempts[rawItem.attempt];

  // Should always be true but bugs can happen.
  if (attemptNode) {
    // Update logic is the same for attempts since they have the same shape
    // as the top-level node.
    node.attempts[rawItem.attempt] = update(attemptNode, rawItem);
  }

  return node;
}
