import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { cn } from '@inngest/components/utils/classNames';

export type Data = {
  title: string;
  description: string;
  current: number;
  limit: number | null;
  overageAllowed?: boolean;
};

export async function LimitBar({ data, className }: { data: Data; className?: string }) {
  const { title, description, current, limit, overageAllowed } = data;
  const isUnlimited = limit === null;
  return (
    <div className={cn(className)}>
      <p className="text-subtle mb-1 text-xs font-medium">{title}</p>
      <p className="text-subtle mb-2 text-xs">{description}</p>
      <ProgressBar value={current} limit={limit} overageAllowed={overageAllowed} />
      <div className="text-right">
        <span
          className={cn(
            'text-medium text-basis text-sm',
            !isUnlimited && current > limit && overageAllowed && 'text-warning',
            !isUnlimited && current > limit && !overageAllowed && 'text-error'
          )}
        >
          {current.toLocaleString()}
        </span>
        <span className="text-muted text-sm">
          /{isUnlimited ? 'unlimited' : limit.toLocaleString()}
        </span>
      </div>
    </div>
  );
}
