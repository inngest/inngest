import { type MetadataItemProps } from '@inngest/components/Metadata/MetadataItem';
import { type FunctionRun } from '@inngest/components/types/functionRun';
import { type FunctionVersion } from '@inngest/components/types/functionVersion';
import { formatMilliseconds, shortDate } from '@inngest/components/utils/date';
import type { HistoryParser } from '@inngest/components/utils/historyParser';

export function renderRunMetadata({
  functionRun,
  functionVersion,
  history,
}: {
  functionRun?: Pick<FunctionRun, 'endedAt' | 'id' | 'startedAt' | 'status'>;
  functionVersion?: Pick<FunctionVersion, 'url' | 'version'>;
  history?: HistoryParser;
}) {
  // The current startedAt is in reality the queuedAt timestamp. We are getting the real startedAt from the first history item
  const startedAt = history?.runStartedAt;
  const startedAtLabel = startedAt ? shortDate(new Date(startedAt)) : '-';

  let duration: number | undefined;
  if (startedAt && functionRun?.endedAt) {
    duration = new Date(functionRun?.endedAt).getTime() - new Date(startedAt).getTime();
  }

  let endedAtLabel = 'Function Completed';
  let tootltipLabel = 'completed';
  if (functionRun?.status === 'CANCELLED') {
    endedAtLabel = 'Function Cancelled';
    tootltipLabel = 'cancelled';
  } else if (functionRun?.status === 'FAILED') {
    endedAtLabel = 'Function Failed';
    tootltipLabel = 'failed';
  }

  const metadataItems: MetadataItemProps[] = [
    { label: 'Run ID', value: functionRun?.id ?? '', size: 'large', type: 'code' },
    {
      label: 'Function Queued',
      value: functionRun?.startedAt ? shortDate(new Date(functionRun?.startedAt)) : '-',
      title: functionRun?.startedAt?.toLocaleString(),
    },
    {
      label: 'Function Started',
      value: startedAtLabel,
      title: startedAt?.toLocaleString(),
    },
    {
      label: endedAtLabel,
      value: functionRun?.endedAt ? shortDate(new Date(functionRun?.endedAt)) : '-',
      title: functionRun?.endedAt?.toLocaleString(),
    },
    {
      label: 'Duration',
      value: duration ? formatMilliseconds(duration) : '-',
      tooltip: `Time between function started and function ${tootltipLabel}`,
    },
  ];

  if (functionVersion) {
    metadataItems.push(
      {
        label: 'URL',
        size: 'large',
        value: `${functionVersion.url}`,
      },
      {
        label: 'Function Version',
        value: `${functionVersion.version}`,
      }
    );
  }

  return metadataItems;
}
