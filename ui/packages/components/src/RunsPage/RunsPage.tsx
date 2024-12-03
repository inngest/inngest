'use client';

import { useCallback, useMemo, useRef, useState, type UIEventHandler } from 'react';
import dynamic from 'next/dynamic';
import { NewButton } from '@inngest/components/Button';
import StatusFilter from '@inngest/components/Filter/StatusFilter';
import TimeFieldFilter from '@inngest/components/Filter/TimeFieldFilter';
import { Pill } from '@inngest/components/Pill';
import { SelectGroup, type Option } from '@inngest/components/Select/Select';
import { TableFilter } from '@inngest/components/Table';
import { DEFAULT_TIME } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  FunctionRunTimeField,
  isFunctionRunStatus,
  isFunctionTimeField,
  type FunctionRunStatus,
} from '@inngest/components/types/functionRun';
import { cn } from '@inngest/components/utils/classNames';
import { durationToString, parseDuration } from '@inngest/components/utils/date';
import { RiArrowRightUpLine, RiRefreshLine, RiSearchLine } from '@remixicon/react';
import { type VisibilityState } from '@tanstack/react-table';
import { useLocalStorage } from 'react-use';

import type { RangeChangeProps } from '../DatePicker/RangePicker';
import EntityFilter from '../Filter/EntityFilter';
import { RunDetailsV2 } from '../RunDetailsV2';
import {
  useBatchedSearchParams,
  useSearchParam,
  useStringArraySearchParam,
  useValidatedArraySearchParam,
  useValidatedSearchParam,
} from '../hooks/useSearchParam';
import type { Features } from '../types/features';
import { TimeFilter } from './TimeFilter';
import { isColumnID, useScopedColumns, type ColumnID } from './columns';
import type { Run, ViewScope } from './types';

// Disable SSR in Runs Table, to prevent hydration errors. It requires windows info on visibility columns
const RunsTable = dynamic(() => import('@inngest/components/RunsPage/RunsTable'), {
  ssr: false,
});

const CodeSearch = dynamic(() => import('@inngest/components/CodeSearch/CodeSearch'), {
  ssr: false,
});

type Props = {
  cancelRun: React.ComponentProps<typeof RunDetailsV2>['cancelRun'];
  data: Run[];
  defaultVisibleColumns?: ColumnID[];
  features: Pick<Features, 'history'>;
  getRun: React.ComponentProps<typeof RunDetailsV2>['getRun'];
  getTraceResult: React.ComponentProps<typeof RunDetailsV2>['getResult'];
  getTrigger: React.ComponentProps<typeof RunDetailsV2>['getTrigger'];
  hasMore: boolean;
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  onRefresh?: () => void;
  onScroll: UIEventHandler<HTMLDivElement>;
  onScrollToTop: () => void;
  pathCreator: React.ComponentProps<typeof RunDetailsV2>['pathCreator'];
  pollInterval?: number;
  rerun: React.ComponentProps<typeof RunDetailsV2>['rerun'];
  rerunFromStep: React.ComponentProps<typeof RunDetailsV2>['rerunFromStep'];
  apps?: Option[];
  functions?: Option[];
  functionIsPaused?: boolean;
  scope: ViewScope;
  totalCount: number | undefined;
  temporaryAlert?: React.ReactElement;
  hasSearchFlag?: boolean;
  stepAIEnabled?: boolean;
};

