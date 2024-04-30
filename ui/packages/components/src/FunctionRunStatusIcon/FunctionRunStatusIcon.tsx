import type React from 'react';
import { IconStatusCanceled } from '@inngest/components/icons/status/Canceled';
import { IconStatusCompleted } from '@inngest/components/icons/status/Completed';
import { IconStatusFailed } from '@inngest/components/icons/status/Failed';
import { IconStatusQueued } from '@inngest/components/icons/status/Queued';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';

// Explicitly assign the Record type but use satisfies to ensure:
// - Accessing an unexpected status gives an undefined
// - Keys must be exhaustive of all known statuses
const icons: Record<string, React.ComponentType> = {
  CANCELLED: IconStatusCanceled,
  COMPLETED: IconStatusCompleted,
  FAILED: IconStatusFailed,
  RUNNING: IconStatusRunning,
  QUEUED: IconStatusQueued,
} as const satisfies { [key in FunctionRunStatus]: React.ComponentType };

type Props = {
  status: FunctionRunStatus;
  className?: string;
};

/** @deprecated For new designs use RunStatusIcons instead. */
export function FunctionRunStatusIcon({ status, className }: Props) {
  const Icon = icons[status] ?? IconStatusQueued;

  const title = 'Function ' + status.toLowerCase();
  return <Icon className={className} title={title} />;
}
