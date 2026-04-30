import { useMemo } from 'react';
import { Card } from '@inngest/components/Card';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import { cn } from '@inngest/components/utils/classNames';
import { RiTrophyLine } from '@remixicon/react';
import {
  Bar,
  BarChart,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { computeChartSizing, truncateCenter } from '@/lib/experiments/chart';
import { colorForMetric } from '@/lib/experiments/colors';
import type { ScoredVariant } from '@/lib/experiments/score';
import { ChartTooltip } from './ChartTooltip';
import { ScoreCalculationExplainer } from './ScoreCalculationExplainer';
import { VariantAxisTick } from './VariantAxisTick';

type BackgroundLineProps = {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
};

function BackgroundLineShape({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
}: BackgroundLineProps) {
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
  scoredVariants: ScoredVariant[];
  metrics: ExperimentScoringMetric[];
  className?: string;
};

type RowData = {
  variantName: string;
  total: number;
  [metricKey: string]: string | number;
};

export function ScoreSummaryCard({
  scoredVariants,
  metrics,
  className,
}: Props) {
  const enabledMetrics = useMemo(
    () => metrics.filter((m) => m.enabled),
    [metrics],
  );

  const { rows, ranked } = useMemo(() => {
    const built = scoredVariants.map(({ variant, result }) => {
      const row: RowData = {
        variantName: variant.variantName,
        total: result.total,
      };
      for (const seg of result.segments) {
        row[seg.metricKey] = seg.contribution;
      }
      return { variantName: variant.variantName, total: result.total, row };
    });

    const sorted = [...built].sort((a, b) => b.total - a.total);
    return { rows: built.map((b) => b.row), ranked: sorted };
  }, [scoredVariants]);

  const maxPossible = useMemo(
    () => enabledMetrics.reduce((acc, m) => acc + m.points, 0),
    [enabledMetrics],
  );

  const { chartHeight, yAxisWidth } = useMemo(() => {
    const sizing = computeChartSizing(rows.map((r) => r.variantName));
    // Reserve room for the metric legend below the bars.
    return { ...sizing, chartHeight: sizing.chartHeight + 28 };
  }, [rows]);
  const topVariant = ranked[0] ?? null;
  const runnerUp = ranked[1] ?? null;

  return (
    <Card
      className={cn('overflow-visible', className)}
      contentClassName="overflow-visible"
    >
      <Card.Header className="rounded-t-md border-b-0 py-2 pl-3 pr-2">
        <div className="flex items-center gap-1.5">
          <span className="text-basis text-sm font-medium">Score Summary</span>
          <ScoreCalculationExplainer />
        </div>
      </Card.Header>
      <Card.Content className="flex gap-6 rounded-b-md px-2 py-0">
        <div className="min-w-0 flex-1">
          <ResponsiveContainer width="100%" height={chartHeight}>
            <BarChart
              data={rows}
              layout="vertical"
              barSize={10}
              margin={{ top: 0, right: 16, bottom: 0, left: 4 }}
            >
              <XAxis
                type="number"
                domain={[0, maxPossible || 100]}
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
              <Legend
                verticalAlign="bottom"
                align="left"
                height={24}
                iconType="circle"
                iconSize={8}
                wrapperStyle={{ fontSize: 12 }}
              />
              {enabledMetrics.map((m, i) => (
                <Bar
                  key={m.key}
                  dataKey={m.key}
                  stackId="score"
                  fill={colorForMetric(i)}
                  name={m.displayName}
                  background={i === 0 ? <BackgroundLineShape /> : undefined}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Callouts */}
        <div className="flex w-64 shrink-0 flex-col gap-1">
          {topVariant && (
            <div className="bg-primary-3xSubtle flex items-center gap-2 rounded px-2 py-1">
              <RiTrophyLine className="text-primary-intense h-[18px] w-[18px] shrink-0" />
              <p
                className="text-primary-intense min-w-0 truncate text-sm"
                title={topVariant.variantName}
              >
                Recommended: {truncateCenter(topVariant.variantName)}
              </p>
            </div>
          )}
          {runnerUp && (
            <div className="flex items-center gap-2 px-2 py-1">
              <span className="text-subtle shrink-0 text-sm">#2</span>
              <p
                className="text-subtle min-w-0 truncate text-sm"
                title={runnerUp.variantName}
              >
                Runner up: {truncateCenter(runnerUp.variantName)}
              </p>
            </div>
          )}
        </div>
      </Card.Content>
    </Card>
  );
}
