import AccordionTimeline, { AccordionTimelineItem } from '../AccordionTimeline/AccordionTimeline';
import type { HistoryNode } from './historyParser';
import { TimelineNode } from './TimelineNode/TimelineNode';

type Props = {
  getOutput: (id: string) => Promise<string>;
  history: Record<string, HistoryNode>;
};

export function Timeline({ getOutput, history }: Props) {
  const nodes = Object.values(history).sort(sortAscending);

  return (
    <div>
      {nodes.length === 0 ? (
        <div className=" text-white text-center">No history yet</div>
      ) : (
        <AccordionTimeline>
          {nodes.map((node) => {
            if (!isVisible(node)) {
              return null
            }

            const { outputItemID } = node;
            let getContent: (() => Promise<string>) | undefined;
            if (node.scope === 'step' && outputItemID) {
              getContent = () => {
                return getOutput(outputItemID);
              };
            }

            return (
              <AccordionTimelineItem
                getContent={getContent}
                header={<TimelineNode getOutput={getOutput} node={node} key={node.groupID} />}
                id={node.groupID}
                key={node.groupID}
              />
            );
          })}
        </AccordionTimeline>
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
