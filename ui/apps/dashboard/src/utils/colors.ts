export const getRGBValues = (cssVar: string) =>
  typeof window !== 'undefined' &&
  typeof document !== 'undefined' &&
  window.getComputedStyle(document.documentElement).getPropertyValue(cssVar);

export const cssToRGB = (cssVar: string) => `rgb(${getRGBValues(cssVar)})`;
