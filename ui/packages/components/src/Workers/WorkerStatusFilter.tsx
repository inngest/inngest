import StatusFilter from '../Filter/StatusFilter';
import {
  convertWorkerStatus,
  groupedWorkerStatuses,
  isWorkerStatus,
  type GroupedWorkerStatus,
  type WorkerStatus,
} from '../types/workers';

type Props = {
  selectedStatuses: WorkerStatus[];
  onStatusesChange: (value: GroupedWorkerStatus[]) => void;
};

export default function WorkerStatusFilter({ selectedStatuses, onStatusesChange }: Props) {
  // Convert selectedStatuses from WorkerStatus[] to GroupedWorkerStatus[]
  const convertedStatuses = selectedStatuses.map(convertWorkerStatus);

  return (
    <StatusFilter
      selectedStatuses={convertedStatuses}
      onStatusesChange={onStatusesChange}
      availableStatuses={[...groupedWorkerStatuses]}
      isValidStatus={isWorkerStatus}
    />
  );
}
