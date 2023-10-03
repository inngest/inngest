import { type MetadataItemProps } from '@/components/Metadata/MetadataItem';
import { FunctionRunStatus } from '@/store/generated';
import { formatMilliseconds, shortDate } from '@/utils/date';

type Run = {
  finishedAt?: string;
  id: string;
  startedAt?: string;
  status?: FunctionRunStatus | null;
};

export default function renderRunMetadata(functionRun: Run): MetadataItemProps[] {
  if (!functionRun.startedAt) {
    throw new Error('missing startedAt');
  }
  const metadataItems: MetadataItemProps[] = [
    { label: 'Run ID', value: functionRun.id, size: 'large', type: 'code' },
    { label: 'Function Started', value: shortDate(new Date(functionRun.startedAt)) },
  ];
  if (functionRun.status == FunctionRunStatus.Completed) {
    if (!functionRun.finishedAt) {
      throw new Error('missing finishedAt');
    }
    metadataItems.push({
      label: 'Function Completed',
      value: shortDate(new Date(functionRun.finishedAt)),
    });
    if (functionRun.startedAt && functionRun.finishedAt) {
      const duration =
        new Date(functionRun.finishedAt).getTime() - new Date(functionRun.startedAt).getTime();
      metadataItems.push({ label: 'Duration', value: formatMilliseconds(duration) });
    }
  }
  if (functionRun.status == FunctionRunStatus.Failed) {
    if (!functionRun.finishedAt) {
      throw new Error('missing finishedAt');
    }
    metadataItems.push({
      label: 'Function Failed',
      value: shortDate(new Date(functionRun.finishedAt)),
    });
    if (functionRun.startedAt && functionRun.finishedAt) {
      const duration =
        new Date(functionRun.finishedAt).getTime() - new Date(functionRun.startedAt).getTime();
      metadataItems.push({ label: 'Duration', value: formatMilliseconds(duration) });
    }
  }
  if (functionRun.status == FunctionRunStatus.Cancelled) {
    if (!functionRun.finishedAt) {
      throw new Error('missing finishedAt');
    }
    metadataItems.push({
      label: 'Function Cancelled',
      value: shortDate(new Date(functionRun.finishedAt)),
    });
  }
  if (functionRun.status == FunctionRunStatus.Running) {
    metadataItems.push({
      label: 'Function Completed',
      value: '-',
    });
  }

  return metadataItems;
}
