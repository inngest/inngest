import type React from 'react';
import { IconStatusCancelled } from '@inngest/components/icons/status/Cancelled';
import { IconStatusCompleted } from '@inngest/components/icons/status/Completed';
import { IconStatusFailed } from '@inngest/components/icons/status/Failed';
import { IconStatusPaused } from '@inngest/components/icons/status/Paused';
import { IconStatusQueued } from '@inngest/components/icons/status/Queued';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';
import { IconStatusSkipped } from '@inngest/components/icons/status/Skipped';

import { getStatusTextClass } from '../Status/statusClasses';
import { statusTitles } from '../Status/statusTitles';
import { cn } from '../utils/classNames';

const icons: Record<string, React.ComponentType> = {
  CANCELED: IconStatusCancelled,
  CANCELLED: IconStatusCancelled,
  COMPLETED: IconStatusCompleted,
  FAILED: IconStatusFailed,
  RUNNING: IconStatusRunning,
  QUEUED: IconStatusQueued,
  SKIPPED_PAUSED: IconStatusSkipped,
  PAUSED: IconStatusPaused,
} as const;

type Props = {
  status: string;
  className?: string;
};

export function RunStatusIcon({ status, className }: Props) {
  const txtClass = getStatusTextClass(status);
  const title = statusTitles[status] || 'Unknown';
  const Icon = icons[status] ?? IconStatusQueued;
  return <Icon className={cn('h-6 w-6', txtClass, className)} title={title} />;
}
