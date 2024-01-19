import { ExclamationTriangleIcon } from '@heroicons/react/20/solid';
import { classNames } from '@inngest/components/utils/classNames';

type Props = {
  className?: string;
  error: string;
};

export function SyncErrorCard({ className, error }: Props) {
  return (
    <div
      className={classNames(
        'flex items-center gap-2 overflow-hidden rounded-lg border border-red-500 bg-red-100 px-4 py-2 text-red-800',
        className
      )}
    >
      <ExclamationTriangleIcon className="h-4 w-4 text-red-500" /> {error}
    </div>
  );
}
