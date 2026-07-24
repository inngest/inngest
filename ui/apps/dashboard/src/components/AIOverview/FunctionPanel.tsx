import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import {
  useBatchedSearchParams,
  useSearchParam,
} from '@inngest/components/hooks/useSearchParams';
import { SelectGroup } from '@inngest/components/Select/Select';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
} from '@inngest/components/utils/date';
import { RiArrowRightUpLine } from '@remixicon/react';
import { useNavigate, useRouterState } from '@tanstack/react-router';
import { useQuery } from 'urql';

import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import { useEnvironment } from '@/components/Environments/environment-context';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { AIOverviewEmptyState } from './EmptyState';
import {
  formatCost,
  formatMs,
  formatSeconds,
  formatSecondsAxis,
  headlineCaveat,
  msPointsToSeconds,
} from './utils';
import { renderRunLink } from './renderIdentifiers';
import { CategoricalChart } from '../InsightsMetrics/CategoricalChart';
import { HeadlineStats } from '../InsightsMetrics/HeadlineStats';
import { RangePlot } from '../InsightsMetrics/RangePlot';
import { RankedTable } from '../InsightsMetrics/RankedTable';
import { TrendChart } from '../InsightsMetrics/TrendChart';
import { TREND_BUCKET_LIMIT } from '../InsightsMetrics/queries';
import { useInsightsMetric } from '../InsightsMetrics/useInsightsMetric';
import {
  toDimensionedTrendPoints,
  toListItems,
  toScalarValues,
  toTrendPoints,
} from '../InsightsMetrics/types';

const DEFAULT_DURATION = { days: 7 };

