import StatusFilter from '@inngest/components/Filter/StatusFilter';
import {
  groupedWorkerStatuses,
  isWorkerStatus,
  type GroupedWorkerStatus,
  type WorkerStatus,
} from '@inngest/components/types/workers';
import { convertWorkerStatus } from '@inngest/components/utils/workerParser';

type Props = {
  selectedStatuses: WorkerStatus[];
  onStatusesChange: (value: GroupedWorkerStatus[]) => void;
};

export default function WorkerStatusFilter({ selectedStatuses, onStatusesChange }: Props) {
  // Convert selectedStatuses from WorkerStatus[] to unique GroupedWorkerStatus[]
  const convertedStatuses = Array.from(new Set(selectedStatuses.map(convertWorkerStatus)));

  return (
    <StatusFilter
      selectedStatuses={convertedStatuses}
      onStatusesChange={onStatusesChange}
      availableStatuses={[...groupedWorkerStatuses]}
      isValidStatus={isWorkerStatus}
    />
  );
}
