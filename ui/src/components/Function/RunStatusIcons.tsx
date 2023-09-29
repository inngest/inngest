import {
  IconStatusCircleArrowPath,
  IconStatusCircleCheck,
  IconStatusCircleCross,
  IconStatusCircleMinus,
  IconStatusCircleMoon,
} from '@/icons';
import { FunctionRunStatus } from '@/store/generated';
import { FunctionRunExtraStatus } from '@/utils/constants';

type FunctionRunStatusIconsProps = {
  status: FunctionRunStatus | FunctionRunExtraStatus;
  className?: string;
};

const functionRunStatusIcons = {
  [FunctionRunStatus.Running]: IconStatusCircleArrowPath,
  [FunctionRunStatus.Completed]: IconStatusCircleCheck,
  [FunctionRunStatus.Failed]: IconStatusCircleCross,
  [FunctionRunStatus.Cancelled]: IconStatusCircleMinus,
  [FunctionRunExtraStatus.WaitingFor]: IconStatusCircleMoon,
  [FunctionRunExtraStatus.Sleeping]: IconStatusCircleMoon,
} as const satisfies Record<FunctionRunStatus | FunctionRunExtraStatus, React.ComponentType>;

export function FunctionRunStatusIcons({ status, className }: FunctionRunStatusIconsProps) {
  const FunctionRunStatusIcon = functionRunStatusIcons[status];

  return <FunctionRunStatusIcon className={className} />;
}
