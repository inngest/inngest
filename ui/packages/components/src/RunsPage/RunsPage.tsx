import { useCallback, useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import TimeFieldFilter from '@inngest/components/Filter/TimeFieldFilter';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
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
import useLocalStorage from 'react-use/lib/useLocalStorage';

import CodeSearch from '../CodeSearch/CodeSearch';
import type { RangeChangeProps } from '../DatePicker/RangePicker';
import EntityFilter from '../Filter/EntityFilter';
import { RunDetailsV3 } from '../RunDetailsV3/RunDetailsV3';
import {
  useBatchedSearchParams,
  useSearchParam,
  useStringArraySearchParam,
  useValidatedArraySearchParam,
  useValidatedSearchParam,
} from '../hooks/useSearchParams';
import type { Features } from '../types/features';
import RunsStatusFilter from './RunsStatusFilter';
import RunsTable from './RunsTable';
import { isColumnID, useScopedColumns, type ColumnID } from './columns';
import type { Run, ViewScope } from './types';

type Props = {
  data: Run[];
  defaultVisibleColumns?: ColumnID[];
  features: Pick<Features, 'history' | 'tracesPreview'>;
  getTrigger: React.ComponentProps<typeof RunDetailsV3>['getTrigger'];
  hasMore: boolean;
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  onRefresh?: () => void;
  onScrollToTop: () => void;
  pollInterval?: number;
  apps?: Option[];
  functions?: Option[];
  functionIsPaused?: boolean;
  scope: ViewScope;
  totalCount: number | undefined;
  searchError?: Error;
  error?: Error | null;
  infiniteScrollTrigger?: (containerRef: HTMLDivElement | null) => React.ReactNode;
};

export function RunsPage({
  defaultVisibleColumns,
  getTrigger,
  data,
  features,
  hasMore,
  isLoadingInitial,
  isLoadingMore,
  onRefresh,
  onScrollToTop,
  apps,
  functions,
  pollInterval,
  functionIsPaused,
  scope,
  totalCount,
  searchError,
  error,
  infiniteScrollTrigger,
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
        <div className={`border-subtle `}>
          <RunDetailsV3
            initialRunData={rowData}
            getTrigger={getTrigger}
            pollInterval={pollInterval}
            runID={rowData.id}
            standalone={false}
            newStack={true}
          />
        </div>
      );
    },
    [getTrigger, pollInterval, features.tracesPreview]
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
    <main className="bg-canvasBase text-basis no-scrollbar flex flex-1 flex-col overflow-hidden focus-visible:outline-none">
      <div className="bg-canvasBase sticky top-0 z-10 flex flex-col">
        <div className="flex h-11 items-center justify-between gap-1.5 px-3">
          <div className="flex items-center gap-1.5">
            <Button
              icon={<RiSearchLine />}
              size="small"
              kind="secondary"
              iconSide="left"
              appearance="outlined"
              label={showSearch ? 'Hide search' : 'Show search'}
              onClick={() => setShowSearch((prev) => !prev)}
              className={cn(
                search
                  ? 'after:bg-secondary-moderate after:mb-3 after:ml-0.5 after:h-2 after:w-2 after:rounded'
                  : '',
                'h-[26px] w-[103px] rounded'
              )}
            />
            <SelectGroup>
              <TimeFieldFilter
                selectedTimeField={timeField}
                onTimeFieldChange={onTimeFieldChange}
              />
              <TimeFilter
                className="rounded-l-none border-l-0"
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
            <RunsStatusFilter
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
          </div>
          <div className="flex items-center gap-2">
            <TotalCount totalCount={totalCount} />
            <TableFilter
              columnVisibility={columnVisibility}
              setColumnVisibility={setColumnVisibility}
              options={options}
            />
          </div>
        </div>

        {showSearch && (
          <>
            <div className="bg-codeEditor flex items-center justify-between px-4 pt-4">
              <div className="flex items-center gap-2">
                <p className="text-subtle text-sm">Search your runs by using a CEL query</p>
                <Pill kind="primary">Beta</Pill>
              </div>
              <Button
                appearance="outlined"
                label="Read the docs"
                icon={<RiArrowRightUpLine />}
                iconSide="right"
                size="small"
                target="_blank"
                href="https://www.inngest.com/docs/platform/monitor/inspecting-function-runs#searching-function-runs?ref=app-runs-search"
              />
            </div>
            <CodeSearch
              onSearch={onSearchChange}
              placeholder="event.data.userId == “1234” or output.count > 10"
              value={search}
              searchError={searchError}
              preset="runs"
            />
          </>
        )}
      </div>

      <div className="flex-1 overflow-y-auto pb-2" ref={containerRef}>
        <RunsTable
          data={data}
          isLoading={isLoadingInitial}
          error={error}
          onRefresh={onRefresh}
          renderSubComponent={renderSubComponent}
          getRowCanExpand={() => true}
          visibleColumns={columnVisibility}
          scope={scope}
        />
        {infiniteScrollTrigger?.(containerRef.current)}
        {!hasMore && data.length > 1 && (
          <div className="flex flex-col items-center pt-8">
            <p className="text-muted">No additional runs found.</p>
            <Button
              label="Back to top"
              kind="primary"
              appearance="ghost"
              onClick={() => scrollToTop(true)}
            />
          </div>
        )}
        {onRefresh && !error && (
          <div className="flex flex-col items-center pt-2">
            <Button
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
    return (
      <span className={cn('text-muted text-xs font-semibold', className)}>{formatted} run</span>
    );
  }
  return (
    <span className={cn('text-muted text-xs font-semibold', className)}>{formatted} runs</span>
  );
}
