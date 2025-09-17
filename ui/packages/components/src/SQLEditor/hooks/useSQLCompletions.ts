'use client';

import { useEffect } from 'react';
import { useMonaco } from '@monaco-editor/react';
import type { languages } from 'monaco-editor';

import type { SQLCompletionConfig } from '../types';

export function useSQLCompletions(config: SQLCompletionConfig) {
  const monaco = useMonaco();

  useEffect(() => {
    if (!monaco) return;

    const { columns, keywords, functions, tables } = config;

    const disposable = monaco.languages.registerCompletionItemProvider('sql', {
      provideCompletionItems: (model, position) => {
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        const suggestions: languages.CompletionItem[] = [];

        columns.forEach((column) => {
          if (labelMatchesPrefix(column, word.word)) {
            suggestions.push({
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: column,
              label: column,
              range,
            });
          }
        });

        functions.forEach((func) => {
          if (labelMatchesPrefix(func.name, word.word)) {
            suggestions.push({
              kind: monaco.languages.CompletionItemKind.Function,
              insertText: func.signature,
              insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              label: func.name,
              range,
            });
          }
        });

        keywords.forEach((keyword) => {
          if (labelMatchesPrefix(keyword, word.word)) {
            suggestions.push({
              kind: monaco.languages.CompletionItemKind.Keyword,
              insertText: keyword,
              label: keyword,
              range,
            });
          }
        });

        tables.forEach((table) => {
          if (labelMatchesPrefix(table, word.word)) {
            suggestions.push({
              kind: monaco.languages.CompletionItemKind.Class,
              insertText: table,
              label: table,
              range,
            });
          }
        });

        return { suggestions };
      },
    });

    return () => {
      disposable.dispose();
    };
  }, [monaco, config]);
}

function labelMatchesPrefix(label: string, prefix: string): boolean {
  if (prefix === '') return true;
  return label.toLowerCase().startsWith(prefix.toLowerCase());
}
