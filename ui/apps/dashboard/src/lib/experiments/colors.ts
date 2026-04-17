/**
 * Palette resolves to Tailwind-defined CSS variables. These are chart-safe
 * tokens from the design system using moderate/subtle tones.
 */
export const METRIC_PALETTE = [
  'rgb(var(--color-primary-moderate))',
  'rgb(var(--color-quaternary-cool-moderate))',
  'rgb(var(--color-secondary-xSubtle))',
  'rgb(var(--color-accent-xSubtle))',
  'rgb(var(--color-tertiary-xSubtle))',
  'rgb(var(--color-quaternary-warmer-moderate))',
] as const;

export function colorForMetric(index: number): string {
  return METRIC_PALETTE[index % METRIC_PALETTE.length];
}
