import { RiBuilding2Line, RiFlashlightLine, RiSettings4Line } from '@remixicon/react';

import { cn } from '../utils/classNames';
import { SegmentedSpanBar } from './SegmentedSpanBar';
import { formatDuration } from './timingBreakdown';
import type { SpanTimingBreakdown, TimingCategoryTotal } from './types';

type Props = {
  breakdown: SpanTimingBreakdown;
  /** Organization name for server category label (defaults to "YOUR") */
  orgName?: string;
  className?: string;
};

/**
 * Icon component mapping for timing categories
 */
const CategoryIcon = ({ icon }: { icon: TimingCategoryTotal['icon'] }) => {
  const iconClass = 'h-4 w-4';
  switch (icon) {
    case 'gear':
      return <RiSettings4Line className={iconClass} />;
    case 'lightning':
      return <RiFlashlightLine className={iconClass} />;
    case 'building':
      return <RiBuilding2Line className={iconClass} />;
  }
};

/**
 * Get display label for a category, substituting org name for server category
 */
function getCategoryLabel(category: TimingCategoryTotal, orgName?: string): string {
  if (category.category === 'customer_server' && orgName) {
    return `${orgName.toUpperCase()} SERVER`;
  }
  return category.label;
}

/**
 * TimingBreakdownPanel displays the detailed timing breakdown for a step.run span.
 *
 * Features:
 * - Header with segmented bar showing timing proportions
 * - Start/end timestamps (FR-009)
 * - Category sections with icons and total durations
 * - Individual segment details with labels and formatted durations
 */
export function TimingBreakdownPanel({ breakdown, orgName, className }: Props) {
  return (
    <div className={cn('flex flex-col gap-4', className)}>
      {/* Header Section */}
      <div className="flex flex-col gap-2">
        {/* Segmented Bar */}
        <SegmentedSpanBar segments={breakdown.barSegments} height="h-3" />

        {/* Timestamps Row */}
        <div className="text-muted flex justify-between text-xs">
          <span>{breakdown.startTime}</span>
          <span className="text-basis font-medium">
            {formatDuration(breakdown.totalDurationMs)}
          </span>
          <span>{breakdown.endTime}</span>
        </div>
      </div>

      {/* Category Breakdown */}
      <div className="flex flex-col gap-3">
        {breakdown.categories.map((category) => (
          <CategorySection
            key={category.category}
            category={category}
            orgName={orgName}
            totalDurationMs={breakdown.totalDurationMs}
          />
        ))}
      </div>
    </div>
  );
}

/**
 * Individual category section with icon, label, and segment details
 */
function CategorySection({
  category,
  orgName,
  totalDurationMs,
}: {
  category: TimingCategoryTotal;
  orgName?: string;
  totalDurationMs: number;
}) {
  const percentage =
    totalDurationMs > 0 ? Math.round((category.totalMs / totalDurationMs) * 100) : 0;

  return (
    <div className="flex flex-col gap-1.5">
      {/* Category Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div
            className={cn(
              'flex h-6 w-6 items-center justify-center rounded',
              category.category === 'inngest' && 'bg-slate-200 text-slate-600',
              category.category === 'connecting' && 'bg-amber-100 text-amber-600',
              category.category === 'customer_server' && 'bg-emerald-100 text-emerald-600'
            )}
          >
            <CategoryIcon icon={category.icon} />
          </div>
          <span className="text-basis text-sm font-medium">
            {getCategoryLabel(category, orgName)}
          </span>
        </div>
        <div className="text-muted flex items-center gap-2 text-sm">
          <span>{formatDuration(category.totalMs)}</span>
          <span className="text-subtle">({percentage}%)</span>
        </div>
      </div>

      {/* Segment Details */}
      <div className="ml-8 flex flex-col gap-1">
        {category.segments.map((segment, index) => (
          <div key={`${segment.segmentType}-${index}`} className="flex items-center gap-2">
            <div className={cn('h-2 w-2 rounded-sm', segment.color)} style={{ minWidth: '8px' }} />
            <span className="text-muted text-xs">{segment.label}</span>
            <span className="text-subtle text-xs">{formatDuration(segment.durationMs)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
