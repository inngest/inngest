import { useMemo, useState } from 'react';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Alert } from '@inngest/components/Alert';
import { Error } from '@inngest/components/Error/Error';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { useBatchedSearchParams, useSearchParam } from '@inngest/components/hooks/useSearchParams';
import { SelectGroup } from '@inngest/components/Select/Select';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
} from '@inngest/components/utils/date';
import type { SortingState } from '@tanstack/react-table';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { useFunction } from '@/queries/functions';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { CategoricalChart } from '@/components/InsightsMetrics/CategoricalChart';
import { CHART_COLORS } from '@/components/InsightsMetrics/colors';
import { HeadlineStats } from '@/components/InsightsMetrics/HeadlineStats';
import { RangePlot } from '@/components/InsightsMetrics/RangePlot';
import { RankedTable } from '@/components/InsightsMetrics/RankedTable';
import { Section, SectionGroupHeading } from '@/components/InsightsMetrics/Section';
import { TrendChart } from '@/components/InsightsMetrics/TrendChart';
import { TREND_BUCKET_LIMIT } from '@/components/InsightsMetrics/queries';
import {
  sumValues,
  toListItems,
  toScalarValues,
  toTrendPoints,
  type TooltipExtra,
} from '@/components/InsightsMetrics/types';
import { sortingToOrderBy, useInsightsMetric } from '@/components/InsightsMetrics/useInsightsMetric';
import { ViewToggle, type ViewMode } from '@/components/InsightsMetrics/ViewToggle';
import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import { useEnvironment } from '@/components/Environments/environment-context';
import {
  formatCost,
  formatCostAxis,
  formatMs,
  formatSeconds,
  formatSecondsAxis,
  headlineCaveat,
  tokenBreakdown,
  withTotalTokens,
} from '@/components/AIOverview/utils';
import { renderFunctionLink, renderRunLink } from '@/components/AIOverview/renderIdentifiers';

const DEFAULT_DURATION = { days: 7 };

// "Cost over time" gets twice the buckets of every other trend chart here —
// it's the sole full-width chart in the Cost section, so it has the
// horizontal room to render a finer-grained trend.
const COST_TREND_BUCKET_LIMIT = TREND_BUCKET_LIMIT * 2;

// Shared by "Cost over time" and "Cost by model" — both plot ai_token_trend/
// ai_model_distribution's `cost` column, which carries input/output tokens
// alongside it in the same row, so the tooltip can show token counts
// without adding a visual series for them.
const TOKEN_TOOLTIP_EXTRAS: TooltipExtra[] = [
  { valueName: 'input_tokens', label: 'Input tokens', format: formatCompactNumber, defaultValue: 0 },
  { valueName: 'output_tokens', label: 'Output tokens', format: formatCompactNumber, defaultValue: 0 },
];

type Props = {
  functionSlug: string;
};

