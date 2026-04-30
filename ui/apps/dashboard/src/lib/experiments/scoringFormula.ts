import type {
  ExperimentDetail,
  ExperimentScoringMetric,
} from '@inngest/components/Experiments';

import {
  serializeExperimentScoringFormula,
  type ExperimentScoringFormula,
  type ExperimentScoringFormulaMetric,
} from '@/lib/experiments/urlState';

type MetricRange = { min: number; max: number };

export function deriveDefaultScoringMetrics(
  detail: ExperimentDetail,
): ExperimentScoringMetric[] {
  const ranges = new Map<string, MetricRange>();

  for (const variant of detail.variants) {
    for (const metric of variant.metrics) {
      const range = ranges.get(metric.key);
      if (!range) {
        ranges.set(metric.key, { min: metric.avg, max: metric.avg });
        continue;
      }

      if (metric.avg < range.min) range.min = metric.avg;
      if (metric.avg > range.max) range.max = metric.avg;
    }
  }

  const keys = [...ranges.keys()];
  const points = distributePoints(keys.length);

  return keys.map((key, index) => {
    const range = ranges.get(key)!;
    const normalizedRange = normalizeRange(range);

    return {
      key,
      enabled: true,
      points: points[index],
      minValue: normalizedRange.min,
      maxValue: normalizedRange.max,
      invert: false,
      labelBest: 'Best',
      labelWorst: 'Worst',
      displayName: key,
    };
  });
}

export function getScoringMetricsForFormula({
  detail,
  formula,
}: {
  detail: ExperimentDetail;
  formula: ExperimentScoringFormula | null;
}): ExperimentScoringMetric[] {
  const defaults = deriveDefaultScoringMetrics(detail);
  if (!formula) return defaults;

  const formulaByKey = new Map(
    formula.metrics.map((metric) => [metric.key, metric]),
  );

  return defaults.map((defaultMetric) => {
    const override = formulaByKey.get(defaultMetric.key);
    return override ? { ...defaultMetric, ...override } : defaultMetric;
  });
}

export function updateScoringMetric(
  metrics: ExperimentScoringMetric[],
  key: string,
  patch: Partial<ExperimentScoringMetric>,
): ExperimentScoringMetric[] {
  return metrics.map((metric) =>
    metric.key === key ? { ...metric, ...patch } : metric,
  );
}

export function serializeScoringFormulaOverrides(
  metrics: ExperimentScoringMetric[],
  defaults: ExperimentScoringMetric[],
): string | undefined {
  const overrides = getScoringFormulaOverrides(metrics, defaults);
  return overrides.length > 0
    ? serializeExperimentScoringFormula(overrides)
    : undefined;
}

function getScoringFormulaOverrides(
  metrics: ExperimentScoringMetric[],
  defaults: ExperimentScoringMetric[],
): ExperimentScoringFormulaMetric[] {
  const defaultsByKey = new Map(defaults.map((metric) => [metric.key, metric]));
  const overrides: ExperimentScoringFormulaMetric[] = [];

  for (const metric of metrics) {
    const defaultMetric = defaultsByKey.get(metric.key)!;

    const override: ExperimentScoringFormulaMetric = { key: metric.key };
    if (metric.enabled !== defaultMetric.enabled) {
      override.enabled = metric.enabled;
    }
    if (metric.points !== defaultMetric.points) {
      override.points = metric.points;
    }
    if (metric.minValue !== defaultMetric.minValue) {
      override.minValue = metric.minValue;
    }
    if (metric.maxValue !== defaultMetric.maxValue) {
      override.maxValue = metric.maxValue;
    }
    if (metric.invert !== defaultMetric.invert) {
      override.invert = metric.invert;
    }
    if (metric.labelBest !== defaultMetric.labelBest) {
      override.labelBest = metric.labelBest;
    }
    if (metric.labelWorst !== defaultMetric.labelWorst) {
      override.labelWorst = metric.labelWorst;
    }
    if (metric.displayName !== defaultMetric.displayName) {
      override.displayName = metric.displayName;
    }

    if (Object.keys(override).length > 1) overrides.push(override);
  }

  return overrides;
}

export function getPointsLeft(metrics: ExperimentScoringMetric[]): number {
  const allocated = metrics
    .filter((metric) => metric.enabled)
    .reduce((sum, metric) => sum + metric.points, 0);
  return 100 - allocated;
}

function distributePoints(count: number): number[] {
  if (count <= 0) return [];

  const base = Math.floor(100 / count);
  let remainder = 100 - base * count;

  return Array.from({ length: count }, () => {
    const pointValue = base + (remainder > 0 ? 1 : 0);
    remainder -= 1;
    return pointValue;
  });
}

function normalizeRange(range: MetricRange): MetricRange {
  if (range.max > range.min) return range;

  return {
    min: range.min,
    max: range.min + 1,
  };
}
