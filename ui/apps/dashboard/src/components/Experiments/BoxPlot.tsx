import { useState } from 'react';
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
const LINE_HEIGHT = 2;

export type RowData = {
  variantName: string;
  value: number;
  /** Index of the variant in the shared list, used to pick a stable palette color. */
  variantIndex: number;

  avg: number;
  stddev: number;

  min: number;
  q1: number;
  med: number;
  q3: number;
  max: number;
  color: string;
  subtleColor: string;
};

export function rowsForMetric(
  variants: ExperimentVariantMetrics[],
  metricKey: string,
): RowData[] {
  return variants
    .map((v, variantIndex) => {
      const m = v.metrics.find((vm) => vm.key === metricKey);
      return m
        ? {
            variantName: v.variantName,
            variantIndex,
            value: m.avg,
            avg: m.avg,
            stddev: m.stddev,
            min: m.min,
            q1: m.q1,
            med: m.med,
            q3: m.q3,
            max: m.max,
            color: colorForVariant(variantIndex),
            subtleColor: subtleColorForVariant(variantIndex),
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
  const range = payload.max - payload.min;
  if (range === 0) {
    return (
      <g>
        <rect
          x={x}
          y={y}
          width={width}
          height={height}
          fill={payload?.subtleColor}
          stroke={payload?.color}
        />
      </g>
    );
  }

  const cy = y + height / 2;
  const cyLine = cy + LINE_HEIGHT / 2;

  const quantiles = [
    payload.min,
    payload.q1,
    payload.med,
    payload.q3,
    payload.max,
  ];
  const quantileOffsets = quantiles.map(
    (q) => ((payload.min + q) / range) * width,
  );
  const quantileXs = quantileOffsets.map((offset) => x + offset);

  const zscores = [-1, 0, 1].map((z) => payload.avg + z * payload.stddev);
  const zOffsets = zscores
    .filter((z) => z >= payload.min && z <= payload.max)
    .map((z) => ((payload.min + z) / range) * width);
  const zXs = zOffsets.map((offset) => x + offset);

  return (
    <g>
      <rect
        x={quantileXs[1]}
        y={y}
        width={quantileXs[3] - quantileXs[1]}
        height={height}
        fill={payload?.subtleColor}
        stroke={payload?.color}
        strokeWidth={2}
      />
      {quantileXs.map((qx, i) => (
        <line
          key={`quantile-${i}`}
          x1={qx}
          x2={qx}
          y1={y}
          y2={y + height}
          stroke={payload?.color}
          strokeWidth={2}
        />
      ))}
      <line
        x1={quantileXs[0]}
        x2={quantileXs[1]}
        y1={cyLine}
        y2={cyLine}
        stroke={payload?.color}
        strokeWidth={2}
      />
      <line
        x1={quantileXs[3]}
        x2={quantileXs[4]}
        y1={cyLine}
        y2={cyLine}
        stroke={payload?.color}
        strokeWidth={2}
      />
      {zXs.map((zx, i) => (
        <line
          key={`zscore-${i}`}
          x1={zx}
          x2={zx}
          y1={y}
          y2={y + height}
          stroke={payload?.color}
          strokeWidth={2}
          strokeDasharray="2 2"
        />
      ))}
    </g>
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

type HoverLineProps = {
  xAxisMap?: Record<
    string,
    { x: number; width: number; height: number; y: number }
  >;
  yAxisMap?: Record<string, { y: number; height: number }>;
  hoverX: number | null;
};

function HoverLine({ xAxisMap, yAxisMap, hoverX }: HoverLineProps) {
  if (hoverX === null || !xAxisMap || !yAxisMap) return null;

  const xAxis = Object.values(xAxisMap)[0];
  const yAxis = Object.values(yAxisMap)[0];
  if (!xAxis || !yAxis) return null;

  const chartLeft = xAxis.x;
  const chartRight = xAxis.x + xAxis.width;
  const plotX = Math.min(Math.max(hoverX, chartLeft), chartRight);
  const y1 = yAxis.y;
  const y2 = yAxis.y + yAxis.height;

  return (
    <line
      x1={plotX}
      x2={plotX}
      y1={y1}
      y2={y2}
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
};

export function BoxPlot({ rows, domain, metricDisplayName }: Props) {
  const { chartHeight, yAxisWidth } = computeChartSizing(
    rows.map((r) => r.variantName),
  );
  const boxDataKey: (entry: RowData) => [number, number] = (entry) => [
    entry.min,
    entry.max,
  ];
  const [hoverX, setHoverX] = useState<number | null>(null);

  return (
    <ResponsiveContainer width="100%" height={chartHeight}>
      <RechartsBarChart
        data={rows}
        layout="vertical"
        barSize={BOX_HEIGHT * 2}
        margin={{ top: 0, right: 0, bottom: 0, left: 4 }}
        onMouseMove={(state) => {
          setHoverX(state.isTooltipActive ? state.chartX ?? null : null);
        }}
        onMouseLeave={() => setHoverX(null)}
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
        <Customized component={<HoverLine hoverX={hoverX} />} />
      </RechartsBarChart>
    </ResponsiveContainer>
  );
}
