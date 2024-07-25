'use-client';

import resolveConfig from 'tailwindcss/resolveConfig';

import tailwindConfig from '../../tailwind.config';

const {
  theme: { backgroundColor, textColor, borderColor },
} = resolveConfig(tailwindConfig);

function resolveColor(colorValue) {
  if (typeof window === 'undefined') {
    // We're in a server-side environment
    return '#ffffff'; // Default dark color
  }

  // Extract the CSS variable name from the color value
  const variableName = colorValue.match(/var\((.*?)\)/)[1];

  // Get the computed style
  const computedStyle = window.getComputedStyle(document.documentElement);

  // Get the RGB values
  const rgbValues = computedStyle.getPropertyValue(variableName).trim();

  // If we couldn't get the RGB values, return a default color
  if (!rgbValues) {
    return '#ffffff';
  }

  // Split the RGB values and convert to numbers
  const [r, g, b] = rgbValues.split(' ').map(Number);

  // Convert to hex
  return rgbToHex(r, g, b);
}

function rgbToHex(r, g, b) {
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
