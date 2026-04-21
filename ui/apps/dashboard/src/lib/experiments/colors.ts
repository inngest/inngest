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
