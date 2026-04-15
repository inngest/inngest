import { useMemo, useState } from 'react';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Search } from '@inngest/components/Forms/Search';
import { Pill } from '@inngest/components/Pill/Pill';
import { Table } from '@inngest/components/Table';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { RiFlaskLine, RiTimeLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';

import type { ExperimentListItem } from './types';

const columnHelper = createColumnHelper<ExperimentListItem>();

function formatDuration(date: Date): string {
  if (!date || date.getTime() === 0) return '-';
  return formatDistanceToNow(date, { addSuffix: false });
}

function formatNumber(n: number): string {
  return n.toLocaleString();
}

function formatStrategy(strategy: string): string {
  if (!strategy) return '-';
  return `experiment.${strategy}`;
}

const columns = [
  columnHelper.accessor('experimentName', {
    header: 'Experiment name',
    cell: (info) => (
      <div className="flex items-center gap-2">
        <span className="bg-primary-moderate h-2 w-2 flex-shrink-0 rounded-full" />
        <span className="text-basis truncate text-sm font-medium">{info.getValue()}</span>
      </div>
    ),
    size: 220,
  }),
  columnHelper.accessor('selectionStrategy', {
    header: 'Experiment type',
    cell: (info) => (
      <span className="text-muted font-mono text-xs">{formatStrategy(info.getValue())}</span>
    ),
    size: 160,
  }),
  columnHelper.accessor('variantCount', {
    header: 'Variants',
    cell: (info) => {
      const count = info.getValue();
      return (
        <span className="text-muted text-sm">
          {count} {count === 1 ? 'variant' : 'variants'}
        </span>
      );
    },
    size: 120,
  }),
  columnHelper.accessor('firstSeen', {
    header: 'Time running',
    cell: (info) => {
      const date = info.getValue();
      return (
        <div className="text-muted flex items-center gap-1 text-sm">
          <RiTimeLine className="h-3.5 w-3.5 flex-shrink-0" />
          <span>{formatDuration(date)}</span>
        </div>
      );
    },
    size: 140,
  }),
  columnHelper.accessor('totalRuns', {
    header: 'Total run count',
    cell: (info) => (
      <span className="text-basis text-sm font-medium tabular-nums">
        {formatNumber(info.getValue())}
      </span>
    ),
    size: 130,
  }),
];

type ExperimentsTableProps = {
  data?: ExperimentListItem[];
  isPending: boolean;
  error: Error | null;
  refetch: () => void;
  getRowHref?: (item: ExperimentListItem) => string;
  onRowClick?: (item: ExperimentListItem) => void;
};

export function ExperimentsTable({
  data,
  isPending,
  error,
  refetch,
  getRowHref,
  onRowClick,
}: ExperimentsTableProps) {
  const [searchInput, setSearchInput] = useState('');
  const [searchFilter, setSearchFilter] = useState('');

  const debouncedSearch = useDebounce(() => {
    setSearchFilter(searchInput);
  }, 300);

  const filteredData = useMemo(() => {
    if (!data) return [];
    if (!searchFilter) return data;
    const lower = searchFilter.toLowerCase();
    return data.filter((item) => item.experimentName.toLowerCase().includes(lower));
  }, [data, searchFilter]);

  if (error) {
    return <ErrorCard error={error} reset={() => refetch()} />;
  }

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 mx-3 flex h-11 items-center gap-1.5">
        <Pill
          kind="primary"
          appearance="outlined"
          icon={<span className="bg-primary-moderate mr-1 inline-block h-2 w-2 rounded-full" />}
          iconSide="left"
        >
          Active experiments
        </Pill>
        <Search
          name="search"
          placeholder="Search by experiment name"
          value={searchInput}
          className="w-[220px]"
          onUpdate={(value) => {
            setSearchInput(value);
            debouncedSearch();
          }}
        />
      </div>
      <div className="flex-1 overflow-y-auto">
        <Table
          columns={columns}
          data={filteredData}
          isLoading={isPending}
          getRowHref={getRowHref ? (row) => getRowHref(row.original) : undefined}
          onRowClick={onRowClick ? (row) => onRowClick(row.original) : undefined}
          blankState={
            <div className="flex flex-col items-center justify-center py-16">
              <RiFlaskLine className="text-disabled mb-3 h-10 w-10" />
              <p className="text-muted text-sm">
                {searchFilter
                  ? `No experiments found matching "${searchFilter}"`
                  : 'No experiments found'}
              </p>
              <p className="text-subtle mt-1 text-xs">
                Experiments will appear here once your functions start running experiment steps.
              </p>
            </div>
          }
        />
      </div>
    </div>
  );
}
