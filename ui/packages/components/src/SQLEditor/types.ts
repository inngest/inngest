import type { Cache } from './hooks/useCache';

// A column entry can be a bare name (existing callers) or a name with an
// optional description shown in the suggestion's documentation panel and/or
// a type shown in the suggestion's detail (the same way a data property's
// type is already shown, and a function's signature is shown in place of
// its bare name).
export type SQLCompletionColumn = string | { name: string; description?: string; type?: string };

export interface SQLCompletionConfig {
  columns: readonly SQLCompletionColumn[];
  keywords: readonly string[];
  // `signature` is the Monaco insertion snippet (e.g. "concat_ws($1)") --
  // what gets typed when the suggestion is accepted, not what's shown for
  // it. `detail`, shown in the suggestion's own right-aligned column (the
  // same way a column's type is), is the function's real, human-readable
  // call signature (e.g. "concat_ws(separator, string, ...)"), so despite
  // the similar names the two fields serve different purposes and often
  // hold different text.
  functions: readonly {
    name: string;
    signature: string;
    description?: string;
    detail?: string;
  }[];
  tables: readonly string[];
  eventNames?: readonly string[];
  dataProperties?: readonly { name: string; type: string }[];
  fetchEventNames?: (search: string) => Promise<string[]>;
  fetchEventSchema?: (eventName: string) => Promise<Array<{ name: string; type: string }>>;
  eventNamesCache?: Cache<string[]>;
  schemasCache?: Cache<Array<{ name: string; type: string }>>;
}
