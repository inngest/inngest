import type { ReactElement } from 'react';
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
 * Returns all stat keys within `proximityThreshold` (data-space units) of the
 * nearest snap position to `hoverValue`. Covers both exact overlaps (e.g.
 * min === q1) and stats that are visually close together.
 */
function nearestStatKeys(
  hoverValue: number,
  p: BoxPlotData,
  proximityThreshold: number,
): string[] {
  let bestDist = Infinity;
  let bestValue: number | null = null;

  for (const stat of SNAP_STATS) {
    const v = stat.valueFn(p);
    if (v < p.min || v > p.max) continue;
    const dist = Math.abs(v - hoverValue);
    if (dist < bestDist) {
      bestDist = dist;
      bestValue = v;
    }
  }

  if (bestValue === null) return [];

  return SNAP_STATS.filter((s) => {
    const v = s.valueFn(p);
    return (
      v >= p.min && v <= p.max && Math.abs(v - bestValue) <= proximityThreshold
    );
  }).map((s) => s.key);
}

/**
 * Returns a recharts Tooltip content component with the domain baked in so it
 * can convert pixel coordinates to data-space values and show only the nearest
 * named datapoint(s) under the cursor.
 */
export function makeBoxPlotTooltip(domain: [number, number]) {
  return function BoxPlotTooltipContent({
    active,
    payload,
    label,
    coordinate,
    viewBox,
  }: RechartsContentProps) {
    if (!active || !payload?.length) return null;

    const first = payload[0];
    const title = label ?? first?.payload?.variantName ?? '';
    const metricName = first?.name;

    const cx = coordinate?.x;
    const vx = viewBox?.x ?? 0;
    const vw = viewBox?.width;
    const domainRange = domain[1] - domain[0];
    const hoverValue =
      cx != null && vw != null && vw > 0
        ? domain[0] + ((cx - vx) / vw) * domainRange
        : null;
    // 8px expressed in data-space so nearby stats are grouped together
    const proximityThreshold =
      vw != null && vw > 0 ? (8 / vw) * domainRange : 0;

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
