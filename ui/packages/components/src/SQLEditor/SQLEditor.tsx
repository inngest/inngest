'use client';

import { useCallback, useRef } from 'react';
import { Editor } from '@monaco-editor/react';

import { EDITOR_OPTIONS } from './constants';
import { useMonacoWithTheme } from './hooks/useMonacoWithTheme';

export type SQLEditorProps = {
  content: string;
  onChange: (value: string) => void;
};

// TODO: Remove this component and use the NewCodeBlock component when it's ready.
export function SQLEditor({ content, onChange }: SQLEditorProps) {
  const wrapperRef = useRef<HTMLDivElement>(null);

  useMonacoWithTheme(wrapperRef);

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
