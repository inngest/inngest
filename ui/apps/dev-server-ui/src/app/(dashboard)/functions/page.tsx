'use client';

import { useMemo, useRef, useState } from 'react';
import { BlankSlate } from '@inngest/components/BlankSlate';
import { Table } from '@inngest/components/Table';
import { TriggerTags } from '@inngest/components/Tag/TriggerTags';
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
} from '@tanstack/react-table';

import SendEventButton from '@/components/Event/SendEventButton';
import TriggerCronButton from '@/components/Event/TriggerCronButton';
import SearchInput from '@/components/SearchInput/SearchInput';
import Skeleton from '@/components/Skeleton';
import useDebounce from '@/hooks/useDebounce';
import { FunctionTriggerTypes, useGetFunctionsQuery, type Function } from '@/store/generated';

const columnHelper = createColumnHelper<Function>();
const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Function Name</span>,
    cell: (props) => <p className="text-sm font-medium leading-7">{props.getValue()}</p>,
    sortingFn: 'text',
    filterFn: 'equalsString',
    enableGlobalFilter: true,
  }),
  columnHelper.accessor('triggers', {
    header: () => <span>Triggers</span>,
    cell: (props) => {
      const triggers = props.getValue();
      if (!triggers || triggers.length === 0) {
        return <></>;
      }
      return <TriggerTags triggers={triggers} />;
    },
    enableSorting: false,
    enableGlobalFilter: false,
  }),
  columnHelper.accessor('app', {
    header: () => <span>App</span>,
    cell: (props) => {
      const app = props.getValue();
      return app?.name;
    },
    enableSorting: false,
    enableGlobalFilter: false,
  }),
  columnHelper.accessor('url', {
    header: () => <span>App URL</span>,
    cell: (props) => {
      const cleanUrl = new URL(props.getValue() || '');
      cleanUrl.search = '';
      return <p className="text-sm">{cleanUrl.toString()}</p>;
    },
    enableSorting: false,
    enableGlobalFilter: false,
  }),
  columnHelper.display({
    id: 'triggerCTA',
    size: 55,
    cell: (props) => {
      const getFirstEventValue = () => {
        const eventTrigger = props.row?.original?.triggers?.find(
          (trigger) => trigger.type === FunctionTriggerTypes.Event
        );
        return eventTrigger ? eventTrigger.value : null;
      };
      const isCron = (): boolean => {
        return Boolean(
          props.row?.original?.triggers?.some(
            (trigger) => trigger.type === FunctionTriggerTypes.Cron
          )
        );
      };
      return (
        <>
          {getFirstEventValue() && (
            <SendEventButton
              appearance="outlined"
              label="Trigger"
              data={JSON.stringify({
                name: getFirstEventValue(),
                data: {},
                user: {},
              })}
            />
          )}
          {isCron() && (
            <TriggerCronButton appearance="outlined" functionId={props.row?.original?.slug} />
          )}
        </>
      );
    },
    enableSorting: false,
    enableGlobalFilter: false,
  }),
];

export default function FunctionList() {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'name',
      desc: false,
    },
  ]);
  const [searchInput, setSearchInput] = useState('');
  const [globalFilter, setGlobalFilter] = useState('');
  const debouncedSearch = useDebounce(() => {
    setGlobalFilter(searchInput);
  });

  const { data, isFetching } = useGetFunctionsQuery(undefined, {
    refetchOnMountOrArgChange: true,
  });
  const functions = data?.functions || [];

  const tableData = useMemo(
    () => (isFetching ? Array(8).fill({}) : functions),
    [isFetching, functions]
  );

  const tableColumns = useMemo(
    () =>
      isFetching
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="my-[0.3rem] block h-5" />,
          }))
        : columns,
    [isFetching, functions]
  );

  return (
    <div className="flex min-h-0 min-w-0 flex-col">
      <SearchInput
        placeholder="Search function..."
        value={searchInput}
        onChange={setSearchInput}
        debouncedSearch={debouncedSearch}
        className="py-4"
      />
      <main className="min-h-0 overflow-y-auto" ref={tableContainerRef}>
        <Table
          options={{
            data: tableData,
            columns: tableColumns,
            getCoreRowModel: getCoreRowModel(),
            enablePinning: true,
            initialState: {
              columnPinning: {
                left: ['name'],
              },
            },
            state: {
              sorting,
              globalFilter,
            },
            getSortedRowModel: getSortedRowModel(),
            onSortingChange: setSorting,
            enableSortingRemoval: false,
            getFilteredRowModel: getFilteredRowModel(),
            onGlobalFilterChange: setGlobalFilter,
          }}
          tableContainerRef={tableContainerRef}
          blankState={
            <BlankSlate
              title="Inngest has not detected any functions"
              subtitle="Read our documentation to learn how to serve your functions"
              imageUrl="/images/no-results.png"
              link={{
                text: 'Serving Functions',
                url: 'https://www.inngest.com/docs/sdk/serve',
              }}
            />
          }
        />
      </main>
    </div>
  );
}
