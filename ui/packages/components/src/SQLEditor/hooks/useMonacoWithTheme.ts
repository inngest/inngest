'use client';

import { useEffect, useState } from 'react';
import { useMonaco } from '@monaco-editor/react';

import { createColors, createRules } from '../../utils/monaco';
import { isDark } from '../../utils/theme';

// TODO: Remove this hook and use the NewCodeBlock component when it's ready.
export function useMonacoWithTheme(wrapperRef: React.RefObject<HTMLDivElement>) {
  const [dark, setDark] = useState(isDark());
  const monaco = useMonaco();

  useEffect(() => {
    // We don't have a DOM ref until we're rendered, so check for dark theme parent classes then
    if (wrapperRef.current) {
      setDark(isDark(wrapperRef.current));
    }
  });

  useEffect(() => {
    if (!monaco) return;

    // Always update the theme when monaco or dark mode changes
    monaco.editor.defineTheme('inngest-theme', {
      base: dark ? 'vs-dark' : 'vs',
      inherit: true,
      rules: createRules(dark),
      colors: createColors(dark),
    });

    // Set the theme immediately
    monaco.editor.setTheme('inngest-theme');
  }, [monaco, dark]);
}
