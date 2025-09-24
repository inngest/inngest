import { useRouter } from 'next/navigation';
import { RiLightbulbLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { Button } from '../Button';
import { ErrorCard } from '../Error/ErrorCard';
import { Pill } from '../Pill';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { useGetDebugSession, type DebugSessionRun } from '../SharedContext/useGetDebugSession';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { Skeleton } from '../Skeleton';
import { StatusCell, Table, TextCell, TimeCell } from '../Table';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';

const DEBUG_SESSION_REFETCH_INTERVAL = 1000;

type HistoryProps = {
  functionSlug: string;
  debugSessionID?: string;
  runID?: string;
};

type HistoryTable = DebugSessionRun | null;

export const EmptyHistory = () => {
  return (
    <div className="bg-canvasBase text-basis mx-auto my-6 flex flex-col items-center">
      <div className="text-center">
        <p className="mb-2 text-xl">No debug runs yet for this session.</p>
        <p className="text-subtle max-w-xl text-sm">Initiate a debug run with the controls above</p>
      </div>
      {/* <div className="flex items-center gap-3">{actions}</div> */}
    </div>
  );
};

export const History = ({ functionSlug, debugSessionID, runID }: HistoryProps) => {
  const { pathCreator } = usePathCreator();
  const router = useRouter();

  const { booleanFlag } = useBooleanFlag();
  const { value: pollingDisabled, isReady: pollingFlagReady } = booleanFlag(
    'polling-disabled',
    false
  );

  const { data, loading, error } = useGetDebugSession({
    functionSlug,
    debugSessionID,
    runID,
    refetchInterval: pollingFlagReady && pollingDisabled ? 0 : DEBUG_SESSION_REFETCH_INTERVAL,
  });

  if (loading) {
    return (
      <div className="flex w-full flex-col gap-2">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  if (error || !data) {
    return <ErrorCard error={error || new Error('No data found')} />;
  }

  const load = (debugRunID: string) => {
    const debuggerPath = pathCreator.debugger({
      functionSlug,
      runID,
      debugRunID,
      debugSessionID,
    });

    router.push(debuggerPath);
  };

  const columnHelper = createColumnHelper<HistoryTable>();

  const columns = [
    columnHelper.accessor('startedAt', {
      cell: (rawStartedAt) => {
        const startedAt = rawStartedAt.getValue();
        return startedAt ? <TimeCell date={new Date(startedAt)} /> : <TextCell>-</TextCell>;
      },
      size: 25,
      enableSorting: true,
    }),
    columnHelper.accessor('status', {
      cell: (rawStatus) => {
        const status = rawStatus.getValue();
        return <StatusCell key={status} status={status} label={status} size="small" />;
      },
      enableSorting: false,
    }),
    columnHelper.accessor('tags', {
      cell: () => {
        return (
          <Tooltip>
            <TooltipTrigger>
              <Pill appearance="outlined" kind="primary">
                <div
                  className="flex flex-row items-center gap-1"
                  onClick={(e) => {
                    e.stopPropagation();
                  }}
                >
                  <RiLightbulbLine className="text-muted h-2.5 w-2.5" />

                  {0}
                </div>
              </Pill>
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line text-left">
              Tags coming soon!
            </TooltipContent>
          </Tooltip>
        );
      },
      enableSorting: false,
    }),
    columnHelper.accessor('versions', {
      cell: () => {
        return (
          <Tooltip>
            <TooltipTrigger>
              <Button
                disabled={true}
                kind="secondary"
                appearance="outlined"
                size="small"
                label="View version"
                className="text-muted text-xs"
                onClick={(e) => {
                  e.stopPropagation();
                }}
              />
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line text-left">
              Version history coming soon!
            </TooltipContent>
          </Tooltip>
        );
      },
      enableSorting: false,
    }),
  ];

  if (data.debugRuns?.length === 0) {
    return <EmptyHistory />;
  }

  return (
    <div className="flex w-full flex-col justify-start gap-2">
      <Table
        noHeader={true}
        onRowClick={(row) =>
          row.original &&
          router.push(
            pathCreator.debugger({
              functionSlug,
              runID,
              debugSessionID: runID,
              debugRunID: row.original.debugRunID,
            })
          )
        }
        data={(data.debugRuns ?? []).sort(
          (a, b) =>
            (a?.startedAt ? new Date(a.startedAt).getTime() : 0) -
            (b?.startedAt ? new Date(b.startedAt).getTime() : 0)
        )}
        columns={columns}
      />
    </div>
  );
};
