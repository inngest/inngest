import type { Cache } from './hooks/useCache';

// A column entry can be a bare name (existing callers) or a name with an
// optional description shown in the suggestion's documentation panel.
export type SQLCompletionColumn = string | { name: string; description?: string };

export interface SQLCompletionConfig {
  columns: readonly SQLCompletionColumn[];
  keywords: readonly string[];
  functions: readonly { name: string; signature: string; description?: string }[];
  tables: readonly string[];
  eventNames?: readonly string[];
  dataProperties?: readonly { name: string; type: string }[];
  fetchEventNames?: (search: string) => Promise<string[]>;
  fetchEventSchema?: (eventName: string) => Promise<Array<{ name: string; type: string }>>;
  eventNamesCache?: Cache<string[]>;
  schemasCache?: Cache<Array<{ name: string; type: string }>>;
}
