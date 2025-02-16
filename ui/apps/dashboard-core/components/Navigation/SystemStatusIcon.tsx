import { cn } from '@inngest/components/utils/classNames';

import type { Status } from '../Support/Status';

type SystemStatusIconProps = {
  className?: string;
  status: Status;
};

export default function SystemStatusIcon({ className, status }: SystemStatusIconProps) {
  return (
    <span
      className={cn('mx-1 inline-flex h-2 w-2 rounded-full', className)}
      style={{ backgroundColor: status.indicatorColor }}
      title={`Status updated at ${status.updated_at}`}
    />
  );
}
