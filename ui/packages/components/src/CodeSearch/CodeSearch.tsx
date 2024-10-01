'use client';

import { useEffect, useRef, useState } from 'react';
import { NewButton as Button } from '@inngest/components/Button';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import Editor, { useMonaco } from '@monaco-editor/react';
import { type editor } from 'monaco-editor';

import { isDark } from '../utils/theme';

type MonacoEditorType = editor.IStandaloneCodeEditor | null;

const MAX_HEIGHT = 10 * LINE_HEIGHT;

export default function CodeSearch({ onSearch }: { onSearch: (content: string) => void }) {
  const [content, setContent] = useState<string>('\n');
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
      editorRef.current.setValue('\n');
      setContent('\n');
    }
  };

  const handleSearch = () => {
    const trimmedContent = content.trim();
    if (trimmedContent && trimmedContent !== '\n') {
      onSearch(trimmedContent);
    }
  };

  return (
    <>
      {monaco && (
        <div ref={wrapperRef}>
          <Editor
            defaultLanguage="javascript"
            value={content}
            theme="inngest-theme"
            onMount={handleEditorDidMount}
            onChange={(value) => {
              setContent(value || '');
              updateEditorHeight();
            }}
            options={{
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
      <div className="bg-codeEditor flex items-center pb-4 pl-7">
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
