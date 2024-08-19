'use client';

import { useMemo, useRef, useState } from 'react';
import { BlankSlate } from '@inngest/components/BlankSlate';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { Link } from '@inngest/components/Link/Link';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { Table } from '@inngest/components/Table';
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
} from '@tanstack/react-table';

import SearchInput from '@/components/SearchInput/SearchInput';
import Skeleton from '@/components/Skeleton';
import useDebounce from '@/hooks/useDebounce';
import {
  FunctionTriggerTypes,
  useGetFunctionsQuery,
  useInvokeFunctionMutation,
  type Function,
} from '@/store/generated';

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
      return (
        <HorizontalPillList
          alwaysVisibleCount={2}
          pills={triggers.map((trigger, i) => (
            <Pill className="text-sm font-normal" key={i}>
              <PillContent type={trigger.type}>{trigger.value}</PillContent>
            </Pill>
          ))}
        />
      );
    },
    enableSorting: false,
    enableGlobalFilter: false,
  }),
  columnHelper.accessor('app', {
    header: () => <span>App</span>,
    cell: (props) => (
      <Pill className="text-sm font-normal">
        <PillContent type="APP">{props.getValue()?.name}</PillContent>
      </Pill>
    ),
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
      const doesFunctionAcceptPayload = useMemo(() => {
        return Boolean(
          props.row?.original?.triggers?.some(
            (trigger) => trigger.type === FunctionTriggerTypes.Event
          )
        );
      }, [props.row.original.triggers]);

      const [invokeFunction] = useInvokeFunctionMutation();

      return (
        <InvokeButton
          disabled={false}
          doesFunctionAcceptPayload={doesFunctionAcceptPayload}
          btnAppearance="outlined"
          btnAction={(data) => {
            invokeFunction({
              data,
              functionSlug: props.row.original.slug,
            });
          }}
        />
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
      <Header
        breadcrumb={[{ text: 'Functions' }]}
        infoIcon={
          <Info
            text="List of all function in the development environment."
            action={
              <Link href={'https://www.inngest.com/docs/functions'} className="text-md">
                Learn how to create a function
              </Link>
            }
          />
        }
        action={
          <SearchInput
            placeholder="Search function..."
            value={searchInput}
            onChange={setSearchInput}
            debouncedSearch={debouncedSearch}
          />
        }
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
