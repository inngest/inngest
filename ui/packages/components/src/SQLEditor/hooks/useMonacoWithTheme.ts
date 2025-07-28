'use client';

import { useEffect, useState } from 'react';
import { useMonaco } from '@monaco-editor/react';

import { createColors, createRules } from '../../utils/monaco';
import { isDark } from '../../utils/theme';

export function useMonacoWithTheme(wrapperRef: React.RefObject<HTMLDivElement>) {
  const [dark, setDark] = useState(isDark());
  const monaco = useMonaco();

  useEffect(() => {
    const updateDarkMode = () => {
      setDark(isDark(wrapperRef.current ?? undefined));
    };

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', updateDarkMode);

    const observer = new MutationObserver(updateDarkMode);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme'],
    });

    // Initial check
    updateDarkMode();

    return () => {
      mediaQuery.removeEventListener('change', updateDarkMode);
      observer.disconnect();
    };
  }, [wrapperRef]);

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
