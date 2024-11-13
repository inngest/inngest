import ProgressBar from '@inngest/components/ProgressBar/ProgressBar';
import { cn } from '@inngest/components/utils/classNames';

type Data = {
  title: string;
  description: string;
  current: number;
  limit: number;
};

export async function LimitBar({ data, className }: { data: Data; className?: string }) {
  const { title, description, current, limit } = data;
  return (
    <div className={cn(className)}>
      <p className="text-subtle mb-1 text-xs font-medium">{title}</p>
      <p className="text-subtle mb-2 text-xs">{description}</p>
      <ProgressBar value={current} limit={limit} />
      <div className="text-right">
        <span className={cn('text-medium text-basis text-sm', current > limit && 'text-warning')}>
          {current.toLocaleString()}
        </span>
        <span className="text-muted text-sm">/{limit.toLocaleString()}</span>
      </div>
    </div>
  );
}
