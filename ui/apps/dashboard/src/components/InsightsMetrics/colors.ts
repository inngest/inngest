// This design system's color tokens compile to CSS custom properties with
// their own light/dark values (see globals.css), so a token resolves
// correctly in either theme without any JS dark-mode branching — as long as
// it's used as a real CSS value (a className, or a valid CSS color string),
// not the raw object resolveConfig() returns (see toCssColor below).
export const SURFACE_COLOR = 'rgb(var(--color-background-canvas-base))';
// Neutral gridline/axis/"other"-bucket color shared across every recharts
// chart in this feature.
export const BORDER_SUBTLE_COLOR = 'rgb(var(--color-border-subtle))';

// toCssColor turns a design-token color string (one entry of a [light, dark]
// tuple like `lineColors[n]`) into a value that's actually valid as a
// fill/stroke/backgroundColor. Two format quirks it works around:
// - Tailwind's resolved theme values (e.g. from `@/utils/tailwind`) embed a
//   literal "<alpha-value>" placeholder meant to be substituted at Tailwind's
//   own build time; reading them via resolveConfig() at runtime leaves that
//   placeholder in the string, so replace it with an explicit alpha of 1.
// - A bare `var(--color-x)` reference (rather than one already wrapped in
//   rgb(...)) holds this design system's space-separated "R G B" triplet,
//   which isn't a valid standalone color — wrap it in rgb(...).
export function toCssColor(value: string): string {
  const withAlpha = value.replace(/<alpha-value>/g, '1');
  return /^var\(--[\w-]+\)$/.test(withAlpha) ? `rgb(${withAlpha})` : withAlpha;
}
