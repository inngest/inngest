import { useEffect, useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button/NewButton';
import {
  FONT,
  LINE_HEIGHT,
  celLanguageTokens,
  createColors,
  createRules,
} from '@inngest/components/utils/monaco';
import Editor, { useMonaco, type Monaco } from '@monaco-editor/react';
//
//TANSTACK TODO: these cause errors in tanstack start (though this component still works)
import { languages, type editor } from 'monaco-editor';

import { isDark } from '../utils/theme';

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

const MAX_HEIGHT = 10 * LINE_HEIGHT;

const EVENT_PATHS = [
  'event.data.',
  'event.id',
  'event.name',
  'event.ts',
  'event.v',
  'output',
  'output.',
] as const;

export const EVENT_PATH_PRESETS = {
  runs: [
    'event.data.',
    'event.id',
    'event.name',
    'event.ts',
    'event.v',
    'output',
    'output.',
  ] as const,
  events: ['event.data.', 'event.id', 'event.name', 'event.ts', 'event.v'] as const,
};

type EventPath = (typeof EVENT_PATHS)[number];

const EVENT_PATH_DETAILS: Record<
  EventPath,
  { kind: languages.CompletionItemKind; detail: string }
> = {
  'event.data.': {
    kind: languages.CompletionItemKind.Struct,
    detail: 'Event Data Fields',
  },
  'event.id': {
    kind: languages.CompletionItemKind.Field,
    detail: 'Event Identifier (string)',
  },
  'event.name': {
    kind: languages.CompletionItemKind.Field,
    detail: 'Event Name (string)',
  },
  'event.ts': {
    kind: languages.CompletionItemKind.Field,
    detail: 'Event Timestamp (int64)',
  },
  'event.v': {
    kind: languages.CompletionItemKind.Field,
    detail: 'Event Version (string)',
  },
  output: {
    kind: languages.CompletionItemKind.Variable,
    detail: 'Output Variable',
  },
  'output.': {
    kind: languages.CompletionItemKind.Struct,
    detail: 'Output Fields',
  },
};

const NUMERIC_OPERATORS = ['==', '!=', '>', '>=', '<', '<='];
const STRING_OPERATORS = ['==', '!='];

function getOperatorsForPath(path: string): string[] {
  if (
    path === 'event.ts' ||
    path.startsWith('event.data.') ||
    path === 'output' ||
    path.startsWith('output.')
  ) {
    return NUMERIC_OPERATORS;
  }
  return STRING_OPERATORS;
}

function isOperator(str: string): boolean {
  return [...NUMERIC_OPERATORS, ...STRING_OPERATORS].includes(str);
}

export default function CodeSearch({
  onSearch,
  placeholder,
  value,
  searchError,
  preset,
}: {
  onSearch: (content: string) => void;
  placeholder?: string;
  value?: string;
  searchError?: Error;
  preset: keyof typeof EVENT_PATH_PRESETS;
}) {
  const eventPaths = EVENT_PATH_PRESETS[preset];
  const [content, setContent] = useState<string>(value || '');
  const [dark, setDark] = useState(isDark());
  const editorRef = useRef<MonacoEditorType>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const monacoRef = useRef<Monaco>();

  const monaco = useMonaco();

  useEffect(() => {
    // We don't have a DOM ref until we're rendered, so check for dark theme parent classes then
    if (wrapperRef.current) {
      setDark(isDark(wrapperRef.current));
    }
  });

  useEffect(() => {
    if (!monaco || monacoRef.current) {
      return;
    }
    monacoRef.current = monaco;

    monaco.languages.register({ id: 'cel' });

    monaco.languages.setMonarchTokensProvider('cel', celLanguageTokens);

    monaco.editor.defineTheme('inngest-theme', {
      base: dark ? 'vs-dark' : 'vs',
      inherit: true,
      rules: dark ? createRules(true) : createRules(false),
      colors: dark ? createColors(true) : createColors(false),
    });

    const disposable = monaco.languages.registerCompletionItemProvider('cel', {
      triggerCharacters: ['.', ' '],
      provideCompletionItems: (model, position) => {
        const lineContent = model.getLineContent(position.lineNumber);
        const wordAtPosition = model.getWordUntilPosition(position);

        // Get the text from start of line to current position
        const textUntilPosition = lineContent.substring(0, position.column - 1);

        // Find the start of the current path being typed
        const pathMatch = textUntilPosition.match(/([a-zA-Z_][a-zA-Z0-9_.]*\.?)$/);
        const currentPath = pathMatch ? pathMatch[1] : wordAtPosition.word;
        const pathStartColumn =
          pathMatch && currentPath
            ? position.column - currentPath.length
            : position.column - wordAtPosition.word.length;

        // Check if we just typed a space
        const justTypedSpace = lineContent[position.column - 2] === ' ';

        // Split by space but keep empty parts to accurately track word count
        const parts = textUntilPosition.split(' ').filter((p) => p !== '');

        // Check if we already have an operator
        const hasOperator = parts.some((part) => isOperator(part));

        // If we already have an operator, no more suggestions
        if (hasOperator) {
          return { suggestions: [] };
        }

        // If we're at the start or just starting a new word
        if (parts.length === 0 || (parts.length === 1 && !justTypedSpace)) {
          // Provide path suggestions
          return {
            suggestions: eventPaths
              .filter((path) => path.startsWith(currentPath || ''))
              .map((path) => ({
                label: path,
                kind: EVENT_PATH_DETAILS[path]?.kind || monaco.languages.CompletionItemKind.Field,
                detail: EVENT_PATH_DETAILS[path]?.detail || 'Field',
                insertText: path,
                range: {
                  startLineNumber: position.lineNumber,
                  startColumn: pathStartColumn,
                  endLineNumber: position.lineNumber,
                  endColumn: position.column,
                },
              })),
          };
        }

        // If we just typed a space after a valid path, suggest operators
        if (justTypedSpace && parts.length > 0) {
          const leftSide = parts[0] || '';
          if (
            eventPaths.some((path) => path === leftSide) ||
            eventPaths.some((path) => path.endsWith('.') && leftSide.startsWith(path))
          ) {
            const operators = getOperatorsForPath(leftSide);
            return {
              suggestions: operators.map((op) => ({
                label: op,
                kind: monaco.languages.CompletionItemKind.Operator,
                insertText: op,
                range: {
                  startLineNumber: position.lineNumber,
                  startColumn: position.column,
                  endLineNumber: position.lineNumber,
                  endColumn: position.column,
                },
              })),
            };
          }
        }

        return { suggestions: [] };
      },
    });

    // Clean up function
    return () => {
      disposable.dispose();
      monacoRef.current = undefined;
    };
  }, [monaco, dark]);

  const handleEditorDidMount = (editor: MonacoEditorType) => {
    editorRef.current = editor;
    updateEditorHeight();
  };

  const updateEditorHeight = () => {
    const editor = editorRef.current;
    if (editor) {
      const contentHeight = Math.min(MAX_HEIGHT, editor.getContentHeight());
      wrapperRef.current!.style.height = `${contentHeight}px`;
      editor.layout();
    }
  };

  const handleReset = () => {
    if (editorRef.current) {
      editorRef.current.setValue('');
      setContent('');
      onSearch('');
    }
  };

  const handleSearch = (editorContent?: string) => {
    const updatedContent = editorContent || content;

    // Remove empty lines and trim whitespace
    const processedContent = updatedContent
      .split('\n')
      .filter((line) => line.trim() !== '')
      .join('\n')
      .trim();

    onSearch(processedContent);
  };

  const handleContentChange = (value: string | undefined) => {
    const newContent = value || '';
    setContent(newContent);
    updateEditorHeight();
  };

  return (
    <>
      {searchError && (
        <Alert severity="error" className="flex items-center justify-between text-sm">
          {searchError.message}
        </Alert>
      )}
      {monaco && (
        <div ref={wrapperRef} className="relative">
          {!content && (
            <div
              className="text-disabled pointer-events-none absolute left-11 top-0 z-[1] flex h-full w-full items-center pl-3"
              style={{
                fontFamily: FONT.font,
                fontSize: FONT.size,
                lineHeight: `${LINE_HEIGHT}px`,
              }}
            >
              {placeholder}
            </div>
          )}
          <Editor
            defaultLanguage="cel"
            value={content}
            theme="inngest-theme"
            onMount={(editor) => {
              editor.addAction({
                id: 'searchRun',
                label: 'Search Run',
                keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
                run: () => {
                  const latestContent = editor.getValue();
                  handleSearch(latestContent);
                },
              });
              handleEditorDidMount(editor);
            }}
            onChange={handleContentChange}
            options={{
              lineNumbersMinChars: 4,
              readOnly: false,
              minimap: {
                enabled: false,
              },
              lineNumbers: 'on',
              contextmenu: false,
              scrollBeyondLastLine: false,
              fontFamily: FONT.font,
              fontSize: FONT.size,
              fontWeight: 'light',
              lineHeight: LINE_HEIGHT,
              renderLineHighlight: 'none',
              renderWhitespace: 'none',
              guides: {
                indentation: false,
                highlightActiveBracketPair: false,
                highlightActiveIndentation: false,
              },
              scrollbar: {
                verticalScrollbarSize: 10,
                alwaysConsumeMouseWheel: false,
                vertical: 'visible',
                horizontal: 'hidden',
              },
              padding: {
                top: 10,
                bottom: 10,
              },
              wordWrap: 'off',
              wrappingStrategy: 'advanced',
              overviewRulerLanes: 0,
              suggest: {
                showWords: false,
              },
            }}
          />
        </div>
      )}
      <div className="bg-codeEditor flex items-center gap-4 py-4 pl-4">
        <Button
          onClick={() => handleSearch()}
          label="Search"
          size="small"
          data-sentry-component="code-search-search-button"
        />
        <Button
          onClick={handleReset}
          appearance="ghost"
          size="small"
          kind="secondary"
          label="Reset"
        />
      </div>
    </>
  );
}
