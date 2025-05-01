'use client';

import { useCallback, useEffect, useRef, useState, type UIEventHandler } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import TableBlankState from '@inngest/components/EventTypes/TableBlankState';
import { Search } from '@inngest/components/Forms/Search';
import NewTable from '@inngest/components/Table/NewTable';
import useDebounce from '@inngest/components/hooks/useDebounce';
import {
  EventTypesOrderByDirection,
  EventTypesOrderByField,
  type EventType,
  type EventTypesOrderBy,
  type PageInfo,
} from '@inngest/components/types/eventType';
import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { type Row, type SortingState } from '@tanstack/react-table';

import { useSearchParam } from '../hooks/useSearchParam';
import EventTypesStatusFilter from './EventTypesStatusFilter';
import { useColumns } from './columns';

export function EventTypesTable({
  getEventTypes,
  getEventTypesVolume,
  pathCreator,
  emptyActions,
  eventTypeActions,
}: {
  emptyActions: React.ReactNode;
  eventTypeActions: (props: Row<EventType>) => React.ReactElement;
  pathCreator: {
    function: (params: { functionSlug: string }) => Route;
    eventType: (params: { eventName: string }) => Route;
  };
  getEventTypes: ({
    cursor,
    archived,
  }: {
    cursor: string | null;
    nameSearch: string | null;
    archived: boolean;
    orderBy: EventTypesOrderBy[];
  }) => Promise<{ events: Omit<EventType, 'volume'>[]; pageInfo: PageInfo }>;
  getEventTypesVolume: ({
    cursor,
    archived,
  }: {
    cursor: string | null;
    archived: boolean;
    orderBy: EventTypesOrderBy[];
  }) => Promise<{ events: Pick<EventType, 'volume' | 'name'>[]; pageInfo: PageInfo }>;
}) {
  const router = useRouter();
  const columns = useColumns({ pathCreator, eventTypeActions });
  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'name',
      desc: true,
    },
  ]);

  const [filteredStatus, setFilteredStatus, removeFilteredStatus] = useSearchParam('archived');
  const archived = filteredStatus === 'true';
  const [searchInput, setSearchInput] = useState<string>('');
  const [isScrollable, setIsScrollable] = useState(false);
  const [nameSearch = null, setNameSearch, removeNameSearch] = useSearchParam('nameSearch');
  const [orderBy, setOrderBy] = useState<EventTypesOrderBy[]>([
    {
      field: EventTypesOrderByField.Name,
      direction: EventTypesOrderByDirection.Asc,
    },
  ]);
  const containerRef = useRef<HTMLDivElement>(null);
  const debouncedSearch = useDebounce(() => {
    if (searchInput === '') {
      removeNameSearch();
    } else {
      setNameSearch(searchInput);
    }
  }, 400);

  const onStatusFilterChange = useCallback(
    (value: boolean) => {
      if (value) {
        setFilteredStatus('true'); // Set query param when archived is true
      } else {
        removeFilteredStatus(); // Remove query param when archived is false
      }
    },
    [setFilteredStatus, removeFilteredStatus]
  );

  const {
    isPending, // first load, no data
    error,
    fetchNextPage,
    hasNextPage,
    data: eventTypesData,
    isFetching,
    isFetchingNextPage, // refetching
  } = useInfiniteQuery({
    queryKey: ['event-types', { orderBy, archived, nameSearch }],
    queryFn: ({ pageParam }: { pageParam: string | null }) =>
      getEventTypes({ orderBy, cursor: pageParam, archived, nameSearch }),
    placeholderData: keepPreviousData,
    getNextPageParam: (lastPage) => {
      if (!lastPage || !lastPage.pageInfo.hasNextPage) {
        return undefined;
      }
      return lastPage.pageInfo.endCursor;
    },
    initialPageParam: null,
  });

  const lastPageCursor = eventTypesData?.pages.at(-1)?.pageInfo.endCursor ?? null;

  const { data: volumeData, isPending: isVolumePending } = useQuery({
    queryKey: ['event-types-volume', { orderBy, archived, cursor: lastPageCursor }],
    queryFn: () => {
      return getEventTypesVolume({ orderBy, archived, cursor: lastPageCursor });
    },
    enabled: !!eventTypesData, // only run once we have event data
    placeholderData: keepPreviousData,
  });

  const mergedData = useCallback(() => {
    if (!eventTypesData?.pages) {
      return undefined;
    }
    if (eventTypesData.pages.length === 0) {
      return [];
    }

    const allEvents = eventTypesData.pages.flatMap((page) => page.events);

    const volumeMap = new Map<string, EventType['volume']>();

    volumeData?.events.forEach((event) => {
      volumeMap.set(event.name, event.volume);
    });

    return allEvents.map((event) => ({
      ...event,
      volume: volumeMap.get(event.name) || {
        totalVolume: 0,
        dailyVolumeSlots: [],
      },
    }));
  }, [eventTypesData, volumeData]);

  const data = mergedData();
  const hasEventTypesData = data && data.length > 0;

  useEffect(() => {
    const el = containerRef.current;
    if (el) {
      setIsScrollable(el.scrollHeight > el.clientHeight);
    }
  }, [data]);

  const onScroll: UIEventHandler<HTMLDivElement> = useCallback(
    (event) => {
      if (hasEventTypesData && hasNextPage && !isFetchingNextPage) {
        const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;

        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isFetching) {
          fetchNextPage();
        }
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage, mergedData]
  );

  useEffect(() => {
    const sortEntry = sorting[0];
    if (!sortEntry) return;

    const sortColumn = sortEntry.id;
    if (sortColumn) {
      const orderBy: EventTypesOrderBy[] = [
        {
          field: EventTypesOrderByField.Name,
          direction: sortEntry.desc
            ? EventTypesOrderByDirection.Desc
            : EventTypesOrderByDirection.Asc,
        },
      ];
      setOrderBy(orderBy);
    }
  }, [sorting, setOrderBy]);

  const scrollToTop = useCallback(
    (smooth = false) => {
      if (containerRef.current) {
        containerRef.current.scrollTo({
          top: 0,
          behavior: smooth ? 'smooth' : 'auto',
        });
      }
    },
    [containerRef.current]
  );

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 m-3 flex items-center gap-2">
        <EventTypesStatusFilter
          archived={archived}
          pathCreator={'/'}
          onStatusChange={onStatusFilterChange}
        />
        <Search
          name="search"
          placeholder="Search by event type"
          value={searchInput}
          className="h-[30px] w-56 py-3"
          onUpdate={(value) => {
            setSearchInput(value);
            debouncedSearch();
          }}
        />
      </div>
      <div className="h-[calc(100%-58px)] overflow-y-auto" onScroll={onScroll} ref={containerRef}>
        <NewTable
          columns={columns}
          data={mergedData() || []}
          isLoading={isPending}
          // TODO: Re-enable this when API supports sorting by event name
          // sorting={sorting}
          // setSorting={setSorting}
          blankState={<TableBlankState actions={emptyActions} />}
          onRowClick={(row) => router.push(pathCreator.eventType({ eventName: row.original.name }))}
        />
        {!hasNextPage && hasEventTypesData && isScrollable && (
          <div className="flex flex-col items-center pt-8">
            <p className="text-muted text-sm">No additional event types found.</p>
            <Button
              label="Back to top"
              kind="primary"
              appearance="ghost"
              onClick={() => scrollToTop(true)}
            />
          </div>
        )}
      </div>
    </div>
  );
}
