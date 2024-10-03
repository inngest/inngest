'use client';

import { useEffect, useRef, useState } from 'react';
import { NewButton as Button } from '@inngest/components/Button';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import Editor, { useMonaco } from '@monaco-editor/react';
import { type editor } from 'monaco-editor';

import { isDark } from '../utils/theme';

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

const MAX_HEIGHT = 10 * LINE_HEIGHT;

export default function CodeSearch({
  onSearch,
  placeholder,
}: {
  onSearch: (content: string) => void;
  placeholder?: string;
}) {
  const [content, setContent] = useState<string>('');
  const [dark, setDark] = useState(isDark());
  const editorRef = useRef<MonacoEditorType>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const monaco = useMonaco();

  useEffect(() => {
    // We don't have a DOM ref until we're rendered, so check for dark theme parent classes then
    if (wrapperRef.current) {
      setDark(isDark(wrapperRef.current));
    }
  });

  useEffect(() => {
    if (!monaco) {
      return;
    }

    monaco.languages.register({ id: 'cel' });

    monaco.languages.setMonarchTokensProvider('cel', {
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
    });

    monaco.editor.defineTheme('inngest-theme', {
      base: dark ? 'vs-dark' : 'vs',
      inherit: true,
      rules: dark ? createRules(true) : createRules(false),
      colors: dark ? createColors(true) : createColors(false),
    });
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

  const handleClear = () => {
    if (editorRef.current) {
      editorRef.current.setValue('');
      setContent('');
    }
  };

  const handleSearch = () => {
    const trimmedContent = content.trim();
    if (trimmedContent && trimmedContent !== '') {
      onSearch(trimmedContent);
    }
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
            onMount={handleEditorDidMount}
            onChange={(value) => {
              setContent(value || '');
              updateEditorHeight();
            }}
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
            }}
          />
        </div>
      )}
      <div className="bg-codeEditor flex items-center gap-4 py-4 pl-4">
        <Button onClick={handleSearch} label="Search" size="small" />
        <Button
          onClick={handleClear}
          appearance="ghost"
          size="small"
          kind="secondary"
          label="Clear"
        />
      </div>
    </>
  );
}
