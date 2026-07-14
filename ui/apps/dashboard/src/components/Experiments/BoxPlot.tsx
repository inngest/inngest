import { useMemo, useState } from 'react';
import type { ExperimentVariantMetrics } from '@inngest/components/Experiments';
import {
  Bar,
  BarChart as RechartsBarChart,
  Customized,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { computeChartSizing } from '@/lib/experiments/chart';
import {
  colorForVariant,
  subtleColorForVariant,
} from '@/lib/experiments/colors';
import { BoxPlotTooltip } from './BoxPlotTooltip';
import { VariantAxisTick } from './VariantAxisTick';

const BOX_HEIGHT = 10;
const LINE_HEIGHT = 1;
const LINE_WIDTH = 1;
/** Snap within this many pixels of a named data point. */
const SNAP_PX = 4;

export type RowData = {
  variantName: string;
  value: number;
  /** Index of the variant in the shared list, used to pick a stable palette color. */
  variantIndex: number;

  runCount: number;
  avg: number;
  stddev: number;

  min: number;
  q1: number;
  med: number;
  q3: number;
  max: number;
  color: string;
  subtleColor: string;
  opacity: number;
};

export function rowsForMetric(
  variants: ExperimentVariantMetrics[],
  metricKey: string,
  colorIndexForVariant?: Map<string, number>,
): RowData[] {
  return variants
    .map((v, variantIndex) => {
      const m = v.metrics.find((vm) => vm.key === metricKey);
      const colorIndex = colorIndexForVariant?.get(v.variantName) ?? variantIndex;
      return m
        ? {
            variantName: v.variantName,
            variantIndex: colorIndex,
            value: m.avg,
            runCount: v.runCount,
            avg: m.avg,
            stddev: m.stddev,
            min: m.min,
            q1: m.q1,
            med: m.med,
            q3: m.q3,
            max: m.max,
            color: colorForVariant(colorIndex),
            subtleColor: subtleColorForVariant(colorIndex),
            opacity: 1,
          }
        : null;
    })
    .filter((r): r is RowData => r !== null);
}

type BarShapeProps = {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  fill?: string;
  payload?: RowData;
};

function BoxShape({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
  payload,
}: BarShapeProps) {
  if (payload === undefined) return null;
  const opacity = payload.opacity ?? 1;
  const range = payload.max - payload.min;
  if (range === 0) {
    const cy = y + height / 2;
    const r = BOX_HEIGHT / 2;
    return (
      <>
        {opacity < 1 && (
          <circle cx={x} cy={cy} r={r} fill="rgb(var(--color-background-canvas-base))" />
        )}
        <g opacity={opacity}>
          <circle cx={x} cy={cy} r={r} fill={payload.subtleColor} stroke={payload.color} strokeWidth={LINE_WIDTH} />
        </g>
      </>
    );
  }

  const cy = y + height / 2;
  const cyLine = cy + LINE_HEIGHT / 4;

  const quantiles = [
    payload.min,
    payload.q1,
    payload.med,
    payload.q3,
    payload.max,
  ];
  const quantileOffsets = quantiles.map(
    (q) => ((q - payload.min) / range) * width,
  );
  const quantileXs = quantileOffsets.map((offset) => x + offset);

  return (
    <>
      {opacity < 1 && (
        <rect
          x={quantileXs[1]}
          y={y}
          width={quantileXs[3] - quantileXs[1]}
          height={height}
          fill="rgb(var(--color-background-canvas-base))"
        />
      )}
      <g opacity={opacity}>
        <rect
          x={quantileXs[1]}
          y={y}
          width={quantileXs[3] - quantileXs[1]}
          height={height}
          fill={payload?.subtleColor}
          stroke={payload?.color}
          strokeWidth={LINE_WIDTH}
        />
        {quantileXs.map((qx, i) => (
          <line
            key={`quantile-${i}`}
            x1={qx}
            x2={qx}
            y1={y}
            y2={y + height}
            stroke={payload?.color}
            strokeWidth={LINE_WIDTH}
          />
        ))}
        <line
          x1={quantileXs[0]}
          x2={quantileXs[1]}
          y1={cyLine}
          y2={cyLine}
          stroke={payload?.color}
          strokeWidth={LINE_HEIGHT}
        />
        <line
          x1={quantileXs[3]}
          x2={quantileXs[4]}
          y1={cyLine}
          y2={cyLine}
          stroke={payload?.color}
          strokeWidth={LINE_HEIGHT}
        />
      </g>
    </>
  );
}

function BackgroundLineShape({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
}: BarShapeProps) {
  const cy = y + height / 2;
  return (
    <rect
      x={x}
      y={cy - 0.5}
      width={width}
      height={1}
      fill="rgb(var(--color-border-subtle))"
    />
  );
}

const SNAP_VALUE_FNS: ((r: RowData) => number)[] = [
  (r) => r.min,
  (r) => r.q1,
  (r) => r.med,
  (r) => r.q3,
  (r) => r.max,
];

type RechartScale = { (v: number): number; invert?: (px: number) => number };
type AxisEntry = {
  x: number;
  width: number;
  y: number;
  height: number;
  scale?: RechartScale;
};

type HoverLineProps = {
  xAxisMap?: Record<string, AxisEntry>;
  yAxisMap?: Record<string, { y: number; height: number }>;
  /** Raw cursor x in SVG space. */
  hoverX: number | null;
  /** Hovered row — snapping is restricted to this row's values. */
  activeRow: RowData | null;
};

function HoverLine({
  xAxisMap,
  yAxisMap,
  hoverX,
  activeRow,
}: HoverLineProps) {
  if (hoverX === null || !xAxisMap || !yAxisMap) return null;

  const xAxis = Object.values(xAxisMap)[0];
  const yAxis = Object.values(yAxisMap)[0];
  if (!xAxis || !yAxis) return null;

  const scale = xAxis.scale;
  let plotX = hoverX;

  if (activeRow && scale) {
    // Compare and snap entirely in pixel space using recharts' own scale —
    // scale(v) returns the exact SVG x that recharts uses for that data value.
    let bestDist = Infinity;
    let bestPx: number | null = null;

    for (const fn of SNAP_VALUE_FNS) {
      const v = fn(activeRow);
      if (v < activeRow.min || v > activeRow.max) continue;
      const px = scale(v);
      const dist = Math.abs(px - hoverX);
      if (dist < bestDist) {
        bestDist = dist;
        bestPx = px;
      }
    }

    if (bestDist <= SNAP_PX && bestPx !== null) {
      plotX = bestPx;
    }
  }

  const chartLeft = xAxis.x;
  const chartRight = xAxis.x + xAxis.width;
  plotX = Math.min(Math.max(plotX, chartLeft), chartRight);

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

type Props = {
  rows: RowData[];
  domain: [number, number];
  metricDisplayName: string;
  hoveredVariantName?: string | null;
  onVariantHover?: (name: string | null) => void;
};

export function BoxPlot({ rows, domain, metricDisplayName, hoveredVariantName, onVariantHover }: Props) {
  const { chartHeight, yAxisWidth } = computeChartSizing(
    rows.map((r) => r.variantName),
  );

  const [hoverX, setHoverX] = useState<number | null>(null);
  const [activeRow, setActiveRow] = useState<RowData | null>(null);

  const displayRows = useMemo(
    () =>
      rows.map((r) => ({
        ...r,
        // Only dim when the highlight comes from another chart (this chart has no active row)
        opacity: !hoveredVariantName || activeRow !== null || r.variantName === hoveredVariantName ? 1 : 0.25,
      })),
    [rows, hoveredVariantName, activeRow],
  );

  const boxDataKey: (entry: RowData) => [number, number] = (entry) => [
    entry.min,
    entry.max,
  ];

  return (
    <ResponsiveContainer width="100%" height={chartHeight}>
      <RechartsBarChart
        data={displayRows}
        layout="vertical"
        barSize={BOX_HEIGHT * 2}
        margin={{ top: 0, right: 16, bottom: 0, left: 4 }}
        onMouseMove={(state) => {
          if (!state.isTooltipActive) {
            setHoverX(null);
            setActiveRow(null);
            onVariantHover?.(null);
            return;
          }
          const row = (state.activePayload?.[0]?.payload as RowData | undefined) ?? null;
          setHoverX(state.chartX ?? null);
          setActiveRow(row);
          onVariantHover?.(row?.variantName ?? null);
        }}
        onMouseLeave={() => {
          setHoverX(null);
          setActiveRow(null);
          onVariantHover?.(null);
        }}
      >
        <XAxis
          type="number"
          domain={domain}
          tick={{ fontSize: 12 }}
          axisLine={false}
          tickLine={false}
          tickFormatter={(v: number) => +v.toFixed(2) + ''}
        />
        <YAxis
          type="category"
          dataKey="variantName"
          width={yAxisWidth}
          axisLine={false}
          tickLine={false}
          tick={<VariantAxisTick />}
          interval={0}
        />
        <Tooltip
          content={<BoxPlotTooltip />}
          cursor={{ fill: 'rgb(var(--color-background-canvas-subtle))' }}
          allowEscapeViewBox={{ x: true, y: true }}
          wrapperStyle={{ zIndex: 50, outline: 'none' }}
        />
        <Bar
          dataKey={boxDataKey}
          name={metricDisplayName}
          shape={<BoxShape />}
          background={<BackgroundLineShape />}
        />
        <Customized
          component={<HoverLine hoverX={hoverX} activeRow={activeRow} />}
        />
      </RechartsBarChart>
    </ResponsiveContainer>
  );
}
