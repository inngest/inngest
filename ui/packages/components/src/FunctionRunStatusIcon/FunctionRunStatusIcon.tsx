import type React from 'react';
import { IconStatusCircleCross } from '@inngest/components/icons/StatusCircleCross';
import { IconStatusCompleted } from '@inngest/components/icons/StatusCompleted';
import { IconStatusFailed } from '@inngest/components/icons/StatusFailed';
import { IconStatusQueued } from '@inngest/components/icons/StatusQueued';
import { IconStatusRunning } from '@inngest/components/icons/StatusRunning';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';

// Explicitly assign the Record type but use satisfies to ensure:
// - Accessing an unexpected status gives an undefined
// - Keys must be exhaustive of all known statuses
const icons: Record<string, React.ComponentType> = {
  CANCELLED: IconStatusFailed,
  COMPLETED: IconStatusCompleted,
  FAILED: IconStatusCircleCross,
  RUNNING: IconStatusRunning,
  QUEUED: IconStatusQueued,
} as const satisfies { [key in FunctionRunStatus]: React.ComponentType };

type Props = {
  status: FunctionRunStatus;
  className?: string;
};

export function FunctionRunStatusIcon({ status, className }: Props) {
  const Icon = icons[status] ?? IconStatusQueued;

  const title = 'Function ' + status.toLowerCase();
  return <Icon className={className} title={title} />;
}
