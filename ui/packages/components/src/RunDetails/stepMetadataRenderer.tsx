import { type MetadataItemProps } from '@inngest/components/Metadata/MetadataItem';
import { IconEvent } from '@inngest/components/icons/Event';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { type HistoryNode } from '@inngest/components/utils/historyParser';

export function renderStepMetadata({
  node,
  isAttempt,
}: {
  node: HistoryNode;
  isAttempt?: boolean;
}): MetadataItemProps[] {
  let endedAtLabel = 'Completed At:';
  let tootltipLabel = 'completed';
  if (node.status === 'cancelled') {
    endedAtLabel = 'Cancelled At:';
    tootltipLabel = 'cancelled';
  } else if (node.status === 'failed') {
    endedAtLabel = 'Failed At:';
    tootltipLabel = 'failed';
  } else if (node.status === 'errored') {
    endedAtLabel = 'Errored At:';
    tootltipLabel = 'errored';
  } else if (node.status === 'completed' && node.waitForEventResult?.timeout) {
    endedAtLabel = 'Timed Out At:';
    tootltipLabel = 'timed out';
  }

  let durationMS: number | undefined;
  if (node.scheduledAt && node.endedAt) {
    durationMS = node.endedAt.getTime() - node.scheduledAt.getTime();
  }

  const metadataItems: MetadataItemProps[] = [
    {
      label: 'Started At',
      value: node.scheduledAt ? node.scheduledAt.toLocaleString() : '-',
      title: node?.scheduledAt?.toLocaleString(),
    },
    {
      label: endedAtLabel,
      value: node.endedAt ? node.endedAt.toLocaleString() : '-',
      title: node?.endedAt?.toLocaleString(),
    },
    {
      label: 'Duration',
      value: durationMS ? formatMilliseconds(durationMS) : '-',
      tooltip: `Time between ${isAttempt ? 'attempt' : 'step'} started and ${
        isAttempt ? 'attempt' : 'step'
      } ${tootltipLabel}`,
    },
  ];

  if (node.sleepConfig?.until) {
    metadataItems.push({
      label: 'Sleep Until',
      value: node.sleepConfig?.until?.toLocaleString(),
    });
  }

  if (node.waitForEventConfig) {
    metadataItems.push(
      {
        label: 'Event Name',
        value: (
          <>
            <IconEvent className="inline-block" /> {node.waitForEventConfig.eventName}
          </>
        ),
      },
      {
        label: 'Match Expression',
        value: node.waitForEventConfig.expression ?? 'N/A',
        type: 'code',
      }
    );
  }

  return metadataItems;
}
