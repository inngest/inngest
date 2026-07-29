import { useEffect } from 'react';
import { useMonaco } from '@monaco-editor/react';
import type { languages } from 'monaco-editor';
import { clickhouse, formatDialect } from 'sql-formatter';

export function useSQLFormatter() {
  const monaco = useMonaco();

  useEffect(() => {
    if (!monaco) return;

    // Register SQL document formatting provider
    const disposable = monaco.languages.registerDocumentFormattingEditProvider('sql', {
      provideDocumentFormattingEdits(model): languages.ProviderResult<languages.TextEdit[]> {
        const text = model.getValue();

        try {
          const formatted = formatDialect(text, {
            dialect: clickhouse,
            tabWidth: 2,
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
