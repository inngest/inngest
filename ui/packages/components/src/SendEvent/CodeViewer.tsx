import { useEffect, useRef, useState } from 'react';
import { FONT, LINE_HEIGHT, createColors, createRules } from '@inngest/components/utils/monaco';
import { isDark } from '@inngest/components/utils/theme';
import Editor, { useMonaco } from '@monaco-editor/react';

interface CodeViewerProps {
  code: string;
  language: string;
  onChange?: (value: string | undefined) => void;
}

export function CodeViewer({ code, language, onChange }: CodeViewerProps) {
  const [dark, setDark] = useState(isDark());
  const wrapperRef = useRef<HTMLDivElement>(null);
  const monaco = useMonaco();

  useEffect(() => {
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

  return (
    <div
      className="border-subtle relative flex h-[20rem] w-full flex-col overflow-hidden rounded-md border"
      ref={wrapperRef}
    >
      <div className="border-subtle flex items-center justify-between border-b">
        <p className="text-subtle px-5 py-2.5 text-sm">Code</p>
      </div>
      {monaco ? (
        <Editor
          defaultLanguage={language}
          value={code}
          onChange={onChange}
          theme="inngest-theme"
          options={{
            readOnly: !onChange,
            minimap: {
              enabled: false,
            },
            lineNumbers: 'on',
            contextmenu: false,
            scrollBeyondLastLine: false,
            wordWrap: 'on',
            fontFamily: FONT.font,
            fontSize: FONT.size,
            fontWeight: 'light',
            lineHeight: LINE_HEIGHT,
            padding: {
              top: 10,
              bottom: 10,
            },
          }}
        />
      ) : null}
    </div>
  );
}
