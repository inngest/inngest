'use-client';

import resolveConfig from 'tailwindcss/resolveConfig';

import tailwindConfig from '../../tailwind.config';

const {
  theme: { backgroundColor, textColor, borderColor },
} = resolveConfig(tailwindConfig);

const defaultColor = '#f6f6f6'; // carbon 50

// Transform css variables into format that monaco can read
export function resolveColor(colorValue: string, isDark: boolean): string {
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
}

function rgbToHex(r: number, g: number, b: number): string {
  return (
    '#' +
    [r, g, b]
      .map((x) => {
        const hex = x.toString(16);
        return hex.length === 1 ? '0' + hex : hex;
      })
      .join('')
  );
}

export const createRules = (isDark: boolean) => [
  {
    token: 'delimiter.bracket.json',
    foreground: resolveColor(textColor.codeDelimiterBracketJson, isDark),
  },
  {
    token: 'string.key.json',
    foreground: resolveColor(textColor.codeStringKeyJson, isDark),
  },
  {
    token: 'number.json',
    foreground: resolveColor(textColor.codeNumberJson, isDark),
  },
  {
    token: 'number',
    foreground: resolveColor(textColor.codeNumberJson, isDark),
  },
  {
    token: 'string.value.json',
    foreground: resolveColor(textColor.codeStringValueJson, isDark),
  },
  {
    token: 'keyword.json',
    foreground: resolveColor(textColor.codeKeyword, isDark),
  },
  {
    token: 'comment',
    fontStyle: 'italic',
    foreground: resolveColor(textColor.codeComment, isDark),
  },
  {
    token: 'string',
    foreground: resolveColor(textColor.codeString, isDark),
  },
  {
    token: 'keyword',
    foreground: resolveColor(textColor.codeKeyword, isDark),
  },
  {
    token: 'entity.name.function',
    foreground: resolveColor(textColor.codeEntityNameFunction, isDark),
  },
  {
    token: 'type',
    foreground: resolveColor(textColor.codeStringValueJson, isDark),
  },
];

export const createColors = (isDark: boolean) => ({
  'editor.background': resolveColor(backgroundColor.codeEditor, isDark),
  'editorLineNumber.foreground': resolveColor(textColor.subtle, isDark),
  'editorLineNumber.activeForeground': resolveColor(textColor.basis, isDark),
  'editorWidget.background': resolveColor(backgroundColor.codeEditor, isDark),
  'editorWidget.border': resolveColor(borderColor.subtle, isDark),
  'editorBracketHighlight.foreground1': resolveColor(textColor.codeDelimiterBracketJson, isDark),
  'editorBracketHighlight.foreground2': resolveColor(textColor.codeDelimiterBracketJson, isDark),
  'editorBracketHighlight.foreground3': resolveColor(textColor.codeDelimiterBracketJson, isDark),
  'editorBracketHighlight.foreground4': resolveColor(textColor.codeDelimiterBracketJson, isDark),
});

export const LINE_HEIGHT = 26;
export const FONT = {
  size: 13,
  type: 'monospace',
  font: 'CircularXXMono',
};
