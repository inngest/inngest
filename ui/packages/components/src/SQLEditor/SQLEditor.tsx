'use client';

import { useCallback, useRef } from 'react';
import { Editor } from '@monaco-editor/react';

import { EDITOR_OPTIONS } from './constants';
import type { SQLCompletionConfig } from './createSQLCompletionProvider';
import { useMonacoWithTheme, useSQLCompletions } from './hooks';

export type SQLEditorProps = {
  completionConfig: SQLCompletionConfig;
  content: string;
  onChange: (value: string) => void;
};

export function SQLEditor({ completionConfig, content, onChange }: SQLEditorProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);

  useMonacoWithTheme(wrapperRef);
  useSQLCompletions(completionConfig);

  const handleContentChange = useCallback(
    (newValue: string | undefined) => {
      onChange(newValue ?? '');
    },
    [onChange]
  );

  return (
    <div ref={wrapperRef} className="bg-codeEditor relative h-full">
      <Editor
        defaultLanguage="sql"
        onChange={handleContentChange}
        options={EDITOR_OPTIONS}
        theme="inngest-theme"
        value={content}
      />
    </div>
  );
}
