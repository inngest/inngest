import { useEffect, useRef } from 'react';
import { useMonaco } from '@monaco-editor/react';
import type { languages } from 'monaco-editor';

import type { SQLCompletionConfig } from '../types';

export function useSQLCompletions(config: SQLCompletionConfig) {
  const monaco = useMonaco();
  const pendingRequestsRef = useRef<Map<string, Promise<string[]>>>(new Map());
  const pendingSchemaRequestsRef = useRef<
    Map<string, Promise<Array<{ name: string; type: string }>>>
  >(new Map());
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (!monaco) return;

    const DEBOUNCE_MS = 200;

    const {
      columns,
      keywords,
      functions,
      tables,
      eventNames = [],
      dataProperties = [],
      fetchEventNames,
      fetchEventSchema,
      eventNamesCache,
      schemasCache,
    } = config;

    // Fetch event names with cache and debouncing - checks cache first, then deduplicates concurrent requests
    const fetchWithCache = async (search: string): Promise<string[]> => {
      const cacheKey = search || '__empty__';

      if (!fetchEventNames) {
        throw new Error('fetchEventNames is not defined');
      }

      // step 1: Check cache synchronously first (no debounce for cached results)
      if (eventNamesCache) {
        const cached = eventNamesCache.get(cacheKey);
        if (cached) {
          return cached;
        }
      }

      // step 2: Check if there's already a pending fetch for this exact search
      const existingRequest = pendingRequestsRef.current.get(cacheKey);
      if (existingRequest) {
        return existingRequest;
      }

      // STEP 3: Debounce the backend fetch
      return new Promise<string[]>((resolve, reject) => {
        // Clear any existing debounce timer
        if (debounceTimerRef.current) {
          clearTimeout(debounceTimerRef.current);
        }

        // Set new debounce timer
        debounceTimerRef.current = setTimeout(async () => {
          try {
            // Create the actual backend request
            const request = fetchEventNames(search)
              .then((names) => {
                pendingRequestsRef.current.delete(cacheKey);
                return names;
              })
              .catch((error) => {
                pendingRequestsRef.current.delete(cacheKey);
                throw error;
              });

            pendingRequestsRef.current.set(cacheKey, request);
            const result = await request;
            resolve(result);
          } catch (error) {
            reject(error);
          }
        }, DEBOUNCE_MS);
      });
    };

    // Fetch schema with cache - checks cache first, then deduplicates concurrent requests
    const fetchSchemaWithCache = async (
      eventName: string
    ): Promise<Array<{ name: string; type: string }>> => {
      if (!fetchEventSchema) {
        return [];
      }

      // STEP 1: Check cache synchronously first
      if (schemasCache) {
        const cached = schemasCache.get(eventName);
        if (cached) {
          return cached;
        }
      }

      // STEP 2: Check if there's already a pending fetch for this event
      const existingRequest = pendingSchemaRequestsRef.current.get(eventName);
      if (existingRequest) {
        return existingRequest;
      }

      // STEP 3: Create new request and cache the promise
      const request = fetchEventSchema(eventName)
        .then((props) => {
          pendingSchemaRequestsRef.current.delete(eventName);
          return props;
        })
        .catch(() => {
          pendingSchemaRequestsRef.current.delete(eventName);
          return [];
        });

      pendingSchemaRequestsRef.current.set(eventName, request);
      return request;
    };

    const disposable = monaco.languages.registerCompletionItemProvider('sql', {
      provideCompletionItems: async (model, position) => {
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
          // After data., we need to find where the dot is and replace everything after it
          // This handles cases like "data.a" -> "data.amount" (replacing "a")
          // Find the position of the dot
          const dotIndex = textBeforeCursor.lastIndexOf('.');
          if (dotIndex !== -1) {
            // Set startColumn to right after the dot (dotIndex is 0-based, columns are 1-based)
            // We need to add 2: +1 for the column offset, +1 to skip the dot itself
            startColumn = dotIndex + 2;
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
          // If we have a dynamic fetch function, use it for server-side filtering
          if (fetchEventNames) {
            try {
              const fetchedNames = await fetchWithCache(currentWord);
              fetchedNames.forEach((eventName: string) => {
                suggestions.push({
                  kind: monaco.languages.CompletionItemKind.Value,
                  insertText: eventName,
                  label: eventName,
                  range,
                  detail: 'Event name',
                });
              });
            } catch (error: any) {
              // Fall back to static list on errors
              console.error('Failed to fetch event names:', error);
              eventNames.forEach((eventName: string) => {
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
            }
          } else {
            // Use static list with client-side filtering
            eventNames.forEach((eventName: string) => {
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
          }
          // Return early - only show event names in this context
          return { suggestions };
        }

        // Context-aware: Data properties after "data."
        if (isAfterDataDot) {
          // Extract just the part after "data." for prefix matching
          const afterDataDotMatch = /\bdata\.([a-zA-Z_]*)$/i.exec(textBeforeCursor);
          const prefixAfterDot = afterDataDotMatch?.[1] ?? '';

          // Extract event name from the query to fetch its schema
          // Look for pattern: name = 'event_name'
          const fullText = model.getValue();
          const eventNameMatch = /name\s*=\s*'([^']+)'/i.exec(fullText);

          if (eventNameMatch && eventNameMatch[1]) {
            const eventName = eventNameMatch[1].trim();

            try {
              // Fetch schema for this event name
              const properties = await fetchSchemaWithCache(eventName);

              properties.forEach((prop) => {
                if (labelMatchesPrefix(prop.name, prefixAfterDot)) {
                  suggestions.push({
                    kind: monaco.languages.CompletionItemKind.Property,
                    insertText: prop.name,
                    label: prop.name,
                    range,
                    detail: prop.type,
                  });
                }
              });
            } catch (error) {
              // Fall back to static dataProperties if any
              dataProperties.forEach((prop) => {
                if (labelMatchesPrefix(prop.name, prefixAfterDot)) {
                  suggestions.push({
                    kind: monaco.languages.CompletionItemKind.Property,
                    insertText: prop.name,
                    label: prop.name,
                    range,
                    detail: prop.type,
                  });
                }
              });
            }
          } else {
            // No event name detected, use static dataProperties if any
            dataProperties.forEach((prop) => {
              if (labelMatchesPrefix(prop.name, prefixAfterDot)) {
                suggestions.push({
                  kind: monaco.languages.CompletionItemKind.Property,
                  insertText: prop.name,
                  label: prop.name,
                  range,
                  detail: prop.type,
                });
              }
            });
          }

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
      // Clean up any pending debounce timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [monaco, config]);
}

function labelMatchesPrefix(label: string, prefix: string): boolean {
  if (prefix === '') return true;
  return label.toLowerCase().startsWith(prefix.toLowerCase());
}
