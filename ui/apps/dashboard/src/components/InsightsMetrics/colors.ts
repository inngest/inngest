// This design system's color tokens compile to CSS custom properties with
// their own light/dark values (see globals.css), so a token resolves
// correctly in either theme without any JS dark-mode branching, as long as
// it's referenced as `rgb(var(--color-x))` directly. Defined here as literal
// strings (same approach as lib/experiments/colors.ts) rather than read
// through Tailwind's resolved theme object (`@/utils/tailwind`'s `colors`),
// which embeds a `<alpha-value>` placeholder meant for Tailwind's own build
// step and isn't valid CSS as-is.
export const SURFACE_COLOR = 'rgb(var(--color-background-canvas-base))';
// Neutral gridline/axis/"other"-bucket color shared across every recharts
// chart in this feature.
export const BORDER_SUBTLE_COLOR = 'rgb(var(--color-border-subtle))';

// Default series/category palette — the same five tokens as Metrics/
// utils.ts's `lineColors` (the ECharts dashboards' own default palette),
// reproduced as ready-to-use CSS strings so every chart in this feature can
// use them directly.
export const DEFAULT_PALETTE: readonly string[] = [
  'rgb(var(--color-accent-subtle))',
  'rgb(var(--color-primary-moderate))',
  'rgb(var(--color-secondary-moderate))',
  'rgb(var(--color-tertiary-moderate))',
  'rgb(var(--color-quaternary-cool-xIntense))',
];
