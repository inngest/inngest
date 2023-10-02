import { type MetadataItemProps } from '@/components/Metadata/MetadataItem';
import { FunctionRunStatus } from '@/store/generated';
import { shortDate } from '@/utils/date';

export default function renderRunMetadata(functionRun) {
  const metadataItems = [
    { label: 'Run ID', value: functionRun.id, size: 'large' },
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
      metadataItems.push({ label: 'Duration', value: duration.toString() + 'ms' });
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
      metadataItems.push({ label: 'Duration', value: duration.toString() + 'ms' });
    }
  }
  if (functionRun.status == FunctionRunStatus.Cancelled) {
    metadataItems.push({
      label: 'Function Cancelled',
      value: shortDate(new Date(functionRun.finishedAt)),
    });
  }

  return metadataItems as MetadataItemProps[];
}
