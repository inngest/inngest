'use client';

import { useState } from 'react';
import { type Route } from 'next';
import { ChartBarIcon, FolderIcon, RocketLaunchIcon, XCircleIcon } from '@heroicons/react/20/solid';

import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { Alert } from '@/components/Alert';
import { Badge } from '@/components/Badge/Badge';
import Block from '@/components/Block';
import { ClientFeatureFlag } from '@/components/FeatureFlags/ClientFeatureFlag';
import ListContainer from '@/components/Lists/ListContainer';
import ListItem from '@/components/Lists/ListItem';
import LoadingIcon from '@/icons/LoadingIcon';
import EventIcon from '@/icons/event.svg';
import { useFunction, useFunctionUsage } from '@/queries';
import { relativeTime } from '@/utils/date';
import DashboardTimeRangeFilter, {
  defaultTimeRange,
  getTimeRangeLabel,
} from './DashboardTimeRangeFilter';
import FunctionRunsChart, { type UsageMetrics } from './FunctionRunsChart';
import FunctionThroughputChart from './FunctionThroughputChart';
import LatestFailedFunctionRuns from './LatestFailedFunctionRuns';

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
  const metrics = [
    {
      title: (
        <>
          <ChartBarIcon className="h-5 text-indigo-500" /> Volume
        </>
      ),
      description: `Total of started function runs in the last ${getTimeRangeLabel(
        selectedTimeRange
      )}`,
      value: usageMetrics?.totalRuns,
      unavailable: false,
    },
    {
      title: (
        <>
          <XCircleIcon className="h-5 text-red-500" /> Failure rate
        </>
      ),
      description: `Function runs failed in the last ${getTimeRangeLabel(selectedTimeRange)}`,
      value: `${failureRate}%`,
      unavailable: false,
    },
  ];

  function handleTimeRangeChange(timeRange: TimeRange) {
    setSelectedTimeRange(timeRange);
  }

  return (
    <>
      <div className="flex items-center gap-1.5 border-b border-slate-300 px-5 py-2">
        <DashboardTimeRangeFilter
          selectedTimeRange={selectedTimeRange}
          onTimeRangeChange={handleTimeRangeChange}
        />
      </div>
      <div className="grid-cols-dashboard grid min-h-0 flex-1 bg-slate-100">
        <main className="col-span-3 overflow-y-auto">
          <FunctionRunsChart
            environmentSlug={params.environmentSlug}
            functionSlug={functionSlug}
            timeRange={selectedTimeRange}
          />
          <ClientFeatureFlag flag="fn-metrics-chart">
            <FunctionThroughputChart
              environmentSlug={params.environmentSlug}
              functionSlug={functionSlug}
              timeRange={selectedTimeRange}
            />
          </ClientFeatureFlag>
          <div className={`mt-6 grid grid-cols-${metrics.length} gap-4 px-6`}>
            {metrics.map((m, idx) => (
              <Block key={idx}>
                <div className="relative rounded-lg bg-slate-900 p-4 text-white">
                  {m.unavailable && (
                    <div className="absolute inset-0 rounded-lg bg-slate-900 opacity-80" />
                  )}
                  {m.unavailable && (
                    <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-lg">
                      <Badge>Coming soon</Badge>
                    </div>
                  )}
                  <h3 className="mb-2 flex flex-row items-center gap-2 text-lg font-medium">
                    {m.title}
                  </h3>
                  <p className="text-sm text-slate-400">{m.description}</p>
                  <div className="mt-6 flex flex-row items-center justify-between">
                    <div className="text-2xl">{m.value}</div>
                  </div>
                </div>
              </Block>
            ))}
          </div>

          <div className="mt-4 px-6">
            <LatestFailedFunctionRuns
              environmentSlug={params.environmentSlug}
              functionSlug={functionSlug}
            />
          </div>
        </main>
        <aside className="border-l border-slate-200 bg-white px-6 py-4">
          <div className="flex flex-col gap-4 ">
            <Block title="Latest Version">
              {function_.current && (
                <ListContainer className="bg-white">
                  <ListItem
                    href={
                      `/env/${params.environmentSlug}/functions/${params.slug}/versions` as Route
                    }
                  >
                    <div>
                      <div className="mb-1 flex flex-row items-center gap-2 font-medium">
                        <FolderIcon className="h-4 text-indigo-400" /> {function_.current?.version}
                      </div>
                      <div className="text-xs text-slate-500">
                        {relativeTime(function_.current?.validFrom)}
                      </div>
                    </div>
                  </ListItem>
                </ListContainer>
              )}
            </Block>
            <Block title="Latest Deployment">
              <ListContainer className="bg-white">
                <ListItem
                  href={
                    `/env/${params.environmentSlug}/deploys/${function_.current?.deploy?.id}` as Route
                  }
                >
                  <div>
                    <div className="mb-1 flex flex-row items-center gap-2 font-medium">
                      <RocketLaunchIcon className="h-4 text-indigo-400" />{' '}
                      {function_.current?.deploy?.id}
                    </div>
                    <div className="text-xs text-slate-500">
                      {relativeTime(function_.current?.deploy?.createdAt)}
                    </div>
                  </div>
                </ListItem>
              </ListContainer>
            </Block>
            <Block title="Triggers">
              <ListContainer className="bg-white">
                {function_.current?.triggers?.map((t, idx) => (
                  <ListItem
                    key={idx}
                    href={
                      `/env/${params.environmentSlug}/events/${encodeURIComponent(
                        t.eventName ?? 'unknown'
                      )}` as Route
                    }
                  >
                    <div className="mb-1 flex flex-row items-center gap-2 font-medium">
                      <EventIcon className="h-3 text-indigo-400" /> {t.eventName || t.schedule}
                    </div>
                  </ListItem>
                ))}
              </ListContainer>
            </Block>

            <Block title="URL">
              <ListContainer className="bg-white">
                <ListItem>
                  <div className="mb-1 flex flex-row items-center gap-2 overflow-scroll whitespace-nowrap font-medium">
                    {function_.url}
                  </div>
                </ListItem>
              </ListContainer>
            </Block>
          </div>
        </aside>
      </div>
    </>
  );
}