// FunctionAI is a single-function-scoped cut of AIOverviewDashboard: every
// "by function" breakdown (top functions by usage/cost, latency by
// function) is meaningless when already scoped to one function, and the
// session-scoped sections (cost per session over time, most expensive
// sessions) are dropped here too — sessions can span several functions, so
// they read better at the env-wide level. Both are cut along with the
// function selector; everything else mirrors the env-wide dashboard.
export const FunctionAI = ({ functionSlug }: Props) => {
  const environment = useEnvironment();
  const workspaceID = environment.id;

  const [{ data, fetching: isFetchingFunction }] = useFunction({ functionSlug });
  const function_ = data?.workspace.workflow;
  const functionID = function_?.id;

  // A single-entry map (rather than an env-wide function lookup query) is
  // enough here — every renderFunctionLink call below resolves to this same
  // function.
  const functionsBySlug = useMemo(() => {
    if (!function_) return new Map<string, { name: string; slug: string }>();
    return new Map([[function_.slug, { name: function_.name, slug: function_.slug }]]);
  }, [function_]);

  const features = useAccountFeatures();
  const daysAgoMax = features.data?.history ?? 7;

  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const batchUpdate = useBatchedSearchParams();
  const [latencyByModelView, setLatencyByModelView] = useState<ViewMode>('chart');
  // Changing this re-issues the ai_latency_by_model query with a new orderBy
  // (see useInsightsMetric below) — sorting the table re-sorts the chart's
  // rows too, since both views share one query.
  const [latencyByModelSort, setLatencyByModelSort] = useState<SortingState>([]);

  const parsedDuration = duration ? parseDuration(duration) : '';
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  const range = useMemo(() => {
    if (parsedStart && parsedEnd) {
      return { from: parsedStart, to: parsedEnd };
    }
    const to = new Date();
    const dur = parsedDuration || DEFAULT_DURATION;
    return { from: subtractDuration(to, dur), to };
  }, [start, end, duration]);

  const timeRange = useMemo(
    () => ({ from: range.from.toISOString(), to: range.to.toISOString() }),
    [range],
  );

  const functionIDs = functionID ? [functionID] : null;
  // Every metric below pauses on `!functionID` — an unpaused call would fire
  // with an empty/null functionIDs list while the function lookup is still
  // resolving and get back env-wide (not "no data yet") rows.
  const pause = !functionID;

  const headline = useInsightsMetric('ai_headline', {
    workspaceID,
    functionIDs,
    range: timeRange,
    pause,
  });
  const tokenTrend = useInsightsMetric('ai_token_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
    pause,
  });
  // Separate from tokenTrend (same registry key, double the buckets) so
  // "Cost over time" can render at finer granularity than "Tokens over
  // time" without doubling the latter's bucket count too.
  const costTrend = useInsightsMetric('ai_token_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: COST_TREND_BUCKET_LIMIT,
    pause,
  });
  const modelDistribution = useInsightsMetric('ai_model_distribution', {
    workspaceID,
    functionIDs,
    range: timeRange,
    pause,
  });
  const runsByModel = useInsightsMetric('ai_runs_by_model', {
    workspaceID,
    functionIDs,
    range: timeRange,
    pause,
  });
  const runVolumeTrend = useInsightsMetric('ai_run_volume_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
    pause,
  });
  const latencyByModel = useInsightsMetric('ai_latency_by_model', {
    workspaceID,
    functionIDs,
    range: timeRange,
    orderBy: sortingToOrderBy(latencyByModelSort),
    pause,
  });
  const mostExpensiveRuns = useInsightsMetric('ai_most_expensive_runs', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
    pause,
  });
  const mostExpensiveSteps = useInsightsMetric('ai_most_expensive_steps', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
    pause,
  });
  const costPerRunTrend = useInsightsMetric('ai_avg_cost_per_run_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
    pause,
  });
  const slowRuns = useInsightsMetric('ai_slow_runs', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 10,
    pause,
  });

  // A non-bucketed aggregate, unaffected by the trend queries' backend
  // FILL (which zero-fills every bucket in range unconditionally, so
  // `points.length` alone can't tell "no data" apart from "zero-filled") —
  // used as the shared `hasData` signal for every trend chart below.
  const hasAnyCalls = toScalarValues(headline.data).some(
    (v) => v.name === 'calls' && v.value > 0,
  );

  const error = [
    headline,
    tokenTrend,
    costTrend,
    modelDistribution,
    runsByModel,
    runVolumeTrend,
    latencyByModel,
    mostExpensiveRuns,
    mostExpensiveSteps,
    costPerRunTrend,
    slowRuns,
  ].some((m) => m.error);

  const defaultRange =
    parsedStart && parsedEnd
      ? { type: 'absolute' as const, start: parsedStart, end: parsedEnd }
      : {
          type: 'relative' as const,
          duration: parsedDuration || DEFAULT_DURATION,
        };

  if (isFetchingFunction) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  if (!function_) {
    return (
      <div className="mt-4 flex place-content-center">
        <Alert severity="warning">Function not yet deployed to this environment</Alert>
      </div>
    );
  }

  return (
    <div className="bg-canvasBase mx-auto flex h-full w-full max-w-[1500px] flex-col px-6">
      {error && <Error message="There was an error loading this function's AI metrics." />}

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
            title="Cost over time"
            className="lg:col-span-2"
            query={costTrend.data?.query}
            queryName="AI cost over time"
          >
            <TrendChart
              points={toTrendPoints(costTrend.data)}
              isLoading={costTrend.fetching && !costTrend.data}
              hasData={hasAnyCalls}
              chartType="bar"
              format={formatCost}
              axisFormat={formatCostAxis}
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
            title="Cost per run over time"
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
          <Section title="Cost by model" query={modelDistribution.data?.query} queryName="AI cost by model">
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              valueName="cost"
              colors={CHART_COLORS}
              format={formatCost}
              tooltipExtras={TOKEN_TOOLTIP_EXTRAS}
              showYAxisLine={false}
              showTooltipValueName={false}
              showValueLabels
            />
          </Section>
          <Section
            title="Most expensive runs"
            query={mostExpensiveRuns.data?.query}
            queryName="AI most expensive runs"
          >
            <RankedTable
              headerStyle="subtle"
              density="compact"
              items={toListItems(mostExpensiveRuns.data)}
              isLoading={mostExpensiveRuns.fetching && !mostExpensiveRuns.data}
              identifierLabel="Run"
              renderIdentifier={(id) => renderRunLink(id, environment.slug)}
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, environment.slug, functionsBySlug),
              }}
              columns={[
                { valueName: 'cost', label: 'Cost', format: formatCost },
                { valueName: (item) => sumValues(item, ['input_tokens', 'output_tokens']), label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
          <Section
            title="Most expensive steps"
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
                render: (id) => renderFunctionLink(id, environment.slug, functionsBySlug),
              }}
              columns={[
                { valueName: 'cost', label: 'Total cost', format: formatCost },
                { valueName: (item) => sumValues(item, ['input_tokens', 'output_tokens']), label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
        </div>

        <SectionGroupHeading>Performance</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
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
            query={slowRuns.data?.query}
            queryName="AI slowest runs"
          >
            <RankedTable
              headerStyle="subtle"
              density="compact"
              items={toListItems(slowRuns.data)}
              isLoading={slowRuns.fetching && !slowRuns.data}
              identifierLabel="Run"
              renderIdentifier={(id) => renderRunLink(id, environment.slug)}
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, environment.slug, functionsBySlug),
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
