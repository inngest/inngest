'use client';

import { Card } from '@inngest/components/Card';
import { MetadataItem } from '@inngest/components/Metadata';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import type { HistoryParser } from '@inngest/components/utils/historyParser';

type Props = {
  history: HistoryParser;
};

export function CancellationSummary({ history }: Props) {
  const { cancellation } = history;
  if (!cancellation) {
    return null;
  }

  return (
    <Card accentColor="bg-status-cancelled">
      <Card.Header>Cancelled</Card.Header>

      <Card.Content>
        {/* TODO: Make this a link */}
        <MetadataItem
          label="Event ID"
          value={
            <>
              <EventsIcon className="inline-block h-4 w-4" /> {cancellation.eventID}
            </>
          }
        />

        <MetadataItem
          label="Match Expression"
          type="code"
          value={cancellation.expression ?? 'N/A'}
        />
      </Card.Content>
    </Card>
  );
}
