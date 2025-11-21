'use client';

import { useEffect } from 'react';
import { useMonaco } from '@monaco-editor/react';
import type { languages } from 'monaco-editor';

import type { SQLCompletionConfig } from '../types';

export function useSQLCompletions(config: SQLCompletionConfig) {
  const monaco = useMonaco();

  useEffect(() => {
    if (!monaco) return;

    const { columns, keywords, functions, tables, eventNames = [], dataProperties = [] } = config;

    const disposable = monaco.languages.registerCompletionItemProvider('sql', {
      provideCompletionItems: (model, position) => {
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        // Get text before cursor to detect context
        const textBeforeCursor = model.getValueInRange({
          startLineNumber: position.lineNumber,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });

        // Check if we're after "name = '"
        const isAfterNameEquals = /name\s*=\s*'[^']*$/i.test(textBeforeCursor);

        // Check if we're after "data."
        const isAfterDataDot = /\bdata\.[a-zA-Z_]*$/i.test(textBeforeCursor);

        const suggestions: languages.CompletionItem[] = [];

        // Context-aware: Event names after "name = '"
        if (isAfterNameEquals) {
          eventNames.forEach((eventName) => {
            if (labelMatchesPrefix(eventName, word.word)) {
              suggestions.push({
                kind: monaco.languages.CompletionItemKind.Value,
                insertText: eventName,
                label: eventName,
                range,
                detail: 'Event name',
              });
            }
          });
          // Return early - only show event names in this context
          return { suggestions };
        }

        // Context-aware: Data properties after "data."
        if (isAfterDataDot) {
          dataProperties.forEach((prop) => {
            if (labelMatchesPrefix(prop.name, word.word)) {
              suggestions.push({
                kind: monaco.languages.CompletionItemKind.Property,
                insertText: prop.name,
                label: prop.name,
                range,
                detail: prop.type,
              });
            }
          });
          // Return early - only show data properties in this context
          return { suggestions };
        }

        // Default autocomplete (keywords, tables, functions, columns)

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
