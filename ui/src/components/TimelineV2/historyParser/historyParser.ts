import { HistoryType, type RunHistoryItem } from '@/store/generated';
import { type HistoryNode } from './types';
import { updateNode } from './updateNode';

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
  history: Record<string, HistoryNode> = {};

  constructor(rawHistory?: RunHistoryItem[]) {
    if (rawHistory) {
      for (const item of rawHistory) {
        this.append(item);
      }
    }
  }

  append(rawItem: RunHistoryItem) {
    let node: HistoryNode;
    if (this.history[rawItem.groupID]) {
      node = { ...this.history[rawItem.groupID] };
    } else {
      node = {
        attempt: rawItem.attempt,
        groupID: rawItem.groupID,
        status: 'scheduled',
      };
    }

    node = updateNode(node, rawItem);

    this.history = {
      ...this.history,
      [node.groupID]: node,
    };

    if (rawItem.type === HistoryType.FunctionCancelled && node.endedAt) {
      this.cancelNodes(node.endedAt);
    }
  }

  private cancelNodes(endedAt: Date) {
    for (const node of Object.values(this.history)) {
      if (!node.endedAt) {
        this.history = {
          ...this.history,
          [node.groupID]: {
            ...node,
            endedAt,
            status: 'cancelled',
          },
        };
      }
    }
  }
}
