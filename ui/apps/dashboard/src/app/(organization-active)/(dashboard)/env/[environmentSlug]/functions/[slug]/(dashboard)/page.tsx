'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { ChartBarIcon, ChevronRightIcon, XCircleIcon } from '@heroicons/react/20/solid';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { useCron } from '@inngest/components/hooks/useCron';
import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';
import { ErrorBoundary } from '@sentry/nextjs';
import { titleCase } from 'title-case';

import FunctionConfiguration from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/(dashboard)/FunctionConfiguration';
import type { TimeRange } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { Badge as LegacyBadge } from '@/components/Badge/Badge';
import Block from '@/components/Block';
import { Time } from '@/components/Time';
import LoadingIcon from '@/icons/LoadingIcon';
import { useFunction, useFunctionUsage } from '@/queries';
import { relativeTime } from '@/utils/date';
import { useSearchParam } from '@/utils/useSearchParam';
import DashboardTimeRangeFilter, {
  defaultTimeRange,
  getTimeRangeByKey,
} from './DashboardTimeRangeFilter';
import FunctionRunRateLimitChart from './FunctionRateLimitChart';
import FunctionRunsChart, { type UsageMetrics } from './FunctionRunsChart';
import FunctionThroughputChart from './FunctionThroughputChart';
import LatestFailedFunctionRuns from './LatestFailedFunctionRuns';
import SDKRequestThroughputChart from './SDKRequestThroughput';
import StepBacklogChart from './StepBacklogChart';
import StepsRunningChart from './StepsRunningChart';

