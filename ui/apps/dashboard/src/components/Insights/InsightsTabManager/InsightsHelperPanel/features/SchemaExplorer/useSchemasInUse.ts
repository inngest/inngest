import { useMemo } from 'react';

import type { SchemaEntry } from './SchemasContext/types';

export function useSchemasInUse(entries: SchemaEntry[]): { schemasInUse: SchemaEntry[] } {
  const schemasInUse = useMemo(() => entries.slice(0, 5), [entries]);
  return { schemasInUse };
}
