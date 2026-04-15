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
  const topVariant = ranked[0] ?? null;
  const runnerUp = ranked[1] ?? null;

  return (
    <Card>
      <Card.Header>
        <span className="text-basis text-sm font-medium">Score Summary</span>
      </Card.Header>
      <Card.Content className="flex gap-6">
        <div className="min-w-0 flex-1">
          <ResponsiveContainer width="100%" height={chartHeight}>
            <BarChart
              data={rows}
              layout="vertical"
              margin={{ top: 4, right: 16, bottom: 4, left: 4 }}
            >
              <XAxis type="number" domain={[0, maxPossible || 100]} hide />
              <YAxis
                type="category"
                dataKey="variantName"
                width={100}
                tick={{ fontSize: 12 }}
              />
              <Tooltip />
              {enabledMetrics.map((m) => (
                <Bar
                  key={m.key}
                  dataKey={m.key}
                  stackId="score"
                  fill={colorForMetric(m.key)}
                  name={m.displayName}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Callouts */}
        <div className="flex w-48 shrink-0 flex-col justify-center gap-3">
          {topVariant && (
            <div className="flex items-start gap-2">
              <RiTrophyLine className="text-accent-intense mt-0.5 h-4 w-4 shrink-0" />
              <div>
                <p className="text-muted text-xs">Recommended</p>
                <p className="text-basis text-sm font-medium">
                  {topVariant.variantName}
                </p>
                <p className="text-muted text-xs tabular-nums">
                  {topVariant.total.toFixed(1)} pts
                </p>
              </div>
            </div>
          )}
          {runnerUp && (
            <div className="flex items-start gap-2 opacity-60">
              <span className="text-muted mt-0.5 text-xs font-medium">#2</span>
              <div>
                <p className="text-muted text-xs">Runner up</p>
                <p className="text-basis text-sm">{runnerUp.variantName}</p>
                <p className="text-muted text-xs tabular-nums">
                  {runnerUp.total.toFixed(1)} pts
                </p>
              </div>
            </div>
          )}
        </div>
      </Card.Content>
    </Card>
  );
}
