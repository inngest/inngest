import { useCallback, useMemo } from "react";
import { availableClickhouseFunctions } from "@inngest/components/SQLEditor/hooks/availableClickhouseFunctions";
import { useCache } from "@inngest/components/SQLEditor/hooks/useCache";
import type { SQLCompletionConfig } from "@inngest/components/SQLEditor/types";

import { useFetchAllEventTypes } from "@/components/EventTypes/useFetchAllEventTypes";
import { useEventTypeSchemas } from "../../InsightsTabManager/InsightsHelperPanel/features/SchemaExplorer/SchemasContext/useEventTypeSchemas";

const KEYWORDS = [
  "AND",
  "AS",
  "ASC",
  "BETWEEN",
  "DESC",
  "DISTINCT",
  "FALSE",
  "FROM",
  "GROUP BY",
  "IS",
  "LIKE",
  "LIMIT",
  "NOT",
  "NULL",
  "OFFSET",
  "OR",
  "ORDER BY",
  "SELECT",
  "TRUE",
  "WHERE",
] as const;

const TABLES = ["events"] as const;

// Common columns available on the events table
const COLUMNS = [
  "name",
  "data",
  "id",
  "ts",
  "ts_dt",
  "received_at",
  "received_at_dt",
  "v",
] as const;

// Convert ClickHouse functions to the format expected by autocomplete
const CLICKHOUSE_FUNCTIONS = availableClickhouseFunctions.map((name) => ({
  name,
  signature: `${name}($1)`,
}));

export function useSQLCompletionConfig(): SQLCompletionConfig {
  const fetchAllEventTypes = useFetchAllEventTypes();
  const getEventTypeSchemas = useEventTypeSchemas();

  // Cache for fetched event names with 5 minute TTL
  const eventNamesCache = useCache<string[]>({
    ttl: 5 * 60 * 1000,
    name: "eventNames",
  });

  // Cache for fetched schemas with 5 minute TTL
  const schemasCache = useCache<Array<{ name: string; type: string }>>({
    ttl: 5 * 60 * 1000,
    name: "eventSchemas",
  });

  // Create a function to fetch event names dynamically with nameSearch
  // Supports pagination up to a maximum number of pages with caching
  // NOTE: This function is called from fetchWithCache in useSQLCompletions
  // The cache is checked BEFORE calling this function
  const fetchEventNames = useCallback(
    async (search: string): Promise<string[]> => {
      const cacheKey = search || "__empty__";
      const allEvents = await fetchAllEventTypes(search);
      const eventNames = allEvents.map((e) => e.name);

      // Update cache
      eventNamesCache.set(cacheKey, eventNames);

      return eventNames;
    },
    [fetchAllEventTypes, eventNamesCache],
  );

  // Create a function to fetch schema for a specific event name
  const fetchEventSchema = useCallback(
    async (
      eventName: string,
    ): Promise<Array<{ name: string; type: string }>> => {
      try {
        const result = await getEventTypeSchemas({
          cursor: null,
          nameSearch: eventName,
        });
        const propsMap = new Map<string, string>();

        result.events.forEach((event) => {
          // Only process if the event name matches exactly
          if (event.name !== eventName) {
            return;
          }

          try {
            if (!event.schema) {
              return;
            }

            const schema = JSON.parse(event.schema);
            const dataProps = schema?.properties?.data?.properties;

            if (!dataProps || typeof dataProps !== "object") {
              return;
            }

            Object.entries(dataProps).forEach(([key, value]: [string, any]) => {
              const type = value?.type || "unknown";
              if (!propsMap.has(key)) {
                propsMap.set(key, type);
              }
            });
          } catch {
            // Ignore parse errors
          }
        });

        const props = Array.from(propsMap.entries()).map(([name, type]) => ({
          name,
          type,
        }));

        // Update cache
        schemasCache.set(eventName, props);

        return props;
      } catch (error) {
        return [];
      }
    },
    [getEventTypeSchemas, schemasCache],
  );

  return useMemo<SQLCompletionConfig>(
    () => ({
      columns: COLUMNS,
      keywords: KEYWORDS,
      functions: CLICKHOUSE_FUNCTIONS,
      tables: TABLES,
      eventNames: [],
      dataProperties: [], // Will be populated dynamically based on selected event
      fetchEventNames,
      fetchEventSchema,
      eventNamesCache,
      schemasCache,
    }),
    [fetchEventNames, fetchEventSchema, eventNamesCache, schemasCache],
  );
}
