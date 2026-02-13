import type { editor } from 'monaco-editor';

import { FONT, LINE_HEIGHT } from '../utils/monaco';

export const EDITOR_OPTIONS: editor.IEditorOptions = {
  autoClosingBrackets: 'beforeWhitespace',
  autoClosingQuotes: 'beforeWhitespace',
  contextmenu: false,
  fixedOverflowWidgets: true,
  fontFamily: FONT.font,
  fontSize: FONT.size,
  fontWeight: 'light',
  guides: {
    indentation: false,
    highlightActiveBracketPair: false,
    highlightActiveIndentation: false,
  },
  lineHeight: LINE_HEIGHT,
  lineNumbers: 'on',
  lineNumbersMinChars: 4,
  minimap: {
    enabled: false,
  },
  overviewRulerLanes: 0,
  padding: {
    top: 10,
    bottom: 10,
  },
  readOnly: false,
  renderLineHighlight: 'none',
  renderWhitespace: 'none',
  scrollBeyondLastLine: false,
  scrollbar: {
    alwaysConsumeMouseWheel: false,
    horizontal: 'visible',
    vertical: 'visible',
    verticalScrollbarSize: 10,
  },
  quickSuggestions: {
    other: true,
    comments: false,
    strings: true,
  },
  suggest: {
    showFunctions: true,
    showKeywords: true,
    showSnippets: true,
    showWords: false,
  },
  suggestOnTriggerCharacters: true,
  acceptSuggestionOnEnter: 'on',
  wordWrap: 'off',
  wrappingStrategy: 'advanced',
};
