import type { HistoryNode } from './historyParser';
import { isEndStatus } from './historyParser/types';
import { TimelineNode } from './TimelineNode/TimelineNode';

type Props = {
  history: Record<string, HistoryNode>;
};

export function Timeline({ history }: Props) {
  const nodes = Object.values(history).sort(sortAscending);

  let content: JSX.Element | (JSX.Element | null)[];
  if (nodes.length === 0) {
    content = <div className="text-center">No history yet</div>;
  } else {
    content = nodes.map((node) => {
      if (!isVisible(node)) {
        return null;
      }

      return <TimelineNode className="my-2" node={node} key={node.groupID} />;
    });
  }

  return <div className="p-4 text-white">{content}</div>;
}

function sortAscending(a: HistoryNode, b: HistoryNode) {
  if (a.startedAt && b.startedAt) {
    return a.startedAt.getTime() - b.startedAt.getTime();
  } else {
    return 0;
  }
}

function isVisible(node: HistoryNode) {
  if (!isEndStatus(node.status)) {
    // Always should in-progress nodes.
    return true;
  }

  if (node.scope === "function") {
    // Show nodes like "function completed".
    return true
  }

  if (node.name) {
    // Pure discovery nodes (like planning parallel steps) don't have names.
    return true;
  }

  if (node.waitForEventResult) {
    // Wait for event may not have a name but we still want to see it.
    return true;
  }

  return false;
}
