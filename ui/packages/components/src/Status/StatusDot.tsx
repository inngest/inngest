import { cn } from '../utils/classNames';
import { getStatusBackgroundClass, getStatusBorderClass } from './statusClasses';
import { statusTitles } from './statusTitles';

export type StatusDotProps = {
  status: string;
  className?: string;
  size?: 'small' | 'base';
};

export function StatusDot({ status, size = 'base', className }: StatusDotProps) {
  const backgroundClass = getStatusBackgroundClass(status);
  const borderClass = getStatusBorderClass(status);

  const title = statusTitles[status] || 'Unknown';
  return (
    <div
      className={cn(
        size === 'small' ? 'block h-2 w-2 shrink-0 rounded-full' : 'h-3.5 w-3.5',
        'rounded-full border',
        backgroundClass,
        borderClass,
        className
      )}
      title={title}
    />
  );
}
