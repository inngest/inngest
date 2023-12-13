import { IconStatusCircleArrowPath } from '@inngest/components/icons/StatusCircleArrowPath';
import { IconStatusCircleCheck } from '@inngest/components/icons/StatusCircleCheck';
import { IconStatusCircleCross } from '@inngest/components/icons/StatusCircleCross';
import { IconStatusCircleMinus } from '@inngest/components/icons/StatusCircleMinus';

import { Time } from '@/components/Time';

const syncStatuses = ['error', 'pending', 'success'] as const;
type SyncStatus = (typeof syncStatuses)[number];
function isSyncStatus(status: string): status is SyncStatus {
  return syncStatuses.includes(status as SyncStatus);
}

const syncStatusIcons = {
  error: IconStatusCircleCross,
  pending: IconStatusCircleArrowPath,
  success: IconStatusCircleCheck,
} as const satisfies { [key in SyncStatus]: React.ComponentType };

const syncStatusText = {
  error: 'Error',
  pending: 'Pending',
  success: 'Success',
} as const satisfies { [key in SyncStatus]: string };

type Props = {
  status: string;
};

export function SyncStatus({ status }: Props) {
  let Icon;
  let text: string;
  if (isSyncStatus(status)) {
    Icon = syncStatusIcons[status];
    text = syncStatusText[status];
  } else {
    Icon = IconStatusCircleMinus;
    text = 'Unknown';
  }

  return (
    <div className="flex">
      <Icon className="h-6 w-6" />
      {text}
    </div>
  );
}
