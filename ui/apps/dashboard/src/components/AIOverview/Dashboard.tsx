import { useMemo, useState } from 'react';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import {
  useBatchedSearchParams,
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParams';
import { SelectGroup } from '@inngest/components/Select/Select';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
} from '@inngest/components/utils/date';
import { useRouterState } from '@tanstack/react-router';
import type { SortingState } from '@tanstack/react-table';
import { useQuery } from 'urql';

import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { CHART_COLORS } from '../InsightsMetrics/colors';
import { Section, SectionGroupHeading } from '../InsightsMetrics/Section';
import { AIOverviewEmptyState } from './EmptyState';
import {
  COST_TOOLTIP,
  formatCost,
  formatCostAxis,
  formatCostAxisBar,
  formatMs,
  formatSeconds,
  formatSecondsAxis,
  headlineCaveat,
  tokenBreakdown,
  withTotalTokens,
} from './utils';
import {
  renderFunctionLink,
  renderRunLink,
  renderSessionKeyLink,
  renderSessionLink,
} from './renderIdentifiers';
import { CategoricalChart } from '../InsightsMetrics/CategoricalChart';
import { ChartLegend } from '../InsightsMetrics/ChartLegend';
import { HeadlineStats } from '../InsightsMetrics/HeadlineStats';
import { RangePlot } from '../InsightsMetrics/RangePlot';
import { RankedTable } from '../InsightsMetrics/RankedTable';
import { ViewToggle, type ViewMode } from '../InsightsMetrics/ViewToggle';
import { TrendChart } from '../InsightsMetrics/TrendChart';
import { TREND_BUCKET_LIMIT } from '../InsightsMetrics/queries';
import { sortingToOrderBy, useInsightsMetric } from '../InsightsMetrics/useInsightsMetric';
import {
  sumValues,
  toListItems,
  toScalarValues,
  toTrendPoints,
  type TooltipExtra,
} from '../InsightsMetrics/types';

const DEFAULT_DURATION = { days: 7 };

const formatRuns = (value: number) => `${formatCompactNumber(value)} runs`;

// Shared by "Estimated cost over time" and "Estimated cost by model" — both plot ai_token_trend/
// ai_model_distribution's `cost` column, which carries input/output tokens
// alongside it in the same row (see InsightsMetrics/queries.ts), so the
// tooltip can show token counts without adding a visual series for them.
const TOKEN_TOOLTIP_EXTRAS: TooltipExtra[] = [
  { valueName: 'input_tokens', label: 'Input tokens', format: formatCompactNumber, defaultValue: 0 },
  { valueName: 'output_tokens', label: 'Output tokens', format: formatCompactNumber, defaultValue: 0 },
];

const FunctionLookupDocument = graphql(`
  query AIOverviewFunctionLookup($envSlug: String!, $page: Int, $pageSize: Int) {
    envBySlug(slug: $envSlug) {
      workflows @paginated(perPage: $pageSize, page: $page) {
        data {
          id
          name
          slug
        }
      }
    }
  }
`);

