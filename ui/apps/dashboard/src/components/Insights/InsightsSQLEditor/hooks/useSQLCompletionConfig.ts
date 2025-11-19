'use client';

import { useEffect, useMemo, useState } from 'react';
import { availableClickhouseFunctions } from '@inngest/components/SQLEditor/hooks/availableClickhouseFunctions';
import type { SQLCompletionConfig } from '@inngest/components/SQLEditor/types';
import { useQuery } from '@tanstack/react-query';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useAllEventTypes } from '@/components/EventTypes/useEventTypes';
import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import { useEventTypeSchemas } from '../../InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/useEventTypeSchemas';

const KEYWORDS = [
  'AND',
  'AS',
  'ASC',
  'BETWEEN',
  'DESC',
  'DISTINCT',
  'FALSE',
  'FROM',
  'GROUP BY',
  'IS',
  'LIKE',
  'LIMIT',
  'NOT',
  'NULL',
  'OFFSET',
  'OR',
  'ORDER BY',
  'SELECT',
  'TRUE',
  'WHERE',
] as const;

const TABLES = ['events'] as const;

// Common columns available on the events table
const COLUMNS = ['name', 'data'] as const;

// Convert ClickHouse functions to the format expected by autocomplete
const CLICKHOUSE_FUNCTIONS = availableClickhouseFunctions.map((name) => ({
  name,
  signature: `${name}($1)`,
}));

// Matches occurrences of name = '<event_name>' (single quotes only)
const POSSIBLE_EVENT_NAME_REGEX = /name\s*=\s*'([^']+)'/gi;

export function useSQLCompletionConfig(): SQLCompletionConfig {
  const env = useEnvironment();
  const { query } = useInsightsStateMachineContext();
  const getAllEventTypes = useAllEventTypes();
  const getEventTypeSchemas = useEventTypeSchemas();

  // Fetch all event names for autocomplete
  const { data: allEventTypes } = useQuery({
    queryKey: ['all-event-types', env.id],
    queryFn: getAllEventTypes,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  // Extract event names from current query
  const eventNamesInQuery = useMemo(() => {
    const names = new Set<string>();
    const matches = query.matchAll(POSSIBLE_EVENT_NAME_REGEX);
    for (const match of matches) {
      const name = match[1]?.trim();
      if (name) names.add(name);
    }
    return Array.from(names).slice(0, 5); // Limit to 5
  }, [query]);

  // Fetch schemas for event names in query
  const [dataProperties, setDataProperties] = useState<Array<{ name: string; type: string }>>([]);

  useEffect(() => {
    if (eventNamesInQuery.length === 0) {
      setDataProperties([]);
      return;
    }

    // Fetch schemas for all event names in query
    Promise.all(
      eventNamesInQuery.map((name) =>
        getEventTypeSchemas({ cursor: null, nameSearch: name }).catch(() => ({ events: [] }))
      )
    ).then((results) => {
      const propsMap = new Map<string, string>();

      results.forEach((result) => {
        result.events.forEach((event) => {
          try {
            const schema = JSON.parse(event.schema || '{}');
            const dataProps = schema?.properties?.data?.properties;

            if (dataProps && typeof dataProps === 'object') {
              Object.entries(dataProps).forEach(([key, value]: [string, any]) => {
                const type = value?.type || 'unknown';
                // Union of all properties from all schemas
                if (!propsMap.has(key)) {
                  propsMap.set(key, type);
                }
              });
            }
          } catch {
            // Ignore parse errors
          }
        });
      });

      const props = Array.from(propsMap.entries()).map(([name, type]) => ({ name, type }));
      setDataProperties(props);
    });
  }, [eventNamesInQuery, getEventTypeSchemas]);

  return useMemo<SQLCompletionConfig>(
    () => ({
      columns: COLUMNS,
      keywords: KEYWORDS,
      functions: CLICKHOUSE_FUNCTIONS,
      tables: TABLES,
      eventNames: allEventTypes?.map((et) => et.name) || [],
      dataProperties,
    }),
    [allEventTypes, dataProperties]
  );
}
