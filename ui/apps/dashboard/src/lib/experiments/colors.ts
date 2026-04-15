/**
 * Palette resolves to Tailwind-defined CSS variables. These are chart-safe
 * tokens from the design system.
 *
 * Available intense tokens: primary, secondary, tertiary, accent (intense),
 * plus xIntense variants and quaternary-cool-xIntense.
 */
export const METRIC_PALETTE = [
  'rgb(var(--color-primary-intense))',
  'rgb(var(--color-secondary-intense))',
  'rgb(var(--color-accent-intense))',
  'rgb(var(--color-tertiary-intense))',
  'rgb(var(--color-quaternary-cool-xIntense))',
  'rgb(var(--color-primary-xIntense))',
  'rgb(var(--color-secondary-xIntense))',
  'rgb(var(--color-tertiary-xIntense))',
] as const;

function djb2(str: string): number {
  let hash = 5381;
  for (let i = 0; i < str.length; i++) {
    hash = (hash * 33) ^ str.charCodeAt(i);
  }
  return hash >>> 0;
}

export function colorForMetric(key: string): string {
  const idx = djb2(key) % METRIC_PALETTE.length;
  const color = METRIC_PALETTE[idx];
  if (!color) throw new Error('colorForMetric: palette index out of bounds');
  return color;
}
