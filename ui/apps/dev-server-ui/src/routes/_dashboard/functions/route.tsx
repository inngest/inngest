import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router';

import { useMemo, useRef, useState } from 'react';

import { Button } from '@inngest/components/Button/NewButton';
import { Search } from '@inngest/components/Forms/Search';
import TableBlankState from '@inngest/components/Functions/TableBlankState';
import { Header } from '@inngest/components/Header/NewHeader';
import { Info } from '@inngest/components/Info/Info';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { Link } from '@inngest/components/Link/NewLink';
import { Pill, PillContent } from '@inngest/components/Pill/NewPill';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Table } from '@inngest/components/Table/OldTable';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { useSearchParam } from '@inngest/components/hooks/useNewSearchParams';
import { RiExternalLinkLine } from '@remixicon/react';
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  type Row,
  type SortingState,
} from '@tanstack/react-table';
import { toast } from 'sonner';

import {
  FunctionTriggerTypes,
  useGetFunctionsQuery,
  useInvokeFunctionMutation,
  type Function,
} from '@/store/generated';
import { HorizontalPillList } from '@inngest/components/Pill/HorizontalPillList';

export const Route = createFileRoute('/_dashboard/functions')({
  component: FunctionListComponent,
});
const columnHelper = createColumnHelper<Function>();
const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Function name</span>,
    cell: (props) => (
      <p className="text-sm font-medium leading-7">{props.getValue()}</p>
    ),
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
      const navigate = useNavigate();
      const doesFunctionAcceptPayload = useMemo(() => {
        return Boolean(
          props.row?.original?.triggers?.some(
            (trigger) => trigger.type === FunctionTriggerTypes.Event,
          ),
        );
      }, [props.row.original.triggers]);

      const [invokeFunction] = useInvokeFunctionMutation();

      return (
        <InvokeButton
          kind="secondary"
          appearance="outlined"
          disabled={false}
          doesFunctionAcceptPayload={doesFunctionAcceptPayload}
          btnAction={async ({ data, user }) => {
            await invokeFunction({
              data,
              functionSlug: props.row.original.slug,
              user,
            });
            toast.success('Function invoked');
            navigate({ to: '/runs' });
          }}
        />
      );
    },
    enableSorting: false,
    enableGlobalFilter: false,
  }),
];

function FunctionListComponent() {
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
    [isFetching, functions],
  );

  const tableColumns = useMemo(
    () =>
      isFetching
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="my-[0.3rem] h-5" />,
          }))
        : columns,
    [isFetching, functions],
  );

  const navigate = useNavigate();

  function handleOpenSlideOver({
    e,
    functionSlug,
  }: {
    e: React.MouseEvent<HTMLElement>;
    functionSlug: string;
  }) {
    if (e.target instanceof HTMLElement) {
      const params = new URLSearchParams({ slug: functionSlug });
      const url = `/functions/config?${params.toString()}`;
      navigate({ to: url });
    }
  }

  const customRowProps = (row: Row<Function>) => ({
    style: {
      cursor: 'pointer',
    },
    onClick: (e: React.MouseEvent<HTMLElement>) => {
      handleOpenSlideOver({
        e,
        functionSlug: row.original.slug,
      });
    },
  });

  return (
    <div className="flex min-h-0 min-w-0 flex-col">
      <Header
        breadcrumb={[{ text: 'Functions' }]}
        infoIcon={
          <Info
            text="List of all function in the development environment."
            action={
              <Link
                className="text-sm"
                href={'https://www.inngest.com/docs/functions'}
              >
                Learn how to create a function
              </Link>
            }
          />
        }
      />
      <Outlet />
      <div className="mx-3 flex h-11 min-h-11 items-center">
        <Search
          name="search"
          placeholder="Search by function name"
          value={searchInput}
          className="w-[182px]"
          onUpdate={(value) => {
            setSearchInput(value);
            debouncedSearch();
          }}
        />
      </div>
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
          customRowProps={customRowProps}
          tableContainerRef={tableContainerRef}
          blankState={
            <TableBlankState
              actions={
                <Button
                  label="Go to docs"
                  href="https://www.inngest.com/docs/sdk/serve"
                  target="_blank"
                  icon={<RiExternalLinkLine />}
                  iconSide="left"
                />
              }
            />
          }
        />
      </main>
    </div>
  );
}
