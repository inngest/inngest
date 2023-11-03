import type { HistoryNode, RawHistoryItem } from './types';
import { updateNode } from './updateNode';

const runEndGroupID = 'function-run-end';
const runStartGroupID = 'function-run-start';

/**
 * Parses and groups raw history. Each history node has enough data to display a
 * history node as our users envision it. For example, if a user calls
 * `step.waitForEvent` then they expect a single history node, rather than the
 * multiple history items we actually store in our DB. Unlike raw history,
 * history nodes are mutable.
 *
 * IMPORTANT: The append method expects to be called in ascending chronological
 * order. Appending older raw history items will result in bugs. This should be
 * changed in the future, but that increases complexity.
 */
export class HistoryParser {
  private groups: Record<string, HistoryNode> = {};
  runStartedAt?: Date;

  constructor(rawHistory?: RawHistoryItem[]) {
    if (rawHistory) {
      for (const item of rawHistory) {
        this.append(item);
      }
    }
  }

  append(rawItem: RawHistoryItem) {
    const groupID = rawItem.groupID ?? 'unknown';

    if (rawItem.type === 'FunctionStarted') {
      this.runStartedAt = new Date(rawItem.createdAt);
      this.createFunctionRunStartNode(new Date(rawItem.createdAt));
    }

    let node: HistoryNode;
    const existingNode = this.groups[groupID];
    if (existingNode) {
      node = { ...existingNode };
    } else {
      node = {
        attempt: rawItem.attempt,
        groupID,
        scheduledAt: new Date(rawItem.createdAt),
        sleepConfig: undefined,
        status: 'scheduled',
        waitForEventConfig: undefined,
        waitForEventResult: undefined,
        invokeFunctionConfig: undefined,
        invokeFunctionResult: undefined,
      };
    }

    if (rawItem.type === 'FunctionFailed' && node.scope === 'step') {
      // Put FunctionFailed into its own node. Its group ID is the same as
      // StepFailed but don't want to mess up the StepFailed node's data.
      node.groupID = runEndGroupID;
    }

    node = updateNode(node, rawItem);

    this.groups = {
      ...this.groups,
      [node.groupID]: node,
    };

    if (rawItem.type === 'FunctionCancelled') {
      this.cancelNodes(new Date(rawItem.createdAt));
    }

    this.handleCompletedSleepNodes(new Date(rawItem.createdAt));
  }

  /**
   * Mark all in-progress nodes as cancelled.
   *
   * Run cancellation doesn't create StepCancelled history items for in-progress
   * steps.
   */
  private cancelNodes(endedAt: Date) {
    for (const node of Object.values(this.groups)) {
      if (!node.endedAt) {
        this.groups = {
          ...this.groups,
          [node.groupID]: {
            ...node,
            endedAt,
            status: 'cancelled',
          },
        };
      }
    }
  }

  /**
   * Creates a node for the function run start. This is needed because the
   * FunctionStarted history item represents 2 things: starting the function and
   * scheduling the first step. Since those 2 things need separate nodes, we
   * need this method to create the function scope node.
   */
  private createFunctionRunStartNode(startedAt: Date) {
    // Create a dedicated node for function run start.
    const node: HistoryNode = {
      attempt: 0,
      endedAt: startedAt,
      groupID: runStartGroupID,
      scheduledAt: startedAt,
      scope: 'function',
      startedAt,
      status: 'started',
    } as const;

    this.groups = {
      ...this.groups,
      [node.groupID]: node,
    };
  }

  getGroups({ sort = false }: { sort?: boolean } = {}): HistoryNode[] {
    const unsortedGroups = Object.values(this.groups);
    if (!sort) {
      return unsortedGroups;
    }

    return unsortedGroups.sort((a, b) => {
      // Always put run start group at the top.
      if (a.groupID === runStartGroupID) {
        return -1;
      }
      if (b.groupID === runStartGroupID) {
        return 1;
      }

      // Always put run end group at the bottom.
      if (a.groupID === runEndGroupID) {
        return 1;
      }
      if (b.groupID === runEndGroupID) {
        return -1;
      }

      // Sort by ascending time.
      return a.scheduledAt.getTime() - b.scheduledAt.getTime();
    });
  }

  /**
   * Mark sleep nodes as completed if their wake time is reached.
   *
   * Sleeps don't have a StepCompleted history item after they complete. So we
   * need to mark them as completed whenever their wake time is reached.
   */
  private handleCompletedSleepNodes(time: Date) {
    for (const node of this.getGroups()) {
      if (node.status === 'sleeping' && node.sleepConfig) {
        const isCompleted = node.sleepConfig.until <= time;
        if (isCompleted) {
          this.groups = {
            ...this.groups,
            [node.groupID]: {
              ...node,
              endedAt: node.sleepConfig.until,
              status: 'completed',
            },
          };
        }
      }
    }
  }
}
