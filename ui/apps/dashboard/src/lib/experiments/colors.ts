/**
 * Palette resolves to Tailwind-defined CSS variables. All entries use the
 * "moderate" tone so that contrast against `canvasBase` is uniform in both
 * light and dark mode.
 */
export const METRIC_PALETTE = [
  'rgb(var(--color-primary-moderate))',
  'rgb(var(--color-quaternary-cool-moderate))',
  'rgb(var(--color-secondary-moderate))',
  'rgb(var(--color-accent-moderate))',
  'rgb(var(--color-tertiary-moderate))',
  'rgb(var(--color-quaternary-warmer-moderate))',
] as const;

export function colorForMetric(index: number): string {
  return METRIC_PALETTE[index % METRIC_PALETTE.length];
}

export function colorForVariant(index: number): string {
  return METRIC_PALETTE[index % METRIC_PALETTE.length];
}
