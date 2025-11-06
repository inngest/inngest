import type { SchemaNode } from '@inngest/components/SchemaViewer/types';

export type UseSchemasArgs = {
  search: string;
};

export type SchemaEntry = {
  displayName: string;
  isShared: boolean;
  key: string;
  node: SchemaNode;
};

export type UseSchemasReturn = {
  entries: SchemaEntry[];
  error: string | null;
  fetchNextPage: () => void;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isLoading: boolean;
};

export type SchemasContextValue = UseSchemasReturn & {
  setSearch: (value: string) => void;
};
