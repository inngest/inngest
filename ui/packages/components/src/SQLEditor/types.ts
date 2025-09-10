export interface SQLCompletionConfig {
  columns: readonly string[];
  keywords: readonly string[];
  functions: readonly { name: string; signature: string }[];
  tables: readonly string[];
}
