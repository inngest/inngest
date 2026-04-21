import { useMemo } from 'react';
import { Card } from '@inngest/components/Card';
import { Pill } from '@inngest/components/Pill';
import type {
  ExperimentScoringMetric,
  ExperimentVariantMetrics,
} from '@inngest/components/Experiments';
import { RiTrophyLine } from '@remixicon/react';
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { computeChartSizing, truncateCenter } from '@/lib/experiments/chart';
import {
  colorForVariant,
  subtleColorForVariant,
} from '@/lib/experiments/colors';
import { findExtremum } from '@/lib/experiments/score';
import { ChartTooltip } from './ChartTooltip';
import { VariantAxisTick } from './VariantAxisTick';

const DOT_RADIUS = 5;
const LINE_HEIGHT = 2;

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

type Props = {
  metric: ExperimentScoringMetric;
  variants: ExperimentVariantMetrics[];
};

type RowData = {
  variantName: string;
  value: number;
  /** Index of the variant in the shared list, used to pick a stable palette color. */
  variantIndex: number;
};

export function MetricPanel({ metric, variants }: Props) {
  const rows: RowData[] = useMemo(
    () =>
      variants
        .map((v, variantIndex) => {
          const m = v.metrics.find((vm) => vm.key === metric.key);
          return m
            ? { variantName: v.variantName, value: m.avg, variantIndex }
            : null;
        })
        .filter((r): r is RowData => r !== null),
    [variants, metric.key],
  );

  const winner = useMemo(
    () =>
      findExtremum(rows, (r) => r.value, metric.invert)?.variantName ?? null,
    [rows, metric.invert],
  );

  const domain = useMemo<[number, number]>(() => {
    let hi = metric.maxValue;
    for (const row of rows) {
      hi = Math.max(hi, row.value);
    }
    if (hi <= 0) hi = 1;
    return [0, hi];
  }, [rows, metric.maxValue]);

  const { chartHeight, yAxisWidth } = useMemo(
    () => computeChartSizing(rows.map((r) => r.variantName)),
    [rows],
  );

  return (
    <Card className="overflow-visible" contentClassName="overflow-visible">
      <Card.Header className="flex-row items-center justify-between rounded-t-md border-b-0 py-2 pl-3 pr-2">
        <span className="text-basis text-sm font-medium">
          {metric.displayName}
        </span>
        {winner && (
          <Pill
            kind="default"
            appearance="solidBright"
            icon={<RiTrophyLine className="h-3 w-3" />}
            iconSide="left"
          >
            <span title={winner}>{truncateCenter(winner)}</span>
          </Pill>
        )}
      </Card.Header>
      <Card.Content className="flex items-center justify-center rounded-b-md px-2 py-0">
        <ResponsiveContainer width="100%" height={chartHeight}>
          <BarChart
            data={rows}
            layout="vertical"
            barSize={DOT_RADIUS * 2}
            margin={{ top: 0, right: DOT_RADIUS + 2, bottom: 0, left: 4 }}
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
              name={metric.displayName}
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
          </BarChart>
        </ResponsiveContainer>
      </Card.Content>
    </Card>
  );
}
