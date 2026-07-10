/**
 * Palette resolves to Tailwind-defined CSS variables. All entries use the
 * "moderate" tone so that contrast against `canvasBase` is uniform in both
 * light and dark mode.
 */
export const METRIC_PALETTE = [
  'rgb(var(--color-primary-moderate))',
  'rgb(var(--color-secondary-subtle))',
  'rgb(var(--color-quaternary-warmer-moderate))',
  'rgb(var(--color-quaternary-cool-moderate))',
  'rgb(var(--color-tertiary-moderate))',
] as const;

/**
 * Lighter counterparts of METRIC_PALETTE, used for fills that sit behind the
 * primary stroke (e.g. the circle caps on lollipop charts).
 */
export const METRIC_PALETTE_SUBTLE = [
  'rgb(var(--color-primary-3xSubtle))',
  'rgb(var(--color-secondary-3xSubtle))',
  'rgb(var(--color-quaternary-warmer-3xSubtle))',
  'rgb(var(--color-quaternary-cool-3xSubtle))',
  'rgb(var(--color-tertiary-3xSubtle))',
] as const;

export function colorForMetric(index: number): string {
  return METRIC_PALETTE[index % METRIC_PALETTE.length];
}

export function colorForVariant(index: number): string {
  return METRIC_PALETTE[index % METRIC_PALETTE.length];
}

export function subtleColorForVariant(index: number): string {
  return METRIC_PALETTE_SUBTLE[index % METRIC_PALETTE_SUBTLE.length];
}

type MetricLike = { key: string };

/**
 * Maps each metric's key to its chart color. Colors are assigned by position in
 * the full metrics list so a metric keeps the same color regardless of whether
 * it (or any other metric) is enabled. The Score Summary chart builds its
 * segment colors from the same map, so the two views stay in sync.
 */
export function buildMetricColorMap(
  metrics: MetricLike[],
): Record<string, string> {
  const map: Record<string, string> = {};
  metrics.forEach((metric, index) => {
    map[metric.key] = colorForMetric(index);
  });
  return map;
}
