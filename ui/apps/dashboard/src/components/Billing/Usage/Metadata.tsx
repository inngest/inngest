import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

export default function UsageMetadata({
  fetching,
  title,
  value,
  className,
}: {
  fetching?: boolean;
  title: string;
  value?: string;
  className?: string;
}) {
  return (
    <div className={className}>
      <dt className="text-subtle text-xs font-medium">{title}</dt>
      {fetching ? (
        <Skeleton className="block h-5 w-full" />
      ) : (
        <dd className="text-basis truncate text-sm font-medium">{value}</dd>
      )}
    </div>
  );
}
