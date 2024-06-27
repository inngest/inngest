'use client';

import { Card } from '@inngest/components/Card';
import { MetadataItem } from '@inngest/components/Metadata';
import { IconEvent } from '@inngest/components/icons/Event';
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
      <Card.Header className="bg-white dark:bg-slate-800/40">Cancelled</Card.Header>

      <Card.Content className="bg-white dark:bg-slate-800/40">
        {/* TODO: Make this a link */}
        <MetadataItem
          label="Event ID"
          value={
            <>
              <IconEvent className="inline-block" /> {cancellation.eventID}
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
