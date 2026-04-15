import type {
  ExperimentScoringMetric,
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
