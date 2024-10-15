import { Skeleton } from '@inngest/components/Skeleton';

export function CardItem({
  className,
  detail,
  term,
  loading = false,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
  loading?: boolean;
}) {
  return (
    <div className={className}>
      <dt className="text-muted pb-2 text-sm">{term}</dt>
      {!loading && <dd className="text-basis leading-8">{detail ?? ''}</dd>}
      {loading && <Skeleton className="mb-2 block h-6 w-full" />}
    </div>
  );
}
