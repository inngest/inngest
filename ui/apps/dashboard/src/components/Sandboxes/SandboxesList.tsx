import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { Table, TableBlankState } from '@inngest/components/Table';
import { StatusCell, TimeCell } from '@inngest/components/Table/Cell';
import { SandboxesIcon } from '@inngest/components/icons/sections/Sandboxes';
import {
  useBatchedSearchParams,
  useSearchParam,
} from '@inngest/components/hooks/useSearchParams';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
} from '@inngest/components/utils/date';
import { useNavigate } from '@tanstack/react-router';
import type { ColumnDef } from '@tanstack/react-table';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import type { SandboxStatus } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

import {
  compactSandboxID,
  formatDuration,
  formatMemory,
  sandboxStatus,
} from './utils';

const DEFAULT_DURATION = { days: 7 };

const SandboxesListDocument = graphql(`
  query SandboxesList(
    $envSlug: String!
    $from: Time!
    $until: Time!
    $after: String
  ) {
    envBySlug(slug: $envSlug) {
      sandboxes(
        first: 50
        after: $after
        filter: { from: $from, until: $until }
      ) {
        edges {
          cursor
          node {
            id
            name
            status
            vcpu
            memoryMB
            createdAt
            startedAt
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
        summary {
          active
          totalCreated
          p95StartTimeMS
          failureRate
          launchUnknown
        }
      }
    }
  }
`);

type SandboxRow = {
  id: string;
  name: string;
  status: SandboxStatus;
  vcpu: number;
  memoryMB: number;
  createdAt: string;
  startedAt?: string | null;
};

const columns: ColumnDef<SandboxRow>[] = [
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const display = sandboxStatus(row.original.status);
      return <StatusCell status={display.colorStatus} label={display.label} />;
    },
  },
  {
    accessorKey: 'name',
    header: 'Sandbox',
    cell: ({ row }) => (
      <div className="flex min-w-0 flex-col py-1">
        <span className="text-basis truncate text-sm font-medium">
          {row.original.name}
        </span>
        <span className="text-light font-mono text-xs">
          {compactSandboxID(row.original.id)}
        </span>
      </div>
    ),
  },
  {
    accessorKey: 'createdAt',
    header: 'Created',
    cell: ({ row }) => (
      <TimeCell date={row.original.createdAt} format="relative" />
    ),
  },
  {
    accessorKey: 'startedAt',
    header: 'Started',
    cell: ({ row }) =>
      row.original.startedAt ? (
        <TimeCell date={row.original.startedAt} format="relative" />
      ) : (
        <span className="text-light text-sm">—</span>
      ),
  },
  {
    id: 'startTime',
    header: 'Start time',
    cell: ({ row }) => (
      <span className="text-muted text-sm">
        {row.original.startedAt
          ? formatDuration(
              new Date(row.original.startedAt).getTime() -
                new Date(row.original.createdAt).getTime(),
            )
          : '—'}
      </span>
    ),
  },
  {
    id: 'size',
    header: 'Size',
    cell: ({ row }) => (
      <span className="text-muted whitespace-nowrap text-sm">
        {row.original.vcpu} vCPU · {formatMemory(row.original.memoryMB)}
      </span>
    ),
  },
];

