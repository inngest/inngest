'use client';

import { useCallback, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useInfiniteQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, type Row } from '@tanstack/react-table';

import { BlankSlate } from '@/components/Blank';
import SendEventButton from '@/components/Event/SendEventButton';
import Table from '@/components/Table';
import TriggerTag from '@/components/Trigger/TriggerTag';
import { client } from '@/store/baseApi';
import { GetTriggersStreamDocument, type StreamItem } from '@/store/generated';
import { selectEvent, selectRun } from '@/store/global';
import { useAppDispatch } from '@/store/hooks';
import { fullDate } from '@/utils/date';
import FunctionRunList from './FunctionRunList';

// import SourceBadge from './SourceBadge';

const columnHelper = createColumnHelper<StreamItem>();

const columns = [
  columnHelper.accessor('createdAt', {
    header: () => <span>Started At</span>,
    cell: (props) => (
      <time dateTime={fullDate(new Date(props.getValue()))} suppressHydrationWarning={true}>
        {fullDate(new Date(props.getValue()))}
      </time>
    ),
  }),
  // The Source BE is not built yet
  // columnHelper.accessor((row) => row.source.name, {
  //   id: 'source',
  //   cell: (props) => <SourceBadge row={props.row} />,
  //   header: () => <span>Source</span>,
  // }),
  columnHelper.accessor('type', {
    header: () => <span>Trigger</span>,
    cell: (props) => (
      <TriggerTag value={props.row.original.trigger} type={props.row.original.type} />
    ),
  }),
  columnHelper.accessor('runs', {
    header: () => <span>Function</span>,
    cell: (props) => <FunctionRunList functionRuns={props.getValue()} />,
  }),
];

export default function Stream() {
  const fetchTriggersStream = async ({ pageParam, direction }) => {
    const variables = {
      limit: 20, // Page size
      before: direction === 'forward' ? pageParam : null,
      after: direction === 'backward' ? pageParam : null,
    };

    const data = await client.request(GetTriggersStreamDocument, variables);
    return data.stream;
  };

  const { data, fetchNextPage, fetchPreviousPage, isFetching, isLoading } = useInfiniteQuery({
    queryKey: ['triggers-stream'],
    queryFn: fetchTriggersStream,
    initialPageParam: null,
    getNextPageParam: (lastPage) => {
      const lastTrigger = lastPage[lastPage.length - 1];
      if (lastTrigger) {
        return lastTrigger.createdAt; // Use the createdAt of the last trigger as cursor
      }
      return undefined;
    },
    getPreviousPageParam: (firstPage) => {
      const firstTrigger = firstPage[0];
      if (firstTrigger) {
        return firstTrigger.createdAt; // Use the createdAt of the first trigger as cursor
      }
      return undefined;
    },
  });

  // We must flatten the array of arrays from the useInfiniteQuery hook
  const triggers = data?.pages.reduce((acc, page) => {
    return [...acc, ...page];
  });
  const tableContainerRef = useRef<HTMLDivElement>(null);

  const fetchMoreOnBottomReached = useCallback(
    (containerRefElement?: HTMLDivElement | null) => {
      if (containerRefElement && triggers?.length > 0) {
        const { scrollHeight, scrollTop, clientHeight } = containerRefElement;
        // Threshold for triggering the fetch
        const threshold = 200;

        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < threshold;

        // Check if scrolled to the top
        const reachedTop = scrollTop < threshold;

        if ((reachedBottom || reachedTop) && !isFetching) {
          console.log(reachedBottom ? 'bottom scroll' : 'top scroll');
          if (reachedBottom) {
            fetchNextPage();
          } else {
            fetchPreviousPage();
          }
        }
      }
    },
    [fetchNextPage, fetchPreviousPage, isFetching],
  );

  useEffect(() => {
    fetchMoreOnBottomReached(tableContainerRef.current);
  }, [fetchMoreOnBottomReached]);

  const dispatch = useAppDispatch();
  const router = useRouter();

  function handleOpenSlideOver({
    triggerID,
    e,
  }: {
    triggerID: string;
    e: React.MouseEvent<HTMLElement>;
  }) {
    if (e.target instanceof HTMLElement) {
      const runID = e.target.dataset.key;
      router.push(`/newstream/trigger/${triggerID}`);
      dispatch(selectEvent(triggerID));
      if (runID) {
        dispatch(selectRun(runID));
      }
    }
  }

  const customRowProps = (row: Row<StreamItem>) => ({
    style: {
      verticalAlign: row.original.runs && row.original.runs.length > 1 ? 'baseline' : 'initial',
      cursor: 'pointer',
    },
    onClick: (e: React.MouseEvent<HTMLElement>) =>
      handleOpenSlideOver({ triggerID: row.original.id, e }),
  });

  return (
    <div className="flex flex-col min-h-0 min-w-0">
      <div className="flex justify-end px-5 py-2">
        <SendEventButton
          label="Test Event"
          data={JSON.stringify({
            name: '',
            data: {},
            user: {},
          })}
        />
      </div>
      <div
        className="min-h-0 overflow-y-auto"
        onScroll={(e) => fetchMoreOnBottomReached(e.target as HTMLDivElement)}
        ref={tableContainerRef}
      >
        <Table
          options={{
            data: triggers ?? [],
            columns,
            getCoreRowModel: getCoreRowModel(),
            enableSorting: false,
            enablePinning: true,
            initialState: {
              columnPinning: {
                left: ['startedAt'],
              },
            },
          }}
          customRowProps={customRowProps}
          blankState={
            <BlankSlate
              title="Inngest hasn't received any events"
              subtitle="Read our documentation to learn how to send events to Inngest."
              imageUrl="/images/no-events.png"
              link={{
                text: 'Sending Events',
                url: 'https://www.inngest.com/docs/events',
              }}
            />
          }
        />
      </div>
    </div>
  );
}