type FunctionDashboardProps = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function FunctionDashboardPage({ params }: FunctionDashboardProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data, fetching: isFetchingFunction }] = useFunction({
    functionSlug,
  });
  const function_ = data?.workspace.workflow;

  const [timeRangeParam, setTimeRangeParam] = useSearchParam('timeRange');
  const selectedTimeRange: TimeRange =
    getTimeRangeByKey(timeRangeParam ?? '24h') ?? defaultTimeRange;

  const [{ data: usage }] = useFunctionUsage({
    functionSlug,
    timeRange: selectedTimeRange,
  });

  if (isFetchingFunction) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  if (!function_) {
    return (
      <>
        <div className="mt-4 flex place-content-center">
          <Alert severity="warning">Function not yet deployed to this environment</Alert>
        </div>
      </>
    );
  }

  const usageMetrics: UsageMetrics | undefined = usage?.reduce(
    (acc, u) => {
      acc.totalRuns += u.values.totalRuns;
      acc.totalFailures += u.values.failures;
      return acc;
    },
    {
      totalRuns: 0,
      totalFailures: 0,
    }
  );

  const failureRate = !usageMetrics?.totalRuns
    ? '0.00'
    : (((usageMetrics.totalFailures || 0) / (usageMetrics.totalRuns || 0)) * 100).toFixed(2);

  const triggers = function_.current?.triggers || [];

  function handleTimeRangeChange(timeRange: TimeRange) {
    if (timeRange.key) {
      setTimeRangeParam(timeRange.key);
    }
  }

  let appRoute = `/env/${params.environmentSlug}/apps/${function_.appName}` as Route;

  return (
    <>
      <div className="grid-cols-dashboard grid min-h-0 flex-1 bg-slate-100">
        <main className="col-span-3 overflow-y-auto">
          <div className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <div className="flex gap-14">
              <div className="inline-flex gap-3">
                <h3 className="inline-flex items-center gap-2 font-medium text-slate-600">
                  <ChartBarIcon className="h-5 text-indigo-500" />
                  Volume
                </h3>
                <span className="text-xl font-medium text-slate-800">
                  {usageMetrics?.totalRuns.toLocaleString(undefined, {
                    notation: 'compact',
                    compactDisplay: 'short',
                  })}
                </span>
              </div>
              <div className="inline-flex gap-3">
                <h3 className="inline-flex items-center gap-2 font-medium text-slate-600">
                  <XCircleIcon className="h-5 text-rose-500" /> Failure rate
                </h3>
                <span className="text-xl font-medium text-slate-800">{`${failureRate}%`}</span>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <LegacyBadge size="sm">Beta</LegacyBadge>
              <DashboardTimeRangeFilter
                selectedTimeRange={selectedTimeRange}
                onTimeRangeChange={handleTimeRangeChange}
              />
            </div>
          </div>
          <FunctionRunsChart functionSlug={functionSlug} timeRange={selectedTimeRange} />
          <FunctionThroughputChart functionSlug={functionSlug} timeRange={selectedTimeRange} />
          <StepsRunningChart functionSlug={functionSlug} timeRange={selectedTimeRange} />
          <StepBacklogChart functionSlug={functionSlug} timeRange={selectedTimeRange} />
          <SDKRequestThroughputChart functionSlug={functionSlug} timeRange={selectedTimeRange} />
          <FunctionRunRateLimitChart functionSlug={functionSlug} timeRange={selectedTimeRange} />
          <div className="my-4 px-6">
            <LatestFailedFunctionRuns
              environmentSlug={params.environmentSlug}
              functionSlug={functionSlug}
            />
          </div>
        </main>
        <aside className="overflow-y-auto border-l border-slate-200 bg-white px-6 py-4">
          <ErrorBoundary
            fallback={({ error, resetError }) => (
              <div className="flex items-center justify-center">
                <div className="rounded p-4">
                  <h2>Something went wrong!</h2>
                  <div className="my-6 overflow-scroll rounded bg-slate-200 p-2">
                    {error.toString()}
                  </div>
                  <Button
                    btnAction={
                      // Attempt to recover by trying to re-render the segment
                      () => resetError()
                    }
                    label="Try again"
                  />
                </div>
              </div>
            )}
          >
            <div className="flex flex-col gap-10">
              <Block title="App">
                <Link
                  href={appRoute}
                  className="shadow-outline-secondary-light block rounded bg-white p-4 hover:bg-slate-50"
                >
                  <div className="flex min-w-0 items-center">
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{function_.appName}</p>
                      {function_.current?.deploy?.createdAt && (
                        <Time
                          className="text-xs text-slate-500"
                          format="relative"
                          value={new Date(function_.current.deploy.createdAt)}
                        />
                      )}
                    </div>
                    <ChevronRightIcon className="h-5" />
                  </div>
                </Link>
              </Block>
              <Block title="Triggers">
                <div className="space-y-3">
                  {triggers.map((trigger) =>
                    trigger.eventName ? (
                      <Link
                        key={trigger.eventName}
                        href={`/env/${params.environmentSlug}/events/${encodeURIComponent(
                          trigger.eventName
                        )}`}
                        className="shadow-outline-secondary-light block rounded bg-white p-4 hover:bg-slate-50"
                      >
                        <div className="flex min-w-0 items-center">
                          <div className="min-w-0 flex-1 space-y-1">
                            <div className="flex min-w-0 items-center">
                              <IconEvent className="w-8 shrink-0 pr-2 text-indigo-500" />
                              <p className="truncate font-medium">{trigger.eventName}</p>
                            </div>
                            <dl className="text-xs">
                              {trigger.condition && (
                                <div className="flex gap-1">
                                  <dt className="text-slate-500">If</dt>
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <dd className="truncate font-mono text-slate-800">
                                        {trigger.condition}
                                      </dd>
                                    </TooltipTrigger>
                                    <TooltipContent className="font-mono text-xs">
                                      {trigger.condition}
                                    </TooltipContent>
                                  </Tooltip>
                                </div>
                              )}
                            </dl>
                          </div>
                          <ChevronRightIcon className="h-5" />
                        </div>
                      </Link>
                    ) : trigger.schedule ? (
                      <ScheduleTrigger
                        key={trigger.schedule}
                        schedule={trigger.schedule}
                        condition={trigger.condition}
                      />
                    ) : null
                  )}
                </div>
              </Block>
              {function_.configuration?.cancellations &&
                function_.configuration.cancellations.length > 0 && (
                  <Block title="Cancellation">
                    <div className="space-y-3">
                      {function_.configuration.cancellations.map((cancellation) => {
                        return (
                          <Link
                            key={cancellation.event}
                            href={`/env/${params.environmentSlug}/events/${encodeURIComponent(
                              cancellation.event
                            )}`}
                            className="shadow-outline-secondary-light block rounded bg-white p-4 hover:bg-slate-50"
                          >
                            <div className="flex min-w-0 items-center">
                              <div className="min-w-0 flex-1 space-y-1">
                                <div className="flex min-w-0 items-center">
                                  <IconEvent className="w-8 shrink-0 pr-2 text-indigo-500" />
                                  <p className="truncate font-medium">{cancellation.event}</p>
                                </div>
                                <dl className="text-xs">
                                  {cancellation.condition && (
                                    <div className="flex gap-1">
                                      <dt className="text-slate-500">If</dt>
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <dd className="truncate font-mono text-slate-800">
                                            {cancellation.condition}
                                          </dd>
                                        </TooltipTrigger>
                                        <TooltipContent className="font-mono text-xs">
                                          {cancellation.condition}
                                        </TooltipContent>
                                      </Tooltip>
                                    </div>
                                  )}
                                  {cancellation.timeout && (
                                    <div className="flex gap-1">
                                      <dt className="text-slate-500">Timeout</dt>
                                      <dd className="text-slate-800">{cancellation.timeout}</dd>
                                    </div>
                                  )}
                                </dl>
                              </div>
                              <ChevronRightIcon className="h-5" />
                            </div>
                          </Link>
                        );
                      })}
                    </div>
                  </Block>
                )}
              {function_.failureHandler && (
                <Block title="Failure Handler">
                  <div className="space-y-3">
                    <Link
                      href={`/env/${params.environmentSlug}/functions/${encodeURIComponent(
                        function_.failureHandler.slug
                      )}`}
                      className="shadow-outline-secondary-light block rounded bg-white p-4 hover:bg-slate-50"
                    >
                      <div className="flex min-w-0 items-center">
                        <div className="min-w-0 flex-1">
                          <div className="flex min-w-0 items-center">
                            <IconFunction className="w-8 shrink-0 pr-2 text-indigo-500" />
                            <p className="truncate font-medium">{function_.failureHandler.name}</p>
                          </div>
                        </div>
                        <ChevronRightIcon className="h-5" />
                      </div>
                    </Link>
                  </div>
                </Block>
              )}
              {function_.configuration && (
                <FunctionConfiguration configuration={function_.configuration} />
              )}
            </div>
          </ErrorBoundary>
        </aside>
      </div>
    </>
  );
}