export function SandboxesList() {
  const environment = useEnvironment();
  const navigate = useNavigate();
  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const [cursor] = useSearchParam('cursor');
  const [cursorEnv] = useSearchParam('cursorEnv');
  const batchUpdate = useBatchedSearchParams();
  const pageCursor = cursorEnv === environment.slug ? cursor : undefined;

  const range = useMemo(() => {
    const parsedStart = toDate(start);
    const parsedEnd = toDate(end);
    if (parsedStart && parsedEnd)
      return { from: parsedStart, until: parsedEnd };

    const until = new Date();
    return {
      from: subtractDuration(
        until,
        (duration && parseDuration(duration)) || DEFAULT_DURATION,
      ),
      until,
    };
  }, [duration, end, start]);

  const { data, error, isLoading, refetch } = useGraphQLQuery({
    query: SandboxesListDocument,
    variables: {
      envSlug: environment.slug,
      from: range.from.toISOString(),
      until: range.until.toISOString(),
      after: pageCursor,
    },
  });

  const connection = data?.envBySlug?.sandboxes;
  const rows = connection?.edges.map(({ node }) => node) ?? [];
  const summary = connection?.summary;
  const defaultRange =
    start && end
      ? {
          type: 'absolute' as const,
          start: new Date(start),
          end: new Date(end),
        }
      : {
          type: 'relative' as const,
          duration: (duration && parseDuration(duration)) || DEFAULT_DURATION,
        };

  return (
    <main className="bg-canvasBase h-full overflow-y-auto px-6 pb-10">
      <div className="flex items-center py-5">
        <TimeFilter
          daysAgoMax={30}
          defaultValue={defaultRange}
          onDaysChange={(next: RangeChangeProps) => {
            batchUpdate({
              cursor: null,
              cursorEnv: null,
              duration:
                next.type === 'relative'
                  ? durationToString(next.duration)
                  : null,
              start: next.type === 'absolute' ? next.start.toISOString() : null,
              end: next.type === 'absolute' ? next.end.toISOString() : null,
            });
          }}
        />
      </div>

      <div className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <SummaryCard label="Active sandboxes" value={summary?.active} />
        <SummaryCard
          label="Total sandboxes created"
          value={summary?.totalCreated}
        />
        <SummaryCard
          label="p95 start time"
          value={
            summary?.p95StartTimeMS == null
              ? undefined
              : formatDuration(summary.p95StartTimeMS)
          }
        />
        <SummaryCard
          label="Failure rate"
          value={
            summary == null
              ? undefined
              : new Intl.NumberFormat(undefined, {
                  style: 'percent',
                  maximumFractionDigits: 2,
                }).format(summary.failureRate)
          }
          detail={
            summary?.launchUnknown
              ? `${summary.launchUnknown} launch result unknown`
              : undefined
          }
        />
      </div>

      {error ? (
        <ErrorCard error={error} reset={() => refetch()} />
      ) : (
        <div className="border-subtle overflow-x-auto rounded-md border">
          <div className="min-w-[850px]">
            <Table
              columns={columns}
              data={rows}
              isLoading={isLoading}
              getRowHref={(row) =>
                pathCreator.sandbox({
                  envSlug: environment.slug,
                  sandboxID: row.original.id,
                })
              }
              onRowClick={(row) => {
                navigate({
                  to: pathCreator.sandbox({
                    envSlug: environment.slug,
                    sandboxID: row.original.id,
                  }),
                });
              }}
              blankState={
                <TableBlankState
                  icon={<SandboxesIcon />}
                  title="No sandboxes in this time range"
                  description="Create a sandbox or select a wider time range."
                  actions={null}
                />
              }
            />
          </div>
          {(pageCursor || connection?.pageInfo.hasNextPage) && (
            <div className="border-subtle flex justify-end gap-2 border-t p-2">
              {pageCursor && (
                <Button
                  kind="secondary"
                  appearance="outlined"
                  label="First page"
                  onClick={() => batchUpdate({ cursor: null, cursorEnv: null })}
                />
              )}
              {connection?.pageInfo.hasNextPage &&
                connection.pageInfo.endCursor && (
                  <Button
                    kind="secondary"
                    appearance="outlined"
                    label="Next page"
                    onClick={() =>
                      batchUpdate({
                        cursor: connection.pageInfo.endCursor,
                        cursorEnv: environment.slug,
                      })
                    }
                  />
                )}
            </div>
          )}
        </div>
      )}
    </main>
  );
}

function SummaryCard({
  label,
  value,
  detail,
}: {
  label: string;
  value: number | string | undefined;
  detail?: string;
}) {
  return (
    <div className="border-subtle bg-canvasBase min-h-[92px] rounded-md border p-4">
      <div className="text-muted mb-1 text-sm">{label}</div>
      <div className="text-basis text-3xl font-medium leading-8">
        {value ?? '—'}
      </div>
      {detail && <div className="text-light mt-1 text-xs">{detail}</div>}
    </div>
  );
}
