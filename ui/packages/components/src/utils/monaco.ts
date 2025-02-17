'use-client';

import { languages } from 'monaco-editor';
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
  {
    token: 'predefined', // Matches built-in commands like 'npx', 'curl'
    foreground: resolveColor(textColor.codeEntityNameFunction, isDark),
  },
  {
    token: 'identifier',
    foreground: resolveColor(textColor.codeKeyword, isDark),
  },
  {
    token: 'delimiter',
    foreground: resolveColor(textColor.codeDelimiterBracketJson, isDark),
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

export const shellLanguageTokens: languages.IMonarchLanguage = {
  defaultToken: '',
  ignoreCase: true,
  tokenPostfix: '.shell',
  brackets: [
    { token: 'delimiter.bracket', open: '{', close: '}' },
    { token: 'delimiter.parenthesis', open: '(', close: ')' },
    { token: 'delimiter.square', open: '[', close: ']' },
  ],
  keywords: [
    'if',
    'then',
    'do',
    'else',
    'elif',
    'while',
    'until',
    'for',
    'in',
    'esac',
    'fi',
    'fin',
    'fil',
    'done',
    'exit',
    'set',
    'unset',
    'export',
    'function',
  ],
  builtins: [
    'ab',
    'awk',
    'bash',
    'beep',
    'bun',
    'cat',
    'cc',
    'cd',
    'chown',
    'chmod',
    'chroot',
    'clear',
    'cp',
    'curl',
    'cut',
    'diff',
    'echo',
    'find',
    'gawk',
    'gcc',
    'get',
    'git',
    'grep',
    'hg',
    'kill',
    'killall',
    'ln',
    'ls',
    'make',
    'mkdir',
    'mv',
    'nc',
    'node',
    'npm',
    'npx',
    'openssl',
    'ping',
    'pnpm',
    'ps',
    'restart',
    'rm',
    'rmdir',
    'sed',
    'service',
    'sh',
    'shopt',
    'shred',
    'source',
    'sort',
    'sleep',
    'ssh',
    'start',
    'stop',
    'su',
    'sudo',
    'svn',
    'tee',
    'telnet',
    'top',
    'touch',
    'vi',
    'vim',
    'wall',
    'wc',
    'wget',
    'who',
    'write',
    'yarn',
    'yes',
    'zsh',
  ],
  symbols: /[=><!~?&|+\-*\/\^;\.,]+/,
  escapes: /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{1,4}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})/,
  tokenizer: {
    root: [
      [
        /[a-zA-Z][\w-]*(@[a-zA-Z0-9.-]+)?/,
        { cases: { '@builtins': 'predefined', '@default': 'type' } },
      ],
      [/[\w-]+/, 'identifier'],
      [/[a-zA-Z][\w-@]*/, { cases: { '@builtins': 'predefined', '@default': 'identifier' } }],
      { include: '@whitespace' },
      { include: '@strings' },
      [/[{}()\[\]]/, '@brackets'],
      [/@symbols/, 'delimiter'],
      [/\d*\.\d+([eE][-+]?\d+)?/, 'number.float'],
      [/0[xX][0-9a-fA-F]+/, 'number.hex'],
      [/\d+/, 'number'],
      [/[;,.]/, 'delimiter'],
    ],
    whitespace: [
      [/[ \t\r\n]+/, 'white'],
      [/^\s*#.*$/, 'comment'],
    ],
    strings: [
      [/'/, { token: 'string.quote', bracket: '@open', next: '@stringBody' }],
      [/"/, { token: 'string.quote', bracket: '@open', next: '@dblStringBody' }],
    ],
    stringBody: [
      [/'/, { token: 'string.quote', bracket: '@close', next: '@pop' }],
      [/./, 'string'],
    ],
    dblStringBody: [
      [/"/, { token: 'string.quote', bracket: '@close', next: '@pop' }],
      [/./, 'string'],
    ],
  },
};

export const celLanguageTokens: languages.IMonarchLanguage = {
  tokenizer: {
    root: [
      // Identifying keywords
      [/\b(true|false|null)\b/, 'keyword.constant'],
      [/\b(in|map|list|as|and|or|not)\b/, 'keyword.operator'],
      [/\b(int|bool|string|double|bytes)\b/, 'keyword.type'],

      // Identifying function calls (e.g. size, exists, all, etc.)
      [/\b(size|exists|all|has)\b/, 'function'],

      // Identifying identifiers (variables or field names)
      [/[a-zA-Z_]\w*/, 'identifier'],

      // Identifying string literals (single and double-quoted)
      [/"([^"\\]|\\.)*"/, 'string'], // Double-quoted string with escaped characters
      [/'([^'\\]|\\.)*'/, 'string'], // Single-quoted string with escaped characters

      // Identifying numbers (only if not within a string)
      [/\b\d+(\.\d+)?\b/, 'number'],

      // Identifying comments (single-line and multi-line)
      [/\/\/.*$/, 'comment'],
      [/\/\*.*\*\//, 'comment'],

      // Identifying operators
      [/[=!<>]=|[-+*/%]/, 'operator'],

      // Identifying parentheses, brackets, and curly braces
      [/[\[\](){}]/, '@brackets'],

      // Identifying whitespace
      [/\s+/, 'white'],
    ],
  },
};
