import { type MetadataItemProps } from '@/components/Metadata/MetadataItem';
import { FunctionRunStatus } from '@/store/generated';
import { formatMilliseconds, shortDate } from '@/utils/date';

export default function renderRunMetadata(functionRun) {
  const metadataItems = [
    { label: 'Run ID', value: functionRun.id, size: 'large', type: 'code' },
    { label: 'Function Started', value: shortDate(new Date(functionRun.startedAt)) },
  ];
  if (functionRun.status == FunctionRunStatus.Completed) {
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

  return metadataItems as MetadataItemProps[];
}
