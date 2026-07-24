import { useEffect, useMemo, useState } from 'react';
import { Bar, BarChart, Customized, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import { lineColors } from '@/components/Metrics/utils';
import { truncateCenter } from '@/lib/experiments/chart';
import { RankedChartSkeleton } from './ChartSkeleton';
import { BORDER_SUBTLE_COLOR, toCssColor } from './colors';
import { valuesToMap, type InsightsMetricItem } from './types';

// RangePlot is a copy of Experiments' BoxPlot (apps/dashboard/src/
// components/Experiments/BoxPlot.tsx) adapted for a percentile shape rather
// than a true quartile one: no Q1–Q3 box (this data has no Q1, and an extra
// point — min/p50/p95/p99/max — that a real boxplot's five positions have
// no room for), just a vertical tick at each configured value, connected by
// one line. `ticks` is fully caller-configurable — the component only ever
// reads values by `valueName`, so any subset of percentiles/labels works.
//
// The floating span itself uses recharts' native tuple `dataKey` (a
// function returning `[min, max]`) rather than the two-bar invisible-
// stacking trick — this is the same mechanism BoxPlot already uses, and
// sidesteps a recharts quirk where per-`<Bar>` `barSize` stops being
// respected once color/shape overrides get layered on a stacked pair;
// `barSize` set once at the chart level (as BoxPlot does) doesn't have that
// problem.

export type RangePlotTick = {
  // Which NamedValue.name to read from each item's values.
  valueName: string;
  label: string;
};

const DEFAULT_TICKS: RangePlotTick[] = [
  { valueName: 'min', label: 'Min' },
  { valueName: 'p50', label: 'p50' },
  { valueName: 'p95', label: 'p95' },
  { valueName: 'p99', label: 'p99' },
  { valueName: 'max', label: 'Max' },
];

const TICK_LINE_HEIGHT = 24;
const CONNECTING_LINE_WIDTH = 1.5;
/** Snap within this many pixels of a tick value. */
const SNAP_PX = 4;
/**
 * Once the crosshair snaps to a tick, any other tick within this fraction of
 * the row's full min–max range counts as "close" and highlights alongside
 * it — ticks that are visually on top of each other (e.g. p95 and p99
 * nearly equal) should call attention to both, not just whichever the
 * cursor happened to land nearest to.
 */
const CLOSE_RANGE_FRACTION = 0.02;

const formatSeconds = (value: number) => `${value.toFixed(2)}s`;

type RowData = {
  identifier: string;
  // Every configured tick's own value for this row, keyed by valueName —
  // absent ticks are simply missing keys rather than null, since recharts
  // only ever reads `min`/`max` (below) off this shape directly.
  values: Record<string, number>;
  min: number;
  max: number;
  color: string;
};

type BarShapeProps = {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  payload?: RowData;
};

// renderYAxisTick draws each category label as a single line of SVG text,
// middle-truncated with an ellipsis — recharts' default category-axis tick
// wraps long labels across multiple lines when constrained by `width`; a
// plain <text> element never wraps, so labels stay on one line instead.
function renderYAxisTick({ x, y, payload }: { x: number; y: number; payload: { value: string } }) {
  return (
    <text x={x} y={y} dy={4} textAnchor="end" fontSize={10} className="fill-basis">
      <title>{payload.value}</title>
      {truncateCenter(payload.value, 30)}
    </text>
  );
}

// TickShape draws a vertical line at every configured tick's value, all
// connected by one horizontal line — the box-plot whisker shape without the
// Q1–Q3 box.
function makeTickShape(ticks: RangePlotTick[]) {
  return function TickShape({ x = 0, y = 0, width = 0, height = 0, payload }: BarShapeProps) {
    if (!payload) return <g />;
    const range = payload.max - payload.min;
    const cy = y + height / 2;

    if (range === 0) {
      return <circle cx={x} cy={cy} r={height / 2} fill={payload.color} />;
    }

    const tickXs = ticks.map((t) => {
      const v = payload.values[t.valueName];
      return typeof v === 'number' ? x + ((v - payload.min) / range) * width : null;
    });
    const presentXs = tickXs.filter((v): v is number => v !== null);

    return (
      <g>
        <line
          x1={Math.min(...presentXs)}
          x2={Math.max(...presentXs)}
          y1={cy}
          y2={cy}
          stroke={payload.color}
          strokeWidth={CONNECTING_LINE_WIDTH}
        />
        {ticks.map((t, i) => {
          const tx = tickXs[i];
          if (tx === null) return null;
          return (
            <line
              key={t.valueName}
              x1={tx}
              x2={tx}
              y1={y}
              y2={y + height}
              stroke={payload.color}
              strokeWidth={CONNECTING_LINE_WIDTH}
            />
          );
        })}
      </g>
    );
  };
}

// BackgroundLineShape draws a full-width 1px line centered on each row —
// the "track" the floating span and its ticks sit on, same as BoxPlot's.
function BackgroundLineShape({ x = 0, y = 0, width = 0, height = 0 }: BarShapeProps) {
  const cy = y + height / 2;
  return <rect x={x} y={cy - 0.5} width={width} height={1} fill={BORDER_SUBTLE_COLOR} />;
}

type RechartScale = { (v: number): number; invert?: (px: number) => number };
type AxisEntry = { x: number; width: number; y: number; height: number; scale?: RechartScale };

type HoverLineProps = {
  xAxisMap?: Record<string, AxisEntry>;
  yAxisMap?: Record<string, { y: number; height: number }>;
  hoverX: number | null;
  activeRow: RowData | null;
  ticks: RangePlotTick[];
  // Reports every tick (by valueName) that should highlight — the one the
  // crosshair snapped to, plus any others within CLOSE_RANGE_FRACTION of
  // it — or an empty array when nothing's within snapping range. Lets the
  // tooltip bold the matching stat rows so it's clear exactly which
  // value(s) are highlighted, including a cluster of near-identical ones.
  onSnap: (valueNames: string[]) => void;
};

// HoverLine draws a dashed vertical crosshair that snaps to the nearest
// configured tick value (in pixel space, via recharts' own x-scale) when
// the cursor is close enough — same interaction as BoxPlot's.
function HoverLine({ xAxisMap, yAxisMap, hoverX, activeRow, ticks, onSnap }: HoverLineProps) {
  const xAxis = xAxisMap ? Object.values(xAxisMap)[0] : undefined;
  const yAxis = yAxisMap ? Object.values(yAxisMap)[0] : undefined;
  const scale = xAxis?.scale;

  let bestDist = Infinity;
  let bestPx: number | null = null;
  let bestName: string | null = null;
  if (hoverX !== null && activeRow && scale) {
    for (const t of ticks) {
      const v = activeRow.values[t.valueName];
      if (typeof v !== 'number' || v < activeRow.min || v > activeRow.max) continue;
      const px = scale(v);
      const dist = Math.abs(px - hoverX);
      if (dist < bestDist) {
        bestDist = dist;
        bestPx = px;
        bestName = t.valueName;
      }
    }
  }
  const snapped = bestDist <= SNAP_PX;

  // Once snapped to an anchor tick, pull in any other tick within 5% of
  // the row's range — ticks nearly on top of each other should highlight
  // together rather than arbitrarily picking whichever is a hair closer.
  let closeNames: string[] = [];
  if (snapped && activeRow && bestName !== null) {
    const anchorValue = activeRow.values[bestName];
    const threshold = (activeRow.max - activeRow.min) * CLOSE_RANGE_FRACTION;
    closeNames = ticks
      .filter((t) => {
        const v = activeRow.values[t.valueName];
        return typeof v === 'number' && Math.abs(v - anchorValue) <= threshold;
      })
      .map((t) => t.valueName);
  }

  // Effect, not a direct call — this runs during Customized's render pass,
  // so calling onSnap (a setState in the parent) synchronously here would
  // be a side effect during render.
  useEffect(() => {
    onSnap(closeNames);
  }, [closeNames.join(','), onSnap]);

  if (hoverX === null || !xAxis || !yAxis) return null;

  const plotX = Math.min(
    Math.max(snapped && bestPx !== null ? bestPx : hoverX, xAxis.x),
    xAxis.x + xAxis.width,
  );

  return (
    <line
      x1={plotX}
      x2={plotX}
      y1={yAxis.y}
      y2={yAxis.y + yAxis.height}
      stroke="rgb(var(--color-foreground-subtle))"
      strokeWidth={1}
      strokeDasharray="3 3"
      pointerEvents="none"
    />
  );
}

type RangePlotTooltipEntry = { payload?: RowData };

// RangePlotTooltip mirrors BoxPlotTooltip: a bordered card with a title row
// (separated by a bottom border) and a column of label/value stat rows.
function RangePlotTooltip({
  active,
  payload,
  label,
  ticks,
  format,
  boldValueNames,
}: {
  active?: boolean;
  payload?: RangePlotTooltipEntry[];
  label?: string;
  ticks: RangePlotTick[];
  format: (value: number) => string;
  // valueNames of the ticks the hover crosshair is currently snapped to
  // (the nearest one plus any others close enough in value to it) — their
  // stat rows render bold so it's clear which value(s) are highlighted.
  boldValueNames?: string[];
}) {
  if (!active || !payload?.length) return null;
  const row = payload[0]?.payload;
  if (!row) return null;

  const stats = ticks
    .map((t) => {
      const value = row.values[t.valueName];
      return typeof value === 'number'
        ? { valueName: t.valueName, label: t.label, value: format(value) }
        : null;
    })
    .filter((s): s is { valueName: string; label: string; value: string } => s !== null);

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {label && (
        <div className="border-subtle mb-1.5 border-b pb-1.5">
          <span className="text-basis text-sm font-medium">{label}</span>
        </div>
      )}
      <div className="flex flex-col gap-1">
        {stats.map(({ valueName, label: statLabel, value }) => {
          const isBold = boldValueNames?.includes(valueName) ?? false;
          return (
            <div key={statLabel} className="flex items-baseline justify-between gap-4">
              <span className={isBold ? 'text-basis font-bold' : 'text-muted'}>{statLabel}</span>
              <span className={`text-basis tabular-nums ${isBold ? 'font-bold' : 'font-semibold'}`}>
                {value}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

type Props = {
  items: InsightsMetricItem[] | undefined;
  // Ordered set of value fields to mark as ticks — the span itself covers
  // the smallest to largest value present among these per row. Defaults to
  // the common min/p50/p95/p99/max latency shape.
  ticks?: RangePlotTick[];
  // Formats the tooltip values (and the x-axis ticks, unless `axisFormat`
  // overrides it) — defaults to seconds, but any unit works.
  format?: (value: number) => string;
  // Overrides `format` for the x-axis ticks specifically — for callers
  // whose tooltip wants a fixed precision but whose axis reads better
  // compact (e.g. formatSeconds' fixed-sig-fig tooltip vs
  // formatSecondsAxis' trailing-zero-collapsed ticks). Defaults to `format`.
  axisFormat?: (value: number) => string;
  // One color per row (in row/sorted order), cycling if there are more rows
  // than colors — matches CategoricalChart's `colors` prop (e.g. the same
  // palette passed to a "Cost by model" chart).
  colors?: readonly (readonly [string, string])[];
  // Hides the vertical y-axis line (ticks/labels stay) — for callers where
  // it's redundant against the chart's own card border.
  showYAxisLine?: boolean;
  isLoading?: boolean;
  group?: string;
  className?: string;
};

export function RangePlot({
  items,
  ticks = DEFAULT_TICKS,
  format = formatSeconds,
  axisFormat = format,
  colors,
  showYAxisLine = true,
  isLoading = false,
  group,
  className,
}: Props) {
  const [hoverX, setHoverX] = useState<number | null>(null);
  const [activeRow, setActiveRow] = useState<RowData | null>(null);
  const [boldValueNames, setBoldValueNames] = useState<string[]>([]);

  const rows = useMemo<RowData[]>(() => {
    if (!items) return [];
    return items.map((item, idx) => {
      const valuesMap = valuesToMap(item.values);
      const values: Record<string, number> = {};
      ticks.forEach((t) => {
        const raw = valuesMap.get(t.valueName);
        if (raw !== undefined) values[t.valueName] = raw / 1000;
      });
      const present = Object.values(values);
      return {
        identifier: item.identifier,
        values,
        min: present.length ? Math.min(...present) : 0,
        max: present.length ? Math.max(...present) : 0,
        color: colors ? toCssColor(colors[idx % colors.length][0]) : toCssColor(lineColors[2][0]),
      };
    });
  }, [items, ticks, colors]);

  const tickShape = useMemo(() => makeTickShape(ticks), [ticks]);
  const spanDataKey = (entry: RowData): [number, number] => [entry.min, entry.max];

  if ((!items || items.length === 0) || isLoading) {
    return (
      <div className={className}>
        <RankedChartSkeleton animate={isLoading} className="h-full min-h-[220px]" />
      </div>
    );
  }

  return (
    <div className={className}>
      <div className="relative h-[220px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={rows}
            layout="vertical"
            syncId={group}
            barSize={TICK_LINE_HEIGHT}
            margin={{ top: 8, right: 16, bottom: 8, left: 8 }}
            onMouseMove={(state) => {
              if (!state.isTooltipActive) {
                setHoverX(null);
                setActiveRow(null);
                setBoldValueNames([]);
                return;
              }
              setHoverX(state.chartX ?? null);
              setActiveRow((state.activePayload?.[0]?.payload as RowData | undefined) ?? null);
            }}
            onMouseLeave={() => {
              setHoverX(null);
              setActiveRow(null);
              setBoldValueNames([]);
            }}
          >
            <XAxis
              type="number"
              tickFormatter={axisFormat}
              tick={{ fontSize: 12 }}
              className="fill-basis"
              axisLine={false}
              tickLine={false}
            />
            <YAxis
              type="category"
              dataKey="identifier"
              tick={renderYAxisTick}
              axisLine={showYAxisLine}
              tickLine={showYAxisLine}
              width={140}
              interval={0}
            />
            <Tooltip
              cursor={{ fill: BORDER_SUBTLE_COLOR, opacity: 0.2 }}
              content={<RangePlotTooltip ticks={ticks} format={format} boldValueNames={boldValueNames} />}
              // Vertical-only — a wide row's tooltip near the top/bottom
              // edge can still escape upward/downward, but it stays
              // clamped within the plot horizontally rather than
              // overflowing the card.
              allowEscapeViewBox={{ x: false, y: true }}
              // recharts' tooltip wrapper is `tabIndex={-1} role="dialog"`,
              // programmatically focused whenever the tooltip shows —
              // including on plain mouse hover, not just real keyboard
              // navigation (tabIndex -1 means it's never reachable via
              // sequential Tab in the first place). Without this, the
              // browser's default focus outline shows on every hover.
              wrapperStyle={{ zIndex: 50, outline: 'none' }}
            />
            <Bar
              dataKey={spanDataKey}
              isAnimationActive={false}
              legendType="none"
              shape={tickShape}
              background={<BackgroundLineShape />}
            />
            <Customized
              component={
                <HoverLine hoverX={hoverX} activeRow={activeRow} ticks={ticks} onSnap={setBoldValueNames} />
              }
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