type ScheduleTriggerProps = {
  schedule: string;
  condition: string | null;
};

function ScheduleTrigger({ schedule, condition }: ScheduleTriggerProps) {
  const { nextRun } = useCron(schedule);

  return (
    <div className="rounded border border-slate-200 bg-white p-4">
      <div className="flex min-w-0 items-center">
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex min-w-0 items-center">
            <IconClock className="w-8 shrink-0 pr-2 text-indigo-500" />
            <p className="truncate font-medium">{schedule}</p>
          </div>
          <dl className="text-xs">
            {condition && (
              <div className="flex gap-1">
                <dt className="text-slate-500">If</dt>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <dd className="truncate font-mono text-slate-800">{condition}</dd>
                  </TooltipTrigger>
                  <TooltipContent className="font-mono text-xs">{condition}</TooltipContent>
                </Tooltip>
              </div>
            )}
            {nextRun && (
              <div className="flex gap-1">
                <dt className="text-slate-500">Next Run</dt>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <dd className="truncate text-slate-800">{titleCase(relativeTime(nextRun))}</dd>
                  </TooltipTrigger>
                  <TooltipContent className="font-mono text-xs">
                    {nextRun.toISOString()}
                  </TooltipContent>
                </Tooltip>
              </div>
            )}
          </dl>
        </div>
      </div>
    </div>
  );
}
