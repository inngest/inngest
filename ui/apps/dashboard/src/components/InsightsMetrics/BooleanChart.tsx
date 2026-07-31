import { useMemo, useState } from 'react';
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
import { AxisTick } from './AxisTick';
import { BooleanChartTooltip } from './BooleanChartTooltip';

const DOT_RADIUS = 5;

// One horizontal lollipop row: a single scalar value (e.g. fraction-true for
// a boolean score, or a boolean-kind experiment metric's average) plotted as
// a dot at the end of a line from the axis — BoxPlot's RowData without the
// quartile stats, since a boolean has nothing to draw a box from.
export type RowData = {
  label: string;
  avg: number;
  count?: number;
  color: string;
  subtleColor: string;
  opacity: number;
};

type BarShapeProps = {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  fill?: string;
  payload?: { subtleColor?: string; opacity?: number };
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
  const dotFill = payload?.subtleColor ?? 'rgb(var(--color-background-canvas-base))';
  return (
    <g opacity={payload?.opacity ?? 1}>
      <line
        x1={x}
        y1={cy}
        x2={x + width}
        y2={cy}
        stroke={fill}
        strokeWidth={1.25}
      />
      <circle
        cx={x + width}
        cy={cy}
        r={DOT_RADIUS}
        fill={dotFill}
        stroke={fill}
      />
    </g>
  );
}

function BackgroundLineShape({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
  payload,
}: BarShapeProps) {
  if ((payload?.opacity ?? 1) < 1) return null;
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
    const snapPx = scale(activeRow.avg);
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
  hoveredLabel?: string | null;
  onLabelHover?: (label: string | null) => void;
  format?: (value: number) => string;
  countLabel?: string;
};

export function BooleanChart({
  rows,
  domain,
  metricDisplayName,
  hoveredLabel,
  onLabelHover,
  format,
  countLabel,
}: Props) {
  const { chartHeight, yAxisWidth } = computeChartSizing(rows.map((r) => r.label));
  const [hoverX, setHoverX] = useState<number | null>(null);
  const [activeRow, setActiveRow] = useState<RowData | null>(null);

  const displayRows = useMemo(
    () =>
      rows.map((r) => ({
        ...r,
        // Only dim when the highlight comes from another chart (this chart has no active row)
        opacity: !hoveredLabel || activeRow !== null || r.label === hoveredLabel ? 1 : 0.25,
      })),
    [rows, hoveredLabel, activeRow],
  );

  return (
    <ResponsiveContainer width="100%" height={chartHeight}>
      <BarChart
        data={displayRows}
        layout="vertical"
        barSize={DOT_RADIUS * 2}
        margin={{ top: 0, right: 16, bottom: 0, left: 4 }}
        onMouseMove={(state) => {
          if (!state.isTooltipActive) {
            setHoverX(null);
            setActiveRow(null);
            onLabelHover?.(null);
            return;
          }
          const row = (state.activePayload?.[0]?.payload as RowData | undefined) ?? null;
          setHoverX(state.chartX ?? null);
          setActiveRow(row);
          onLabelHover?.(row?.label ?? null);
        }}
        onMouseLeave={() => {
          setHoverX(null);
          setActiveRow(null);
          onLabelHover?.(null);
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
          dataKey="label"
          width={yAxisWidth}
          tick={<AxisTick />}
          axisLine={false}
          tickLine={false}
          interval={0}
        />
        <Tooltip
          content={<BooleanChartTooltip format={format} countLabel={countLabel} />}
          cursor={{ fill: 'rgb(var(--color-background-canvas-subtle))' }}
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 50, outline: 'none' }}
        />
        <Bar
          dataKey="avg"
          name={metricDisplayName}
          shape={<LineDotShape />}
          background={<BackgroundLineShape />}
        >
          {rows.map((row) => (
            <Cell key={row.label} fill={row.color} />
          ))}
        </Bar>
        <Customized
          component={<HoverLine hoverX={hoverX} activeRow={activeRow} />}
        />
      </BarChart>
    </ResponsiveContainer>
  );
}
