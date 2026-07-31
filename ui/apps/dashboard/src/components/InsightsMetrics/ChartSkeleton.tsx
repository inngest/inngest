import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

type SkeletonProps = {
  className?: string;
  // Whether the shimmer runs — callers pass false for a confirmed-empty
  // state (nothing in flight, so a shimmer would be misleading) and true (or
  // omit — it's the default) while genuinely loading.
  animate?: boolean;
};

// Fixed, decreasing widths — the real row count isn't known until data
// arrives (loading), or there simply isn't any (empty), so this mimics a
// plausible ranked shape rather than a specific result.
const RANKED_WIDTHS = ['w-[92%]', 'w-[76%]', 'w-[60%]', 'w-[46%]', 'w-[32%]'];

// RankedChartSkeleton mimics a horizontal ranked bar chart (CategoricalChart,
// LatencyRangeChart) for both the loading and empty states, so a chart
// never shows a spinner or bare text where its data would otherwise be.
export function RankedChartSkeleton({ className, animate = true }: SkeletonProps) {
  return (
    <div className={`flex h-full flex-col justify-center gap-3 ${className ?? ''}`}>
      {RANKED_WIDTHS.map((width) => (
        <Skeleton key={width} animate={animate} className={`h-4 ${width}`} />
      ))}
    </div>
  );
}

// Fixed pixel heights (not percentages) mimicking a plausible bar/line
// silhouette. Callers (TrendChart's empty state in particular) mount this
// inside a wrapper whose own height isn't always CSS-definite — a percentage
// height only resolves against a definite ancestor height, so a bar sized
// with e.g. `h-[35%]` can silently collapse to 0 and disappear. Fixed pixel
// heights have no such dependency, so the bars always render.
const TREND_HEIGHTS = [
  'h-[84px]',
  'h-[132px]',
  'h-[96px]',
  'h-[168px]',
  'h-[120px]',
  'h-[204px]',
  'h-[144px]',
  'h-[108px]',
  'h-[156px]',
  'h-[120px]',
  'h-[180px]',
  'h-[96px]',
];

// TrendChartSkeleton mimics a bar/line time-series chart's plot area for
// both the loading and empty states.
export function TrendChartSkeleton({ className, animate = true }: SkeletonProps) {
  return (
    <div className={`flex items-end gap-2 px-1 pb-6 ${className ?? ''}`}>
      {TREND_HEIGHTS.map((height, i) => (
        <Skeleton key={i} animate={animate} direction="up" className={`w-full ${height}`} />
      ))}
    </div>
  );
}

// A single filled-area silhouette (rather than discrete bars), for
// TrendChart's 'area' chartType. The shimmer itself is the shared Skeleton
// styling; clip-path carves it into a mountain-line shape so it still reads
// as "area chart" rather than a generic block. A fixed pixel height (not
// `h-full`) — the wrapping div only ever gets a `min-h-*` from callers, which
// doesn't establish a definite height for a percentage-based child to
// resolve against, the same pitfall TrendChartSkeleton's bars had.
const AREA_CLIP_PATH =
  '[clip-path:polygon(0%_100%,0%_55%,15%_40%,30%_52%,45%_25%,60%_38%,75%_15%,90%_32%,100%_20%,100%_100%)]';

export function TrendAreaChartSkeleton({ className, animate = true }: SkeletonProps) {
  return (
    <div className={`px-1 pb-6 ${className ?? ''}`}>
      <Skeleton animate={animate} className={`h-[180px] w-full ${AREA_CLIP_PATH}`} />
    </div>
  );
}

// Fixed, decreasing label widths (like RANKED_WIDTHS) plus a fixed-width
// value at the end of each row, mirroring ChartLegend's real row shape
// (swatch + identifier + value).
const LEGEND_LABEL_WIDTHS = ['w-[70%]', 'w-[55%]', 'w-[62%]', 'w-[42%]', 'w-[50%]'];

// LegendSkeleton mimics ChartLegend's rows (swatch, identifier, value) for
// both the loading and empty states.
export function LegendSkeleton({ className, animate = true }: SkeletonProps) {
  return (
    <ul className={`flex flex-col gap-3 ${className ?? ''}`}>
      {LEGEND_LABEL_WIDTHS.map((width) => (
        <li key={width} className="flex items-center gap-2">
          <Skeleton animate={animate} className="h-2.5 w-2.5 shrink-0" />
          <Skeleton animate={animate} className={`h-3 ${width}`} />
          <Skeleton animate={animate} className="h-3 w-8 shrink-0" />
        </li>
      ))}
    </ul>
  );
}

const TABLE_ROW_COUNT = 5;

// TableRowsSkeleton mimics a handful of RankedTable rows for its blank
// state — shown whenever there's no data, whether the range is genuinely
// empty or the first load just hasn't resolved yet (the shared Table
// component already renders its own per-column skeleton cells while
// `isLoading`, so this only needs to cover the empty case). `columnWidths`
// lets the caller roughly match its real column shapes (e.g. a wide
// identifier column followed by narrower value columns). Never shimmers —
// this only ever renders once the range is confirmed empty (Table's own
// `isLoading` skeleton cells cover the loading case), so a shimmer here
// would misleadingly imply something is still in flight.
export function TableRowsSkeleton({ columnWidths }: { columnWidths: string[] }) {
  return (
    <div className="flex flex-col gap-4 py-1">
      {Array.from({ length: TABLE_ROW_COUNT }, (_, row) => (
        <div key={row} className="flex items-center gap-6">
          {columnWidths.map((width, col) => (
            <Skeleton key={col} animate={false} className={`h-3 ${width}`} />
          ))}
        </div>
      ))}
    </div>
  );
}