// FunctionAIPanel is the function-page AI tab — the same reusable display
// components as AIOverviewDashboard, scoped to one function and without the
// top-functions rankings (redundant when already scoped to a single
// function). Run/step-level lists (most expensive runs/steps, slow runs)
// stay, since a single function can still have many runs. Score averages
// aren't merged in yet — the partial-data blank-slate behavior (scores
// present, no gen_ai.* metadata) is still an open design question per the
// spec.
export const FunctionAIPanel = ({ functionID }: { functionID: string }) => {
  const environment = useEnvironment();
  const workspaceID = environment.id;
  const envSlug = environment.slug;

  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const batchUpdate = useBatchedSearchParams();

  const parsedDuration = duration ? parseDuration(duration) : '';
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  const loadedAt = useRouterState({ select: (s) => s.loadedAt });

  const range = useMemo(() => {
    if (parsedStart && parsedEnd) {
      return { from: parsedStart, to: parsedEnd };
    }
    const to = new Date();
    const dur = parsedDuration || DEFAULT_DURATION;
    return { from: subtractDuration(to, dur), to };
  }, [start, end, duration, loadedAt]);

  const timeRange = useMemo(
    () => ({ from: range.from.toISOString(), to: range.to.toISOString() }),
    [range],
  );

  const functionIDs = [functionID];

  const [{ data: accountData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const daysAgoMax = accountData?.account.entitlements.history.limit ?? 7;

  // One InsightsMetric request per widget (see InsightsMetrics/queries.ts).
  const headline = useInsightsMetric('ai_headline', { workspaceID, functionIDs, range: timeRange });
  const tokenTrend = useInsightsMetric('ai_token_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const avgCostPerRunTrend = useInsightsMetric('ai_avg_cost_per_run_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const tokenTrendByModel = useInsightsMetric('ai_token_trend_by_model', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const runVolumeTrend = useInsightsMetric('ai_run_volume_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const latencyTrend = useInsightsMetric('ai_latency_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const modelDistribution = useInsightsMetric('ai_model_distribution', {
    workspaceID,
    functionIDs,
    range: timeRange,
  });
  const runsByModel = useInsightsMetric('ai_runs_by_model', {
    workspaceID,
    functionIDs,
    range: timeRange,
  });
  const latencyByModel = useInsightsMetric('ai_latency_by_model', {
    workspaceID,
    functionIDs,
    range: timeRange,
  });
  const mostExpensiveRuns = useInsightsMetric('ai_most_expensive_runs', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 5,
  });
  const mostExpensiveSteps = useInsightsMetric('ai_most_expensive_steps', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 5,
  });
  const slowRuns = useInsightsMetric('ai_slow_runs', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 5,
  });

  const error = [
    headline,
    tokenTrend,
    avgCostPerRunTrend,
    tokenTrendByModel,
    runVolumeTrend,
    latencyTrend,
    modelDistribution,
    runsByModel,
    latencyByModel,
    mostExpensiveRuns,
    mostExpensiveSteps,
    slowRuns,
  ].some((m) => m.error);

  const latencyTrendPoints = useMemo(
    () => msPointsToSeconds(toTrendPoints(latencyTrend.data), ['p50', 'p95', 'p99']),
    [latencyTrend.data],
  );

  const isDefaultView = !start && !end && !duration;
  const hasAnyCalls = toScalarValues(headline.data).some(
    (v) => v.name === 'calls' && v.value > 0,
  );
  const showEmptyState =
    isDefaultView && !headline.fetching && !headline.error && headline.data && !hasAnyCalls;

  const defaultRange =
    parsedStart && parsedEnd
      ? { type: 'absolute' as const, start: parsedStart, end: parsedEnd }
      : {
          type: 'relative' as const,
          duration: parsedDuration || DEFAULT_DURATION,
        };

  return (
    <div className="bg-canvasBase mx-auto flex h-full w-full flex-col">
      <div className="bg-canvasBase flex flex-row items-center gap-1.5 px-3 py-[9px]">
        <SelectGroup>
          <span className="border-muted bg-modalBase text-muted box-content flex h-[24px] items-center rounded border px-1.5 text-[13px]">
            Time range
          </span>
          <TimeFilter
            className="rounded-l-none border-l-0"
            daysAgoMax={daysAgoMax}
            defaultValue={defaultRange}
            onDaysChange={(r: RangeChangeProps) => {
              batchUpdate({
                duration: r.type === 'relative' ? durationToString(r.duration) : null,
                start: r.type === 'absolute' ? r.start.toISOString() : null,
                end: r.type === 'absolute' ? r.end.toISOString() : null,
              });
            }}
          />
        </SelectGroup>
      </div>

      {error && <Error message="There was an error loading AI metrics for this function." />}

      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto px-3 pb-6 [&::-webkit-scrollbar]:hidden">
        {showEmptyState && <AIOverviewEmptyState compact className="mb-4 mt-3" />}
        <Section title="Overview" plain>
          <HeadlineStats
            values={toScalarValues(headline.data)}
            isLoading={headline.fetching && !headline.data}
            tiles={[
              { valueName: 'runs', label: 'AI runs', format: formatCompactNumber },
              { valueName: 'calls', label: 'AI calls', format: formatCompactNumber },
              {
                valueName: 'cost',
                label: 'Cost',
                format: formatCost,
                tooltip: headlineCaveat(toScalarValues(headline.data)),
              },
              {
                valueName: 'avg_cost_per_run',
                label: 'Avg cost / run',
                format: formatCost,
              },
              {
                valueName: 'input_tokens',
                label: 'input',
                groupLabel: 'Total tokens',
                format: formatCompactNumber,
                secondary: {
                  valueName: 'output_tokens',
                  label: 'output',
                  format: formatCompactNumber,
                },
              },
              {
                valueName: 'p95_latency',
                label: 'AI Call p95 Latency',
                format: (value) => formatSeconds(value / 1000),
              },
            ]}
          />
        </Section>

        <SectionGroupHeading>Usage</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section
            title="Run volume over time"
            query={runVolumeTrend.data?.query}
            queryName="AI run volume over time"
          >
            <TrendChart
              points={toTrendPoints(runVolumeTrend.data)}
              isLoading={runVolumeTrend.fetching && !runVolumeTrend.data}
              group="aiFunctionPanel"
              chartType="bar"
              series={[{ valueName: 'runs', label: 'Runs' }]}
            />
          </Section>
          <Section title="Runs by model" query={runsByModel.data?.query} queryName="AI runs by model">
            <CategoricalChart
              items={toListItems(runsByModel.data)}
              isLoading={runsByModel.fetching && !runsByModel.data}
              valueName="runs"
              valueLabel="Runs"
              format={formatCompactNumber}
              showValueLabels
            />
          </Section>
          <Section
            title="Tokens over time"
            query={tokenTrend.data?.query}
            queryName="AI tokens over time"
          >
            <TrendChart
              points={toTrendPoints(tokenTrend.data)}
              isLoading={tokenTrend.fetching && !tokenTrend.data}
              group="aiFunctionPanel"
              chartType="area"
              series={[
                { valueName: 'input_tokens', label: 'Input tokens' },
                { valueName: 'output_tokens', label: 'Output tokens' },
              ]}
            />
          </Section>
          <Section
            title="Tokens by model — input"
            query={modelDistribution.data?.query}
            queryName="AI tokens by model"
          >
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              valueName="input_tokens"
              valueLabel="Input tokens"
              format={formatCompactNumber}
            />
          </Section>
          <Section
            title="Tokens by model — output"
            query={modelDistribution.data?.query}
            queryName="AI tokens by model"
          >
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              valueName="output_tokens"
              valueLabel="Output tokens"
              format={formatCompactNumber}
            />
          </Section>
          <Section
            title="Tokens over time by model — input"
            query={tokenTrendByModel.data?.query}
            queryName="AI tokens over time by model"
          >
            <TrendChart
              points={toDimensionedTrendPoints(tokenTrendByModel.data)}
              isLoading={tokenTrendByModel.fetching && !tokenTrendByModel.data}
              group="aiFunctionPanel"
              chartType="bar"
              stacked
              valueName="input_tokens"
            />
          </Section>
          <Section
            title="Tokens over time by model — output"
            query={tokenTrendByModel.data?.query}
            queryName="AI tokens over time by model"
          >
            <TrendChart
              points={toDimensionedTrendPoints(tokenTrendByModel.data)}
              isLoading={tokenTrendByModel.fetching && !tokenTrendByModel.data}
              group="aiFunctionPanel"
              chartType="bar"
              stacked
              valueName="output_tokens"
            />
          </Section>
        </div>

        <SectionGroupHeading>Cost</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section title="Cost over time" query={tokenTrend.data?.query} queryName="AI cost over time">
            <TrendChart
              points={toTrendPoints(tokenTrend.data)}
              isLoading={tokenTrend.fetching && !tokenTrend.data}
              group="aiFunctionPanel"
              series={[{ valueName: 'cost', label: 'Cost' }]}
            />
          </Section>
          <Section
            title="Avg cost per run over time"
            query={avgCostPerRunTrend.data?.query}
            queryName="AI avg cost per run over time"
          >
            <TrendChart
              points={toTrendPoints(avgCostPerRunTrend.data)}
              isLoading={avgCostPerRunTrend.fetching && !avgCostPerRunTrend.data}
              group="aiFunctionPanel"
              series={[{ valueName: 'avg_cost_per_run', label: 'Avg cost / run' }]}
            />
          </Section>
          <Section title="Cost by model" query={modelDistribution.data?.query} queryName="AI cost by model">
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              valueName="cost"
              format={formatCost}
              showTooltipValueName={false}
            />
          </Section>
          <Section
            title="Cost over time by model"
            query={tokenTrendByModel.data?.query}
            queryName="AI cost over time by model"
          >
            <TrendChart
              points={toDimensionedTrendPoints(tokenTrendByModel.data)}
              isLoading={tokenTrendByModel.fetching && !tokenTrendByModel.data}
              group="aiFunctionPanel"
              valueName="cost"
            />
          </Section>
          <Section
            title="Most expensive runs"
            query={mostExpensiveRuns.data?.query}
            queryName="AI most expensive runs"
          >
            <RankedTable
              items={toListItems(mostExpensiveRuns.data)}
              isLoading={mostExpensiveRuns.fetching && !mostExpensiveRuns.data}
              identifierLabel="Run"
              renderIdentifier={(id) => renderRunLink(id, envSlug)}
              columns={[
                { valueName: 'cost', label: 'Cost', format: formatCost },
                { valueName: 'tokens', label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
          <Section
            title="Most expensive steps"
            query={mostExpensiveSteps.data?.query}
            queryName="AI most expensive steps"
          >
            <RankedTable
              items={toListItems(mostExpensiveSteps.data)}
              isLoading={mostExpensiveSteps.fetching && !mostExpensiveSteps.data}
              identifierLabel="Step"
              columns={[
                { valueName: 'cost', label: 'Total cost', format: formatCost },
                { valueName: 'tokens', label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
        </div>

        <SectionGroupHeading>Performance</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section
            title="Latency over time"
            query={latencyTrend.data?.query}
            queryName="AI latency over time"
          >
            <TrendChart
              points={latencyTrendPoints}
              isLoading={latencyTrend.fetching && !latencyTrend.data}
              group="aiFunctionPanel"
              format={formatSeconds}
              axisFormat={formatSecondsAxis}
              allowDecimals
              series={[
                { valueName: 'p50', label: 'p50' },
                { valueName: 'p95', label: 'p95' },
                { valueName: 'p99', label: 'p99' },
              ]}
            />
          </Section>
          <Section
            title="AI Call Latency by model"
            query={latencyByModel.data?.query}
            queryName="AI Call Latency by model"
          >
            <RangePlot
              items={toListItems(latencyByModel.data)}
              isLoading={latencyByModel.fetching && !latencyByModel.data}
              format={formatSeconds}
              axisFormat={formatSecondsAxis}
            />
          </Section>
          <Section
            title="Slowest runs"
            className="lg:col-span-2"
            query={slowRuns.data?.query}
            queryName="AI slowest runs"
          >
            <RankedTable
              items={toListItems(slowRuns.data)}
              isLoading={slowRuns.fetching && !slowRuns.data}
              identifierLabel="Run"
              renderIdentifier={(id) => renderRunLink(id, envSlug)}
              columns={[
                { valueName: 'latency_ms', label: 'Total AI latency', format: formatMs },
              ]}
            />
          </Section>
        </div>
      </div>
    </div>
  );
};

function SectionGroupHeading({ children }: { children: React.ReactNode }) {
  return <h2 className="text-basis mb-3 mt-6 text-base font-semibold">{children}</h2>;
}

function Section({
  title,
  className,
  children,
  query,
  queryName,
  plain = false,
}: {
  title: string;
  className?: string;
  children: React.ReactNode;
  // The exact Insights-dialect SQL that produced this card's data (the
  // insightsMetric result's `query` field) — when present, an "Open in
  // Insights" link opens that same query for the user to inspect/modify.
  query?: string;
  queryName?: string;
  // Skip the bordered/background card chrome — just the header row and
  // children, unboxed.
  plain?: boolean;
}) {
  const env = useEnvironment();
  const navigate = useNavigate();

  return (
    <section
      className={`mb-4 ${plain ? '' : 'border-subtle bg-canvasBase rounded-md border p-4'} ${className ?? ''}`}
    >
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 className="text-basis text-sm font-medium">{title}</h2>
        {query && (
          <Button
            size="small"
            kind="secondary"
            appearance="outlined"
            icon={<RiArrowRightUpLine />}
            iconSide="left"
            label="Open in Insights"
            onClick={() =>
              navigate({
                to: pathCreator.insights({ envSlug: env.slug }),
                search: { sql: query, name: queryName ?? title },
              })
            }
          />
        )}
      </div>
      {children}
    </section>
  );
}
