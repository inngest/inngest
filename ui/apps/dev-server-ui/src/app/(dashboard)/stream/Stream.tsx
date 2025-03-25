'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { BlankSlate } from '@inngest/components/BlankSlate';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { Pill, PillContent } from '@inngest/components/Pill';
import { Table } from '@inngest/components/Table';
import { fullDate } from '@inngest/components/utils/date';
import { RiArrowDownSLine } from '@remixicon/react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, type Row } from '@tanstack/react-table';

import { queryClient } from '@/app/StoreProvider';
import SendEventButton from '@/components/Event/SendEventButton';
import { client } from '@/store/baseApi';
import {
  GetTriggersStreamDocument,
  type FunctionRun,
  type GetTriggersStreamQuery,
  type StreamItem,
} from '@/store/generated';
import FunctionRunList from './FunctionRunList';

const columnHelper = createColumnHelper<StreamItem>();

const columns = [
  columnHelper.accessor('createdAt', {
    header: () => <span>Queued at</span>,
    cell: (props) => (
      <time dateTime={fullDate(new Date(props.getValue()))} suppressHydrationWarning={true}>
        {fullDate(new Date(props.getValue()))}
      </time>
    ),
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('type', {
    header: () => <span>Trigger</span>,
    cell: (props) => (
      <Pill appearance="outlined">
        <PillContent type={props.row.original.type}>{props.row.original.trigger}</PillContent>
      </Pill>
    ),
    size: 300,
    minSize: 300,
  }),
  columnHelper.accessor('runs', {
    header: () => <span>Function</span>,
    cell: (props) => {
      let validFunctionRuns: FunctionRun[] = [];
      for (const run of props.getValue() ?? []) {
        if (run) {
          validFunctionRuns.push(run);
        }
      }

      return (
        <FunctionRunList inBatch={props.row.original.inBatch} functionRuns={validFunctionRuns} />
      );
    },
    size: 350,
    minSize: 350,
  }),
];

export default function Stream() {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const [freezeStream, setFreezeStream] = useState(false);
  const [showInternalEvents, setShowInternalEvents] = useState(false);
  const [tableScrollTopPosition, setTableScrollTopPosition] = useState(0);

  useEffect(() => {
    // Save table's parent scroll top value
    if (tableContainerRef.current) {
      const handleScroll = () => {
        const scrollTop = tableContainerRef.current?.scrollTop;
        if (scrollTop !== undefined) {
          setTableScrollTopPosition(scrollTop);
        }
      };

      tableContainerRef.current.addEventListener('scroll', handleScroll);

      return () => {
        tableContainerRef.current?.removeEventListener('scroll', handleScroll);
      };
    }
  }, []);

  const fetchTriggersStream = async ({ pageParam }: { pageParam: string | null }) => {
    const variables = {
      limit: 40, // Page size
      before: tableScrollTopPosition > 0 ? pageParam : null,
      includeInternalEvents: showInternalEvents,
    };

    const data: GetTriggersStreamQuery = await client.request(GetTriggersStreamDocument, variables);

    return data.stream;
  };

  const { data, fetchNextPage, isFetching, refetch } = useInfiniteQuery({
    queryKey: ['triggers-stream'],
    queryFn: fetchTriggersStream,
    refetchInterval: freezeStream || tableScrollTopPosition > 0 ? false : 2500,
    initialPageParam: null,
    getNextPageParam: (lastPage, pages) => {
      if (!lastPage) {
        return undefined;
      }

      const lastTrigger = lastPage[lastPage.length - 1];
      if (!lastTrigger) {
        return undefined;
      }

      return lastTrigger.id; // Use the id of the last trigger as cur sor
    },
  });

  useEffect(() => {
    if (!data?.pages || data.pages.length === 0) {
      return;
    }

    // Clear the cache due to internal event visibility toggling
    queryClient.setQueryData(['triggers-stream'], () => ({
      pages: [],
      pageParams: [null],
    }));

    refetch();
  }, [showInternalEvents]);

  // We must flatten the array of arrays from the useInfiniteQuery hook
  const triggers = useMemo(() => {
    if (!data?.pages) {
      return undefined;
    }
    if (data.pages.length === 0) {
      return [];
    }

    return data.pages.reduce((acc, page) => {
      return [...acc, ...page];
    });
  }, [data?.pages]);

  const fetchMoreOnScroll = useCallback(
    (containerRefElement?: HTMLDivElement | null) => {
      if (containerRefElement && triggers && triggers.length > 0) {
        const { scrollHeight, scrollTop, clientHeight } = containerRefElement;
        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isFetching) {
          fetchNextPage();
        }
      }
    },
    [fetchNextPage, isFetching]
  );

  const scrollToTop = () => {
    if (tableContainerRef.current) {
      tableContainerRef.current.scrollTo({
        top: 0,
        behavior: 'smooth', // Enable smooth scrolling
      });
    }
  };

  useEffect(() => {
    const hasMoreThanOnePage = data && data.pages?.length > 1;

    // If user scrolled down multiple pages and then to the top of the table, we clear the cache to only have 1 page again
    if (tableScrollTopPosition === 0 && hasMoreThanOnePage && !isFetching) {
      //@ts-ignore
      queryClient.setQueryData(['triggers-stream'], (data) => ({
        // @ts-ignore
        pages: data?.pages?.slice(0, 1),
        pageParams: [null],
      }));
    }
  }, [tableScrollTopPosition, isFetching]);

  const router = useRouter();

  function handleOpenSlideOver({
    triggerID,
    e,
    firstRunID,
  }: {
    triggerID: string;
    e: React.MouseEvent<HTMLElement>;
    firstRunID?: string;
  }) {
    if (e.target instanceof HTMLElement) {
      const runID = e.target.dataset.key || firstRunID;
      const params = new URLSearchParams({ event: triggerID });
      if (runID) {
        params.append('run', runID);
      }
      const url = `/stream/trigger?${params.toString()}`;
      router.push(url);
    }
  }

  const customRowProps = (row: Row<StreamItem>) => ({
    style: {
      verticalAlign: row.original.runs && row.original.runs.length > 1 ? 'top' : 'middle',
      cursor: 'pointer',
    },
    onClick: (e: React.MouseEvent<HTMLElement>) => {
      const firstRunID =
        row.original.runs && row.original.runs?.length > 0 ? row.original.runs[0]?.id : undefined;
      handleOpenSlideOver({
        triggerID: row.original.id,
        e,
        firstRunID: firstRunID,
      });
    },
  });

  return (
    <div className="flex min-h-0 min-w-0 flex-col">
      <Header
        breadcrumb={[{ text: 'Stream' }]}
        action={
          <div className="flex justify-end gap-1 py-2">
            <Button
              kind="secondary"
              appearance="outlined"
              label={`${showInternalEvents ? 'Hide' : 'Show'} internal events`}
              onClick={() => setShowInternalEvents((prev) => !prev)}
            />
            <Button
              kind="secondary"
              appearance="outlined"
              label={freezeStream ? 'Enable auto-refresh' : 'Disable auto-refresh'}
              onClick={() => setFreezeStream((prev) => !prev)}
            />
            <SendEventButton
              label="Test event"
              data={JSON.stringify({
                name: '',
                data: {},
                user: {},
              })}
            />
          </div>
        }
      />
      <div
        className="bg-canvasBase min-h-0 overflow-y-auto pb-10"
        onScroll={(e) => fetchMoreOnScroll(e.target as HTMLDivElement)}
        ref={tableContainerRef}
      >
        <Table
          options={{
            // @ts-expect-error TODO: Fix this
            data: triggers ?? [],
            columns,
            getCoreRowModel: getCoreRowModel(),
            enableSorting: false,
            enablePinning: true,
            initialState: {
              columnPinning: {
                left: ['createdAt'],
              },
            },
            defaultColumn: {
              minSize: 0,
              size: Number.MAX_SAFE_INTEGER,
              maxSize: Number.MAX_SAFE_INTEGER,
            },
          }}
          tableContainerRef={tableContainerRef}
          customRowProps={customRowProps}
          blankState={
            <BlankSlate
              title="Inngest hasn't received any events"
              subtitle="Read our documentation to learn how to send events to Inngest."
              imageUrl="/images/no-events.png"
              link={{
                text: 'Sending events',
                url: 'https://www.inngest.com/docs/events',
              }}
            />
          }
        />
      </div>
      {tableScrollTopPosition > 0 && (
        <span className="absolute bottom-5 right-5 animate-bounce">
          <Button
            kind="secondary"
            onClick={scrollToTop}
            icon={<RiArrowDownSLine className="rotate-180" />}
          />
        </span>
      )}
    </div>
  );
}
