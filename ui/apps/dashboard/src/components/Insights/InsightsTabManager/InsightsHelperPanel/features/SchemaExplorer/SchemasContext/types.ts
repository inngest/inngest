import type { SchemaNode } from '@inngest/components/SchemaViewer/types';
import type { PageInfo } from '@inngest/components/types/eventType';

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
  error: Error | null;
  fetchNextPage: () => void;
  hasFetchedMax: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isLoading: boolean;
};

export type SchemasContextValue = UseSchemasReturn & {
  setSearch: (value: string) => void;
};

export type SchemaEvent = {
  archived: boolean;
  functions: { id: string; slug: string; name: string }[];
  latestSchema: string;
  name: string;
};

export type SchemaEventPage = {
  events: SchemaEvent[];
  pageInfo: PageInfo;
};
