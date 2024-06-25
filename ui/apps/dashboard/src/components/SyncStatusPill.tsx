import { IconStatusRunning } from '@inngest/components/icons/status/Running';
import { type SyncStatus } from '@inngest/components/types/sync';
import { cn } from '@inngest/components/utils/classNames';
import { RiCheckLine, RiErrorWarningLine, RiSubtractLine } from '@remixicon/react';

const syncStatusIcons: Record<string, React.ComponentType> = {
  duplicate: RiCheckLine,
  error: RiErrorWarningLine,
  pending: IconStatusRunning,
  success: RiCheckLine,
} as const satisfies { [key in SyncStatus]: unknown };

const syncStatusText: Record<string, string> = {
  duplicate: 'No Change',
  error: 'Error',
  pending: 'Syncing',
  success: 'Success',
} as const satisfies { [key in SyncStatus]: unknown };

const syncStatusColor: Record<string, string> = {
  duplicate: 'bg-canvasBase text-status-completedText border-status-completed',
  error: 'bg-canvasBase text-status-failedText border-status-failed',
  pending: 'bg-canvasBase text-status-runningText border-status-running',
  success: 'bg-canvasBase text-status-completedText border-status-completed',
} as const satisfies { [key in SyncStatus]: unknown };

const syncStatusIconColor: Record<string, string> = {
  duplicate: 'text-status-completedText',
  error: 'text-status-failedText',
  pending: 'text-status-runningText',
  success: 'text-status-completedText',
} as const satisfies { [key in SyncStatus]: unknown };

type Props = {
  status: string;
  iconOnly?: boolean;
};

export function SyncStatusPill({ status, iconOnly = false }: Props) {
  const Icon = syncStatusIcons[status] ?? RiSubtractLine;
  const text = syncStatusText[status] ?? 'Unknown';
  const color =
    syncStatusColor[status] ?? 'bg-canvasBase text-status-cancelledText border-status-cancelled';
  const iconColor = syncStatusIconColor[status] ?? 'text-status-cancelledText';

  const iconProps = {
    className: cn(iconColor, 'h-4 w-4'),
    title: text,
  };

  return (
    <div
      className={cn(
        color,
        iconOnly ? 'px-1.5' : 'px-3',
        'flex h-8 w-fit items-center gap-2 whitespace-nowrap rounded-full border'
      )}
    >
      <Icon {...iconProps} />
      {!iconOnly && text}
    </div>
  );
}