export const AIOverviewDashboard = ({ envSlug }: { envSlug: string }) => {
  const environment = useEnvironment();
  const workspaceID = environment.id;

  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const batchUpdate = useBatchedSearchParams();
  const [selectedFns, setFns, removeFns] = useStringArraySearchParam('fns');
  const [latencyByModelView, setLatencyByModelView] = useState<ViewMode>('chart');
  const [latencyByFunctionView, setLatencyByFunctionView] = useState<ViewMode>('chart');
  // Changing these re-issues the ai_latency_by_model/ai_latency_by_function
  // queries with a new orderBy (see useInsightsMetric below) — sorting the
  // table re-sorts the chart's rows too, since both views share one query.
  const [latencyByModelSort, setLatencyByModelSort] = useState<SortingState>([]);
  const [latencyByFunctionSort, setLatencyByFunctionSort] = useState<SortingState>([]);

  const parsedDuration = duration ? parseDuration(duration) : '';
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  // `loadedAt` bumps on router.invalidate(), so RefreshButton refires queries.
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

  const functionIDs = selectedFns?.length ? selectedFns : null;

  const [{ data: accountData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const daysAgoMax = accountData?.account.entitlements.history.limit ?? 7;

  // One InsightsMetric request per widget (see InsightsMetrics/queries.ts) —
  // topFunctionsByRuns/topFunctionsByCost omit functionIDs since they're
  // env-wide rankings, so they don't refetch when the function filter
  // changes.
  const headline = useInsightsMetric('ai_headline', { workspaceID, functionIDs, range: timeRange });
  const tokenTrend = useInsightsMetric('ai_token_trend', {
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
  const runVolumeTrend = useInsightsMetric('ai_run_volume_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const latencyByModel = useInsightsMetric('ai_latency_by_model', {
    workspaceID,
    functionIDs,
    range: timeRange,
    orderBy: sortingToOrderBy(latencyByModelSort),
  });
  const latencyByFunction = useInsightsMetric('ai_latency_by_function', {
    workspaceID,
    functionIDs,
    range: timeRange,
    orderBy: sortingToOrderBy(latencyByFunctionSort),
  });
  const topFunctionsByRuns = useInsightsMetric('ai_top_functions_by_runs', {
    workspaceID,
    range: timeRange,
    limit: 5,
  });
  const topFunctionsByCost = useInsightsMetric('ai_top_functions_by_cost', {
    workspaceID,
    range: timeRange,
    limit: 5,
  });
  const mostExpensiveRuns = useInsightsMetric('ai_most_expensive_runs', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
  });
  const mostExpensiveSteps = useInsightsMetric('ai_most_expensive_steps', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
  });
  const mostExpensiveSessions = useInsightsMetric('ai_most_expensive_sessions', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
  });
  const costPerRunTrend = useInsightsMetric('ai_avg_cost_per_run_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const slowRuns = useInsightsMetric('ai_slow_runs', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
  });

  const error = [
    headline,
    tokenTrend,
    modelDistribution,
    runsByModel,
    runVolumeTrend,
    latencyByModel,
    latencyByFunction,
    topFunctionsByRuns,
    topFunctionsByCost,
    mostExpensiveRuns,
    mostExpensiveSteps,
    mostExpensiveSessions,
    costPerRunTrend,
    slowRuns,
  ].some((m) => m.error);

  const [{ data: lookupData }] = useQuery({
    query: FunctionLookupDocument,
    variables: { envSlug, page: 1, pageSize: 1000 },
  });
  // Keyed by slug, not id: the backend's "identifier" column for these
  // rankings is `function_id AS identifier`, and the Insights transpiler's
  // output slug-translation (buildSlugOutputColumns in pkg/insights) turns
  // that UUID into the function's slug before it reaches the frontend — so
  // `identifier` here is already a slug, not the workflow's id.
  const functionsBySlug = useMemo(() => {
    const m = new Map<string, { name: string; slug: string }>();
    for (const wf of lookupData?.envBySlug?.workflows.data ?? []) {
      m.set(wf.slug, { name: wf.name, slug: wf.slug });
    }
    return m;
  }, [lookupData]);

  const isDefaultView = !start && !end && !duration && !selectedFns?.length;
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
    <div className="bg-canvasBase mx-auto flex h-full w-full max-w-[1500px] flex-col px-6">
      {showEmptyState && <AIOverviewEmptyState compact className="mt-3" />}
      {error && <Error message="There was an error loading the AI Overview." />}

      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto pb-6 [&::-webkit-scrollbar]:hidden">
        <div className="bg-canvasBase flex flex-row items-center gap-1.5 pb-3 pt-6">
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
          <EntityFilter
            type="function"
            onFilterChange={(fns) => (fns.length ? setFns(fns) : removeFns())}
            selectedEntities={selectedFns || []}
            entities={lookupData?.envBySlug?.workflows.data || []}
          />
        </div>
        <Section plain>
          <HeadlineStats
            values={withTotalTokens(toScalarValues(headline.data))}
            isLoading={headline.fetching && !headline.data}
            tiles={[
              { valueName: 'runs', label: 'AI runs', format: formatCompactNumber },
              {
                valueName: 'cost',
                label: 'Estimated Cost',
                format: formatCost,
                tooltip: headlineCaveat(toScalarValues(headline.data)),
              },
              {
                valueName: 'p95_latency',
                label: 'AI Call p95 Latency',
                format: (value) => formatSeconds(value / 1000),
              },
              {
                valueName: 'total_tokens',
                label: 'Total tokens',
                format: formatCompactNumber,
                tooltip: tokenBreakdown(toScalarValues(headline.data)),
              },
            ]}
          />
        </Section>

        <SectionGroupHeading>Usage</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section
            title="Runs over time"
            query={runVolumeTrend.data?.query}
            queryName="AI run volume over time"
          >
            <TrendChart
              points={toTrendPoints(runVolumeTrend.data)}
              isLoading={runVolumeTrend.fetching && !runVolumeTrend.data}
              hasData={hasAnyCalls}
              chartType="bar"
              series={[
                {
                  valueName: 'runs',
                  label: 'Runs',
                  color: CHART_COLORS[2],
                },
              ]}
              defaultValue={0}
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
              hasData={hasAnyCalls}
              chartType="bar"
              stacked
              legendIcon="rect"
              series={[
                {
                  valueName: 'input_tokens',
                  label: 'Input',
                  color: CHART_COLORS[2],
                },
                {
                  valueName: 'output_tokens',
                  label: 'Output',
                  color: CHART_COLORS[0],
                },
              ]}
              tooltipExtras={[{ valueName: 'cost', label: 'Cost', format: formatCost, defaultValue: 0 }]}
              defaultValue={0}
            />
          </Section>
          <Section
            title="Top functions by usage"
            query={topFunctionsByRuns.data?.query}
            queryName="AI top functions by usage"
            className="lg:col-span-2"
          >
            <div className="flex flex-col gap-4 lg:flex-row">
              <CategoricalChart
                items={toListItems(topFunctionsByRuns.data)}
                isLoading={topFunctionsByRuns.fetching && !topFunctionsByRuns.data}
                valueName="runs"
                valueLabel="Runs"
                colors={CHART_COLORS}
                format={formatRuns}
                formatIdentifier={(id) => functionsBySlug.get(id)?.name ?? id}
                showYAxisLine={false}
                className="min-w-0 lg:w-2/3"
              />
              <ChartLegend
                items={toListItems(topFunctionsByRuns.data)}
                isLoading={topFunctionsByRuns.fetching && !topFunctionsByRuns.data}
                valueName="runs"
                colors={CHART_COLORS}
                format={formatRuns}
                renderIdentifier={(id) => renderFunctionLink(id, envSlug, functionsBySlug)}
                className="w-full lg:w-1/3"
              />
            </div>
          </Section>
          <Section title="Runs by model" query={runsByModel.data?.query} queryName="AI runs by model">
            <CategoricalChart
              items={toListItems(runsByModel.data)}
              isLoading={runsByModel.fetching && !runsByModel.data}
              valueName="runs"
              valueLabel="Runs"
              color={CHART_COLORS[2]}
              format={formatCompactNumber}
              showYAxisLine={false}
              showValueLabels
            />
          </Section>
          <Section
            title="Tokens by model"
            query={modelDistribution.data?.query}
            queryName="AI tokens by model"
          >
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              series={[
                {
                  valueName: 'input_tokens',
                  label: 'Input',
                  color: CHART_COLORS[2],
                },
                {
                  valueName: 'output_tokens',
                  label: 'Output',
                  color: CHART_COLORS[0],
                },
              ]}
              stacked
              format={formatCompactNumber}
              showYAxisLine={false}
              showValueLabels
              tooltipExtras={[{ valueName: 'cost', label: 'Cost', format: formatCost, defaultValue: 0 }]}
            />
          </Section>
        </div>

        <SectionGroupHeading>Cost</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section
            title="Estimated cost over time"
            tooltip={COST_TOOLTIP}
            query={tokenTrend.data?.query}
            queryName="AI cost over time"
          >
            <TrendChart
              points={toTrendPoints(tokenTrend.data)}
              isLoading={tokenTrend.fetching && !tokenTrend.data}
              hasData={hasAnyCalls}
              chartType="bar"
              format={formatCost}
              axisFormat={formatCostAxisBar}
              allowDecimals
              series={[
                {
                  valueName: 'cost',
                  label: 'Cost',
                  color: CHART_COLORS[0],
                },
              ]}
              tooltipExtras={TOKEN_TOOLTIP_EXTRAS}
              defaultValue={0}
            />
          </Section>
          <Section
            title="Estimated cost per run over time"
            tooltip={COST_TOOLTIP}
            query={costPerRunTrend.data?.query}
            queryName="AI cost per run over time"
          >
            <TrendChart
              points={toTrendPoints(costPerRunTrend.data)}
              isLoading={costPerRunTrend.fetching && !costPerRunTrend.data}
              hasData={hasAnyCalls}
              format={formatCost}
              axisFormat={formatCostAxis}
              allowDecimals
              series={[{ valueName: 'avg_cost_per_run', label: 'Cost per run', color: CHART_COLORS[0] }]}
              defaultValue={0}
            />
          </Section>
          <Section
            title="Estimated cost by model"
            tooltip={COST_TOOLTIP}
            query={modelDistribution.data?.query}
            queryName="AI cost by model"
          >
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              valueName="cost"
              colors={CHART_COLORS}
              format={formatCost}
              valueLabelFormat={formatCostAxisBar}
              tooltipExtras={TOKEN_TOOLTIP_EXTRAS}
              showYAxisLine={false}
              showTooltipValueName={false}
              showValueLabels
            />
          </Section>
          <Section
            title="Estimated cost by function"
            tooltip={COST_TOOLTIP}
            query={topFunctionsByCost.data?.query}
            queryName="AI cost by function"
          >
            <CategoricalChart
              items={toListItems(topFunctionsByCost.data)}
              isLoading={topFunctionsByCost.fetching && !topFunctionsByCost.data}
              valueName="cost"
              colors={CHART_COLORS}
              format={formatCost}
              valueLabelFormat={formatCostAxisBar}
              formatIdentifier={(id) => functionsBySlug.get(id)?.name ?? id}
              showYAxisLine={false}
              showTooltipValueName={false}
              showValueLabels
              tooltipExtras={TOKEN_TOOLTIP_EXTRAS}
            />
          </Section>
          <Section
            title="Most expensive runs"
            tooltip={COST_TOOLTIP}
            query={mostExpensiveRuns.data?.query}
            queryName="AI most expensive runs"
          >
            <RankedTable
              headerStyle="subtle"
              density="compact"
              items={toListItems(mostExpensiveRuns.data)}
              isLoading={mostExpensiveRuns.fetching && !mostExpensiveRuns.data}
              identifierLabel="Run"
              renderIdentifier={(id) => renderRunLink(id, envSlug)}
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, envSlug, functionsBySlug),
              }}
              columns={[
                { valueName: 'cost', label: 'Cost', format: formatCost },
                { valueName: (item) => sumValues(item, ['input_tokens', 'output_tokens']), label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
          <Section
            title="Most expensive steps"
            tooltip={COST_TOOLTIP}
            query={mostExpensiveSteps.data?.query}
            queryName="AI most expensive steps"
          >
            <RankedTable
              headerStyle="subtle"
              density="compact"
              items={toListItems(mostExpensiveSteps.data)}
              isLoading={mostExpensiveSteps.fetching && !mostExpensiveSteps.data}
              identifierLabel="Step"
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, envSlug, functionsBySlug),
              }}
              columns={[
                { valueName: 'cost', label: 'Total cost', format: formatCost },
                { valueName: (item) => sumValues(item, ['input_tokens', 'output_tokens']), label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
          <Section
            title="Most expensive sessions"
            tooltip={COST_TOOLTIP}
            className="lg:col-span-2"
            query={mostExpensiveSessions.data?.query}
            queryName="AI most expensive sessions"
          >
            <RankedTable
              headerStyle="subtle"
              density="compact"
              items={toListItems(mostExpensiveSessions.data)}
              isLoading={mostExpensiveSessions.fetching && !mostExpensiveSessions.data}
              identifierLabel="Session"
              renderIdentifier={(id, item) => renderSessionLink(id, item.sessionKey ?? '', envSlug)}
              sessionKeyColumn={{
                label: 'Session key',
                render: (key) => renderSessionKeyLink(key, envSlug),
              }}
              columns={[
                { valueName: 'cost', label: 'Cost', format: formatCost },
                { valueName: (item) => sumValues(item, ['input_tokens', 'output_tokens']), label: 'Tokens used', format: formatCompactNumber },
                { valueName: 'runs', label: 'Runs', format: formatCompactNumber },
              ]}
            />
          </Section>
        </div>

        <SectionGroupHeading>Performance</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section
            title="AI Call Latency by function"
            query={latencyByFunction.data?.query}
            queryName="AI Call Latency by function"
          >
            <ViewToggle mode={latencyByFunctionView} onChange={setLatencyByFunctionView} />
            {latencyByFunctionView === 'chart' ? (
              <RangePlot
                items={toListItems(latencyByFunction.data)}
                isLoading={latencyByFunction.fetching && !latencyByFunction.data}
                format={formatSeconds}
                axisFormat={formatSecondsAxis}
                colors={CHART_COLORS}
                showYAxisLine={false}
                formatIdentifier={(id) => functionsBySlug.get(id)?.name ?? id}
              />
            ) : (
              <RankedTable
                headerStyle="subtle"
                density="compact"
                items={toListItems(latencyByFunction.data)}
                isLoading={latencyByFunction.fetching && !latencyByFunction.data}
                identifierLabel="Function"
                sortable
                sorting={latencyByFunctionSort}
                onSortingChange={setLatencyByFunctionSort}
                renderIdentifier={(id) => renderFunctionLink(id, envSlug, functionsBySlug)}
                columns={[
                  { valueName: 'min', label: 'Min', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'p50', label: 'p50', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'p95', label: 'p95', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'p99', label: 'p99', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'max', label: 'Max', format: (v) => formatSeconds(v / 1000) },
                ]}
              />
            )}
          </Section>
          <Section
            title="AI Call Latency by model"
            query={latencyByModel.data?.query}
            queryName="AI Call Latency by model"
          >
            <ViewToggle mode={latencyByModelView} onChange={setLatencyByModelView} />
            {latencyByModelView === 'chart' ? (
              <RangePlot
                items={toListItems(latencyByModel.data)}
                isLoading={latencyByModel.fetching && !latencyByModel.data}
                format={formatSeconds}
                axisFormat={formatSecondsAxis}
                colors={CHART_COLORS}
                showYAxisLine={false}
              />
            ) : (
              <RankedTable
                headerStyle="subtle"
                density="compact"
                items={toListItems(latencyByModel.data)}
                isLoading={latencyByModel.fetching && !latencyByModel.data}
                identifierLabel="Model"
                sortable
                sorting={latencyByModelSort}
                onSortingChange={setLatencyByModelSort}
                columns={[
                  { valueName: 'min', label: 'Min', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'p50', label: 'p50', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'p95', label: 'p95', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'p99', label: 'p99', format: (v) => formatSeconds(v / 1000) },
                  { valueName: 'max', label: 'Max', format: (v) => formatSeconds(v / 1000) },
                ]}
              />
            )}
          </Section>
          <Section
            title="Slowest runs"
            className="lg:col-span-2"
            query={slowRuns.data?.query}
            queryName="AI slowest runs"
          >
            <RankedTable
              headerStyle="subtle"
              density="compact"
              items={toListItems(slowRuns.data)}
              isLoading={slowRuns.fetching && !slowRuns.data}
              identifierLabel="Run"
              renderIdentifier={(id) => renderRunLink(id, envSlug)}
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, envSlug, functionsBySlug),
              }}
              columns={[
                { valueName: 'latency_ms', label: 'Total AI latency', format: formatMs },
                { valueName: (item) => sumValues(item, ['input_tokens', 'output_tokens']), label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
        </div>
      </div>
    </div>
  );
};
