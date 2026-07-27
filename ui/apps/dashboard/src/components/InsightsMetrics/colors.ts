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

// Default series/category palette — a fixed-order pastel palette (green,
// blue, yellow, orange, purple), matched against a reference mock. Used as
// the default fill/stroke color by every chart in this package whenever a
// caller doesn't pass its own `color`/`colors`/`series[].color`. Reuses the
// design system's "subtle" tier tokens; yellow has no dedicated categorical
// slot, so it references the honey scale's step 300 directly (verified
// against the live app — the warning/honey *semantic* tokens render as a
// burnt orange-rust in light mode, not yellow).
export const CHART_COLORS: readonly string[] = [
  'rgb(var(--color-primary-subtle))', // green
  'rgb(var(--color-secondary-subtle))', // blue
  'rgb(var(--color-honey-300))', // yellow
  'rgb(var(--color-quaternary-warmer-xSubtle))', // orange
  'rgb(var(--color-quaternary-cool-xSubtle))', // purple
];

// Lighter counterparts of CHART_COLORS, index-paired — for a fill that sits
// behind a CHART_COLORS stroke (e.g. a box plot's IQR box, a candle body),
// the same pairing lib/experiments/colors.ts' METRIC_PALETTE/
// METRIC_PALETTE_SUBTLE already establishes for that package's own palette.
export const CHART_COLORS_SUBTLE: readonly string[] = [
  'rgb(var(--color-primary-3xSubtle))', // green
  'rgb(var(--color-secondary-3xSubtle))', // blue
  'rgb(var(--color-honey-100))', // yellow
  'rgb(var(--color-quaternary-warmer-3xSubtle))', // orange
  'rgb(var(--color-quaternary-cool-3xSubtle))', // purple
];
