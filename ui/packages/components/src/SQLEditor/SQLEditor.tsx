'use client';

import { useCallback, useRef } from 'react';
import { Editor, type Monaco } from '@monaco-editor/react';
import type { editor } from 'monaco-editor';

import { EDITOR_OPTIONS } from './constants';
import { useMonacoWithTheme } from './hooks/useMonacoWithTheme';
import { useSQLCompletions } from './hooks/useSQLCompletions';
import type { SQLCompletionConfig } from './types';

export type SQLEditorMountCallback = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => void;

export type SQLEditorProps = {
  completionConfig: SQLCompletionConfig;
  content: string;
  onChange: (value: string) => void;
  onMount?: SQLEditorMountCallback;
};

export function SQLEditor({ completionConfig, content, onChange, onMount }: SQLEditorProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);

  useMonacoWithTheme(wrapperRef);
  useSQLCompletions(completionConfig);

  const handleContentChange = useCallback(
    (newValue: string | undefined) => {
      onChange(newValue ?? '');
    },
    [onChange]
  );

  const handleEditorMount = useCallback(
    (editorInstance: editor.IStandaloneCodeEditor, monaco: Monaco) => {
      onMount?.(editorInstance, monaco);
    },
    [onMount]
  );

  return (
    <div ref={wrapperRef} className="bg-codeEditor relative h-full">
      <Editor
        defaultLanguage="sql"
        onChange={handleContentChange}
        onMount={handleEditorMount}
        options={EDITOR_OPTIONS}
        theme="inngest-theme"
        value={content}
      />
    </div>
  );
}
