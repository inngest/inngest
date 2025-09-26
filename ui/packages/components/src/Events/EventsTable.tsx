'use client';

import { useCallback, useEffect, useRef, useState, type UIEventHandler } from 'react';
import dynamic from 'next/dynamic';
import { Button } from '@inngest/components/Button';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import TableBlankState from '@inngest/components/EventTypes/TableBlankState';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { InfiniteScrollTrigger } from '@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger';
import { Pill } from '@inngest/components/Pill';
import { Table } from '@inngest/components/Table';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import {
  DEFAULT_TIME,
  useCalculatedStartTime,
} from '@inngest/components/hooks/useCalculatedStartTime';
import { type Event, type PageInfo } from '@inngest/components/types/event';
import { type EventType } from '@inngest/components/types/eventType';
import { cn } from '@inngest/components/utils/classNames';
import { durationToString, parseDuration } from '@inngest/components/utils/date';
import { RiArrowRightUpLine, RiSearchLine } from '@remixicon/react';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';

import type { RangeChangeProps } from '../DatePicker/RangePicker';
import EntityFilter from '../Filter/EntityFilter';
import { useShared } from '../SharedContext/SharedContext';
import { usePathCreator } from '../SharedContext/usePathCreator';
import {
  useBatchedSearchParams,
  useBooleanSearchParam,
  useSearchParam,
  useStringArraySearchParam,
} from '../hooks/useSearchParam';
import type { Features } from '../types/features';
import { parseCelSearchError } from '../utils/searchErrorParser';
import { EventDetails } from './EventDetails';
import TotalCount from './TotalCount';
import { useColumns } from './columns';

const CodeSearch = dynamic(() => import('@inngest/components/CodeSearch/CodeSearch'), {
  ssr: false,
});

