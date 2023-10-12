import {
  IconStatusCircleArrowPath,
  IconStatusCircleCheck,
  IconStatusCircleCross,
  IconStatusCircleMinus,
} from '@/icons';
import { FunctionRunStatus } from '@/store/generated';

type FunctionRunStatusIconsProps = {
  status: FunctionRunStatus;
  className?: string;
};

const functionRunStatusIcons = {
  [FunctionRunStatus.Running]: IconStatusCircleArrowPath,
  [FunctionRunStatus.Completed]: IconStatusCircleCheck,
  [FunctionRunStatus.Failed]: IconStatusCircleCross,
  [FunctionRunStatus.Cancelled]: IconStatusCircleMinus,
} as const satisfies Record<FunctionRunStatus, React.ComponentType>;

export function FunctionRunStatusIcons({ status, className }: FunctionRunStatusIconsProps) {
  const FunctionRunStatusIcon = functionRunStatusIcons[status];
  const title = 'Function ' + status.toLowerCase();
  return <FunctionRunStatusIcon className={className} title={title} />;
}
