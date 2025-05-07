import { useQuery } from '@tanstack/react-query';

import type { EventTypesTable } from './EventTypesTable';

export function useEventTypeVolume(
  eventName: string,
  getEventTypeVolume: React.ComponentProps<typeof EventTypesTable>['getEventTypeVolume']
) {
  return useQuery({
    queryKey: ['event-type-volume', eventName],
    queryFn: () => getEventTypeVolume({ eventName }),
    staleTime: 5 * 60 * 1000, // cache for 5 min
  });
}
