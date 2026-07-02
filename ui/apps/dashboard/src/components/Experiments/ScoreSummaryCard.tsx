import { useMemo, useState } from 'react';
import { Card } from '@inngest/components/Card';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import { cn } from '@inngest/components/utils/classNames';
import {
  Bar,
  BarChart,
  Cell,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { computeChartSizing } from '@/lib/experiments/chart';
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
  payload?: { opacity?: number };
};

function BackgroundLineShape({
  x = 0,
  y = 0,
  width = 0,
  height = 0,
  payload,
}: BackgroundLineProps) {
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

type Props = {
  scoredVariants: ScoredVariant[];
  metrics: ExperimentScoringMetric[];
  className?: string;
  hoveredVariantName?: string | null;
  onVariantHover?: (name: string | null) => void;
};

type RowData = {
  variantName: string;
  total: number;
  runCount: number;
  opacity: number;
  [metricKey: string]: string | number;
};

export function ScoreSummaryCard({
  scoredVariants,
  metrics,
  className,
  hoveredVariantName,
  onVariantHover,
}: Props) {
  const [activeVariantName, setActiveVariantName] = useState<string | null>(null);

  const enabledMetrics = useMemo(
    () => metrics.filter((m) => m.enabled),
    [metrics],
  );

  // Only dim when the highlight comes from another chart (not from this one).
  const effectiveHighlight = activeVariantName ? null : (hoveredVariantName ?? null);

  const rows = useMemo(() => {
    return scoredVariants.map(({ variant, result }) => {
      const row: RowData = {
        variantName: variant.variantName,
        total: result.total,
        runCount: variant.runCount,
        opacity: effectiveHighlight && variant.variantName !== effectiveHighlight ? 0.25 : 1,
      };
      for (const seg of result.segments) {
        row[seg.metricKey] = seg.contribution;
      }
      return row;
    });
  }, [scoredVariants, effectiveHighlight]);

  const maxPossible = useMemo(
    () => enabledMetrics.reduce((acc, m) => acc + m.points, 0),
    [enabledMetrics],
  );

  const { chartHeight, yAxisWidth } = useMemo(() => {
    const sizing = computeChartSizing(rows.map((r) => r.variantName));
    // Reserve room for the metric legend below the bars.
    return { ...sizing, chartHeight: Math.max(200, sizing.chartHeight + 28) };
  }, [rows]);

  return (
    <Card
      className={cn('overflow-visible', className)}
      contentClassName="overflow-visible"
    >
      <Card.Header className="rounded-t-md border-b-0 py-2 pl-3 pr-2">
        <div className="flex items-center gap-1.5">
          <span className="text-basis text-sm">Score Summary</span>
          <ScoreCalculationExplainer />
        </div>
      </Card.Header>
      <Card.Content className="flex gap-6 rounded-b-md px-2 py-2">
        <div className="min-w-0 flex-1">
          <ResponsiveContainer width="100%" height={chartHeight}>
            <BarChart
              data={rows}
              layout="vertical"
              barSize={10}
              margin={{ top: 0, right: 16, bottom: 0, left: 4 }}
              onMouseMove={(state) => {
                if (!state.isTooltipActive) {
                  setActiveVariantName(null);
                  onVariantHover?.(null);
                  return;
                }
                const name = (state.activePayload?.[0]?.payload as RowData | undefined)?.variantName ?? null;
                setActiveVariantName(name);
                onVariantHover?.(name);
              }}
              onMouseLeave={() => {
                setActiveVariantName(null);
                onVariantHover?.(null);
              }}
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
                axisLine={false}
                tickLine={false}
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
                // Keep the colored dot but render the label in the muted text
                // tone instead of the series color (recharts' default).
                formatter={(value: string) => (
                  <span style={{ color: 'rgb(var(--color-foreground-muted))' }}>
                    {value}
                  </span>
                )}
              />
              {enabledMetrics.map((m, i) => (
                <Bar
                  key={m.key}
                  dataKey={m.key}
                  stackId="score"
                  fill={colorForMetric(i)}
                  name={m.displayName}
                  background={i === 0 ? <BackgroundLineShape /> : undefined}
                >
                  {rows.map((row) => (
                    <Cell key={row.variantName} fill={colorForMetric(i)} opacity={row.opacity} />
                  ))}
                </Bar>
              ))}
            </BarChart>
          </ResponsiveContainer>
        </div>
      </Card.Content>
    </Card>
  );
}
