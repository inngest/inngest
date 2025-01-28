import { getStatusBackgroundClass, getStatusBorderClass } from '../Status/statusClasses';
import { groupedWorkerStatuses, type GroupedWorkerStatus } from '../types/workers';
import { cn } from '../utils/classNames';

type Props = {
  counts: Record<GroupedWorkerStatus, number>;
  className?: string;
};

export default function WorkersCounter({ counts, className }: Props) {
  return (
    <div className="flex gap-2">
      {groupedWorkerStatuses.map((status) => {
        const backgroundClass = getStatusBackgroundClass(status);
        const borderClass = getStatusBorderClass(status);
        return (
          <div
            key={status}
            className={cn('flex items-center gap-0.5', className)}
            title={`${status}: ${counts[status] || '0'}`}
          >
            <div
              className={cn(
                'h-[10px] w-[10px] rounded-full border',
                backgroundClass,
                borderClass,
                className
              )}
            />
            <span className="text-subtle text-sm">{counts[status] || '0'}</span>
          </div>
        );
      })}
    </div>
  );
}
