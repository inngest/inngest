import type React from 'react';
import { IconStatusCompleted } from '@inngest/components/icons/status/Completed';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';

import type { ReplayStatus } from '../types/replay';

const icons = {
  CREATED: IconStatusRunning,
  ENDED: IconStatusCompleted,
} as const satisfies { [key in ReplayStatus]: React.ComponentType };

type Props = {
  status: ReplayStatus;
  className?: string;
};

export function ReplayStatusIcon({ status, className }: Props) {
  const Icon = icons[status];

  const title = 'Replay ' + status.toLowerCase();
  return <Icon className={className} title={title} />;
}
