'use client';

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import useDebounce from '@inngest/components/hooks/useDebounce';

import type { SchemasContextValue, UseSchemasArgs, UseSchemasReturn } from './types';
import { useSchemasQuery } from './useSchemasQuery';

const SchemasContext = createContext<SchemasContextValue | undefined>(undefined);

export function SchemasProvider({ children }: { children: ReactNode }) {
  const [search, setSearch] = useState('');
  const query = useSchemasQuery(search);

  return (
    <SchemasContext.Provider value={{ ...query, setSearch }}>{children}</SchemasContext.Provider>
  );
}

function useSchemasContext(): SchemasContextValue {
  const ctx = useContext(SchemasContext);
  if (!ctx) throw new Error('useSchemas must be used within SchemasProvider');

  return ctx;
}

export function useSchemas({ search }: UseSchemasArgs): UseSchemasReturn {
  const ctx = useSchemasContext();

  const debouncedSetSearch = useDebounce(() => ctx.setSearch(search), 300);
  useEffect(() => {
    debouncedSetSearch();
  }, [search, debouncedSetSearch]);

  return ctx;
}
