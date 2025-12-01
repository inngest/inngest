import type { Cache } from './hooks/useCache';

export interface SQLCompletionConfig {
  columns: readonly string[];
  keywords: readonly string[];
  functions: readonly { name: string; signature: string }[];
  tables: readonly string[];
  eventNames?: readonly string[];
  dataProperties?: readonly { name: string; type: string }[];
  fetchEventNames?: (search: string) => Promise<string[]>;
  fetchEventSchema?: (eventName: string) => Promise<Array<{ name: string; type: string }>>;
  eventNamesCache?: Cache<string[]>;
  schemasCache?: Cache<Array<{ name: string; type: string }>>;
}
