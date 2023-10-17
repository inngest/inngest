import { type MetadataItemProps } from '@inngest/components/Metadata/MetadataItem';
import { type FunctionRun } from '@inngest/components/types/functionRun';
import { formatMilliseconds, shortDate } from '@inngest/components/utils/date';
import type { RawHistoryItem } from '@inngest/components/utils/historyParser';

export function renderRunMetadata(
  functionRun: Pick<FunctionRun, 'endedAt' | 'id' | 'startedAt' | 'status'>,
  history: RawHistoryItem[]
): MetadataItemProps[] {
  if (!functionRun.startedAt) {
    throw new Error('missing startedAt');
  }

  // The current startedAt is in reality the queuedAt timestamp. We are getting the real startedAt from the first history item
  const startedAt = history[0]?.createdAt;
  const startedAtLabel = startedAt ? shortDate(new Date(startedAt)) : '-';
  const metadataItems: MetadataItemProps[] = [
    { label: 'Run ID', value: functionRun.id, size: 'large', type: 'code' },
    {
      label: 'Function Scheduled',
      value: shortDate(new Date(functionRun.startedAt)),
      title: functionRun.startedAt.toLocaleString(),
    },
  ];

  if (startedAt) {
    metadataItems.push({
      label: 'Function Started',
      value: startedAtLabel,
      title: startedAt.toLocaleString(),
    });
  }

  if (functionRun.status === 'COMPLETED') {
    if (!functionRun.endedAt) {
      throw new Error('missing endedAt');
    }
    metadataItems.push({
      label: 'Function Completed',
      value: shortDate(new Date(functionRun.endedAt)),
      title: functionRun.endedAt.toLocaleString(),
    });
    if (startedAt && functionRun.endedAt) {
      const duration = new Date(functionRun.endedAt).getTime() - new Date(startedAt).getTime();
      metadataItems.push({
        label: 'Duration',
        value: formatMilliseconds(duration),
        tooltip: 'Time between function started and function completed',
      });
    }
  }
  if (functionRun.status === 'FAILED') {
    if (!functionRun.endedAt) {
      throw new Error('missing endedAt');
    }
    metadataItems.push({
      label: 'Function Failed',
      value: shortDate(new Date(functionRun.endedAt)),
      title: functionRun.endedAt.toLocaleString(),
    });
    if (startedAt && functionRun.endedAt) {
      const duration = new Date(functionRun.endedAt).getTime() - new Date(startedAt).getTime();
      metadataItems.push({
        label: 'Duration',
        value: formatMilliseconds(duration),
        tooltip: 'Time between function started and function failed',
      });
    }
  }
  if (functionRun.status === 'CANCELLED') {
    if (!functionRun.endedAt) {
      throw new Error('missing endedAt');
    }
    metadataItems.push({
      label: 'Function Cancelled',
      value: shortDate(new Date(functionRun.endedAt)),
      title: functionRun.endedAt.toLocaleString(),
    });
    if (startedAt && functionRun.endedAt) {
      const duration = new Date(functionRun.endedAt).getTime() - new Date(startedAt).getTime();
      metadataItems.push({
        label: 'Duration',
        value: formatMilliseconds(duration),
        tooltip: 'Time between function started and function cancelled',
      });
    }
  }
  if (functionRun.status === 'RUNNING') {
    metadataItems.push(
      {
        label: 'Function Completed',
        value: '-',
      },
      {
        label: 'Duration',
        value: '-',
      }
    );
  }

  return metadataItems;
}
