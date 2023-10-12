import * as AccordionPrimitive from '@radix-ui/react-accordion';

import { TimelineNode } from './TimelineNode/TimelineNode';
import type { HistoryNode } from './historyParser';

type Props = {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  history: Record<string, HistoryNode>;
};

export function Timeline({ getOutput, history }: Props) {
  const nodes = Object.values(history).sort(sortAscending);

  return (
    <div>
      {nodes.length === 0 ? (
        <div className=" text-white text-center">No history yet</div>
      ) : (
        <AccordionPrimitive.Root
          type="multiple"
          className="text-slate-100 w-full last:border-b last:border-slate-800/50"
        >
          {nodes.map((node, i) => {
            let position: 'first' | 'last' | 'middle' = 'middle';
            if (!isVisible(node)) {
              return null;
            }
            if (i === 0) {
              position = 'first';
            } else if (i === nodes.length - 1) {
              position = 'last';
            }

            return (
              <TimelineNode
                key={node.groupID}
                position={position}
                getOutput={getOutput}
                node={node}
              />
            );
          })}
        </AccordionPrimitive.Root>
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

  if (node.sleepConfig) {
    // Sleeps may not have a name but we still want to see it.
    return true;
  }

  if (node.waitForEventResult) {
    // Waits may not have a name but we still want to see it.
    return true;
  }

  return false;
}
