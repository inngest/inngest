import { ArrowPathIcon, CheckIcon, MinusIcon, XMarkIcon } from '@heroicons/react/20/solid';
import { classNames } from '@inngest/components/utils/classNames';

const syncStatuses = ['error', 'pending', 'success'] as const;
type SyncStatus = (typeof syncStatuses)[number];
function isSyncStatus(status: string): status is SyncStatus {
  return syncStatuses.includes(status as SyncStatus);
}

const syncStatusIcons = {
  error: XMarkIcon,
  pending: ArrowPathIcon,
  success: CheckIcon,
} as const satisfies { [key in SyncStatus]: React.ComponentType };

const syncStatusText = {
  error: 'Error',
  pending: 'Pending',
  success: 'Success',
} as const satisfies { [key in SyncStatus]: string };

const syncStatusColor = {
  error: 'text-red-300',
  pending: 'text-sky-300',
  success: 'text-emerald-300',
} as const satisfies { [key in SyncStatus]: string };

type Props = {
  status: string;
};

export function SyncStatus({ status }: Props) {
  let Icon;
  let text: string;
  let color: string;
  if (isSyncStatus(status)) {
    Icon = syncStatusIcons[status];
    text = syncStatusText[status];
    color = syncStatusColor[status];
  } else {
    Icon = MinusIcon;
    text = 'Unknown';
    color = 'text-slate-100';
  }

  return (
    <div
      className={classNames(
        color,
        'py-.5 flex items-center gap-1.5 whitespace-nowrap rounded-full bg-slate-800 px-2'
      )}
    >
      <Icon className="h-4 w-4" />
      {text}
    </div>
  );
}
