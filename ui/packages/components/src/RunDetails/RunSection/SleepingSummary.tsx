import { useEffect, useState } from 'react';
import { Card } from '@inngest/components/Card';
import { MetadataItem } from '@inngest/components/Metadata';
import type { HistoryNode } from '@inngest/components/utils/historyParser';

type Props = {
  history: Record<string, HistoryNode>;
};

export function SleepingSummary({ history }: Props) {
  const sleeps = useActiveSleeps(history);

  if (sleeps.length === 0) {
    return null;
  }

  return (
    <>
      {sleeps.map((sleep, i) => {
        const config = sleep.sleepConfig;
        if (!config) {
          // Should be unreachable but our types don't reflect that.
          return null;
        }

        return (
          <Card
            accentColor="bg-sky-400"
            className={i < sleeps.length - 1 ? 'mb-4' : undefined}
            key={sleep.groupID}
          >
            <Card.Header>Sleeping</Card.Header>

            <Card.Content>
              <MetadataItem label="Sleep until" value={config.until.toLocaleString()} />
            </Card.Content>
          </Card>
        );
      })}
    </>
  );
}

function useActiveSleeps(history: Record<string, HistoryNode>): HistoryNode[] {
  const [nodes, setNodes] = useState<HistoryNode[]>([]);

  useEffect(() => {
    const newWaits: HistoryNode[] = [];
    for (const node of Object.values(history)) {
      if (node.status === 'sleeping') {
        newWaits.push(node);
      }
    }
    setNodes(newWaits);
  }, [history]);

  return nodes;
}
