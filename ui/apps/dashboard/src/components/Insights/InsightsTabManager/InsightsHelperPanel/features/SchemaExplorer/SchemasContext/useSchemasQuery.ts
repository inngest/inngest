'use client';

import { useMemo } from 'react';

import { buildSchemaEntriesFromQueryData } from './queries';
import type { SchemaEntry } from './types';

export function useSchemasQuery() {
  // TODO: Fetch schemas from the API.

  const entries = useMemo<SchemaEntry[]>(() => buildSchemaEntriesFromQueryData(), []);

  return {
    entries,
    error: null,
    fetchNextPage: () => {},
    hasNextPage: false,
    isFetchingNextPage: false,
    isLoading: false,
  };
}
