'use client';

import { useEffect } from 'react';
import { useMonaco } from '@monaco-editor/react';
import type { languages } from 'monaco-editor';
import { format } from 'sql-formatter';

export function useSQLFormatter() {
  const monaco = useMonaco();

  useEffect(() => {
    if (!monaco) return;

    // Register SQL document formatting provider
    const disposable = monaco.languages.registerDocumentFormattingEditProvider('sql', {
      provideDocumentFormattingEdits(model): languages.ProviderResult<languages.TextEdit[]> {
        const text = model.getValue();

        try {
          const formatted = format(text, {
            language: 'sql',
            tabWidth: 2,
            keywordCase: 'upper',
            linesBetweenQueries: 2,
          });

          return [
            {
              range: model.getFullModelRange(),
              text: formatted,
            },
          ];
        } catch (error) {
          // If formatting fails, return no edits
          console.error('SQL formatting error:', error);
          return [];
        }
      },
    });

    // Cleanup on unmount
    return () => {
      disposable.dispose();
    };
  }, [monaco]);
}
