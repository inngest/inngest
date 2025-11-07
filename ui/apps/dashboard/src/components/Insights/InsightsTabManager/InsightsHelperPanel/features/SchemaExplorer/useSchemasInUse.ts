import { useEffect, useMemo, useState } from 'react';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { useQueries, type UseQueryResult } from '@tanstack/react-query';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useInsightsStateMachineContext } from '../../../../InsightsStateMachineContext/InsightsStateMachineContext';
import { buildEntryFromLatestSchema } from './SchemasContext/queries';
import type { SchemaEntry, SchemaEvent } from './SchemasContext/types';
import { useEventTypeSchemas } from './SchemasContext/useEventTypeSchemas';
import { makeTitleOnlyEntry } from './SchemasContext/utils';

export function useSchemasInUse(): { schemasInUse: SchemaEntry[] } {
  const { possibleEventNames } = useDetectPossibleEvents();
  const env = useEnvironment();
  const getEventTypeSchemas = useEventTypeSchemas();

  const results = useQueries({
    queries: possibleEventNames.map((name) => ({
      queryKey: ['schema-explorer-event-type-in-use', env.id, { name }],
      queryFn: () => getEventTypeSchemas({ cursor: null, nameSearch: name }),
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    })),
  });

  const schemasInUse = useMemo<SchemaEntry[]>(() => {
    const eventNameToSchema = makeMapOfEventNameToSchema(results);
    return possibleEventNames.flatMap((name) => {
      const schema = eventNameToSchema[name];
      if (schema === undefined) return [];

      const built = buildEntryFromLatestSchema(schema, name);
      if (built === null) return [makeTitleOnlyEntry(name)];

      return [built];
    });
  }, [possibleEventNames, results]);

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

    // Limit to 5 names (likely more than needed) to limit queries.
    return Array.from(results)
      .sort((a, b) => a.localeCompare(b))
      .slice(0, 5);
  }, [debouncedText]);

  return { possibleEventNames };
}

function makeMapOfEventNameToSchema(
  results: Array<UseQueryResult<{ events: SchemaEvent[] }>>
): Record<string, string> {
  return results.reduce<Record<string, string>>((acc, r) => {
    const events = r.data?.events ?? [];
    for (const evt of events) {
      acc[evt.name] = evt.schema;
    }
    return acc;
  }, {});
}
