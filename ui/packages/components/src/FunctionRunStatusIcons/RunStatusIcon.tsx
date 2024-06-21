import type React from 'react';
import { IconStatusCanceled } from '@inngest/components/icons/status/Canceled';
import { IconStatusCompleted } from '@inngest/components/icons/status/Completed';
import { IconStatusFailed } from '@inngest/components/icons/status/Failed';
import { IconStatusQueued } from '@inngest/components/icons/status/Queued';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';

const icons: Record<string, React.ComponentType> = {
  CANCELLED: IconStatusCanceled,
  COMPLETED: IconStatusCompleted,
  FAILED: IconStatusFailed,
  RUNNING: IconStatusRunning,
  QUEUED: IconStatusQueued,
} as const;

type Props = {
  status: string;
  className?: string;
};

export function RunStatusIcon({ status, className }: Props) {
  const Icon = icons[status] ?? IconStatusQueued;

  const title = status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
  return <Icon className={className} title={title} />;
}
