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
import { InsightsFunctionMetricsDocument, TREND_BUCKET_LIMIT } from '../InsightsMetrics/queries';
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

  const [{ data: accountData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const daysAgoMax = accountData?.account.entitlements.history.limit ?? 7;

  const [{ data, fetching, error }] = useQuery({
    query: InsightsFunctionMetricsDocument,
    variables: {
      workspaceID,
      functionIDs: [functionID],
      range: timeRange,
      trendBucketLimit: TREND_BUCKET_LIMIT,
    },
  });

  const headline = data?.headline;
  const tokenTrend = data?.tokenTrend;
  const avgCostPerRunTrend = data?.avgCostPerRunTrend;
  const tokenTrendByModel = data?.tokenTrendByModel;
  const runVolumeTrend = data?.runVolumeTrend;
  const latencyTrend = data?.latencyTrend;
  const latencyTrendPoints = useMemo(
    () => msPointsToSeconds(toTrendPoints(latencyTrend), ['p50', 'p95', 'p99']),
    [latencyTrend],
  );
  const modelDistribution = data?.modelDistribution;
  const runsByModel = data?.runsByModel;
  const latencyByModel = data?.latencyByModel;
  const mostExpensiveRuns = data?.mostExpensiveRuns;
  const mostExpensiveSteps = data?.mostExpensiveSteps;
  const slowRuns = data?.slowRuns;

  const isDefaultView = !start && !end && !duration;
  const hasAnyCalls = toScalarValues(headline).some(
    (v) => v.name === 'calls' && v.value > 0,
  );
  const showEmptyState =
    isDefaultView && !fetching && !error && headline && !hasAnyCalls;

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

      <div id="chart-tooltip" className="z-[1000]" />

      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto px-3 pb-6 [&::-webkit-scrollbar]:hidden">
        {showEmptyState && <AIOverviewEmptyState compact className="mb-4 mt-3" />}
        <Section title="Overview" plain>
          <HeadlineStats
            values={toScalarValues(headline)}
            isLoading={fetching && !headline}
            tiles={[
              { valueName: 'runs', label: 'AI runs', format: formatCompactNumber },
              { valueName: 'calls', label: 'AI calls', format: formatCompactNumber },
              {
                valueName: 'cost',
                label: 'Cost',
                format: formatCost,
                tooltip: headlineCaveat(toScalarValues(headline)),
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
            query={runVolumeTrend?.query}
            queryName="AI run volume over time"
          >
            <TrendChart
              points={toTrendPoints(runVolumeTrend)}
              isLoading={fetching && !runVolumeTrend}
              group="aiFunctionPanel"
              chartType="bar"
              series={[{ valueName: 'runs', label: 'Runs' }]}
            />
          </Section>
          <Section title="Runs by model" query={runsByModel?.query} queryName="AI runs by model">
            <CategoricalChart
              items={toListItems(runsByModel)}
              isLoading={fetching && !runsByModel}
              valueName="runs"
              valueLabel="Runs"
              format={formatCompactNumber}
              showValueLabels
            />
          </Section>
          <Section
            title="Tokens over time"
            query={tokenTrend?.query}
            queryName="AI tokens over time"
          >
            <TrendChart
              points={toTrendPoints(tokenTrend)}
              isLoading={fetching && !tokenTrend}
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
            query={modelDistribution?.query}
            queryName="AI tokens by model"
          >
            <CategoricalChart
              items={toListItems(modelDistribution)}
              isLoading={fetching && !modelDistribution}
              valueName="input_tokens"
              valueLabel="Input tokens"
              format={formatCompactNumber}
            />
          </Section>
          <Section
            title="Tokens by model — output"
            query={modelDistribution?.query}
            queryName="AI tokens by model"
          >
            <CategoricalChart
              items={toListItems(modelDistribution)}
              isLoading={fetching && !modelDistribution}
              valueName="output_tokens"
              valueLabel="Output tokens"
              format={formatCompactNumber}
            />
          </Section>
          <Section
            title="Tokens over time by model — input"
            query={tokenTrendByModel?.query}
            queryName="AI tokens over time by model"
          >
            <TrendChart
              points={toDimensionedTrendPoints(tokenTrendByModel)}
              isLoading={fetching && !tokenTrendByModel}
              group="aiFunctionPanel"
              chartType="bar"
              stacked
              valueName="input_tokens"
            />
          </Section>
          <Section
            title="Tokens over time by model — output"
            query={tokenTrendByModel?.query}
            queryName="AI tokens over time by model"
          >
            <TrendChart
              points={toDimensionedTrendPoints(tokenTrendByModel)}
              isLoading={fetching && !tokenTrendByModel}
              group="aiFunctionPanel"
              chartType="bar"
              stacked
              valueName="output_tokens"
            />
          </Section>
        </div>

        <SectionGroupHeading>Cost</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section title="Cost over time" query={tokenTrend?.query} queryName="AI cost over time">
            <TrendChart
              points={toTrendPoints(tokenTrend)}
              isLoading={fetching && !tokenTrend}
              group="aiFunctionPanel"
              series={[{ valueName: 'cost', label: 'Cost' }]}
            />
          </Section>
          <Section
            title="Avg cost per run over time"
            query={avgCostPerRunTrend?.query}
            queryName="AI avg cost per run over time"
          >
            <TrendChart
              points={toTrendPoints(avgCostPerRunTrend)}
              isLoading={fetching && !avgCostPerRunTrend}
              group="aiFunctionPanel"
              series={[{ valueName: 'avg_cost_per_run', label: 'Avg cost / run' }]}
            />
          </Section>
          <Section title="Cost by model" query={modelDistribution?.query} queryName="AI cost by model">
            <CategoricalChart
              items={toListItems(modelDistribution)}
              isLoading={fetching && !modelDistribution}
              valueName="cost"
              format={formatCost}
              showTooltipValueName={false}
            />
          </Section>
          <Section
            title="Cost over time by model"
            query={tokenTrendByModel?.query}
            queryName="AI cost over time by model"
          >
            <TrendChart
              points={toDimensionedTrendPoints(tokenTrendByModel)}
              isLoading={fetching && !tokenTrendByModel}
              group="aiFunctionPanel"
              valueName="cost"
            />
          </Section>
          <Section
            title="Most expensive runs"
            query={mostExpensiveRuns?.query}
            queryName="AI most expensive runs"
          >
            <RankedTable
              items={toListItems(mostExpensiveRuns)}
              isLoading={fetching && !mostExpensiveRuns}
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
            query={mostExpensiveSteps?.query}
            queryName="AI most expensive steps"
          >
            <RankedTable
              items={toListItems(mostExpensiveSteps)}
              isLoading={fetching && !mostExpensiveSteps}
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
            query={latencyTrend?.query}
            queryName="AI latency over time"
          >
            <TrendChart
              points={latencyTrendPoints}
              isLoading={fetching && !latencyTrendPoints}
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
            query={latencyByModel?.query}
            queryName="AI Call Latency by model"
          >
            <RangePlot
              items={toListItems(latencyByModel)}
              isLoading={fetching && !latencyByModel}
              format={formatSeconds}
              axisFormat={formatSecondsAxis}
            />
          </Section>
          <Section
            title="Slowest runs"
            className="lg:col-span-2"
            query={slowRuns?.query}
            queryName="AI slowest runs"
          >
            <RankedTable
              items={toListItems(slowRuns)}
              isLoading={fetching && !slowRuns}
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
