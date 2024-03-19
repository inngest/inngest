import { CheckIcon, ExclamationTriangleIcon, MinusIcon } from '@heroicons/react/20/solid';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';
import { type SyncStatus } from '@inngest/components/types/sync';
import { cn } from '@inngest/components/utils/classNames';

const syncStatusIcons: Record<string, React.ComponentType> = {
  duplicate: CheckIcon,
  error: ExclamationTriangleIcon,
  pending: IconStatusRunning,
  success: CheckIcon,
} as const satisfies { [key in SyncStatus]: unknown };

const syncStatusText: Record<string, string> = {
  duplicate: 'No change',
  error: 'Error',
  pending: 'Syncing',
  success: 'Success',
} as const satisfies { [key in SyncStatus]: unknown };

const syncStatusColor: Record<string, string> = {
  duplicate: 'bg-teal-100 text-teal-800 border-teal-500',
  error: 'bg-rose-100 text-rose-800 border-rose-500',
  pending: 'bg-sky-100 text-sky-800 border-sky-500',
  success: 'bg-teal-100 text-teal-800 border-teal-500',
} as const satisfies { [key in SyncStatus]: unknown };

const syncStatusIconColor: Record<string, string> = {
  duplicate: 'text-teal-500',
  error: 'text-rose-700',
  pending: 'text-sky-500',
  success: 'text-teal-500',
} as const satisfies { [key in SyncStatus]: unknown };

type Props = {
  status: string;
  iconOnly?: boolean;
};

export function SyncStatusPill({ status, iconOnly = false }: Props) {
  const Icon = syncStatusIcons[status] ?? MinusIcon;
  const text = syncStatusText[status] ?? 'Unknown';
  const color = syncStatusColor[status] ?? 'bg-slate-100 text-slate-800 border-slate-500';
  const iconColor = syncStatusIconColor[status] ?? 'text-slate-500';

  return (
    <div
      className={cn(
        color,
        iconOnly ? 'px-1.5' : 'px-3',
        'flex h-8 w-fit items-center gap-2 whitespace-nowrap rounded-full border'
      )}
    >
      <Icon className={cn(iconColor, 'h-4 w-4')} title={text} />
      {!iconOnly && text}
    </div>
  );
}
