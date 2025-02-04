import { Pill } from '@inngest/components/Pill/Pill';
import { type AppKind } from '@inngest/components/types/app';
import { type SyncStatus } from '@inngest/components/types/sync';

export const syncStatusText: Record<string, string> = {
  duplicate: 'No change',
  error: 'Error',
  pending: 'Syncing',
  success: 'Success',
} as const satisfies { [key in SyncStatus]: unknown };

export const syncKind: Record<string, AppKind> = {
  duplicate: 'primary',
  error: 'error',
  pending: 'info',
  success: 'primary',
} as const satisfies { [key in SyncStatus]: unknown };

type Props = {
  status: string;
  iconOnly?: boolean;
};

export function SyncStatusPill({ status }: Props) {
  const text = syncStatusText[status] ?? 'Unknown';
  const kind = syncKind[status] ?? 'default';

  return (
    <Pill appearance="outlined" kind={kind}>
      {text}
    </Pill>
  );
}
