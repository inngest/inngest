'use client';

import { Pill } from '@inngest/components/Pill';
import type { HistoryNode, HistoryParser } from '@inngest/components/utils/historyParser';
import * as AccordionPrimitive from '@radix-ui/react-accordion';

import { TimelineNode } from './TimelineNode/TimelineNode';

export type NavigateToRunFn = (options: {
  eventID: string;
  runID: string;
  fnID: string;
}) => React.ReactNode;

type Props = {
  getOutput: (historyItemID: string) => Promise<string | undefined>;
  history: HistoryParser;
  navigateToRun: NavigateToRunFn;
};

export function Timeline({ getOutput, history, navigateToRun }: Props) {
  const nodes = history.getGroups({ sort: true });

  return (
    <div>
      {nodes.length === 0 ? (
        <div className=" text-basis text-center">No history yet</div>
      ) : (
        <AccordionPrimitive.Root
          type="multiple"
          className="border-subtle text-muted w-full last:border-b"
        >
          {nodes.map((node, i) => {
            if (!isVisible(node)) {
              return null;
            }

            return (
              <TimelineNode
                key={node.groupID}
                getOutput={getOutput}
                node={node}
                navigateToRun={navigateToRun}
              >
                {Object.values(node.attempts).length > 0 && (
                  <>
                    <div className="flex items-center gap-2 pt-4">
                      <p className="text-subtle py-4 text-sm">Attempts</p>
                      <Pill appearance="outlined">
                        {Object.values(node.attempts).length.toString() || '0'}
                      </Pill>
                    </div>
                    {Object.values(node.attempts).map((attempt) => (
                      <TimelineNode
                        key={attempt.groupID + attempt.attempt}
                        getOutput={getOutput}
                        node={attempt}
                        isAttempt
                        navigateToRun={navigateToRun}
                      />
                    ))}
                  </>
                )}
              </TimelineNode>
            );
          })}
        </AccordionPrimitive.Root>
      )}
    </div>
  );
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

  if (node.invokeFunctionResult) {
    // Invokes don't return a step name, but the group overall gives information
    return true;
  }

  return false;
}
