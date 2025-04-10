'use client';

import { useCallback, useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import TableBlankState from '@inngest/components/EventTypes/TableBlankState';
import NewTable from '@inngest/components/Table/NewTable';
import { type Event, type PageInfo } from '@inngest/components/types/event';
import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { useColumns } from './columns';

const refreshInterval = 5000;

export function EventsTable({
  getEvents,
  pathCreator,
  emptyActions,
}: {
  emptyActions: React.ReactNode;
  pathCreator: {
    function: (params: { functionSlug: string }) => Route;
    eventType: (params: { eventName: string }) => Route;
  };
  getEvents: ({
    cursor,
    eventName,
    source,
    startTime,
    celQuery,
  }: {
    eventName?: string[];
    cursor?: string | null;
    source?: string;
    startTime?: string;
    celQuery?: string;
  }) => Promise<{ events: Event[]; pageInfo: PageInfo }>;
}) {
  const router = useRouter();
  const columns = useColumns({ pathCreator });
  const [cursor, setCursor] = useState<string | null>(null);
  const eventName = undefined;
  const source = undefined;
  const startTime = undefined;
  const celQuery = undefined;

  const {
    isPending, // first load, no data
    error,
    data: eventsData,
    isFetching, // refetching
    // TODO: implement infinite scrolling
  } = useQuery({
    queryKey: ['events', { cursor, eventName, source, startTime, celQuery }],
    queryFn: useCallback(() => {
      return getEvents({ cursor, eventName, source, startTime, celQuery });
    }, [getEvents, cursor, eventName, source, startTime, celQuery]),
    placeholderData: keepPreviousData,
    refetchInterval: !cursor ? refreshInterval : 0,
  });

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 m-3 flex items-center gap-2">
        {/* Filters */}
      </div>
      <div className="h-[calc(100%-58px)] overflow-y-auto">
        <NewTable
          columns={columns}
          data={eventsData?.events || []}
          isLoading={isPending}
          blankState={<TableBlankState actions={emptyActions} />}
          // onRowClick={(row) => router.push(pathCreator.eventType({ eventName: row.original.name }))}
        />
      </div>
    </div>
  );
}
