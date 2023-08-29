'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useInfiniteQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, type Row } from '@tanstack/react-table';

import { BlankSlate } from '@/components/Blank';
import Button from '@/components/Button';
import SendEventButton from '@/components/Event/SendEventButton';
import Table from '@/components/Table';
import TriggerTag from '@/components/Trigger/TriggerTag';
import { IconChevron } from '@/icons';
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
  const [prevScrollTop, setPrevScrollTop] = useState(0); // Store the previous scrollTop value
  const [freezeStream, setFreezeStream] = useState(false);

  const fetchTriggersStream = async ({ pageParam, direction }) => {
    const variables = {
      limit: 40, // Page size
      before: direction === 'forward' && prevScrollTop > 0 ? pageParam : null,
      after: direction === 'backward' && prevScrollTop > 0 ? pageParam : null,
    };

    const data = await client.request(GetTriggersStreamDocument, variables);
    return data.stream;
  };

  const { data, fetchNextPage, fetchPreviousPage, isFetching, hasNextPage } = useInfiniteQuery({
    queryKey: ['triggers-stream'],
    queryFn: fetchTriggersStream,
    refetchInterval: freezeStream ? false : 2500,
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

  const fetchMoreOnScroll = useCallback(
    (containerRefElement?: HTMLDivElement | null) => {
      if (containerRefElement && triggers?.length > 0) {
        const { scrollHeight, scrollTop, clientHeight } = containerRefElement;
        setPrevScrollTop(scrollTop);
        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        // Check if scrolled to the top
        const reachedTop = scrollTop === 0;

        if ((reachedBottom || reachedTop) && !isFetching) {
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

  const scrollToTop = () => {
    if (tableContainerRef.current) {
      tableContainerRef.current.scrollTo({
        top: 0,
        behavior: 'smooth', // Enable smooth scrolling
      });
    }
  };

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
      <div className="flex justify-end px-5 py-2 gap-1">
        <Button
          label={freezeStream ? 'Resume Stream' : 'Freeze Stream'}
          btnAction={() => setFreezeStream(!freezeStream)}
        />
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
        onScroll={(e) => fetchMoreOnScroll(e.target as HTMLDivElement)}
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
          tableContainerRef={tableContainerRef}
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
      {prevScrollTop > 0 && (
        <span className="absolute bottom-5 right-5">
          <Button
            btnAction={scrollToTop}
            icon={<IconChevron className="text-indigo-100 rotate-180" />}
          />
        </span>
      )}
    </div>
  );
}
