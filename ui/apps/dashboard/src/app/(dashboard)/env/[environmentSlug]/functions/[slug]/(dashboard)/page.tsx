'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ChartBarIcon, ChevronRightIcon, XCircleIcon } from '@heroicons/react/20/solid';
import { MetadataGrid } from '@inngest/components/Metadata';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';

import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { Alert } from '@/components/Alert';
import { Badge } from '@/components/Badge/Badge';
import Block from '@/components/Block';
import { ClientFeatureFlag } from '@/components/FeatureFlags/ClientFeatureFlag';
import { Time } from '@/components/Time';
import LoadingIcon from '@/icons/LoadingIcon';
import { useFunction, useFunctionUsage } from '@/queries';
import DashboardTimeRangeFilter, { defaultTimeRange } from './DashboardTimeRangeFilter';
import FunctionRunsChart, { type UsageMetrics } from './FunctionRunsChart';
import FunctionThroughputChart from './FunctionThroughputChart';
import LatestFailedFunctionRuns from './LatestFailedFunctionRuns';
import SDKRequestThroughputChart from './SDKRequestThroughput';

const functionConfig = {
  priority: "event.data.lastName == 'Doe' ? 120 : 0",
  concurrency: {
    scope: 'Function',
    limit: 1,
    key: 'event.data.userId',
  },
  rateLimit: {
    period: '24h0m0s',
    limit: 1,
    key: 'event.data.userId',
  },
  debounce: {
    period: '10s',
    key: 'event.data.userId',
  },
  eventsBatch: {
    maxSize: 100,
    timeout: '10s',
  },
  retries: 3,
  cancellations: [
    {
      event: 'app/user.deleted',
      timeout: '30m',
      condition: 'event.data.userId == async.data.userId',
    },
  ],
};

