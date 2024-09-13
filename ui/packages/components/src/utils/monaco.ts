'use-client';

import resolveConfig from 'tailwindcss/resolveConfig';

import tailwindConfig from '../../tailwind.config';
import { resolveColor as resolver } from './colors';

const {
  theme: { backgroundColor, textColor, borderColor },
} = resolveConfig(tailwindConfig);

const defaultColor = '#f6f6f6'; // carbon 50

// Transform css variables into format that monaco can read
export function resolveColor(colorValue: string, isDark: boolean): string {
  return resolver(colorValue, isDark, defaultColor);
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
