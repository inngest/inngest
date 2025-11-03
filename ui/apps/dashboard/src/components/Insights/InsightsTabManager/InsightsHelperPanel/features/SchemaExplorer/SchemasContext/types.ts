import type { SchemaNode } from '@inngest/components/SchemaViewer/types';
import type { PageInfo } from '@inngest/components/types/eventType';

export type UseSchemasArgs = {
  search: string;
};

export type SchemaEntry = {
  key: string;
  displayName: string;
  isShared: boolean;
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

// Underlying event types used by the schemas query
export type SchemaEvent = {
  name: string;
  latestSchema: string;
  functions: { id: string; slug: string; name: string }[];
  archived: boolean;
};

export type SchemaEventPage = {
  events: SchemaEvent[];
  pageInfo: PageInfo;
};
