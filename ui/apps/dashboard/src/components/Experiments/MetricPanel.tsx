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
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { colorForMetric } from '@/lib/experiments/colors';

const DOT_RADIUS = 5;
const LINE_HEIGHT = 2;

function LineDotShape(props: any) {
  const { x, y, width, height, fill } = props;
  const cy = y + height / 2;
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
        fill="white"
        stroke={fill}
        strokeWidth={2}
      />
    </g>
  );
}

type Props = {
  metric: ExperimentScoringMetric;
  variants: ExperimentVariantMetrics[];
  colorIndex: number;
};

type RowData = {
  variantName: string;
  value: number;
};

function findWinner(rows: RowData[], invert: boolean): string | null {
  let best: RowData | null = null;

  for (const row of rows) {
    if (!best) {
      best = row;
      continue;
    }
    if (invert ? row.value < best.value : row.value > best.value) {
      best = row;
    }
  }

  return best?.variantName ?? null;
}

export function MetricPanel({ metric, variants, colorIndex }: Props) {
  const rows: RowData[] = useMemo(
    () =>
      variants
        .map((v) => {
          const m = v.metrics.find((vm) => vm.key === metric.key);
          return m ? { variantName: v.variantName, value: m.avg } : null;
        })
        .filter((r): r is RowData => r !== null),
    [variants, metric.key],
  );

  const winner = useMemo(
    () => findWinner(rows, metric.invert),
    [rows, metric.invert],
  );

  const domain = useMemo(() => {
    let hi = metric.maxValue;
    for (const row of rows) {
      hi = Math.max(hi, row.value);
    }
    if (hi <= 0) hi = 1;
    return [0, hi] as const;
  }, [rows, metric.maxValue]);

  const chartHeight = Math.max(120, rows.length * 36);
  const yAxisWidth = useMemo(() => {
    const longest = rows.reduce(
      (max, r) => Math.max(max, r.variantName.length),
      0,
    );
    return Math.max(80, longest * 6.5);
  }, [rows]);

  const color = colorForMetric(colorIndex);

  return (
    <Card className="overflow-visible" contentClassName="overflow-visible">
      <Card.Header className="flex-row items-center justify-between">
        <span className="text-basis text-sm font-medium">
          {metric.displayName}
        </span>
        {winner && (
          <Pill
            kind="primary"
            appearance="solidBright"
            icon={<RiTrophyLine className="h-3 w-3" />}
            iconSide="left"
          >
            {winner}
          </Pill>
        )}
      </Card.Header>
      <Card.Content>
        <ResponsiveContainer width="100%" height={chartHeight}>
          <BarChart
            data={rows}
            layout="vertical"
            barSize={DOT_RADIUS * 2}
            margin={{ top: 4, right: DOT_RADIUS + 2, bottom: 16, left: 4 }}
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
              tick={{ fontSize: 12 }}
            />
            <Tooltip
              allowEscapeViewBox={{ x: true, y: true }}
              wrapperStyle={{ zIndex: 50 }}
            />
            <Bar
              dataKey="value"
              fill={color}
              name={metric.displayName}
              shape={<LineDotShape />}
            />
          </BarChart>
        </ResponsiveContainer>
      </Card.Content>
    </Card>
  );
}
