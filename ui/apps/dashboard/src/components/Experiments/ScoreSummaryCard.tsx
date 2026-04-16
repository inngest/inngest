import { useMemo } from 'react';
import { Card } from '@inngest/components/Card';
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
import { scoreVariant } from '@/lib/experiments/score';

type Props = {
  variants: ExperimentVariantMetrics[];
  metrics: ExperimentScoringMetric[];
};

type RowData = {
  variantName: string;
  total: number;
  [metricKey: string]: string | number;
};

export function ScoreSummaryCard({ variants, metrics }: Props) {
  const enabledMetrics = useMemo(
    () => metrics.filter((m) => m.enabled),
    [metrics],
  );

  const { rows, ranked } = useMemo(() => {
    const scored = variants.map((v) => {
      const result = scoreVariant(v.metrics, metrics);
      const row: RowData = { variantName: v.variantName, total: result.total };

      for (const seg of result.segments) {
        row[seg.metricKey] = seg.contribution;
      }

      return { variantName: v.variantName, total: result.total, row };
    });

    // Sort descending by total score
    const sorted = [...scored].sort((a, b) => b.total - a.total);

    return {
      rows: scored.map((s) => s.row),
      ranked: sorted,
    };
  }, [variants, metrics]);

  const maxPossible = useMemo(
    () => enabledMetrics.reduce((acc, m) => acc + m.points, 0),
    [enabledMetrics],
  );

  const chartHeight = Math.max(120, rows.length * 36);
  const yAxisWidth = useMemo(() => {
    const longest = rows.reduce(
      (max, r) => Math.max(max, r.variantName.length),
      0,
    );
    return Math.max(80, longest * 6.5);
  }, [rows]);
  const topVariant = ranked[0] ?? null;
  const runnerUp = ranked[1] ?? null;

  return (
    <Card contentClassName="overflow-visible">
      <Card.Header>
        <span className="text-basis text-sm font-medium">Score Summary</span>
      </Card.Header>
      <Card.Content className="flex gap-6">
        <div className="min-w-0 flex-1">
          <ResponsiveContainer width="100%" height={chartHeight}>
            <BarChart
              data={rows}
              layout="vertical"
              barSize={10}
              margin={{ top: 4, right: 16, bottom: 16, left: 4 }}
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
                tick={{ fontSize: 12 }}
              />
              <Tooltip
                allowEscapeViewBox={{ x: true, y: true }}
                wrapperStyle={{ zIndex: 50 }}
              />
              {enabledMetrics.map((m, i) => (
                <Bar
                  key={m.key}
                  dataKey={m.key}
                  stackId="score"
                  fill={colorForMetric(i)}
                  name={m.displayName}
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
              <p className="text-primary-intense min-w-0 truncate text-sm">
                Recommended: {topVariant.variantName}
              </p>
            </div>
          )}
          {runnerUp && (
            <div className="flex items-center gap-2 px-2 py-1">
              <span className="text-subtle shrink-0 text-sm">#2</span>
              <p className="text-subtle min-w-0 truncate text-sm">
                Runner up: {runnerUp.variantName}
              </p>
            </div>
          )}
        </div>
      </Card.Content>
    </Card>
  );
}
