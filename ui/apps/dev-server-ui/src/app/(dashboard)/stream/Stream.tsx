'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useInfiniteQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, type Row } from '@tanstack/react-table';

import { queryClient } from '@/app/StoreProvider';
import { BlankSlate } from '@/components/Blank';
import Button from '@/components/Button/Button';
import SendEventButton from '@/components/Event/SendEventButton';
import Table from '@/components/Table';
import TriggerTag from '@/components/Trigger/TriggerTag';
import { IconChevron } from '@/icons';
import { client } from '@/store/baseApi';
import { GetTriggersStreamDocument, type StreamItem } from '@/store/generated';
import { fullDate } from '@/utils/date';
import FunctionRunList from './FunctionRunList';

// import OutputList from './OutputList';

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
    size: 250,
    minSize: 250,
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
    size: 300,
    minSize: 300,
  }),
  columnHelper.accessor('runs', {
    header: () => <span>Function</span>,
    cell: (props) => <FunctionRunList functionRuns={props.getValue()} />,
    size: 350,
    minSize: 350,
  }),
  // columnHelper.accessor((row) => row.runs, {
  //   id: 'output',
  //   cell: (props) => <OutputList functionRuns={props.row.original.runs} />,
  //   header: () => <span>Output</span>,
  // }),
];

export default function Stream() {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const [freezeStream, setFreezeStream] = useState(false);
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

  const fetchTriggersStream = async ({ pageParam }) => {
    const variables = {
      limit: 40, // Page size
      before: tableScrollTopPosition > 0 ? pageParam : null,
    };

    const data = await client.request(GetTriggersStreamDocument, variables);

    // @ts-expect-error
    return data.stream;
  };

  const { data, fetchNextPage, isFetching } = useInfiniteQuery({
    queryKey: ['triggers-stream'],
    queryFn: fetchTriggersStream,
    refetchInterval: freezeStream || tableScrollTopPosition > 0 ? false : 2500,
    initialPageParam: null,
    getNextPageParam: (lastPage, pages) => {
      const lastTrigger = lastPage[lastPage.length - 1];
      if (lastTrigger) {
        return lastTrigger.createdAt; // Use the createdAt of the last trigger as cursor
      }
      return undefined;
    },
  });

  // We must flatten the array of arrays from the useInfiniteQuery hook
  const triggers = data?.pages.reduce((acc, page) => {
    return [...acc, ...page];
  });

  const fetchMoreOnScroll = useCallback(
    (containerRefElement?: HTMLDivElement | null) => {
      if (containerRefElement && triggers?.length > 0) {
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
    isCron,
    e,
    firstRunID,
  }: {
    triggerID: string;
    isCron: boolean;
    e: React.MouseEvent<HTMLElement>;
    firstRunID?: string;
  }) {
    if (e.target instanceof HTMLElement) {
      const runID = e.target.dataset.key || firstRunID;
      const params = new URLSearchParams({
        [isCron ? 'cron' : 'event']: triggerID,
      });
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
      const isCron = row.original.type === 'CRON';
      const firstRunID =
        row.original.runs && row.original.runs?.length > 0 ? row.original.runs[0]?.id : undefined;
      handleOpenSlideOver({
        triggerID: row.original.id,
        e,
        firstRunID: firstRunID,
        isCron: isCron,
      });
    },
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
        className="min-h-0 overflow-y-auto pb-10"
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
                text: 'Sending Events',
                url: 'https://www.inngest.com/docs/events',
              }}
            />
          }
        />
      </div>
      {tableScrollTopPosition > 0 && (
        <span className="absolute bottom-5 right-5 animate-bounce">
          <Button btnAction={scrollToTop} icon={<IconChevron className="rotate-180" />} />
        </span>
      )}
    </div>
  );
}
