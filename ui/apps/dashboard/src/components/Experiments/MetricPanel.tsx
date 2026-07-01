import { useMemo } from 'react';
import { Card } from '@inngest/components/Card';
import { Pill } from '@inngest/components/Pill';
import {
  type ExperimentScoringMetric,
  type ExperimentVariantMetrics,
} from '@inngest/components/Experiments';
import { RiTrophyLine } from '@remixicon/react';

import { truncateCenter } from '@/lib/experiments/chart';
import { findExtremum } from '@/lib/experiments/score';
import { BooleanChart } from './BooleanChart';
import { BoxPlot, rowsForMetric } from './BoxPlot';

type Props = {
  metric: ExperimentScoringMetric;
  variants: ExperimentVariantMetrics[];
};

export function MetricPanel({ metric, variants }: Props) {
  const rows = useMemo(
    () => rowsForMetric(variants, metric.key),
    [variants, metric.key],
  );

  const winner = useMemo(
    () =>
      findExtremum(rows, (r) => r.value, metric.invert)?.variantName ?? null,
    [rows, metric.invert],
  );

  const domain = useMemo<[number, number]>(() => {
    let hi = metric.maxValue;
    let low = metric.minValue;
    for (const row of rows) {
      hi = Math.max(hi, row.max);
      low = Math.min(low, row.min);
    }
    return [low, hi];
  }, [rows, metric.minValue, metric.maxValue]);

  const plot = useMemo(() => {
    switch (metric.kind) {
      case 'BOOLEAN':
        return (
          <BooleanChart
            rows={rows}
            domain={domain}
            metricDisplayName={metric.displayName}
          />
        );
      case 'NUMERIC':
      default:
        return (
          <BoxPlot
            rows={rows}
            domain={domain}
            metricDisplayName={metric.displayName}
          />
        );
    }
  }, [metric.kind, rows, domain, metric.displayName]);

  return (
    <Card className="overflow-visible" contentClassName="overflow-visible">
      <Card.Header className="flex-row items-center justify-between rounded-t-md border-b-0 py-2 pl-3 pr-2">
        <span className="text-basis text-sm">{metric.displayName}</span>
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
        {plot}
      </Card.Content>
    </Card>
  );
}
