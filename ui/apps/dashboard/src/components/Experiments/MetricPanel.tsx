import { useMemo } from 'react';
import { Card } from '@inngest/components/Card';
import { Pill } from '@inngest/components/Pill';
import type {
  ExperimentScoringMetric,
  ExperimentVariantMetrics,
} from '@inngest/components/Experiments';
import { cn } from '@inngest/components/utils/classNames';
import { RiTrophyLine } from '@remixicon/react';

import { colorForMetric } from '@/lib/experiments/colors';

type Props = {
  metric: ExperimentScoringMetric;
  variants: ExperimentVariantMetrics[];
};

type VariantRow = {
  variantName: string;
  avg: number | null;
  min: number | null;
  max: number | null;
};

function findWinner(rows: VariantRow[], invert: boolean): string | null {
  let best: VariantRow | null = null;

  for (const row of rows) {
    if (row.avg == null) continue;
    if (!best || best.avg == null) {
      best = row;
      continue;
    }
    if (invert ? row.avg < best.avg : row.avg > best.avg) {
      best = row;
    }
  }

  return best?.variantName ?? null;
}

export function MetricPanel({ metric, variants }: Props) {
  const rows: VariantRow[] = useMemo(
    () =>
      variants.map((v) => {
        const m = v.metrics.find((vm) => vm.key === metric.key);
        return {
          variantName: v.variantName,
          avg: m?.avg ?? null,
          min: m?.min ?? null,
          max: m?.max ?? null,
        };
      }),
    [variants, metric.key],
  );

  const winner = useMemo(
    () => findWinner(rows, metric.invert),
    [rows, metric.invert],
  );

  // Compute shared x-domain across all variants
  const domain = useMemo(() => {
    let lo = metric.minValue;
    let hi = metric.maxValue;

    for (const row of rows) {
      if (row.min != null) lo = Math.min(lo, row.min);
      if (row.max != null) hi = Math.max(hi, row.max);
    }

    // Avoid zero-width domain
    if (hi <= lo) hi = lo + 1;
    return { lo, hi };
  }, [rows, metric.minValue, metric.maxValue]);

  const color = colorForMetric(metric.key);

  return (
    <Card>
      <Card.Header className="flex-row items-center justify-between">
        <span className="text-basis text-sm font-medium">
          {metric.displayName}
        </span>
        {winner && (
          <Pill
            kind="primary"
            appearance="outlined"
            icon={<RiTrophyLine className="h-3 w-3" />}
            iconSide="left"
          >
            {winner}
          </Pill>
        )}
      </Card.Header>
      <Card.Content className="flex flex-col gap-2">
        {rows.map((row) => {
          const hasData = row.avg != null && row.min != null && row.max != null;

          return (
            <div key={row.variantName} className="flex items-center gap-3">
              <span className="text-muted w-32 shrink-0 truncate text-xs">
                {row.variantName}
              </span>

              <div className="relative h-6 flex-1">
                {hasData ? (
                  <VariantTrack
                    min={row.min!}
                    max={row.max!}
                    avg={row.avg!}
                    domainLo={domain.lo}
                    domainHi={domain.hi}
                    color={color}
                    isWinner={row.variantName === winner}
                  />
                ) : (
                  <div className="bg-canvasSubtle absolute inset-x-0 top-1/2 h-px -translate-y-1/2" />
                )}
              </div>
            </div>
          );
        })}

        {/* Range labels */}
        <div className="ml-[140px] flex justify-between">
          <span className="text-disabled text-[10px] tabular-nums">
            {metric.labelWorst || domain.lo.toFixed(1)}
          </span>
          <span className="text-disabled text-[10px] tabular-nums">
            {metric.labelBest || domain.hi.toFixed(1)}
          </span>
        </div>
      </Card.Content>
    </Card>
  );
}

function VariantTrack({
  min,
  max,
  avg,
  domainLo,
  domainHi,
  color,
  isWinner,
}: {
  min: number;
  max: number;
  avg: number;
  domainLo: number;
  domainHi: number;
  color: string;
  isWinner: boolean;
}) {
  const span = domainHi - domainLo;
  const toPercent = (val: number) => ((val - domainLo) / span) * 100;

  const leftPct = toPercent(min);
  const rightPct = toPercent(max);
  const avgPct = toPercent(avg);

  return (
    <div className="absolute inset-0 flex items-center">
      {/* Subtle background track */}
      <div className="bg-canvasSubtle absolute inset-x-0 top-1/2 h-px -translate-y-1/2" />

      {/* Range line from min to max */}
      <div
        className={cn(
          'absolute top-1/2 h-0.5 -translate-y-1/2',
          isWinner ? 'opacity-100' : 'opacity-60',
        )}
        style={{
          left: `${leftPct}%`,
          width: `${rightPct - leftPct}%`,
          backgroundColor: color,
        }}
      />

      {/* Dot at avg */}
      <div
        className={cn(
          'absolute top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full',
          isWinner ? 'ring-2 ring-white' : '',
        )}
        style={{
          left: `${avgPct}%`,
          backgroundColor: color,
        }}
      />
    </div>
  );
}
