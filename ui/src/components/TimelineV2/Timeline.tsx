import AccordionTimeline from '../AccordionTimeline/AccordionTimeline';
import type { HistoryNode } from './historyParser';
import { isEndStatus } from './historyParser/types';
import { TimelineNode } from './TimelineNode/TimelineNode';

type Props = {
  history: Record<string, HistoryNode>;
};

export function Timeline({ history }: Props) {
  const nodes = Object.values(history).sort(sortAscending);

  return (
    <div>
      {nodes.length === 0 ? (
        <div className=" text-white text-center">No history yet</div>
      ) : (
        <AccordionTimeline
          timelineItems={nodes
            .filter((node) => isVisible(node))
            .map((node, i) => ({
              id: node.groupID,
              header: <TimelineNode node={node} key={node.groupID} />,
              expandable: node.scope === 'function' ? false : true,
              position: i === 0 ? 'first' : i === nodes.length - 1 ? 'last' : 'middle',
              content: <div>Content here</div>,
            }))}
        />
      )}
    </div>
  );
}

function sortAscending(a: HistoryNode, b: HistoryNode) {
  if (a.startedAt && b.startedAt) {
    return a.startedAt.getTime() - b.startedAt.getTime();
  } else {
    return 0;
  }
}

function isVisible(node: HistoryNode) {
  if (node.status !== 'completed') {
    // We'll know if a node is pure discovery when it's completed. Therefore all
    // non-completed nodes are possibly non pure discovery.
    return true;
  }

  if (node.scope === 'function') {
    // Show nodes like "function completed".
    return true;
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
