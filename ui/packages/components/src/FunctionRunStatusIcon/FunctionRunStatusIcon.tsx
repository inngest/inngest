import type React from 'react';
import { IconStatusCircleArrowPath } from '@inngest/components/icons/StatusCircleArrowPath';
import { IconStatusCircleCheck } from '@inngest/components/icons/StatusCircleCheck';
import { IconStatusCircleCross } from '@inngest/components/icons/StatusCircleCross';
import { IconStatusCircleMinus } from '@inngest/components/icons/StatusCircleMinus';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';

// Explicitly assign the Record type but use satisfies to ensure:
// - Accessing an unexpected status gives an undefined
// - Keys must be exhaustive of all known statuses
const icons: Record<string, React.ComponentType> = {
  CANCELLED: IconStatusCircleMinus,
  COMPLETED: IconStatusCircleCheck,
  FAILED: IconStatusCircleCross,
  RUNNING: IconStatusCircleArrowPath,
} as const satisfies { [key in FunctionRunStatus]: React.ComponentType };

type Props = {
  status: FunctionRunStatus;
  className?: string;
};

export function FunctionRunStatusIcon({ status, className }: Props) {
  const Icon = icons[status] ?? IconStatusCircleArrowPath;

  const title = 'Function ' + status.toLowerCase();
  return <Icon className={className} title={title} />;
}
