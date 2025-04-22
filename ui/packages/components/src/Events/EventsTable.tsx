'use client';

import { useCallback, useState } from 'react';
import type { Route } from 'next';
import dynamic from 'next/dynamic';
import { Button } from '@inngest/components/Button';
import TableBlankState from '@inngest/components/EventTypes/TableBlankState';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { Pill } from '@inngest/components/Pill';
import NewTable from '@inngest/components/Table/NewTable';
import { DEFAULT_TIME } from '@inngest/components/hooks/useCalculatedStartTime';
import { type Event, type PageInfo } from '@inngest/components/types/event';
import { cn } from '@inngest/components/utils/classNames';
import { durationToString, parseDuration } from '@inngest/components/utils/date';
import { RiArrowRightUpLine, RiSearchLine } from '@remixicon/react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';

import type { RangeChangeProps } from '../DatePicker/RangePicker';
import EntityFilter from '../Filter/EntityFilter';
import {
  useBatchedSearchParams,
  useSearchParam,
  useStringArraySearchParam,
} from '../hooks/useSearchParam';
import type { Features } from '../types/features';
import { EventDetails } from './EventDetails';
import TotalCount from './TotalCount';
import { useColumns } from './columns';

const CodeSearch = dynamic(() => import('@inngest/components/CodeSearch/CodeSearch'), {
  ssr: false,
});

const refreshInterval = 5000;

export function EventsTable({
  getEvents,
  getEventDetails,
  pathCreator,
  emptyActions,
  expandedRowActions,
  features,
}: {
  emptyActions: React.ReactNode;
  expandedRowActions: (eventName: string) => React.ReactNode;
  pathCreator: {
    eventType: (params: { eventName: string }) => Route;
    runPopout: (params: { runID: string }) => Route;
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
  }) => Promise<{ events: Omit<Event, 'payload'>[]; pageInfo: PageInfo; totalCount: number }>;
  getEventDetails: ({ eventName }: { eventName: string }) => Promise<Omit<Event, 'payload'>>;
  features: Pick<Features, 'history'>;
}) {
  const columns = useColumns({ pathCreator });
  const [cursor, setCursor] = useState<string | null>(null);
  const [showSearch, setShowSearch] = useState(false);
  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const batchUpdate = useBatchedSearchParams();
  const [filteredEvent = [], setFilteredEvent, removeFilteredEvent] =
    useStringArraySearchParam('filterEvent');
  const [search, setSearch, removeSearch] = useSearchParam('search');
  const source = undefined;
  const [expandedIDs, setExpandedIDs] = useState<string[]>([]);

  const {
    isPending, // first load, no data
    error,
    data: eventsData,
    isFetching, // refetching
    // TODO: implement infinite scrolling
  } = useQuery({
    queryKey: ['events', { cursor, eventName: filteredEvent, source, startTime, celQuery: search }],
    queryFn: useCallback(() => {
      return getEvents({ cursor, eventName: filteredEvent, source, startTime, celQuery: search });
    }, [getEvents, cursor, filteredEvent, source, startTime, search]),
    placeholderData: keepPreviousData,
    refetchInterval: !cursor ? refreshInterval : 0,
  });

  const onSearchChange = useCallback(
    (value: string) => {
      if (value.length > 0) {
        setSearch(value);
      } else {
        removeSearch();
      }
    },
    [setSearch]
  );

  const onEventFilterChange = useCallback(
    (value: string[]) => {
      if (value.length > 0) {
        setFilteredEvent(value);
      } else {
        removeFilteredEvent();
      }
    },
    [removeFilteredEvent, setFilteredEvent]
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

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10">
        <div className="m-3 flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            {/* TODO: Wire entity */}
            <EntityFilter
              type="event"
              onFilterChange={onEventFilterChange}
              selectedEntities={filteredEvent}
              entities={[]}
            />
            <Button
              icon={<RiSearchLine />}
              size="large"
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
            <TotalCount totalCount={eventsData?.totalCount} />
          </div>
          <div className="flex">
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
                // TODO: Create "Inspecting an event" doc in Monitor
                href="https://www.inngest.com/docs/platform/monitor/inspecting-events?ref=events-table"
              />
            </div>
            <div className="border-subtle border-b">
              <CodeSearch
                onSearch={onSearchChange}
                placeholder="event.data.userId == “1234” or event.data.billingPlan == 'Enterprise'"
                value={search}
              />
            </div>
          </>
        )}
      </div>

      <div className="h-[calc(100%-58px)] overflow-y-auto">
        <NewTable
          columns={columns}
          data={eventsData?.events || []}
          isLoading={isPending}
          blankState={<TableBlankState actions={emptyActions} />}
          renderSubComponent={({ row }) => (
            <EventDetails
              pathCreator={pathCreator}
              eventName={row.original.name}
              getEventDetails={getEventDetails}
              expandedRowActions={expandedRowActions}
            />
          )}
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
      </div>
    </div>
  );
}
