import { RiBuilding2Line, RiSettings4Line } from '@remixicon/react';

import { cn } from '../utils/classNames';
import { TIMING_COLORS } from './timingBreakdown';

type Props = {
  /** Organization name for server category label (defaults to "YOUR") */
  orgName?: string;
  className?: string;
};

/**
 * TimingLegend displays the color legend for timing categories
 * above the timeline view.
 *
 * US3: Understand Timing Legend
 */
export function TimingLegend({ orgName, className }: Props) {
  const serverLabel = orgName ? `${orgName.toUpperCase()} SERVER` : 'YOUR SERVER';

  return (
    <div className={cn('flex items-center gap-6 text-xs', className)}>
      {/* INNGEST category */}
      <div className="flex items-center gap-1.5">
        <div
          className={cn(
            'flex h-4 w-4 items-center justify-center rounded',
            'bg-slate-200 text-slate-600'
          )}
        >
          <RiSettings4Line className="h-3 w-3" />
        </div>
        <div className={cn('h-2 w-4 rounded-sm', TIMING_COLORS.inngest.base)} />
        <span className="text-muted">INNGEST</span>
      </div>

      {/* Customer Server category */}
      <div className="flex items-center gap-1.5">
        <div
          className={cn(
            'flex h-4 w-4 items-center justify-center rounded',
            'bg-emerald-100 text-emerald-600'
          )}
        >
          <RiBuilding2Line className="h-3 w-3" />
        </div>
        <div className={cn('h-2 w-4 rounded-sm', TIMING_COLORS.customer_server.base)} />
        <span className="text-muted">{serverLabel}</span>
      </div>
    </div>
  );
}
