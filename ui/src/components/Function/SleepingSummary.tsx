import { useEffect, useState } from 'react';

import MetadataItem from '../Metadata/MetadataItem';
import type { HistoryNode } from '../TimelineV2/historyParser';
import { StateSummaryCard } from './StateSummaryCard';

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
          <StateSummaryCard
            className={i < sleeps.length - 1 ? 'mb-4' : undefined}
            key={sleep.groupID}
          >
            <StateSummaryCard.Header barColor="#38BDF8">Sleeping</StateSummaryCard.Header>

            <StateSummaryCard.Content>
              <MetadataItem label="Sleep until" value={config.until.toLocaleString()} />
            </StateSummaryCard.Content>
          </StateSummaryCard>
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
