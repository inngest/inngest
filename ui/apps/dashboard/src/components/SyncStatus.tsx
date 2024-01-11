import { CheckIcon, ExclamationTriangleIcon, MinusIcon } from '@heroicons/react/20/solid';
import { IconHourglass } from '@inngest/components/icons/Hourglass';
import { classNames } from '@inngest/components/utils/classNames';

const syncStatuses = ['error', 'pending', 'success'] as const;
type SyncStatus = (typeof syncStatuses)[number];
function isSyncStatus(status: string): status is SyncStatus {
  return syncStatuses.includes(status as SyncStatus);
}

const syncStatusIcons = {
  error: ExclamationTriangleIcon,
  pending: IconHourglass,
  success: CheckIcon,
} as const satisfies { [key in SyncStatus]: React.ComponentType };

const syncStatusText = {
  error: 'Error',
  pending: 'Syncing',
  success: 'Success',
} as const satisfies { [key in SyncStatus]: string };

const syncStatusColor = {
  error: 'bg-red-100 text-red-800 border-red-500',
  pending: 'bg-sky-100 text-sky-800 border-sky-500',
  success: 'bg-teal-100 text-teal-800 border-teal-500',
} as const satisfies { [key in SyncStatus]: string };

const syncStatusIconColor = {
  error: 'text-red-500',
  pending: 'text-sky-500',
  success: 'text-teal-500',
} as const satisfies { [key in SyncStatus]: string };

type Props = {
  status: string;
};

export function SyncStatus({ status }: Props) {
  let Icon;
  let text: string;
  let color: string;
  let iconColor: string;
  if (isSyncStatus(status)) {
    Icon = syncStatusIcons[status];
    text = syncStatusText[status];
    color = syncStatusColor[status];
    iconColor = syncStatusIconColor[status];
  } else {
    Icon = MinusIcon;
    text = 'Unknown';
    color = 'bg-slate-100 text-slate-800 border-slate-500';
    iconColor = 'text-slate-500';
  }

  return (
    <div
      className={classNames(
        color,
        'flex w-fit items-center gap-2 whitespace-nowrap rounded-full border px-3 py-1'
      )}
    >
      <Icon className={classNames(iconColor, 'h-4 w-4')} />
      {text}
    </div>
  );
}
