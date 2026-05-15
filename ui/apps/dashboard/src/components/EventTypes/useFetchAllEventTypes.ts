import { useCallback } from 'react';

import { useEventTypes } from './useEventTypes';

/**
 * Hook to fetch all event types with pagination support
 * Fetches up to MAX_PAGES (5 pages × 40 events = 200 total)
 */
export function useFetchAllEventTypes() {
  const getEventTypes = useEventTypes();

  return useCallback(
    async (search?: string) => {
      const MAX_PAGES = 5; // Fetch up to 5 pages (40 per page = 200 total)
      const allEvents: Array<{ name: string }> = [];
      let cursor: string | null = null;
      let pageCount = 0;

      while (pageCount < MAX_PAGES) {
        const result = await getEventTypes({
          cursor,
          archived: false,
          nameSearch: search || null,
        });

        allEvents.push(
          ...result.events.map((e) => ({
            name: e.name,
          })),
        );
        pageCount++;

        // Check if there are more pages
        if (result.pageInfo.hasNextPage && result.pageInfo.endCursor) {
          cursor = result.pageInfo.endCursor;
        } else {
          // No more pages
          break;
        }
      }

      return allEvents;
    },
    [getEventTypes],
  );
}
