'use client';

import { useCallback, useEffect, useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import TableBlankState from '@inngest/components/EventTypes/TableBlankState';
import { Search } from '@inngest/components/Forms/Search';
import NewTable from '@inngest/components/Table/NewTable';
import {
  EventTypesOrderByDirection,
  EventTypesOrderByField,
  type EventType,
  type EventTypesOrderBy,
  type PageInfo,
} from '@inngest/components/types/eventType';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { type Row, type SortingState } from '@tanstack/react-table';

import { useSearchParam } from '../hooks/useSearchParam';
import EventTypesStatusFilter from './EventTypesStatusFilter';
import { useColumns } from './columns';

const refreshInterval = 5000;

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
  const [cursor, setCursor] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [orderBy, setOrderBy] = useState<EventTypesOrderBy[]>([
    {
      field: EventTypesOrderByField.Name,
      direction: EventTypesOrderByDirection.Asc,
    },
  ]);

  const onStatusFilterChange = useCallback(
    (value: boolean) => {
      if (value) {
        setFilteredStatus('true'); // Set query param when archived is true
      } else {
        removeFilteredStatus(); // Remove query param when archived is false
      }
      // Reset cursor and page when filter changes
      setCursor(null);
      setPage(1);
    },
    [setFilteredStatus, removeFilteredStatus]
  );

  const {
    isPending, // first load, no data
    error,
    data: eventTypesData,
    isFetching, // refetching
    // TODO: implement infinite scrolling
  } = useQuery({
    queryKey: ['event-types', { orderBy, cursor, archived }],
    queryFn: useCallback(() => {
      return getEventTypes({ orderBy, cursor, archived });
    }, [getEventTypes, orderBy, cursor, archived]),
    placeholderData: keepPreviousData,
    refetchInterval: !cursor || page === 1 ? refreshInterval : 0,
  });

  const { data: volumeData, isPending: isVolumePending } = useQuery({
    queryKey: ['event-types-volume', { orderBy, cursor, archived }],
    queryFn: useCallback(() => {
      return getEventTypesVolume({ orderBy, cursor, archived });
    }, [getEventTypesVolume, orderBy, cursor, archived]),
    placeholderData: keepPreviousData,
    refetchInterval: !cursor || page === 1 ? refreshInterval : 0,
  });

  const mergedData = useCallback(() => {
    if (!eventTypesData?.events) return [];

    const volumeMap = new Map<string, EventType['volume']>();

    volumeData?.events.forEach((event) => {
      volumeMap.set(event.name, event.volume);
    });

    return eventTypesData.events.map((event) => ({
      ...event,
      volume: volumeMap.get(event.name) || {
        totalVolume: 0,
        dailyVolumeSlots: [],
      },
    }));
  }, [eventTypesData, volumeData]);

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
      // Back to first page when we sort changes
      setCursor(null);
      setPage(1);
    }
  }, [sorting, setOrderBy]);

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-50 m-3 flex items-center gap-2">
        <EventTypesStatusFilter
          archived={archived}
          pathCreator={'/'}
          onStatusChange={onStatusFilterChange}
        />
        {/* TODO: Wire search */}
        <Search
          name="search"
          placeholder="Search by event type"
          value={''}
          className="h-[30px] w-56 py-3"
          onUpdate={(value) => {}}
        />
      </div>
      <div className="h-[calc(100%-58px)] overflow-y-auto">
        <NewTable
          columns={columns}
          data={mergedData() || []}
          isLoading={isPending}
          sorting={sorting}
          setSorting={setSorting}
          blankState={<TableBlankState actions={emptyActions} />}
          onRowClick={(row) => router.push(pathCreator.eventType({ eventName: row.original.name }))}
        />
      </div>
    </div>
  );
}
