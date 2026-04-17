import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { RiSettings3Line } from '@remixicon/react';

import { MetricAccordionItem } from '@/components/Experiments/ScoringFormulaSidebar';

import type { MetricStats } from './metricStats';

export function MetricSubLabel({
  variantName,
  avg,
  stats,
  metric,
}: {
  variantName: string;
  avg: number;
  stats: MetricStats | null;
  metric: ExperimentScoringMetric;
}) {
  if (!stats) return null;

  if (variantName === stats.bestVariant) {
    return (
      <span className="text-success text-[10px]">
        {metric.labelBest || 'Best'}
      </span>
    );
  }

  if (variantName === stats.worstVariant) {
    return (
      <span className="text-error text-[10px]">
        {metric.labelWorst || 'Worst'}
      </span>
    );
  }

  // Middle variant: delta vs best. Sign is flipped for inverted metrics so
  // "higher % vs best" always reads as "worse".
  if (stats.bestAvg === 0) return null;

  const rawDelta = ((avg - stats.bestAvg) / stats.bestAvg) * 100;
  const delta = rawDelta * (metric.invert ? -1 : 1);
  const sign = delta >= 0 ? '+' : '';
  return (
    <span className="text-muted text-[10px] tabular-nums">
      {sign}
      {delta.toFixed(1)}% vs best
    </span>
  );
}

export function MetricColumnHeader({
  metric,
  pointsLeft,
  onUpdateMetric,
}: {
  metric: ExperimentScoringMetric;
  pointsLeft: number;
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
}) {
  return (
    <div className="flex w-full items-center gap-1">
      <span className="text-muted min-w-0 flex-1 truncate text-xs font-medium">
        {metric.displayName}
      </span>
      <Popover>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="text-muted hover:text-basis ml-auto flex shrink-0 items-center"
          >
            <RiSettings3Line className="h-3.5 w-3.5" />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start">
          <MetricAccordionItem
            metric={metric}
            pointsLeft={pointsLeft}
            collapsible={false}
            onUpdate={(patch) => onUpdateMetric(metric.key, patch)}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}

export function AddMetricPopover({
  disabledMetrics,
  onEnable,
}: {
  disabledMetrics: ExperimentScoringMetric[];
  onEnable: (key: string) => void;
}) {
  return (
    <div className="flex w-52 flex-col gap-1 p-2">
      <p className="text-muted px-2 py-1 text-xs font-medium">
        Enable a metric
      </p>
      {disabledMetrics.map((m) => (
        <PopoverClose key={m.key} asChild>
          <button
            type="button"
            className="text-basis hover:bg-canvasSubtle rounded px-2 py-1.5 text-left text-sm"
            onClick={() => onEnable(m.key)}
          >
            {m.displayName}
          </button>
        </PopoverClose>
      ))}
    </div>
  );
}
