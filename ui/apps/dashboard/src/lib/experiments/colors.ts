/**
 * Palette resolves to Tailwind-defined CSS variables. All entries use the
 * "moderate" tone so that contrast against `canvasBase` is uniform in both
 * light and dark mode.
 */
export const METRIC_PALETTE = [
  'rgb(var(--color-primary-moderate))',
  'rgb(var(--color-secondary-xSubtle))',
  'rgb(var(--color-accent-subtle))',
  'rgb(var(--color-quaternary-cool-moderate))',
  'rgb(var(--color-tertiary-moderate))',
  'rgb(var(--color-quaternary-warmer-moderate))',
] as const;

/**
 * Lighter counterparts of METRIC_PALETTE, used for fills that sit behind the
 * primary stroke (e.g. the circle caps on lollipop charts).
 */
export const METRIC_PALETTE_SUBTLE = [
  'rgb(var(--color-primary-3xSubtle))',
  'rgb(var(--color-quaternary-cool-3xSubtle))',
  'rgb(var(--color-secondary-3xSubtle))',
  'rgb(var(--color-accent-3xSubtle))',
  'rgb(var(--color-tertiary-3xSubtle))',
  'rgb(var(--color-quaternary-warmer-3xSubtle))',
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

type MetricLike = { key: string; enabled: boolean };

/**
 * Maps each enabled metric's key to its chart color. Colors are assigned by
 * position among *enabled* metrics (not the full list) so the result lines up
 * exactly with the Score Summary chart, which colors its stacked segments the
 * same way. Disabled metrics are omitted (callers fall back to a neutral tone).
 */
export function buildMetricColorMap(
  metrics: MetricLike[],
): Record<string, string> {
  const map: Record<string, string> = {};
  let enabledIndex = 0;
  for (const metric of metrics) {
    if (!metric.enabled) continue;
    map[metric.key] = colorForMetric(enabledIndex);
    enabledIndex += 1;
  }
  return map;
}
