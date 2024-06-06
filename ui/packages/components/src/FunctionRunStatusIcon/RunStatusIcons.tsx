import { getStatusBackgroundClass, getStatusBorderClass } from '../statusClasses';
import { cn } from '../utils/classNames';

type Props = {
  status: string;
  className?: string;
};

export function RunStatusIcon({ status, className }: Props) {
  const backgroundClass = getStatusBackgroundClass(status);
  const borderClass = getStatusBorderClass(status);

  const title = 'Function ' + status.toLowerCase();
  return (
    <div
      className={cn('h-3.5 w-3.5 rounded-full border', backgroundClass, borderClass, className)}
      title={title}
    />
  );
}
