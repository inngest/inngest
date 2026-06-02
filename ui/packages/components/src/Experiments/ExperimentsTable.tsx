import { useMemo, useState, type ReactNode } from 'react';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Search } from '@inngest/components/Forms/Search';
import { Select } from '@inngest/components/Select/Select';
import { StatusDot } from '@inngest/components/Status/StatusDot';
import { Table } from '@inngest/components/Table';
import useDebounce from '@inngest/components/hooks/useDebounce';
import { RiTimeLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';

import { ExperimentsBlankState } from './ExperimentsBlankState';
import { getActiveThreshold, isActive } from './status';
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

export type ExperimentStatusFilter = 'all' | 'active' | 'completed';

const statusOptions = {
  all: { id: 'all', name: 'All experiments' },
  active: { id: 'active', name: 'Active experiments' },
  completed: { id: 'completed', name: 'Completed experiments' },
} as const;

type ExperimentsTableProps = {
  data?: ExperimentListItem[];
  isPending: boolean;
  error: Error | null;
  refetch: () => void;
  // Receives the full row so callers can route by (functionSlug, experimentName).
  // Two functions can declare an experiment with the same name; the slug is the
  // URL disambiguator.
  onRowClick?: (row: ExperimentListItem) => void;
  /** Copy shown in the blank state when no experiments exist (unfiltered). */
  emptyDescription: ReactNode;
};

export function ExperimentsTable({
  data,
  isPending,
  error,
  refetch,
  onRowClick,
  emptyDescription: unfilteredEmptyDescription,
}: ExperimentsTableProps) {
  const [searchInput, setSearchInput] = useState('');
  const [searchFilter, setSearchFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<ExperimentStatusFilter>('active');

  const debouncedSearch = useDebounce(() => {
    setSearchFilter(searchInput);
  }, 300);

  // Compute the active/completed cutoff once per render so the filter AND the
  // per-row status-dot cells share one `Date` allocation instead of N+1.
  const activeThreshold = useMemo(() => getActiveThreshold(), [data]);

  const filteredData = useMemo(() => {
    if (!data) return [];
    let filtered = data;

    if (statusFilter === 'active') {
      filtered = filtered.filter((item) => isActive(item.lastSeen, activeThreshold));
    } else if (statusFilter === 'completed') {
      filtered = filtered.filter((item) => !isActive(item.lastSeen, activeThreshold));
    }

    if (searchFilter) {
      const lower = searchFilter.toLowerCase();
      filtered = filtered.filter((item) => item.experimentName.toLowerCase().includes(lower));
    }

    return filtered;
  }, [data, searchFilter, statusFilter, activeThreshold]);

  const columns = useMemo(
    () => [
      columnHelper.accessor('experimentName', {
        header: 'Experiment name',
        cell: (info) => {
          const active = isActive(info.row.original.lastSeen, activeThreshold);
          return (
            <div className="flex items-center gap-2">
              <StatusDot status={active ? 'ACTIVE' : 'ARCHIVED'} size="small" />
              <span className="text-basis truncate text-sm font-medium">{info.getValue()}</span>
            </div>
          );
        },
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
    ],
    [activeThreshold]
  );

  if (error) {
    return <ErrorCard error={error} reset={() => refetch()} />;
  }

  const isFiltered = Boolean(searchFilter) || statusFilter !== 'all';
  const emptyTitle = searchFilter
    ? `No experiments found matching "${searchFilter}"`
    : statusFilter === 'active'
    ? 'No active experiments'
    : statusFilter === 'completed'
    ? 'No completed experiments'
    : 'No experiments found';
  const emptyDescription = isFiltered
    ? 'Try adjusting your filters to find experiments.'
    : unfilteredEmptyDescription;

  return (
    <div className="bg-canvasBase text-basis no-scrollbar flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 mx-3 flex h-11 items-center gap-1.5">
        <Select
          onChange={(value) => setStatusFilter(value.id as ExperimentStatusFilter)}
          isLabelVisible={false}
          multiple={false}
          value={statusOptions[statusFilter]}
          size="small"
        >
          <Select.Button size="small">
            <div className="flex flex-row items-center gap-2">
              {statusFilter !== 'all' && (
                <StatusDot
                  status={statusFilter === 'active' ? 'ACTIVE' : 'ARCHIVED'}
                  size="small"
                />
              )}
              {statusOptions[statusFilter].name}
            </div>
          </Select.Button>
          <Select.Options>
            <Select.Option option={statusOptions.all}>{statusOptions.all.name}</Select.Option>
            <Select.Option option={statusOptions.active}>
              <div className="flex flex-row items-center gap-2">
                <StatusDot status="ACTIVE" size="small" />
                {statusOptions.active.name}
              </div>
            </Select.Option>
            <Select.Option option={statusOptions.completed}>
              <div className="flex flex-row items-center gap-2">
                <StatusDot status="ARCHIVED" size="small" />
                {statusOptions.completed.name}
              </div>
            </Select.Option>
          </Select.Options>
        </Select>
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
          onRowClick={onRowClick ? (row) => onRowClick(row.original) : undefined}
          blankState={
            <ExperimentsBlankState
              title={emptyTitle}
              description={emptyDescription}
              onRefresh={refetch}
            />
          }
        />
      </div>
    </div>
  );
}
