'use-client';

import resolveConfig from 'tailwindcss/resolveConfig';

import tailwindConfig from '../../tailwind.config';

const {
  theme: { backgroundColor, textColor, borderColor },
} = resolveConfig(tailwindConfig);

const defaultColor = '#f6f6f6'; // carbon 50

// Transform css variables into format that monaco can read
function resolveColor(colorValue: string): string {
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

  // Get the computed style
  const computedStyle = window.getComputedStyle(document.documentElement);

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

export const RULES = [
  {
    token: 'delimiter.bracket.json',
    foreground: resolveColor(textColor.codeDelimiterBracketJson),
  },
  {
    token: 'string.key.json',
    foreground: resolveColor(textColor.codeStringKeyJson),
  },
  {
    token: 'number.json',
    foreground: resolveColor(textColor.codeNumberJson),
  },
  {
    token: 'string.value.json',
    foreground: resolveColor(textColor.codeStringValueJson),
  },
  {
    token: 'keyword.json',
    foreground: resolveColor(textColor.codeKeyword),
  },
  {
    token: 'comment',
    fontStyle: 'italic',
    foreground: resolveColor(textColor.codeComment),
  },
  {
    token: 'string',
    foreground: resolveColor(textColor.codeString),
  },
  {
    token: 'keyword',
    foreground: resolveColor(textColor.codeKeyword),
  },
  {
    token: 'entity.name.function',
    foreground: resolveColor(textColor.codeEntityNameFunction),
  },
];
console.log(backgroundColor.codeEditor);

export const COLORS = {
  'editor.background': resolveColor(backgroundColor.codeEditor),
  'editorLineNumber.foreground': resolveColor(textColor.subtle),
  'editorLineNumber.activeForeground': resolveColor(textColor.basis),
  'editorWidget.background': resolveColor(backgroundColor.codeEditor),
  'editorWidget.border': resolveColor(borderColor.subtle),
  'editorBracketHighlight.foreground1': resolveColor(textColor.warning),
};

export const LINE_HEIGHT = 26;
export const FONT = {
  size: 13,
  type: 'monospace',
  font: 'CircularXXMono',
};
