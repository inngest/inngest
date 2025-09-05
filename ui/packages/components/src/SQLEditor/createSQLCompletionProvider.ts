import { languages } from 'monaco-editor';

export interface SQLCompletionConfig {
  columns: readonly string[];
  keywords: readonly string[];
  functions: readonly { name: string; signature: string }[];
  tables: readonly string[];
}

export function createSQLCompletionProvider(
  config: SQLCompletionConfig
): languages.CompletionItemProvider {
  const { columns, keywords, functions, tables } = config;

  return {
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
        suggestions.push({
          kind: languages.CompletionItemKind.Field,
          insertText: column,
          label: column,
          range,
        });
      });

      functions.forEach((func) => {
        suggestions.push({
          kind: languages.CompletionItemKind.Function,
          insertText: func.signature,
          insertTextRules: languages.CompletionItemInsertTextRule.InsertAsSnippet,
          label: func.name,
          range,
        });
      });

      keywords.forEach((keyword) => {
        suggestions.push({
          kind: languages.CompletionItemKind.Keyword,
          insertText: keyword,
          label: keyword,
          range,
        });
      });

      tables.forEach((table) => {
        suggestions.push({
          kind: languages.CompletionItemKind.Reference,
          insertText: table,
          label: table,
          range,
        });
      });

      return { suggestions };
    },
  };
}
