'use client';

import { useCallback, useEffect, useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import TableBlankState from '@inngest/components/EventTypes/TableBlankState';
import NewTable from '@inngest/components/Table/NewTable';
import {
  EventTypesOrderByDirection,
  EventTypesOrderByField,
  type EventType,
  type EventTypesOrderBy,
  type PageInfo,
} from '@inngest/components/types/eventType';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { type SortingState } from '@tanstack/react-table';

import { useSearchParam } from '../hooks/useSearchParam';
import EventTypesStatusFilter from './EventTypesStatusFilter';
import { useColumns } from './columns';

const refreshInterval = 5000;

export function EventTypesTable({
  getEventTypes,
  pathCreator,
  emptyActions,
}: {
  emptyActions: React.ReactNode;
  pathCreator: {
    function: (params: { functionSlug: string }) => Route;
    eventType: (params: { eventName: string }) => Route;
  };
  getEventTypes: ({
    cursor,
    pageSize,
    archived,
  }: {
    cursor: string | null;
    pageSize: number;
    archived: boolean;
    orderBy: EventTypesOrderBy[];
  }) => Promise<{ events: EventType[]; pageInfo: PageInfo; totalCount: number }>;
}) {
  const router = useRouter();
  const columns = useColumns({ pathCreator });
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
  const [pageSize] = useState(20);
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
    queryKey: ['event-types', { orderBy, cursor, pageSize, archived }],
    queryFn: useCallback(() => {
      return getEventTypes({ orderBy, cursor, pageSize, archived });
    }, [getEventTypes, orderBy, cursor, pageSize, archived]),
    placeholderData: keepPreviousData,
    refetchInterval: !cursor || page === 1 ? refreshInterval : 0,
  });

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
    <div>
      <div className="m-3 flex items-center">
        <EventTypesStatusFilter
          archived={archived}
          pathCreator={'/'}
          onStatusChange={onStatusFilterChange}
        />
      </div>
      <NewTable
        columns={columns}
        data={eventTypesData?.events || []}
        isLoading={isPending}
        sorting={sorting}
        setSorting={setSorting}
        blankState={<TableBlankState actions={emptyActions} />}
        onRowClick={(row) => router.push(pathCreator.eventType({ eventName: row.original.name }))}
      />
    </div>
  );
}
