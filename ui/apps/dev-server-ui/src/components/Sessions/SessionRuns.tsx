import { useCallback, useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';
import RunsStatusFilter from '@inngest/components/RunsPage/RunsStatusFilter';
import {
  IDCell,
  PillCell,
  Table,
  TableBlankState,
  TextCell,
  TimeCell,
} from '@inngest/components/Table';
import { RunStatusCell } from '@inngest/components/Table/Cell';
import { useCalculatedStartTime } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  useBatchedSearchParams,
  useSearchParam,
  useValidatedArraySearchParam,
} from '@inngest/components/hooks/useSearchParams';
import { SessionsIcon } from '@inngest/components/icons/sections/Sessions';
import {
  isFunctionRunStatus,
  type FunctionRunStatus,
} from '@inngest/components/types/functionRun';
import {
  durationToString,
  formatMilliseconds,
  parseDuration,
} from '@inngest/components/utils/date';
import { RiRefreshLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { useGetTrigger } from '@/hooks/useGetTrigger';

import { type SessionRun, useSessionRuns } from './useSessionRuns';

const DEFAULT_RANGE = '7d';
const POLL_INTERVAL = 2500;

const columnHelper = createColumnHelper<SessionRun>();

const columns = [
  columnHelper.accessor('status', {
    header: 'Status',
    enableSorting: false,
    cell: ({ row }) => (
      <div className="flex items-center">
        <RunStatusCell status={row.original.status} runID={row.original.id} />
      </div>
    ),
  }),
  columnHelper.accessor('id', {
    header: 'Run ID',
    enableSorting: false,
    cell: (info) => (
      <div className="flex items-center">
        <IDCell>{info.getValue()}</IDCell>
      </div>
    ),
  }),
  columnHelper.accessor('eventName', {
    header: 'Trigger',
    enableSorting: false,
    cell: (info) => {
      const eventName = info.getValue();
      if (!eventName) return null;
      return (
        <div className="flex items-center">
          <PillCell type="EVENT">{eventName}</PillCell>
        </div>
      );
    },
  }),
  columnHelper.accessor('functionSlug', {
    header: 'Function',
    enableSorting: false,
    cell: (info) => (
      <div className="flex max-w-md items-center">
        <TextCell className="min-w-0">{info.getValue()}</TextCell>
      </div>
    ),
  }),
  columnHelper.accessor('queuedAt', {
    header: 'Queued at',
    enableSorting: false,
    cell: (info) => (
      <div className="flex items-center">
        <TimeCell date={new Date(info.getValue())} />
      </div>
    ),
  }),
  columnHelper.display({
    id: 'duration',
    header: 'Duration',
    cell: ({ row }) => {
      const { startedAt, endedAt } = row.original;
      const durationMS =
        startedAt && endedAt
          ? new Date(endedAt).getTime() - new Date(startedAt).getTime()
          : null;
      return (
        <div className="flex items-center">
          <TextCell>
            {durationMS !== null ? formatMilliseconds(durationMS) : '-'}
          </TextCell>
        </div>
      );
    },
  }),
];

export function SessionRuns({
  sessionKey,
  sessionId,
}: {
  sessionKey: string;
  sessionId: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const getTrigger = useGetTrigger();
  const [expandedRunIDs, setExpandedRunIDs] = useState<string[]>([]);
  const batchUpdate = useBatchedSearchParams();

  const [lastDays] = useSearchParam('last');
  const [startTimeParam] = useSearchParam('start');
  const [endTimeParam] = useSearchParam('end');
  const [filteredStatus = [], setFilteredStatus, removeFilteredStatus] =
    useValidatedArraySearchParam('filterStatus', isFunctionRunStatus);

  const calculatedStartTime = useCalculatedStartTime({
    lastDays,
    startTime: startTimeParam,
    defaultTime: DEFAULT_RANGE,
  });
  const startTime = calculatedStartTime.toISOString();
  const endTime = useMemo(
    () => (endTimeParam ? new Date(endTimeParam) : new Date()).toISOString(),
    [endTimeParam],
  );

  const {
    data: runs = [],
    isPending,
    isFetching,
    error,
    refetch,
  } = useSessionRuns({ sessionKey, sessionId, startTime, endTime });

  const filteredRuns = useMemo(() => {
    if (filteredStatus.length === 0) return runs;
    return runs.filter((run) =>
      filteredStatus.includes(run.status as FunctionRunStatus),
    );
  }, [runs, filteredStatus]);

  const onDaysChange = useCallback(
    (range: RangeChangeProps) => {
      if (range.type === 'relative') {
        batchUpdate({
          last: durationToString(range.duration),
          start: null,
          end: null,
        });
      } else {
        batchUpdate({
          last: null,
          start: range.start.toISOString(),
          end: range.end.toISOString(),
        });
      }
    },
    [batchUpdate],
  );

  const onStatusFilterChange = useCallback(
    (statuses: FunctionRunStatus[]) => {
      if (statuses.length > 0) {
        setFilteredStatus(statuses);
      } else {
        removeFilteredStatus();
      }
    },
    [setFilteredStatus, removeFilteredStatus],
  );

  return (
    <main className="bg-canvasBase text-basis no-scrollbar flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 flex h-11 items-center justify-between gap-1.5 px-3">
        <div className="flex items-center gap-1.5">
          <TimeFilter
            daysAgoMax={7}
            onDaysChange={onDaysChange}
            defaultValue={
              lastDays
                ? { type: 'relative', duration: parseDuration(lastDays) }
                : startTimeParam && endTimeParam
                  ? {
                      type: 'absolute',
                      start: new Date(startTimeParam),
                      end: new Date(endTimeParam),
                    }
                  : { type: 'relative', duration: parseDuration(DEFAULT_RANGE) }
            }
          />
          <RunsStatusFilter
            selectedStatuses={filteredStatus}
            onStatusesChange={onStatusFilterChange}
          />
        </div>
        <TotalCount totalCount={isPending ? undefined : filteredRuns.length} />
      </div>
      <div className="flex-1 overflow-y-auto pb-2" ref={containerRef}>
        {error ? (
          <ErrorCard error={error} reset={() => refetch()} />
        ) : (
          <>
            <Table
              columns={columns}
              data={filteredRuns}
              isLoading={isPending}
              blankState={
                <TableBlankState
                  icon={<SessionsIcon />}
                  title={`No runs found for ${sessionId}`}
                  actions={null}
                />
              }
              expandedIDs={expandedRunIDs}
              onRowClick={(row) =>
                setExpandedRunIDs((prev) =>
                  prev.includes(row.original.id)
                    ? prev.filter((id) => id !== row.original.id)
                    : [...prev, row.original.id],
                )
              }
              renderSubComponent={({ row }) => (
                <RunDetailsV4
                  runID={row.original.id}
                  getTrigger={getTrigger}
                  pollInterval={POLL_INTERVAL}
                  standalone={false}
                  readOnly
                />
              )}
            />
            <div className="flex flex-col items-center pt-2">
              <Button
                kind="secondary"
                appearance="outlined"
                label="Refresh runs"
                icon={<RiRefreshLine />}
                iconSide="left"
                onClick={() => refetch()}
                loading={isFetching}
                disabled={isFetching}
              />
            </div>
          </>
        )}
      </div>
    </main>
  );
}

function TotalCount({ totalCount }: { totalCount: number | undefined }) {
  if (totalCount === undefined) return null;

  const formatted = new Intl.NumberFormat().format(totalCount);
  return (
    <span className="text-muted text-xs font-semibold">
      {formatted} {totalCount === 1 ? 'run' : 'runs'}
    </span>
  );
}