type FunctionDashboardProps = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function FunctionDashboardPage({ params }: FunctionDashboardProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data, fetching: isFetchingFunction }] = useFunction({
    environmentSlug: params.environmentSlug,
    functionSlug,
  });
  const function_ = data?.workspace.workflow;

  const [selectedTimeRange, setSelectedTimeRange] = useState<TimeRange>(defaultTimeRange);

  const [{ data: usage }] = useFunctionUsage({
    environmentSlug: params.environmentSlug,
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
    : (((usageMetrics?.totalFailures || 0) / (usageMetrics?.totalRuns || 0)) * 100).toFixed(2);

  const triggers = function_.current?.triggers || [];

  function handleTimeRangeChange(timeRange: TimeRange) {
    setSelectedTimeRange(timeRange);
  }

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
                  <XCircleIcon className="h-5 text-red-500" /> Failure rate
                </h3>
                <span className="text-xl font-medium text-slate-800">{`${failureRate}%`}</span>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <Badge size="sm">Beta</Badge>
              <DashboardTimeRangeFilter
                selectedTimeRange={selectedTimeRange}
                onTimeRangeChange={handleTimeRangeChange}
              />
            </div>
          </div>
          <FunctionRunsChart
            environmentSlug={params.environmentSlug}
            functionSlug={functionSlug}
            timeRange={selectedTimeRange}
          />
          <FunctionThroughputChart
            environmentSlug={params.environmentSlug}
            functionSlug={functionSlug}
            timeRange={selectedTimeRange}
          />
          <SDKRequestThroughputChart
            environmentSlug={params.environmentSlug}
            functionSlug={functionSlug}
            timeRange={selectedTimeRange}
          />
          <div className="mt-4 px-6">
            <LatestFailedFunctionRuns
              environmentSlug={params.environmentSlug}
              functionSlug={functionSlug}
            />
          </div>
        </main>
        <aside className="overflow-y-auto border-l border-slate-200 bg-white px-6 py-4">
          <div className="flex flex-col gap-10">
            <Block title="App">
              <Link
                href={`/env/${params.environmentSlug}/deploys/${function_.current?.deploy?.id}`}
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
                            {/*{trigger.condition && (*/}
                            {/*  <div className="flex gap-1">*/}
                            {/*    <dt className="text-slate-500">If</dt>*/}
                            {/*    <TooltipProvider>*/}
                            {/*      <Tooltip>*/}
                            {/*        <TooltipTrigger asChild>*/}
                            {/*          <dd className="truncate font-mono text-slate-800">*/}
                            {/*            {trigger.condition}*/}
                            {/*          </dd>*/}
                            {/*        </TooltipTrigger>*/}
                            {/*        <TooltipContent className="font-mono text-xs">*/}
                            {/*          {trigger.condition}*/}
                            {/*        </TooltipContent>*/}
                            {/*      </Tooltip>*/}
                            {/*    </TooltipProvider>*/}
                            {/*  </div>*/}
                            {/*)}*/}
                          </dl>
                        </div>
                        <ChevronRightIcon className="h-5" />
                      </div>
                    </Link>
                  ) : (
                    <div
                      key={trigger.schedule}
                      className="rounded border border-slate-200 bg-white p-4"
                    >
                      <div className="flex min-w-0 items-center">
                        <div className="min-w-0 flex-1 space-y-1">
                          <div className="flex min-w-0 items-center">
                            <IconClock className="w-8 shrink-0 pr-2 text-indigo-500" />
                            <p className="truncate font-medium">{trigger.schedule}</p>
                          </div>
                          <dl className="text-xs">
                            {/*{trigger.condition && (*/}
                            {/*  <div className="flex gap-1">*/}
                            {/*    <dt className="text-slate-500">If</dt>*/}
                            {/*    <TooltipProvider>*/}
                            {/*      <Tooltip>*/}
                            {/*        <TooltipTrigger asChild>*/}
                            {/*          <dd className="truncate font-mono text-slate-800">*/}
                            {/*            {trigger.condition}*/}
                            {/*          </dd>*/}
                            {/*        </TooltipTrigger>*/}
                            {/*        <TooltipContent className="font-mono text-xs">*/}
                            {/*          {trigger.condition}*/}
                            {/*        </TooltipContent>*/}
                            {/*      </Tooltip>*/}
                            {/*    </TooltipProvider>*/}
                            {/*  </div>*/}
                            {/*)}*/}
                            {/*{trigger.nextRun && (*/}
                            {/*  <div className="flex gap-1">*/}
                            {/*    <dt className="text-slate-500">Next Run</dt>*/}
                            {/*    <TooltipProvider>*/}
                            {/*      <Tooltip>*/}
                            {/*        <TooltipTrigger asChild>*/}
                            {/*          <dd className="truncate font-mono text-slate-800">*/}
                            {/*            {trigger.nextRun}*/}
                            {/*          </dd>*/}
                            {/*        </TooltipTrigger>*/}
                            {/*        <TooltipContent className="font-mono text-xs">*/}
                            {/*          {trigger.nextRun}*/}
                            {/*        </TooltipContent>*/}
                            {/*      </Tooltip>*/}
                            {/*    </TooltipProvider>*/}
                            {/*  </div>*/}
                            {/*)}*/}
                          </dl>
                        </div>
                      </div>
                    </div>
                  )
                )}
              </div>
            </Block>
            <ClientFeatureFlag flag="function-config">
              <Block title="Cancellation">
                <div className="space-y-3">
                  {functionConfig.cancellations.map((cancellation) => (
                    <Link
                      key={cancellation.event}
                      href={`/env/${params.environmentSlug}/events/${cancellation.event}`}
                      className="shadow-outline-secondary-light block rounded bg-white p-4 hover:bg-slate-50"
                    >
                      <div className="flex min-w-0 items-center">
                        <div className="min-w-0 flex-1 space-y-1">
                          <div className="flex min-w-0 items-center">
                            <IconEvent className="w-8 shrink-0 pr-2 text-indigo-500" />
                            <p className="truncate font-medium">{cancellation.event}</p>
                          </div>
                          <dl className="text-xs">
                            <div className="flex gap-1">
                              <dt className="text-slate-500">If</dt>
                              <TooltipProvider>
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
                              </TooltipProvider>
                            </div>
                            <div className="flex gap-1">
                              <dt className="text-slate-500">Timeout</dt>
                              <dd className="text-slate-800">{cancellation.timeout}</dd>
                            </div>
                          </dl>
                        </div>
                        <ChevronRightIcon className="h-5" />
                      </div>
                    </Link>
                  ))}
                </div>
              </Block>
              <Block title="Failure Handler">
                <div className="space-y-3">
                  <Link
                    href={`/env/${params.environmentSlug}/deploys/${function_.current?.deploy?.id}`}
                    className="shadow-outline-secondary-light block rounded bg-white p-4 hover:bg-slate-50"
                  >
                    <div className="flex min-w-0 items-center">
                      <div className="min-w-0 flex-1">
                        <div className="flex min-w-0 items-center">
                          <IconFunction className="w-8 shrink-0 pr-2 text-indigo-500" />
                          <p className="truncate font-medium">Failure: Customer Onboarding</p>
                        </div>
                      </div>
                      <ChevronRightIcon className="h-5" />
                    </div>
                  </Link>
                </div>
              </Block>
              <Block title="Configuration">
                <MetadataGrid
                  columns={2}
                  metadataItems={[
                    {
                      label: 'Priority',
                      value: `${functionConfig.priority}`,
                      size: 'large',
                      type: 'code',
                    },
                    { label: 'Retries', value: `${functionConfig.retries}` },
                  ]}
                />
                <h3 className="pb-2 pt-6 text-sm font-medium text-slate-800">Concurrency</h3>
                <MetadataGrid
                  columns={2}
                  metadataItems={[
                    {
                      label: 'Scope',
                      value: `${functionConfig.concurrency.scope}`,
                    },
                    { label: 'Limit', value: `${functionConfig.concurrency.limit}` },
                    {
                      label: 'Key',
                      value: `${functionConfig.concurrency.key}`,
                      type: 'code',
                      size: 'large',
                    },
                  ]}
                />
                <h3 className="pb-2 pt-6 text-sm font-medium text-slate-800">Rate Limit</h3>
                <MetadataGrid
                  columns={2}
                  metadataItems={[
                    {
                      label: 'Period',
                      value: `${functionConfig.rateLimit.period}`,
                    },
                    { label: 'Limit', value: `${functionConfig.rateLimit.limit}` },
                    {
                      label: 'Key',
                      value: `${functionConfig.rateLimit.key}`,
                      type: 'code',
                      size: 'large',
                    },
                  ]}
                />
                <h3 className="pb-2 pt-6 text-sm font-medium text-slate-800">Debounce</h3>
                <MetadataGrid
                  columns={2}
                  metadataItems={[
                    {
                      label: 'Period',
                      value: `${functionConfig.debounce.period}`,
                    },
                    {
                      label: 'Key',
                      value: `${functionConfig.debounce.key}`,
                      type: 'code',
                    },
                  ]}
                />
                <h3 className="pb-2 pt-6 text-sm font-medium text-slate-800">Events Batch</h3>
                <MetadataGrid
                  columns={2}
                  metadataItems={[
                    {
                      label: 'Max Size',
                      value: `${functionConfig.eventsBatch.maxSize}`,
                    },
                    { label: 'Timeout', value: `${functionConfig.eventsBatch.timeout}` },
                  ]}
                />
              </Block>
            </ClientFeatureFlag>
          </div>
        </aside>
      </div>
    </>
  );
}
