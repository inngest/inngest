import { useEffect, useState } from 'react';

import { IconEvent } from '@/icons';
import MetadataItem from '../Metadata/MetadataItem';
import type { HistoryNode } from '../TimelineV2/historyParser';
import { StateSummaryCard } from './StateSummaryCard';

type Props = {
  history: Record<string, HistoryNode>;
};

export function WaitingSummary({ history }: Props) {
  const waits = useActiveWaits(history);

  if (waits.length === 0) {
    return null;
  }

  return (
    <>
      {waits.map((wait, i) => {
        const config = wait.waitForEventConfig;
        if (!config) {
          // Should be unreachable but our types don't reflect that.
          return null;
        }

        return (
          <StateSummaryCard
            className={i < waits.length - 1 ? 'mb-4' : undefined}
            key={wait.groupID}
          >
            <StateSummaryCard.Header barColor="#38BDF8">Waiting for event</StateSummaryCard.Header>

            <StateSummaryCard.Content>
              <MetadataItem
                label="Event name"
                value={
                  <>
                    <IconEvent className="inline-block" /> {config.eventName}
                  </>
                }
              />

              <MetadataItem
                label="Match expression"
                type="code"
                value={config.expression ?? 'N/A'}
              />

              <MetadataItem label="Timeout" value={config.timeout.toLocaleString()} />
            </StateSummaryCard.Content>
          </StateSummaryCard>
        );
      })}
    </>
  );
}

function useActiveWaits(history: Record<string, HistoryNode>): HistoryNode[] {
  const [waits, setWaits] = useState<HistoryNode[]>([]);

  useEffect(() => {
    const newWaits: HistoryNode[] = [];
    for (const node of Object.values(history)) {
      if (node.status === 'waiting') {
        newWaits.push(node);
      }
    }
    setWaits(newWaits);
  }, [history]);

  return waits;
}
