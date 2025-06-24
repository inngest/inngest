import StatusFilter from '../Filter/StatusFilter';
import {
  functionRunStatuses,
  isFunctionRunStatus,
  type FunctionRunStatus,
} from '../types/functionRun';

type Props = {
  selectedStatuses: FunctionRunStatus[];
  onStatusesChange: (value: FunctionRunStatus[]) => void;
  functionIsPaused?: boolean;
};

export default function RunsStatusFilter({
  functionIsPaused,
  selectedStatuses,
  onStatusesChange,
}: Props) {
  const availableStatuses: FunctionRunStatus[] = functionRunStatuses.filter((status) => {
    if (status === 'PAUSED') {
      return !!functionIsPaused;
    } else if (status === 'RUNNING') {
      return !functionIsPaused;
      // Hide skipped runs from filter
    } else if (status === 'SKIPPED') {
      return false;
    } else if (status === 'WAITING') {
      return false;
    }
    return true;
  });

  return (
    <StatusFilter
      selectedStatuses={selectedStatuses}
      onStatusesChange={onStatusesChange}
      availableStatuses={availableStatuses}
      isValidStatus={isFunctionRunStatus}
    />
  );
}
