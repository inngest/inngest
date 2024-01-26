import type React from 'react';
import { IconStatusCircleArrowPath } from '@inngest/components/icons/StatusCircleArrowPath';
import { IconStatusCircleCheck } from '@inngest/components/icons/StatusCircleCheck';

import type { ReplayStatus } from '../types/replay';

const icons = {
  CREATED: IconStatusCircleArrowPath,
  ENDED: IconStatusCircleCheck,
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
