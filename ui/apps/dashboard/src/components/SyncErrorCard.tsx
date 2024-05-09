import { cn } from '@inngest/components/utils/classNames';
import { RiErrorWarningLine } from '@remixicon/react';

type Props = {
  className?: string;
  error: string;
};

export function SyncErrorCard({ className, error }: Props) {
  return (
    <div
      className={cn(
        'flex items-center gap-2 overflow-hidden rounded-lg border border-rose-500 bg-rose-100 px-4 py-2 text-rose-700',
        className
      )}
    >
      <RiErrorWarningLine className="h-4 w-4 text-rose-700" /> {error}
    </div>
  );
}
