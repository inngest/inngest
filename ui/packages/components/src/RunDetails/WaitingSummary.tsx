'use client';

import { useEffect, useState } from 'react';
import { Card } from '@inngest/components/Card';
import { MetadataItem } from '@inngest/components/Metadata';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import type { HistoryNode, HistoryParser } from '@inngest/components/utils/historyParser';

type Props = {
  history: HistoryParser;
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
          <Card
            accentColor="bg-status-running"
            className={i < waits.length - 1 ? 'mb-4' : undefined}
            key={wait.groupID}
          >
            <Card.Header>Waiting for event</Card.Header>

            <Card.Content>
              <MetadataItem
                label="Event Name"
                value={
                  <>
                    <EventsIcon className="inline-block h-4 w-4" /> {config.eventName}
                  </>
                }
              />

              <MetadataItem
                label="Match Expression"
                type="code"
                value={config.expression ?? 'N/A'}
              />

              <MetadataItem label="Timeout" value={config.timeout.toLocaleString()} />
            </Card.Content>
          </Card>
        );
      })}
    </>
  );
}

function useActiveWaits(history: HistoryParser): HistoryNode[] {
  const [waits, setWaits] = useState<HistoryNode[]>([]);

  useEffect(() => {
    const newWaits: HistoryNode[] = [];
    for (const node of history.getGroups()) {
      if (node.status === 'waiting') {
        newWaits.push(node);
      }
    }
    setWaits(newWaits);
  }, [history]);

  return waits;
}
