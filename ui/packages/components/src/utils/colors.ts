export const rgbToHex = (r: number, g: number, b: number): string =>
  '#' +
  [r, g, b]
    .map((x) => {
      const hex = x.toString(16);
      return hex.length === 1 ? '0' + hex : hex;
    })
    .join('');

export const resolveColor = (
  colorValue: string,
  isDark: boolean,
  defaultColor: string = '#f6f6f6' // carbon 50
): string => {
  if (typeof window === 'undefined') {
    return defaultColor;
  }

  // Extract the CSS variable name from the color value
  const match = colorValue.match(/var\((.*?)\)/);
  if (!match || !match[1]) {
    console.warn(`Invalid color value format: ${colorValue}`);
    return defaultColor;
  }
  const variableName = match[1];

  // Get the appropriate root element based on isDark
  const root = isDark
    ? document.querySelector('.dark') || document.documentElement
    : document.documentElement;

  // Get the computed style
  const computedStyle = window.getComputedStyle(root);

  // Get the RGB values
  const rgbValues = computedStyle.getPropertyValue(variableName).trim();

  if (!rgbValues) {
    console.warn(`Could not resolve color for variable: ${variableName}`);
    return defaultColor;
  }

  // Split the RGB values and convert to numbers
  const rgbArray = rgbValues.split(' ').map(Number);

  if (rgbArray.length !== 3 || rgbArray.some(isNaN)) {
    console.warn(`Invalid RGB values: ${rgbValues}`);
    return defaultColor;
  }

  const [r, g, b] = rgbArray;

  if (typeof r !== 'number' || typeof g !== 'number' || typeof b !== 'number') {
    console.warn(`Unexpected non-number values in RGB: ${rgbValues}`);
    return defaultColor;
  }

  return rgbToHex(r, g, b);
};
