import { cn } from '@inngest/components/utils/classNames';

export default function TotalCount({
  className,
  totalCount,
}: {
  className?: string;
  totalCount: number | undefined;
}) {
  if (totalCount === undefined) {
    return null;
  }

  const formatted = new Intl.NumberFormat().format(totalCount);
  if (totalCount === 1) {
    return (
      <span className={cn('text-muted text-xs font-semibold', className)}>{formatted} event</span>
    );
  }
  return (
    <span className={cn('text-muted text-xs font-semibold', className)}>{formatted} events</span>
  );
}
