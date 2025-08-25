'use client';

import { useEffect } from 'react';
import { useMonaco } from '@monaco-editor/react';

import {
  createSQLCompletionProvider,
  type SQLCompletionConfig,
} from '../createSQLCompletionProvider';

export function useSQLCompletions(config: SQLCompletionConfig) {
  const monaco = useMonaco();

  useEffect(() => {
    if (!monaco) return;

    const disposable = monaco.languages.registerCompletionItemProvider(
      'sql',
      createSQLCompletionProvider(config)
    );

    return () => {
      disposable.dispose();
    };
  }, [monaco, config]);
}