export function RunsPage({
  cancelRun,
  defaultVisibleColumns,
  getRun,
  getTraceResult,
  getTrigger,
  rerun,
  rerunFromStep,
  data,
  features,
  hasMore,
  isLoadingInitial,
  isLoadingMore,
  onRefresh,
  onScroll,
  onScrollToTop,
  pathCreator,
  apps,
  functions,
  pollInterval,
  functionIsPaused,
  scope,
  totalCount,
  temporaryAlert,
  hasSearchFlag = false,
  stepAIEnabled = false,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const columns = useScopedColumns(scope);
  const [showSearch, setShowSearch] = useState(false);

  const displayAllColumns = useMemo(() => {
    const out: Record<string, boolean> = {};
    for (const column of columns) {
      if (!isColumnID(column.id)) {
        continue;
      }
      if (
        scope === 'env' &&
        (column.id === 'startedAt' || column.id === 'app' || column.id === 'durationMS')
      ) {
        out[column.id] = false;
        continue;
      }
      if (defaultVisibleColumns && !defaultVisibleColumns.includes(column.id)) {
        out[column.id] = false;
      } else {
        out[column.id] = true;
      }
    }
    return out;
  }, [defaultVisibleColumns, columns, scope]);

  const [columnVisibility, setColumnVisibility] = useLocalStorage<VisibilityState>(
    `VisibleRunsColumns-${scope}`,
    displayAllColumns
  );

  const [filteredStatus = [], setFilteredStatus, removeFilteredStatus] =
    useValidatedArraySearchParam('filterStatus', isFunctionRunStatus);

  const [filteredApp = [], setFilteredApp, removeFilteredApp] =
    useStringArraySearchParam('filterApp');

  const [filteredFunction = [], setFilteredFunction, removeFilteredFunction] =
    useStringArraySearchParam('filterFunction');

  const [timeField = FunctionRunTimeField.QueuedAt, setTimeField] = useValidatedSearchParam(
    'timeField',
    isFunctionTimeField
  );

  const [search, setSearch, removeSearch] = useSearchParam('search');

  const [lastDays] = useSearchParam('last');
  const [startTime] = useSearchParam('start');
  const [endTime] = useSearchParam('end');
  const batchUpdate = useBatchedSearchParams();

  const scrollToTop = useCallback(
    (smooth = false) => {
      if (containerRef.current) {
        containerRef.current.scrollTo({
          top: 0,
          behavior: smooth ? 'smooth' : 'auto',
        });
        onScrollToTop();
      }
    },
    [containerRef.current, onScrollToTop]
  );

  const onStatusFilterChange = useCallback(
    (value: FunctionRunStatus[]) => {
      scrollToTop();
      if (value.length > 0) {
        setFilteredStatus(value);
      } else {
        removeFilteredStatus();
      }
    },
    [removeFilteredStatus, scrollToTop, setFilteredStatus]
  );

  const onAppFilterChange = useCallback(
    (value: string[]) => {
      scrollToTop();
      if (value.length > 0) {
        setFilteredApp(value);
      } else {
        removeFilteredApp();
      }
    },
    [removeFilteredApp, scrollToTop, setFilteredApp]
  );

  const onFunctionFilterChange = useCallback(
    (value: string[]) => {
      scrollToTop();
      if (value.length > 0) {
        setFilteredFunction(value);
      } else {
        removeFilteredFunction();
      }
    },
    [removeFilteredFunction, scrollToTop, setFilteredFunction]
  );

  const onTimeFieldChange = useCallback(
    (value: FunctionRunTimeField) => {
      scrollToTop();
      setTimeField(value);
    },
    [scrollToTop, setTimeField]
  );

  const onDaysChange = useCallback(
    (value: RangeChangeProps) => {
      scrollToTop();
      if (value.type === 'relative') {
        batchUpdate({
          last: durationToString(value.duration),
          start: null,
          end: null,
        });
      } else {
        batchUpdate({
          last: null,
          start: value.start.toISOString(),
          end: value.end.toISOString(),
        });
      }
    },
    [batchUpdate, scrollToTop]
  );

  const onSearchChange = useCallback(
    (value: string) => {
      scrollToTop();
      if (value.length > 0) {
        setSearch(value);
      } else {
        removeSearch();
      }
    },
    [scrollToTop, setSearch]
  );

  const renderSubComponent = useCallback(
    (rowData: Run) => {
      return (
        <div className="border-subtle border-l-4 pb-6">
          <RunDetailsV2
            cancelRun={cancelRun}
            getResult={getTraceResult}
            getRun={getRun}
            initialRunData={rowData}
            getTrigger={getTrigger}
            pathCreator={pathCreator}
            pollInterval={pollInterval}
            rerun={rerun}
            rerunFromStep={rerunFromStep}
            runID={rowData.id}
            standalone={false}
            stepAIEnabled={stepAIEnabled}
          />
        </div>
      );
    },
    [cancelRun, getRun, getTraceResult, getTrigger, pathCreator, pollInterval, rerun]
  );

  const options = useMemo(() => {
    const out = [];
    for (const column of columns) {
      if (!isColumnID(column.id)) {
        continue;
      }

      out.push({
        id: column.id,
        name: column.header?.toString() || column.id,
      });
    }
    return out;
  }, [columns]);

  // Do not disable or show the button as loading if the poll interval is less than 1 second
  // Changing state too quickly can cause the button to flicker
  const disableRefreshButton =
    pollInterval && pollInterval < 1000 ? isLoadingInitial : isLoadingMore || isLoadingInitial;

  return (
    <main className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 flex flex-col">
        <div className="border-subtle flex h-[58px] items-center justify-between gap-2 border-b px-3">
          <div className="flex items-center gap-2">
            <SelectGroup>
              <TimeFieldFilter
                selectedTimeField={timeField}
                onTimeFieldChange={onTimeFieldChange}
              />
              <TimeFilter
                daysAgoMax={features.history}
                onDaysChange={onDaysChange}
                defaultValue={
                  lastDays
                    ? {
                        type: 'relative',
                        duration: parseDuration(lastDays),
                      }
                    : startTime && endTime
                    ? {
                        type: 'absolute',
                        start: new Date(startTime),
                        end: new Date(endTime),
                      }
                    : {
                        type: 'relative',
                        duration: parseDuration(DEFAULT_TIME),
                      }
                }
              />
            </SelectGroup>
            <StatusFilter
              selectedStatuses={filteredStatus}
              onStatusesChange={onStatusFilterChange}
              functionIsPaused={functionIsPaused}
            />
            {apps && (
              <EntityFilter
                type="app"
                onFilterChange={onAppFilterChange}
                selectedEntities={filteredApp}
                entities={apps}
              />
            )}
            {functions && (
              <EntityFilter
                type="function"
                onFilterChange={onFunctionFilterChange}
                selectedEntities={filteredFunction}
                entities={functions}
              />
            )}
            {hasSearchFlag && (
              <NewButton
                icon={<RiSearchLine />}
                iconSide="left"
                appearance="outlined"
                label={showSearch ? 'Hide search' : 'Show search'}
                onClick={() => setShowSearch((prev) => !prev)}
                className={cn(
                  'h-[42px]',
                  search
                    ? 'after:bg-secondary-moderate after:mb-3 after:ml-0.5 after:h-2 after:w-2 after:rounded'
                    : ''
                )}
              />
            )}
            <TotalCount totalCount={totalCount} />
          </div>
          <div className="flex items-center gap-2">
            <TableFilter
              columnVisibility={columnVisibility}
              setColumnVisibility={setColumnVisibility}
              options={options}
            />
          </div>
        </div>

        {hasSearchFlag && showSearch && (
          <>
            <div className="bg-codeEditor flex items-center justify-between px-4 pt-4">
              <div className="flex items-center gap-2">
                <p className="text-subtle text-sm">Search your runs by using a CEL query</p>
                <Pill kind="primary">Beta</Pill>
              </div>
              <NewButton
                appearance="outlined"
                label="Read the docs"
                icon={<RiArrowRightUpLine />}
                iconSide="right"
                size="small"
                href="https://www.inngest.com/docs/platform/monitor/inspecting-function-runs#searching-function-runs?ref=app-runs-search"
              />
            </div>
            <CodeSearch
              onSearch={onSearchChange}
              placeholder="event.data.userId == “1234” or output.count > 10"
              value={search}
            />
          </>
        )}
      </div>

      <div className="h-[calc(100%-58px)] overflow-y-auto" onScroll={onScroll} ref={containerRef}>
        <RunsTable
          data={data}
          isLoading={isLoadingInitial}
          renderSubComponent={renderSubComponent}
          getRowCanExpand={() => true}
          visibleColumns={columnVisibility}
          scope={scope}
        />
        {!hasMore && data.length > 1 && (
          <div className="flex flex-col items-center pt-8">
            <p className="text-muted">No additional runs found.</p>
            <NewButton
              label="Back to top"
              kind="primary"
              appearance="ghost"
              onClick={() => scrollToTop(true)}
            />
          </div>
        )}
        {onRefresh && (
          <div className="flex flex-col items-center pt-2">
            <NewButton
              kind="secondary"
              appearance="outlined"
              label="Refresh runs"
              icon={<RiRefreshLine />}
              iconSide="left"
              onClick={onRefresh}
              loading={disableRefreshButton}
              disabled={disableRefreshButton}
            />
          </div>
        )}
        {temporaryAlert}
      </div>
    </main>
  );
}

function TotalCount({
  className,
  totalCount,
}: {
  className?: string;
  totalCount: number | undefined;
}) {
  if (totalCount === undefined) {
    return null;
  }

  const formatted = new Intl.NumberFormat().format(totalCount);
  if (totalCount === 1) {
    return <span className={className}>{formatted} run</span>;
  }
  return <span className={className}>{formatted} runs</span>;
}
