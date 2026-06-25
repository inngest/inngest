import type { MutableRefObject, ReactElement } from 'react';
import type { BoxPlotData } from './types';
import { formatMetricValue } from './variantsTable/metricStats';

type TooltipEntry = {
  name?: string | number;
  dataKey?: string | number;
  value?: number | string | Array<number | string>;
  color?: string;
  payload?: BoxPlotData;
};

type RechartsContentProps = {
  active?: boolean;
  payload?: TooltipEntry[];
  /** Falls back to `payload[0].payload.variantName` when not provided. */
  label?: string | number;
  coordinate?: { x?: number; y?: number };
  viewBox?: { x?: number; y?: number; width?: number; height?: number };
};

type StatEntry = {
  key: string;
  label: ReactElement | string;
  valueFn: (p: BoxPlotData) => number;
};

const ALL_STATS: StatEntry[] = [
  { key: 'avg', label: 'Avg', valueFn: (p) => p.avg },
  { key: 'stddev', label: 'StdDev', valueFn: (p) => p.stddev },
  {
    key: 'z_neg1',
    label: (
      <>
        Z<sub>-1</sub>
      </>
    ),
    valueFn: (p) => p.avg - p.stddev,
  },
  {
    key: 'z_pos1',
    label: (
      <>
        Z<sub>+1</sub>
      </>
    ),
    valueFn: (p) => p.avg + p.stddev,
  },
  { key: 'min', label: 'Min', valueFn: (p) => p.min },
  { key: 'q1', label: 'Q1', valueFn: (p) => p.q1 },
  { key: 'med', label: 'Median', valueFn: (p) => p.med },
  { key: 'q3', label: 'Q3', valueFn: (p) => p.q3 },
  { key: 'max', label: 'Max', valueFn: (p) => p.max },
];

// Snap candidates — excludes stddev which is a magnitude, not a chart position
const SNAP_STATS = ALL_STATS.filter((s) => s.key !== 'stddev');

/**
 * Returns the keys of stats within `proximityThreshold` of the cursor.
 * Co-located stats (within the same threshold of each other) are grouped
 * together so hovering near a cluster shows the full cluster.
 */
function nearestStatKeys(
  hoverValue: number,
  p: BoxPlotData,
  proximityThreshold: number,
): string[] {
  const candidates = SNAP_STATS.map((s) => ({
    key: s.key,
    v: s.valueFn(p),
  })).filter((c) => c.v >= p.min && c.v <= p.max);

  const dist = (c: { v: number }) => Math.abs(c.v - hoverValue);

  if (candidates.length === 0) return [];

  const exact = candidates.filter((c) => dist(c) < 0.001);
  if (exact.length > 0)
    return candidates
      .filter((c) => dist(c) <= proximityThreshold)
      .map((c) => c.key);

  const below = candidates
    .filter((c) => c.v < hoverValue)
    .sort((a, b) => b.v - a.v)[0];
  const above = candidates
    .filter((c) => c.v > hoverValue)
    .sort((a, b) => a.v - b.v)[0];

  const result = new Set<string>();

  for (const flank of [below, above]) {
    if (!flank) continue;
    // Include the flank and any stats co-located within the same threshold.
    candidates
      .filter((c) => Math.abs(c.v - flank.v) <= proximityThreshold)
      .forEach((c) => result.add(c.key));
  }

  return [...result];
}

/**
 * Returns a recharts Tooltip content component with the domain baked in so it
 * can convert pixel coordinates to data-space values and show only the nearest
 * named datapoint(s) under the cursor.
 */
export function makeBoxPlotTooltip(
  domain: [number, number],
  hoverValueRef: MutableRefObject<number | null>,
  chartWidthRef: MutableRefObject<number>,
) {
  return function BoxPlotTooltipContent({
    active,
    payload,
    label,
  }: RechartsContentProps) {
    if (!active || !payload?.length) return null;

    const first = payload[0];
    const title = label ?? first?.payload?.variantName ?? '';
    const metricName = first?.name;

    const domainRange = domain[1] - domain[0];
    const hoverValue = hoverValueRef.current;
    const chartWidth = chartWidthRef.current;
    const proximityThreshold =
      chartWidth > 0 ? (12 / chartWidth) * domainRange : 0;

    return (
      <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
        {title && <div className="text-basis text-sm font-medium">{title}</div>}
        {metricName && (
          <div className="text-muted mb-1.5 text-xs">{metricName}</div>
        )}
        <div className="flex flex-col gap-1">
          {payload.map((p, i) => {
            if (!p.payload) return null;
            const data = p.payload;
            const keys =
              hoverValue !== null
                ? nearestStatKeys(hoverValue, data, proximityThreshold)
                : [];
            const stats = ALL_STATS.filter((s) => keys.includes(s.key));

            if (stats.length === 0) return null;

            return (
              <div key={i} className="flex gap-3">
                {stats.map(({ key, label, valueFn }) => (
                  <div key={key} className="flex gap-1.5 items-center">
                    <span className="text-muted">{label}</span>
                    <span className="text-basis tabular-nums font-semibold">
                      {formatMetricValue(valueFn(data))}
                    </span>
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      </div>
    );
  };
}
