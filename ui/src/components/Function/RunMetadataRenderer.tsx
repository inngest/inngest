import { type MetadataItemProps } from '@/components/Metadata/MetadataItem';
import { FunctionRunStatus, type FunctionRun } from '@/store/generated';
import { formatMilliseconds, shortDate } from '@/utils/date';

export default function renderRunMetadata(
  functionRun: Pick<FunctionRun, 'finishedAt' | 'id' | 'startedAt' | 'status' | 'history'>,
): MetadataItemProps[] {
  if (!functionRun.startedAt) {
    throw new Error('missing startedAt');
  }
  if (!functionRun.history) {
    throw new Error('missing history');
  }
  // The current startedAt is in reality the queuedAt timestamp. We are getting the real startedAt from the first history item
  const functionStartedTimestamp = functionRun?.history?.[0]?.createdAt;
  const startedAtLabel = functionStartedTimestamp
    ? shortDate(new Date(functionStartedTimestamp))
    : '-';
  const metadataItems: MetadataItemProps[] = [
    { label: 'Run ID', value: functionRun.id, size: 'large', type: 'code' },
    {
      label: 'Function Scheduled',
      value: shortDate(new Date(functionRun.startedAt)),
      title: functionRun.startedAt.toLocaleString(),
    },
    {
      label: 'Function Started',
      value: startedAtLabel,
      title: functionStartedTimestamp.toLocaleString(),
    },
  ];

  if (functionRun.status == FunctionRunStatus.Completed) {
    if (!functionRun.finishedAt) {
      throw new Error('missing finishedAt');
    }
    metadataItems.push({
      label: 'Function Completed',
      value: shortDate(new Date(functionRun.finishedAt)),
      title: functionRun.finishedAt.toLocaleString(),
    });
    if (functionStartedTimestamp && functionRun.finishedAt) {
      const duration =
        new Date(functionRun.finishedAt).getTime() - new Date(functionStartedTimestamp).getTime();
      metadataItems.push({
        label: 'Duration',
        value: formatMilliseconds(duration),
        tooltip: 'Time between function started and function completed',
      });
    }
  }
  if (functionRun.status == FunctionRunStatus.Failed) {
    if (!functionRun.finishedAt) {
      throw new Error('missing finishedAt');
    }
    metadataItems.push({
      label: 'Function Failed',
      value: shortDate(new Date(functionRun.finishedAt)),
      title: functionRun.finishedAt.toLocaleString(),
    });
    if (functionStartedTimestamp && functionRun.finishedAt) {
      const duration =
        new Date(functionRun.finishedAt).getTime() - new Date(functionStartedTimestamp).getTime();
      metadataItems.push({
        label: 'Duration',
        value: formatMilliseconds(duration),
        tooltip: 'Time between function started and function failed',
      });
    }
  }
  if (functionRun.status == FunctionRunStatus.Cancelled) {
    if (!functionRun.finishedAt) {
      throw new Error('missing finishedAt');
    }
    metadataItems.push({
      label: 'Function Cancelled',
      value: shortDate(new Date(functionRun.finishedAt)),
      title: functionRun.finishedAt.toLocaleString(),
    });
    if (functionStartedTimestamp && functionRun.finishedAt) {
      const duration =
        new Date(functionRun.finishedAt).getTime() - new Date(functionStartedTimestamp).getTime();
      metadataItems.push({
        label: 'Duration',
        value: formatMilliseconds(duration),
        tooltip: 'Time between function started and function cancelled',
      });
    }
  }
  if (functionRun.status == FunctionRunStatus.Running) {
    metadataItems.push(
      {
        label: 'Function Completed',
        value: '-',
      },
      {
        label: 'Duration',
        value: '-',
      },
    );
  }

  return metadataItems;
}