export function EventsTable({
  getEvents,
  getEventDetails,
  getEventPayload,
  getEventTypes,
  eventNames,
  singleEventTypePage,
  emptyActions,
  expandedRowActions,
  features,
  standalone = false,
  pollInterval,
  autoRefresh,
}: {
  emptyActions: React.ReactNode;
  expandedRowActions: ({
    eventName,
    payload,
  }: {
    eventName?: string;
    payload?: string;
  }) => React.ReactNode;
  getEvents: ({
    cursor,
    eventNames,
    source,
    startTime,
    endTime,
    celQuery,
    includeInternalEvents,
  }: {
    eventNames: string[] | null;
    cursor: string | null;
    source?: string;
    startTime: string;
    endTime: string | null;
    celQuery?: string;
    includeInternalEvents?: boolean;
  }) => Promise<{ events: Event[]; pageInfo: PageInfo; totalCount: number }>;
  getEventDetails: ({ eventID }: { eventID: string }) => Promise<Event>;
  getEventPayload: ({ eventID }: { eventID: string }) => Promise<Pick<Event, 'payload'>>;
  getEventTypes?: () => Promise<Required<Pick<EventType, 'name' | 'id'>>[]>;
  eventNames?: string[];
  singleEventTypePage?: boolean;
  features: Pick<Features, 'history'>;
  standalone?: boolean;
  pollInterval?: number;
  autoRefresh?: boolean;
}) {
  const { pathCreator } = usePathCreator();
  const { cloud } = useShared();
  const columns = useColumns({ pathCreator, singleEventTypePage });
  const [showSearch, setShowSearch] = useState(false);
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const batchUpdate = useBatchedSearchParams();
  const [filteredEvent, setFilteredEvent, removeFilteredEvent] =
    useStringArraySearchParam('filterEvent');
  const [includeInternalEvents] = useBooleanSearchParam('includeInternal');
  const [search, setSearch, removeSearch] = useSearchParam('search');
  const source = undefined;
  const [expandedIDs, setExpandedIDs] = useState<string[]>([]);
  const containerRef = useRef<HTMLDivElement>(null);
  const [isScrollable, setIsScrollable] = useState(false);

  /* The start date comes from either the absolute start time or the relative time */
  const calculatedStartTime = useCalculatedStartTime({ lastDays, startTime });

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

  const {
    isPending, // first load, no data
    error,
    fetchNextPage,
    hasNextPage,
    data: eventsData,
    isFetching,
    refetch,
    isRefetching,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: [
      'events',
      {
        eventNames: filteredEvent || eventNames || null,
        source,
        startTime: calculatedStartTime.toISOString(),
        endTime: endTime ?? null,
        celQuery: search,
        includeInternalEvents: singleEventTypePage || (includeInternalEvents ?? false),
      },
    ],
    queryFn: ({ pageParam }: { pageParam: string | null }) =>
      getEvents({
        cursor: pageParam,
        eventNames: filteredEvent ?? eventNames ?? null,
        source,
        startTime: calculatedStartTime.toISOString(),
        endTime: endTime ?? null,
        celQuery: search,
        includeInternalEvents: singleEventTypePage || (includeInternalEvents ?? false),
      }),
    getNextPageParam: (lastPage) => {
      if (!lastPage || !lastPage.pageInfo.hasNextPage) {
        return undefined;
      }
      return lastPage.pageInfo.endCursor;
    },
    initialPageParam: null,
    refetchInterval: autoRefresh ? pollInterval : false,
    select: (data) => ({
      ...data,
      events: data.pages.flatMap((page) => page.events),
      totalCount: data.pages[data.pages.length - 1]?.totalCount ?? 0,
    }),
  });

  const searchError = parseCelSearchError(error);
  /* TODO: Find out what to do with the event types filter, since it will affect performance */

  // const { data: eventTypesData } = useQuery({
  //   queryKey: ['event-types'],
  //   queryFn: () => getEventTypes(),
  // });

  // const onEventFilterChange = useCallback(
  //   (value: string[]) => {
  //     if (value.length > 0) {
  //       setFilteredEvent(value);
  //     } else {
  //       removeFilteredEvent();
  //     }
  //   },
  //   [removeFilteredEvent, setFilteredEvent]
  // );

  const onSearchChange = useCallback(
    (value: string) => {
      if (value.length > 0) {
        setSearch(value);
      } else {
        removeSearch();
      }
    },
    [setSearch, removeSearch]
  );

  const onDaysChange = useCallback(
    (value: RangeChangeProps) => {
      if (value.type === 'relative') {
        batchUpdate({
          last: durationToString(value.duration),
          start: null,
          end: null,
        });
      } else {
        batchUpdate({
          last: null,
          start: value.start.toISOString(),
          end: value.end.toISOString(),
        });
      }
    },
    [batchUpdate]
  );

  useEffect(() => {
    const el = containerRef.current;
    if (el) {
      setIsScrollable(el.scrollHeight > el.clientHeight);
    }
  }, [eventsData]);

  const hasEventsData = eventsData?.events && eventsData?.events.length > 0;

  if (error && !searchError) {
    return <ErrorCard error={error} reset={() => refetch()} />;
  }

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10">
        <div className="mx-3 flex h-11 items-center justify-between gap-1.5">
          <div className="flex items-center gap-1.5">
            {/* <EntityFilter
              type="event"
              onFilterChange={onEventFilterChange}
              selectedEntities={filteredEvent ?? []}
              entities={eventTypesData ?? []}
            /> */}
            {/* TODO: Remove disabled prop when search is implemented in Dev Server */}
            {cloud ? (
              <Button
                icon={<RiSearchLine />}
                size="small"
                kind="secondary"
                iconSide="left"
                appearance="outlined"
                label={showSearch ? 'Hide search' : 'Show search'}
                onClick={() => setShowSearch((prev) => !prev)}
                className={cn(
                  search
                    ? 'after:bg-secondary-moderate after:mb-3 after:ml-0.5 after:h-2 after:w-2 after:rounded'
                    : '',
                  'h-[26px] w-[103px] rounded'
                )}
              />
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    disabled
                    icon={<RiSearchLine />}
                    size="small"
                    kind="secondary"
                    iconSide="left"
                    appearance="outlined"
                    label={showSearch ? 'Hide search' : 'Show search'}
                    onClick={() => setShowSearch((prev) => !prev)}
                    className={cn(
                      search
                        ? 'after:bg-secondary-moderate after:mb-3 after:ml-0.5 after:h-2 after:w-2 after:rounded'
                        : ''
                    )}
                  />
                </TooltipTrigger>
                <TooltipContent>Coming soon</TooltipContent>
              </Tooltip>
            )}
            <TimeFilter
              daysAgoMax={features.history}
              onDaysChange={onDaysChange}
              defaultValue={
                lastDays
                  ? {
                      type: 'relative',
                      duration: parseDuration(lastDays),
                    }
                  : startTime && endTime
                  ? {
                      type: 'absolute',
                      start: new Date(startTime),
                      end: new Date(endTime),
                    }
                  : {
                      type: 'relative',
                      duration: parseDuration(DEFAULT_TIME),
                    }
              }
            />
          </div>
          <div className="flex">
            <TotalCount totalCount={eventsData?.totalCount} />
          </div>
        </div>
        {showSearch && (
          <>
            <div className="bg-codeEditor flex items-center justify-between p-4">
              <div className="flex items-center gap-2">
                <p className="text-subtle text-sm">Search your events by using a CEL query</p>
                <Pill kind="primary">Beta</Pill>
              </div>
              <Button
                appearance="outlined"
                label="Read the docs"
                icon={<RiArrowRightUpLine />}
                iconSide="right"
                size="small"
                target="_blank"
                href="https://www.inngest.com/docs/platform/monitor/inspecting-events?ref=events-table"
              />
            </div>
            <div className="border-subtle border-b">
              <CodeSearch
                onSearch={onSearchChange}
                placeholder="event.data.userId == “1234” or event.data.billingPlan == 'Enterprise'"
                value={search}
                preset="events"
                searchError={searchError}
              />
            </div>
          </>
        )}
      </div>

      <div className="flex-1 overflow-y-auto" ref={containerRef}>
        <Table
          columns={columns}
          data={eventsData?.events || []}
          isLoading={isPending || (isFetching && !isFetchingNextPage && !isRefetching)}
          blankState={<TableBlankState actions={emptyActions} />}
          renderSubComponent={({ row }) => {
            const { id, name, runs } = row.original;
            const initialData: Pick<Event, 'name' | 'runs'> = { name, runs };
            return (
              <EventDetails
                initialData={initialData}
                getEventDetails={getEventDetails}
                getEventPayload={getEventPayload}
                expandedRowActions={expandedRowActions}
                standalone={standalone}
                eventID={id}
                pollInterval={pollInterval}
                autoRefresh={autoRefresh}
              />
            );
          }}
          expandedIDs={expandedIDs}
          onRowClick={(row) => {
            if (expandedIDs.includes(row.original.id)) {
              setExpandedIDs((prev) => {
                return prev.filter((id) => id !== row.original.id);
              });
            } else {
              setExpandedIDs((prev) => {
                return [...prev, row.original.id];
              });
            }
          }}
        />
        <InfiniteScrollTrigger
          onIntersect={fetchNextPage}
          hasMore={hasNextPage ?? false}
          isLoading={isFetching || isFetchingNextPage}
        />
        {!hasNextPage && hasEventsData && isScrollable && !isFetchingNextPage && !isFetching && (
          <div className="flex flex-col items-center pb-4 pt-8">
            <p className="text-muted text-sm">No additional events found.</p>
            <Button
              label="Back to top"
              kind="primary"
              appearance="ghost"
              onClick={() => scrollToTop(true)}
            />
          </div>
        )}
        {isFetchingNextPage && (
          <div className="flex flex-col items-center">
            <Button appearance="outlined" label="loading" loading={true} />
          </div>
        )}
      </div>
    </div>
  );
}
