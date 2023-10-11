import { capitalCase } from 'change-case';

import { FunctionRunStatus } from '@/gql/graphql';
import CancelledIcon from '@/icons/status-icons/cancelled.svg';
import CompletedIcon from '@/icons/status-icons/completed.svg';
import FailedIcon from '@/icons/status-icons/failed.svg';
import RunningIcon from '@/icons/status-icons/running.svg';
import cn from '@/utils/cn';

const backgroundColorStyles = {
  [FunctionRunStatus.Cancelled]: 'bg-gray-100',
  [FunctionRunStatus.Completed]: 'bg-teal-100',
  [FunctionRunStatus.Failed]: 'bg-red-100',
  [FunctionRunStatus.Running]: 'bg-sky-100',
} as const;

const labelColorStyles = {
  [FunctionRunStatus.Cancelled]: 'text-gray-900',
  [FunctionRunStatus.Completed]: 'text-teal-900',
  [FunctionRunStatus.Failed]: 'text-red-900',
  [FunctionRunStatus.Running]: 'text-sky-900',
} as const;

const icons = {
  [FunctionRunStatus.Cancelled]: <CancelledIcon className="h-4 w-4 shrink-0" />,
  [FunctionRunStatus.Completed]: <CompletedIcon className="h-4 w-4 shrink-0" />,
  [FunctionRunStatus.Failed]: <FailedIcon className="h-4 w-4 shrink-0" />,
  [FunctionRunStatus.Running]: <RunningIcon className="h-4 w-4 shrink-0" />,
} as const;

type FunctionRunStatusCardProps = {
  status: FunctionRunStatus;
};
export default function FunctionRunStatusCard({ status }: FunctionRunStatusCardProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-center gap-1 rounded-md px-3 py-1.5',
        backgroundColorStyles[status]
      )}
    >
      {icons[status]}
      <span className={cn('text-sm text-teal-900', labelColorStyles[status])}>
        {capitalCase(status)}
      </span>
    </div>
  );
}
