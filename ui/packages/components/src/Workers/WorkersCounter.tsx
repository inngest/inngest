import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

import { getStatusBackgroundClass, getStatusBorderClass } from '../Status/statusClasses';
import { groupedWorkerStatuses, type GroupedWorkerStatus } from '../types/workers';
import { cn } from '../utils/classNames';

type Props = {
  counts: Record<GroupedWorkerStatus, number | null>;
  className?: string;
};

export default function WorkersCounter({ counts, className }: Props) {
  return (
    <Tooltip>
      <TooltipTrigger>
        <div className="flex gap-2">
          {groupedWorkerStatuses
            .filter((status) => counts[status] !== null)
            .map((status) => {
              const backgroundClass = getStatusBackgroundClass(status);
              const borderClass = getStatusBorderClass(status);
              return (
                <div key={status} className={cn('flex items-center gap-0.5', className)}>
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
      </TooltipTrigger>
      <TooltipContent
        side="bottom"
        sideOffset={0}
        align="start"
        className="border-subtle flex flex-col gap-1 rounded-md border p-2 pr-3 text-xs"
      >
        {groupedWorkerStatuses
          .filter((status) => counts[status] !== null)
          .map((status) => (
            <div key={status} className="flex items-center gap-1">
              <div
                className={cn(
                  'h-[10px] w-[10px] rounded-full border',
                  getStatusBackgroundClass(status),
                  getStatusBorderClass(status)
                )}
              />
              <div className="flex w-full items-center justify-between gap-3">
                <div className="text-muted lowercase first-letter:capitalize">{status} workers</div>
                {counts[status] || 0}
              </div>
            </div>
          ))}
      </TooltipContent>
    </Tooltip>
  );
}
