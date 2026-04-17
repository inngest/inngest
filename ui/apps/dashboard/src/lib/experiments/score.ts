import type {
  ExperimentScoringMetric,
  ExperimentVariantMetrics,
  VariantMetric,
} from '@inngest/components/Experiments';

export type ScoreSegment = {
  metricKey: string;
  contribution: number;
};

export type ScoreResult = {
  total: number;
  maxPossible: number;
  segments: ScoreSegment[];
};

export type ScoredVariant = {
  variant: ExperimentVariantMetrics;
  result: ScoreResult;
};

function clamp01(n: number): number {
  if (Number.isNaN(n)) return 0;
  if (n < 0) return 0;
  if (n > 1) return 1;
  return n;
}

export function scoreVariant(
  variantMetrics: VariantMetric[],
  config: ExperimentScoringMetric[],
): ScoreResult {
  const byKey = new Map(variantMetrics.map((m) => [m.key, m]));
  let total = 0;
  let maxPossible = 0;
  const segments: ScoreSegment[] = [];

  for (const cfg of config) {
    if (!cfg.enabled) continue;
    maxPossible += cfg.points;

    const v = byKey.get(cfg.key);
    if (!v) continue;

    const span = cfg.maxValue - cfg.minValue;
    if (span <= 0) continue;

    let norm = clamp01((v.avg - cfg.minValue) / span);
    if (cfg.invert) norm = 1 - norm;

    const contribution = cfg.points * norm;
    total += contribution;
    segments.push({ metricKey: cfg.key, contribution });
  }

  return { total, maxPossible, segments };
}

export function scoreVariants(
  variants: ExperimentVariantMetrics[],
  config: ExperimentScoringMetric[],
): ScoredVariant[] {
  return variants.map((variant) => ({
    variant,
    result: scoreVariant(variant.metrics, config),
  }));
}

/**
 * Returns the item with the highest value (or lowest, when `invert` is true).
 * Returns null for an empty list. Ties resolve to the first item.
 */
export function findExtremum<T>(
  items: readonly T[],
  getValue: (item: T) => number,
  invert = false,
): T | null {
  if (items.length === 0) return null;
  let best = items[0];
  let bestVal = getValue(best);
  for (let i = 1; i < items.length; i++) {
    const item = items[i];
    const val = getValue(item);
    if (invert ? val < bestVal : val > bestVal) {
      best = item;
      bestVal = val;
    }
  }
  return best;
}

/**
 * Best and worst items by `getValue`. When `invert` is true, "best" = lowest.
 * Returns null when fewer than 2 items are provided (no meaningful contrast).
 */
export function findBestAndWorst<T>(
  items: readonly T[],
  getValue: (item: T) => number,
  invert = false,
): { best: T; worst: T } | null {
  if (items.length < 2) return null;
  const best = findExtremum(items, getValue, invert)!;
  const worst = findExtremum(items, getValue, !invert)!;
  return { best, worst };
}
