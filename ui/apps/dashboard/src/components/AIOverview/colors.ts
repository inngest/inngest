// Shared between AIOverviewDashboard and FunctionAIPanel so the two pages'
// charts use identical colors for the same metrics. Literal `rgb(var(--x))`
// strings (see InsightsMetrics/colors.ts) rather than Tailwind's resolved
// theme object, which embeds a `<alpha-value>` placeholder that isn't valid
// CSS as-is.

// Fixed-order pastel palette (green, blue, yellow, orange, purple), matched
// against a reference mock — used both for per-category color (top
// functions by usage) and as the single-hue override for individual charts
// (e.g. green for runs). Reuses the design system's "subtle" tier tokens;
// yellow has no dedicated categorical slot, so it references the honey
// scale's step 300 directly (verified against the live app — the
// warning/honey *semantic* tokens render as a burnt orange-rust in light
// mode, not yellow).
export const CHART_COLORS: readonly string[] = [
  'rgb(var(--color-primary-subtle))', // green
  'rgb(var(--color-secondary-subtle))', // blue
  'rgb(var(--color-honey-300))', // yellow
  'rgb(var(--color-quaternary-warmer-xSubtle))', // orange
  'rgb(var(--color-quaternary-cool-xSubtle))', // purple
];

// 3xSubtle blue/green for the Tokens over time area fill — the design
// system's most muted tier of the same secondary/primary hues
// InsightsMetrics/colors.ts's DEFAULT_PALETTE "moderate" tier uses.
export const SUBTLE_BLUE = 'rgb(var(--color-secondary-3xSubtle))';
export const SUBTLE_GREEN = 'rgb(var(--color-primary-3xSubtle))';

// Tokens by model's stacked bars — matched pixel-for-pixel against a
// reference mock: blue reuses the same secondary 3xSubtle tier as Tokens
// over time, but green needed one tier down (2xSubtle, not 3xSubtle) to
// match — the two charts aren't a matched pair here.
export const TOKENS_BY_MODEL_GREEN = 'rgb(var(--color-primary-2xSubtle))';
