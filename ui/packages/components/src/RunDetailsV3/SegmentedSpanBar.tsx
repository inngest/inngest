import { cn } from '../utils/classNames';
import { TIMING_COLORS } from './timingBreakdown';
import type { TimingCategory } from './types';

type BarSegment = {
  category: TimingCategory;
  widthPercent: number;
  isWaiting?: boolean;
};

type Props = {
  className?: string;
  segments: BarSegment[];
  /** Height of the bar in Tailwind units (default: h-2) */
  height?: string;
};

/**
 * Get the background color class for a timing category
 */
function getCategoryColor(category: TimingCategory): string {
  switch (category) {
    case 'inngest':
      return TIMING_COLORS.inngest.base;
    case 'connecting':
      return TIMING_COLORS.connecting.base;
    case 'customer_server':
      return TIMING_COLORS.customer_server.base;
    default:
      return 'bg-slate-400';
  }
}

/**
 * SegmentedSpanBar renders a horizontal bar with colored segments
 * representing timing category proportions.
 *
 * FR-010: Ensures minimum 2px visible width for each segment
 */
export function SegmentedSpanBar({ className, segments, height = 'h-2' }: Props) {
  if (segments.length === 0) {
    return null;
  }

  return (
    <div className={cn('flex w-full overflow-hidden rounded-sm', height, className)}>
      {segments.map((segment, index) => {
        // FR-010: Ensure minimum visible width (2px) even for very short durations
        // Use minWidth style to guarantee visibility
        const hasWidth = segment.widthPercent > 0;

        return (
          <div
            key={`${segment.category}-${index}`}
            className={cn(
              getCategoryColor(segment.category),
              'transition-all duration-150',
              // Striped pattern indicates in-progress execution on customer server
              segment.isWaiting &&
                'bg-[linear-gradient(45deg,rgba(255,255,255,0.15)_25%,transparent_25%,transparent_50%,rgba(255,255,255,0.15)_50%,rgba(255,255,255,0.15)_75%,transparent_75%,transparent)] bg-[length:10px_10px]'
            )}
            style={{
              width: `${segment.widthPercent}%`,
              // FR-010: Minimum 2px width for visibility when segment has any duration
              minWidth: hasWidth ? '2px' : '0px',
            }}
          />
        );
      })}
    </div>
  );
}
