import { useState } from 'react';
import type { ExperimentVariantMetrics } from '@inngest/components/Experiments';
import {
  Bar,
  BarChart,
  Cell,
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
import { ChartTooltip } from './ChartTooltip';
import { VariantAxisTick } from './VariantAxisTick';

const DOT_RADIUS = 5;
const LINE_HEIGHT = 2;

export type RowData = {
  variantName: string;
  value: number;
  /** Index of the variant in the shared list, used to pick a stable palette color. */
  variantIndex: number;
};

export function rowsForMetric(
  variants: ExperimentVariantMetrics[],
  metricKey: string,
): RowData[] {
  return variants
    .map((v, variantIndex) => {
      const m = v.metrics.find((vm) => vm.key === metricKey);
      return m
        ? { variantName: v.variantName, value: m.avg, variantIndex }
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
  payload?: { variantIndex?: number };
};

function LineDotShape({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
  fill,
  payload,
}: BarShapeProps) {
  const cy = y + height / 2;
  const dotFill =
    payload?.variantIndex !== undefined
      ? subtleColorForVariant(payload.variantIndex)
      : 'rgb(var(--color-background-canvas-base))';
  return (
    <g>
      <rect
        x={x}
        y={cy - LINE_HEIGHT / 2}
        width={width}
        height={LINE_HEIGHT}
        fill={fill}
      />
      <circle
        cx={x + width}
        cy={cy}
        r={DOT_RADIUS}
        fill={dotFill}
        stroke={fill}
        strokeWidth={2}
      />
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

type RechartScale = { (v: number): number };
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
  hoverX: number | null;
  activeRow: RowData | null;
};

function HoverLine({ xAxisMap, yAxisMap, hoverX, activeRow }: HoverLineProps) {
  if (hoverX === null || !xAxisMap || !yAxisMap) return null;

  const xAxis = Object.values(xAxisMap)[0];
  const yAxis = Object.values(yAxisMap)[0];
  if (!xAxis || !yAxis) return null;

  const scale = xAxis.scale;
  let plotX = hoverX;

  if (activeRow && scale) {
    const snapPx = scale(activeRow.value);
    if (Math.abs(snapPx - hoverX) <= 4) {
      plotX = snapPx;
    }
  }

  plotX = Math.min(Math.max(plotX, xAxis.x), xAxis.x + xAxis.width);

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
};

export function BooleanChart({ rows, domain, metricDisplayName }: Props) {
  const { chartHeight, yAxisWidth } = computeChartSizing(
    rows.map((r) => r.variantName),
  );
  const [hoverX, setHoverX] = useState<number | null>(null);
  const [activeRow, setActiveRow] = useState<RowData | null>(null);

  return (
    <ResponsiveContainer width="100%" height={chartHeight}>
      <BarChart
        data={rows}
        layout="vertical"
        barSize={DOT_RADIUS * 2}
        margin={{ top: 0, right: DOT_RADIUS + 2, bottom: 0, left: 4 }}
        onMouseMove={(state) => {
          if (!state.isTooltipActive) {
            setHoverX(null);
            setActiveRow(null);
            return;
          }
          setHoverX(state.chartX ?? null);
          setActiveRow(
            (state.activePayload?.[0]?.payload as RowData | undefined) ?? null,
          );
        }}
        onMouseLeave={() => {
          setHoverX(null);
          setActiveRow(null);
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
          tick={<VariantAxisTick />}
          interval={0}
        />
        <Tooltip
          content={<ChartTooltip />}
          cursor={{ fill: 'rgb(var(--color-background-canvas-subtle))' }}
          allowEscapeViewBox={{ x: true, y: true }}
          wrapperStyle={{ zIndex: 50, outline: 'none' }}
        />
        <Bar
          dataKey="value"
          name={metricDisplayName}
          shape={<LineDotShape />}
          background={<BackgroundLineShape />}
        >
          {rows.map((row) => (
            <Cell
              key={row.variantName}
              fill={colorForVariant(row.variantIndex)}
            />
          ))}
        </Bar>
        <Customized
          component={<HoverLine hoverX={hoverX} activeRow={activeRow} />}
        />
      </BarChart>
    </ResponsiveContainer>
  );
}
