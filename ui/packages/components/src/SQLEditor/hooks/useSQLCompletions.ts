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

        // Calculate word range manually to include forward slashes and other special chars
        // This is especially important for event names like 'app/user.created'
        let startColumn = position.column;
        const lineContent = model.getLineContent(position.lineNumber);

        // If we're in a string context (after name = '), allow more characters including /
        if (isAfterNameEquals) {
          // Find the start of the current word within the string
          // Work backwards from cursor position, including /, ., -, _, alphanumeric
          while (startColumn > 1) {
            const char = lineContent[startColumn - 2]; // -2 because columns are 1-indexed
            if (char && /[a-zA-Z0-9_.\-/]/.test(char)) {
              startColumn--;
            } else {
              break;
            }
          }
        } else if (isAfterDataDot) {
          // After data., allow alphanumeric, underscore, and dot
          while (startColumn > 1) {
            const char = lineContent[startColumn - 2];
            if (char && /[a-zA-Z0-9_.]/.test(char)) {
              startColumn--;
            } else {
              break;
            }
          }
        } else {
          // Default: use Monaco's word definition
          const word = model.getWordUntilPosition(position);
          startColumn = word.startColumn;
        }

        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: startColumn,
          endColumn: position.column,
        };

        // Extract the current word for prefix matching
        const currentWord = lineContent.substring(startColumn - 1, position.column - 1);

        const suggestions: languages.CompletionItem[] = [];

        // Context-aware: Event names after "name = '"
        if (isAfterNameEquals) {
          eventNames.forEach((eventName) => {
            if (labelMatchesPrefix(eventName, currentWord)) {
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
            if (labelMatchesPrefix(prop.name, currentWord)) {
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
          if (labelMatchesPrefix(column, currentWord)) {
            suggestions.push({
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: column,
              label: column,
              range,
            });
          }
        });

        functions.forEach((func) => {
          if (labelMatchesPrefix(func.name, currentWord)) {
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
          if (labelMatchesPrefix(keyword, currentWord)) {
            suggestions.push({
              kind: monaco.languages.CompletionItemKind.Keyword,
              insertText: keyword,
              label: keyword,
              range,
            });
          }
        });

        tables.forEach((table) => {
          if (labelMatchesPrefix(table, currentWord)) {
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
