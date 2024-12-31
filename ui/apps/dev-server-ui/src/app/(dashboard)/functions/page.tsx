'use client';

import { useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { BlankSlate } from '@inngest/components/BlankSlate';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { Link } from '@inngest/components/Link/Link';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Table } from '@inngest/components/Table';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type SortingState,
} from '@tanstack/react-table';
import { toast } from 'sonner';

import SearchInput from '@/components/SearchInput/SearchInput';
import {
  FunctionTriggerTypes,
  useGetFunctionsQuery,
  useInvokeFunctionMutation,
  type Function,
} from '@/store/generated';

const columnHelper = createColumnHelper<Function>();
const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Function name</span>,
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
            <Pill appearance="outlined" key={i}>
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
      <Pill appearance="outlined">
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
      const router = useRouter();
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
          btnAction={async ({ data, user }) => {
            await invokeFunction({
              data,
              functionSlug: props.row.original.slug,
              user,
            });
            toast.success('Function invoked');
            router.push('/runs');
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
  const [value, upsert, remove] = useSearchParam('query');

  const [searchInput, setSearchInput] = useState(value || '');
  const [globalFilter, setGlobalFilter] = useState(value || '');

  const debouncedSearch = useDebounce(() => {
    if (searchInput) {
      upsert(searchInput);
    } else {
      remove();
    }
    setGlobalFilter(searchInput);
  }, 200);

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
            cell: () => <Skeleton className="my-[0.3rem] h-5" />,
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
              <Link
                arrowOnHover
                className="text-sm"
                href={'https://www.inngest.com/docs/functions'}
              >
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
                text: 'Serving functions',
                url: 'https://www.inngest.com/docs/sdk/serve',
              }}
            />
          }
        />
      </main>
    </div>
  );
}
