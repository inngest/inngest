'use client';

import { useCallback, useRef, type UIEventHandler } from 'react';
import { Button } from '@inngest/components/Button';
import StatusFilter from '@inngest/components/Filter/StatusFilter';
import TimeFieldFilter from '@inngest/components/Filter/TimeFieldFilter';
import RunsTable, { type Run } from '@inngest/components/RunsPage/RunsTable';
import { SelectGroup } from '@inngest/components/Select/Select';
import { LoadingMore } from '@inngest/components/Table';
import {
  FunctionRunTimeField,
  isFunctionRunStatus,
  isFunctionTimeField,
  type FunctionRunStatus,
} from '@inngest/components/types/functionRun';
import { RiLoopLeftLine } from '@remixicon/react';

import { RunDetails } from '../RunDetailsV2';
import {
  useSearchParam,
  useValidatedArraySearchParam,
  useValidatedSearchParam,
} from '../hooks/useSearchParam';
import type { Features } from '../types/features';
import { TimeFilter } from './TimeFilter';

type Props = {
  cancelRun: React.ComponentProps<typeof RunDetails>['cancelRun'];
  data: Run[];
  features: Pick<Features, 'history'>;
  functionSlug: string;
  getRun: React.ComponentProps<typeof RunDetails>['getRun'];
  getTraceResult: React.ComponentProps<typeof RunDetails>['getResult'];
  getTrigger: React.ComponentProps<typeof RunDetails>['getTrigger'];
  hasMore: boolean;
  isLoadingInitial: boolean;
  isLoadingMore: boolean;
  onScroll: UIEventHandler<HTMLDivElement>;
  onScrollToTop: () => void;
  pathCreator: React.ComponentProps<typeof RunDetails>['pathCreator'];
  rerun: React.ComponentProps<typeof RunDetails>['rerun'];
};

export function RunsPage({
  cancelRun,
  getRun,
  getTraceResult,
  getTrigger,
  rerun,
  data,
  features,
  hasMore,
  isLoadingInitial,
  isLoadingMore,
  onScroll,
  onScrollToTop,
  pathCreator,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);

  const [filteredStatus = [], setFilteredStatus, removeFilteredStatus] =
    useValidatedArraySearchParam('filterStatus', isFunctionRunStatus);

  const [timeField = FunctionRunTimeField.QueuedAt, setTimeField] = useValidatedSearchParam(
    'timeField',
    isFunctionTimeField
  );
  const [lastDays = '3', setLastDays] = useSearchParam('last');

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

  const onTimeFieldChange = useCallback(
    (value: FunctionRunTimeField) => {
      scrollToTop();
      setTimeField(value);
    },
    [scrollToTop, setTimeField]
  );

  const onDaysChange = useCallback(
    (value: string) => {
      scrollToTop();
      setLastDays(value);
    },
    [scrollToTop, setLastDays]
  );

  const renderSubComponent = useCallback(({ id }: { id: string }) => {
    return (
      <div className="border-subtle border-l-4 pb-6">
        <RunDetails
          cancelRun={cancelRun}
          getResult={getTraceResult}
          getRun={getRun}
          getTrigger={getTrigger}
          pathCreator={pathCreator}
          rerun={rerun}
          runID={id}
          standalone={false}
        />
      </div>
    );
  }, []);

  return (
    <main
      className="bg-canvasBase text-basis h-full min-h-0 overflow-y-auto"
      onScroll={onScroll}
      ref={containerRef}
    >
      <div className="bg-canvasBase sticky top-0 z-[5] flex items-center justify-between gap-2 px-8 py-2">
        <div className="flex items-center gap-2">
          <SelectGroup>
            <TimeFieldFilter selectedTimeField={timeField} onTimeFieldChange={onTimeFieldChange} />
            <TimeFilter
              daysAgoMax={features.history}
              onDaysChange={onDaysChange}
              selectedDays={lastDays}
            />
          </SelectGroup>
          <StatusFilter selectedStatuses={filteredStatus} onStatusesChange={onStatusFilterChange} />
        </div>
        {/* TODO: wire button */}
        <Button
          label="Refresh"
          appearance="text"
          btnAction={() => {}}
          icon={<RiLoopLeftLine />}
          disabled
        />
      </div>
      <RunsTable
        data={data}
        isLoading={isLoadingInitial}
        renderSubComponent={renderSubComponent}
        getRowCanExpand={() => true}
      />
      {isLoadingMore && <LoadingMore />}
      {!hasMore && (
        <div className="flex flex-col items-center py-8">
          <p className="text-subtle">No additional runs found.</p>
          <Button
            label="Back to top"
            kind="primary"
            appearance="text"
            btnAction={() => scrollToTop(true)}
          />
        </div>
      )}
    </main>
  );
}
