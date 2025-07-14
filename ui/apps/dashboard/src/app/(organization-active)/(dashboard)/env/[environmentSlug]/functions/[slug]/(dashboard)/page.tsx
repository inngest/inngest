'use client';

import type { Route } from 'next';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { FunctionConfiguration } from '@inngest/components/FunctionConfiguration';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { cn } from '@inngest/components/utils/classNames';
import { ErrorBoundary } from '@sentry/nextjs';

import type { TimeRange } from '@/types/TimeRangeFilter';
import LoadingIcon from '@/icons/LoadingIcon';
import { useFunction, useFunctionUsage } from '@/queries';
import { pathCreator } from '@/utils/urls';
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

  function handleTimeRangeChange(timeRange: TimeRange) {
    if (timeRange.key) {
      setTimeRangeParam(timeRange.key);
    }
  }

  let appRoute = `/env/${params.environmentSlug}/apps/${function_.app.name}` as Route;
  let billingUrl = pathCreator.billing({
    tab: 'plans',
    ref: 'concurrency-limit-popover',
  });

  return (
    <>
      <div className="grid-cols-dashboard bg-canvasSubtle grid min-h-0 flex-1">
        <main className="col-span-3 overflow-y-auto">
          <div className="border-subtle flex items-center justify-between border-b px-5 py-4">
            <div className="flex gap-14">
              <div className="inline-flex gap-3">
                <h3 className="text-subtle inline-flex items-center gap-2 font-medium">
                  Runs volume
                </h3>
                <span className="text-xl font-medium ">
                  {usageMetrics?.totalRuns.toLocaleString(undefined, {
                    notation: 'compact',
                    compactDisplay: 'short',
                  })}
                </span>
              </div>
              <div className="inline-flex gap-3">
                <h3 className="text-subtle inline-flex items-center gap-2 font-medium">
                  Failure rate
                </h3>
                <span
                  className={cn(
                    'text-xl font-medium',
                    failureRate === '0.00' ? 'text-subtle' : 'text-error'
                  )}
                >{`${failureRate}%`}</span>
              </div>
            </div>
            <div className="flex items-center gap-4">
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
        <aside className="border-subtle bg-canvasSubtle overflow-y-auto">
          <ErrorBoundary
            fallback={({ error, resetError }) => (
              <div className="flex items-center justify-center">
                <div className="rounded-md p-4">
                  <h2>Something went wrong!</h2>
                  <div className="bg-canvasBase my-6 overflow-auto rounded-md p-2">
                    {error.toString()}
                  </div>
                  <Button
                    onClick={
                      // Attempt to recover by trying to re-render the segment
                      () => resetError()
                    }
                    label="Try again"
                    kind="secondary"
                  />
                </div>
              </div>
            )}
          >
            <div className="bg-canvasBase h-full overflow-y-auto">
              <FunctionConfiguration
                inngestFunction={function_}
                deployCreatedAt={function_.current?.deploy?.createdAt}
                getAppLink={() => appRoute}
                getBillingUrl={() => billingUrl}
                getEventLink={(eventName) =>
                  pathCreator.eventType({
                    envSlug: params.environmentSlug,
                    eventName,
                  })
                }
                getFunctionLink={(functionSlug) =>
                  pathCreator.function({
                    envSlug: params.environmentSlug,
                    functionSlug,
                  })
                }
              />
            </div>
          </ErrorBoundary>
        </aside>
      </div>
    </>
  );
}
