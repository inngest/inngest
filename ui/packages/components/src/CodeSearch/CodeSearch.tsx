'use client';

import { useEffect, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  FONT,
  LINE_HEIGHT,
  celLanguageTokens,
  createColors,
  createRules,
} from '@inngest/components/utils/monaco';
import Editor, { useMonaco, type Monaco } from '@monaco-editor/react';
import { languages, type editor } from 'monaco-editor';

import { isDark } from '../utils/theme';

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

const MAX_HEIGHT = 10 * LINE_HEIGHT;
const VALIDATION_DELAY = 500;

const EVENT_PATHS = [
  'event.data.',
  'event.id',
  'event.name',
  'event.ts',
  'event.v',
  'output',
  'output.',
] as const;

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

interface ValidationError {
  message: string;
  startColumn: number;
  endColumn: number;
  lineNumber: number;
}

function validateExpression(content: string): ValidationError | null {
  const lines = content.split('\n');

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const currentLine = lines[lineIndex] ?? '';
    const line = currentLine.trim();

    if (!line) continue;

    const parts = line.split(' ').filter((p) => p !== '');

    // Check if the first word is a valid path
    const firstWord = parts[0];
    if (!firstWord) continue;

    const startColumn = currentLine.indexOf(firstWord) + 1;

    if (
      !EVENT_PATHS.includes(firstWord as EventPath) &&
      !EVENT_PATHS.some((path) => path.endsWith('.') && firstWord.startsWith(path))
    ) {
      return {
        message: `Invalid field: ${firstWord}. Search by event or output.`,
        startColumn,
        endColumn: startColumn + firstWord.length,
        lineNumber: lineIndex + 1,
      };
    }

    // Not enough parts to validate operator and value
    if (parts.length < 3) continue;

    const operator = parts[1];
    if (!operator) continue;

    const value = parts.slice(2).join(' ');
    const valueStartInLine = currentLine.indexOf(value);
    const valueStartColumn = valueStartInLine + 1;

    // Validate operator
    const validOperators = getOperatorsForPath(firstWord);
    if (!validOperators.includes(operator)) {
      return {
        message: `Invalid operator for ${firstWord}: ${operator}. Valid operators are: ${validOperators.join(
          ', '
        )}`,
        startColumn: currentLine.indexOf(operator) + 1,
        endColumn: currentLine.indexOf(operator) + operator.length + 1,
        lineNumber: lineIndex + 1,
      };
    }

    // Validate value type
    if (firstWord === 'event.id' || firstWord === 'event.name' || firstWord === 'event.v') {
      const isValidString =
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"));

      if (!isValidString) {
        return {
          message: `${firstWord} must be a string`,
          startColumn: valueStartColumn,
          endColumn: valueStartColumn + value.length,
          lineNumber: lineIndex + 1,
        };
      }
    } else if (firstWord === 'event.ts') {
      if (!/^\d+$/.test(value)) {
        return {
          message: `${firstWord} must be an integer`,
          startColumn: valueStartColumn,
          endColumn: valueStartColumn + value.length,
          lineNumber: lineIndex + 1,
        };
      }
    }
  }
  return null;
}

export default function CodeSearch({
  onSearch,
  placeholder,
  value,
}: {
  onSearch: (content: string) => void;
  placeholder?: string;
  value?: string;
}) {
  const [content, setContent] = useState<string>(value || '');
  const [dark, setDark] = useState(isDark());
  const editorRef = useRef<MonacoEditorType>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const monacoRef = useRef<Monaco>();
  const [hasValidationError, setHasValidationError] = useState(false);
  const validationTimerRef = useRef<NodeJS.Timeout>();

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

        // Check if we just typed a space
        const justTypedSpace = lineContent[position.column - 2] === ' ';

        // Get the text before the current position
        const textUntilPosition = lineContent.substring(0, position.column - 1);

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
            suggestions: EVENT_PATHS.filter((path) => path.startsWith(wordAtPosition.word)).map(
              (path) => ({
                label: path,
                kind: EVENT_PATH_DETAILS[path]?.kind || monaco.languages.CompletionItemKind.Field,
                detail: EVENT_PATH_DETAILS[path]?.detail || 'Field',
                insertText: path,
                range: {
                  startLineNumber: position.lineNumber,
                  startColumn: position.column - wordAtPosition.word.length,
                  endLineNumber: position.lineNumber,
                  endColumn: position.column,
                },
              })
            ),
          };
        }

        // If we just typed a space after a valid path, suggest operators
        if (justTypedSpace && parts.length > 0) {
          const leftSide = parts[0] || '';
          if (
            EVENT_PATHS.includes(leftSide as EventPath) ||
            EVENT_PATHS.some((path) => path.endsWith('.') && leftSide.startsWith(path))
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

  const updateMarkers = (error: ValidationError | null) => {
    if (!editorRef.current || !monacoRef.current) return;

    const model = editorRef.current.getModel();
    if (!model) return;

    if (error) {
      const marker: editor.IMarkerData = {
        severity: monacoRef.current.MarkerSeverity.Error,
        message: error.message,
        startLineNumber: error.lineNumber,
        startColumn: error.startColumn,
        endLineNumber: error.lineNumber,
        endColumn: error.endColumn,
      };
      monacoRef.current.editor.setModelMarkers(model, 'owner', [marker]);
      setHasValidationError(true);
    } else {
      monacoRef.current.editor.setModelMarkers(model, 'owner', []);
      setHasValidationError(false);
    }
  };

  const handleReset = () => {
    if (editorRef.current) {
      editorRef.current.setValue('');
      setContent('');
      updateMarkers(null);
      onSearch('');
    }
  };

  const handleSearch = (editorContent?: string) => {
    const updatedContent = editorContent || content;
    if (!hasValidationError) {
      // Remove empty lines and trim whitespace
      const processedContent = updatedContent
        .split('\n')
        .filter((line) => line.trim() !== '')
        .join('\n')
        .trim();

      onSearch(processedContent);
    }
  };

  const handleContentChange = (value: string | undefined) => {
    const newContent = value || '';
    setContent(newContent);
    updateEditorHeight();

    // Clear existing timer
    if (validationTimerRef.current) {
      clearTimeout(validationTimerRef.current);
    }

    // Set new timer for validation
    validationTimerRef.current = setTimeout(() => {
      const error = validateExpression(newContent);
      updateMarkers(error);
    }, VALIDATION_DELAY);
  };

  return (
    <>
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
        <Button onClick={() => handleSearch()} label="Search" size="small" />
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
