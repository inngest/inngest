import { useEffect, useMemo, useState } from 'react';
import useDebounce from '@inngest/components/hooks/useDebounce';

import { useInsightsStateMachineContext } from '../../../../InsightsStateMachineContext/InsightsStateMachineContext';
import type { SchemaEntry } from './SchemasContext/types';

export function useSchemasInUse(entries: SchemaEntry[]): { schemasInUse: SchemaEntry[] } {
  const { possibleEventNames } = useDetectPossibleEvents();

  const schemasInUse = useMemo(
    () => entries.filter((entry) => possibleEventNames.includes(entry.node.name)),
    [entries, possibleEventNames]
  );
  return { schemasInUse };
}

// Matches occurrences of name = '<event_name>' (single quotes only),
// allowing optional whitespace around the equals sign, and captures the event name.
const POSSIBLE_EVENT_NAME_REGEX = /name\s*=\s*'([^']+)'/g;
function useDetectPossibleEvents(): { possibleEventNames: string[] } {
  const { query } = useInsightsStateMachineContext();

  const [debouncedText, setDebouncedText] = useState(query);
  const debouncedUpdate = useDebounce(() => setDebouncedText(query), 1000);
  useEffect(() => {
    debouncedUpdate();
  }, [debouncedUpdate, query]);

  const possibleEventNames = useMemo(() => {
    const results = new Set<string>();
    for (const match of debouncedText.matchAll(POSSIBLE_EVENT_NAME_REGEX)) {
      const name = match[1]?.trim();
      if (name) results.add(name);
    }

    return Array.from(results).sort((a, b) => a.localeCompare(b));
  }, [debouncedText]);

  return { possibleEventNames };
}
