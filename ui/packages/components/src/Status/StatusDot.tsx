import { cn } from '../utils/classNames';
import { getStatusBackgroundClass, getStatusBorderClass } from './statusClasses';
import { statusTitles } from './statusTitles';

type Props = {
  status: string;
  className?: string;
};

export function StatusDot({ status, className }: Props) {
  const backgroundClass = getStatusBackgroundClass(status);
  const borderClass = getStatusBorderClass(status);

  const title = statusTitles[status] || 'Unknown';
  return (
    <div
      className={cn('h-3.5 w-3.5 rounded-full border', backgroundClass, borderClass, className)}
      title={title}
    />
  );
}
