import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
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
import { RiArrowRightUpLine } from '@remixicon/react';
import { useNavigate, useRouterState } from '@tanstack/react-router';
import { useQuery } from 'urql';

import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import { lineColors } from '@/components/Metrics/utils';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { colors } from '@/utils/tailwind';
import { pathCreator } from '@/utils/urls';
import { AIOverviewEmptyState } from './EmptyState';
import {
  formatCost,
  formatCostAxis,
  formatMs,
  formatSeconds,
  formatSecondsAxis,
  headlineCaveat,
  msPointsToSeconds,
} from './utils';
import { renderRunLink, renderSessionKeyLink, renderSessionLink } from './renderIdentifiers';
import { CategoricalChart } from '../InsightsMetrics/CategoricalChart';
import { ChartLegend } from '../InsightsMetrics/ChartLegend';
import { HeadlineStats } from '../InsightsMetrics/HeadlineStats';
import { RangePlot } from '../InsightsMetrics/RangePlot';
import { RankedTable } from '../InsightsMetrics/RankedTable';
import { TrendChart } from '../InsightsMetrics/TrendChart';
import { TREND_BUCKET_LIMIT } from '../InsightsMetrics/queries';
import { useInsightsMetric } from '../InsightsMetrics/useInsightsMetric';
import {
  toListItems,
  toScalarValues,
  toTrendPoints,
} from '../InsightsMetrics/types';

const DEFAULT_DURATION = { days: 7 };

const formatRuns = (value: number) => `${formatCompactNumber(value)} runs`;

// extendedColors casts in the design-system tiers this file reaches for
// beyond DefaultColors — see @/utils/tailwind's `colors` export.
const extendedColors = colors as typeof colors & {
  primary: { subtle: string; '2xSubtle': string; '3xSubtle': string };
  secondary: { subtle: string; '3xSubtle': string };
  quaternary: { warmerxSubtle: string; coolxSubtle: string };
};

// Fixed-order pastel palette (green, blue, yellow, orange, purple), matched
// against a reference mock — used both for per-category color (top
// functions by usage) and as the single-hue override for individual charts
// (e.g. green for runs). Reuses the design system's "subtle" tier tokens;
// yellow has no dedicated categorical slot, so it references the honey
// scale's step 300 directly via a CSS var (verified against the live app —
// the warning/honey *semantic* tokens render as a burnt orange-rust in
// light mode, not yellow).
const CHART_COLORS: readonly (readonly [string, string])[] = [
  [extendedColors.primary.subtle, '#66bd8b'], // green
  [extendedColors.secondary.subtle, '#52b2fd'], // blue
  ['var(--color-honey-300)', '#fcc43f'], // yellow
  [extendedColors.quaternary.warmerxSubtle, '#ffae7f'], // orange
  [extendedColors.quaternary.coolxSubtle, '#cec6fd'], // purple
];

// 3xSubtle blue/green tuple for the Tokens over time area fill — the design
// system's most muted tier of the same secondary/primary hues lineColors'
// "moderate" tier uses.
const SUBTLE_BLUE: readonly [string, string] = [extendedColors.secondary['3xSubtle'], '#e3f0fd'];
const SUBTLE_GREEN: readonly [string, string] = [extendedColors.primary['3xSubtle'], '#e2f3ea'];

// Tokens by model's stacked bars — matched pixel-for-pixel against a
// reference mock: blue reuses the same secondary 3xSubtle tier as Tokens
// over time, but green needed one tier down (2xSubtle, not 3xSubtle) to
// match — the two charts aren't a matched pair here.
const TOKENS_BY_MODEL_GREEN: readonly [string, string] = [
  extendedColors.primary['2xSubtle'],
  '#c4efd4',
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
  });
  const latencyTrend = useInsightsMetric('ai_latency_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
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
    limit: 5,
  });
  const mostExpensiveSteps = useInsightsMetric('ai_most_expensive_steps', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 5,
  });
  const mostExpensiveSessions = useInsightsMetric('ai_most_expensive_sessions', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: 5,
  });
  const costPerRunTrend = useInsightsMetric('ai_avg_cost_per_run_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
  });
  const costPerSessionTrend = useInsightsMetric('ai_cost_per_session_trend', {
    workspaceID,
    functionIDs,
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
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
    modelDistribution,
    runsByModel,
    runVolumeTrend,
    latencyByModel,
    latencyTrend,
    topFunctionsByRuns,
    topFunctionsByCost,
    mostExpensiveRuns,
    mostExpensiveSteps,
    mostExpensiveSessions,
    costPerRunTrend,
    costPerSessionTrend,
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

  const latencyTrendPoints = useMemo(
    () => msPointsToSeconds(toTrendPoints(latencyTrend.data), ['p50', 'p95', 'p99']),
    [latencyTrend.data],
  );

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
    <div className="bg-canvasBase mx-auto flex h-full w-full flex-col">
      {showEmptyState && <AIOverviewEmptyState compact className="mx-3 mt-3" />}
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
        <EntityFilter
          type="function"
          onFilterChange={(fns) => (fns.length ? setFns(fns) : removeFns())}
          selectedEntities={selectedFns || []}
          entities={lookupData?.envBySlug?.workflows.data || []}
        />
      </div>

      {error && <Error message="There was an error loading the AI Overview." />}

      <div id="chart-tooltip" className="z-[1000]" />

      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto px-3 pb-6 [&::-webkit-scrollbar]:hidden">
        <Section plain>
          <HeadlineStats
            values={toScalarValues(headline.data)}
            isLoading={headline.fetching && !headline.data}
            tiles={[
              { valueName: 'runs', label: 'AI runs', format: formatCompactNumber },
              {
                valueName: 'cost',
                label: 'Cost',
                format: formatCost,
                tooltip: headlineCaveat(toScalarValues(headline.data)),
              },
              {
                valueName: 'p95_latency',
                label: 'AI Call p95 Latency',
                format: (value) => formatSeconds(value / 1000),
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
              chartType="bar"
              series={[{ valueName: 'runs', label: 'Runs', color: CHART_COLORS[0] }]}
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
              chartType="area"
              stacked
              legendIcon="rect"
              series={[
                {
                  valueName: 'input_tokens',
                  label: 'Input',
                  color: lineColors[2],
                  areaColor: SUBTLE_BLUE,
                },
                {
                  valueName: 'output_tokens',
                  label: 'Output',
                  color: lineColors[1],
                  areaColor: SUBTLE_GREEN,
                },
              ]}
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
              color={CHART_COLORS[0]}
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
                  color: SUBTLE_BLUE,
                  borderColor: lineColors[2],
                },
                {
                  valueName: 'output_tokens',
                  label: 'Output',
                  color: TOKENS_BY_MODEL_GREEN,
                  borderColor: lineColors[1],
                },
              ]}
              stacked
              format={formatCompactNumber}
              showYAxisLine={false}
            />
          </Section>
        </div>

        <SectionGroupHeading>Cost</SectionGroupHeading>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Section title="Cost over time" query={tokenTrend.data?.query} queryName="AI cost over time">
            <TrendChart
              points={toTrendPoints(tokenTrend.data)}
              isLoading={tokenTrend.fetching && !tokenTrend.data}
              chartType="bar"
              format={formatCost}
              axisFormat={formatCostAxis}
              allowDecimals
              series={[{ valueName: 'cost', label: 'Cost', color: CHART_COLORS[0] }]}
            />
          </Section>
          <Section title="Cost by model" query={modelDistribution.data?.query} queryName="AI cost by model">
            <CategoricalChart
              items={toListItems(modelDistribution.data)}
              isLoading={modelDistribution.fetching && !modelDistribution.data}
              valueName="cost"
              colors={CHART_COLORS}
              format={formatCost}
              showYAxisLine={false}
              showTooltipValueName={false}
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
              format={formatCost}
              axisFormat={formatCostAxis}
              allowDecimals
              series={[{ valueName: 'avg_cost_per_run', label: 'Cost per run', color: CHART_COLORS[0] }]}
            />
          </Section>
          <Section
            title="Cost per session over time"
            query={costPerSessionTrend.data?.query}
            queryName="AI cost per session over time"
          >
            <TrendChart
              points={toTrendPoints(costPerSessionTrend.data)}
              isLoading={costPerSessionTrend.fetching && !costPerSessionTrend.data}
              format={formatCost}
              axisFormat={formatCostAxis}
              allowDecimals
              series={[
                { valueName: 'avg_cost_per_session', label: 'Cost per session', color: CHART_COLORS[0] },
              ]}
            />
          </Section>
          <Section
            title="Cost by function"
            query={topFunctionsByCost.data?.query}
            queryName="AI cost by function"
            className="lg:col-span-2"
          >
            <div className="flex flex-col gap-4 lg:flex-row">
              <CategoricalChart
                items={toListItems(topFunctionsByCost.data)}
                isLoading={topFunctionsByCost.fetching && !topFunctionsByCost.data}
                valueName="cost"
                colors={CHART_COLORS}
                format={formatCost}
                formatIdentifier={(id) => functionsBySlug.get(id)?.name ?? id}
                showYAxisLine={false}
                showTooltipValueName={false}
                className="min-w-0 lg:w-2/3"
              />
              <ChartLegend
                items={toListItems(topFunctionsByCost.data)}
                isLoading={topFunctionsByCost.fetching && !topFunctionsByCost.data}
                valueName="cost"
                colors={CHART_COLORS}
                format={formatCost}
                renderIdentifier={(id) => renderFunctionLink(id, envSlug, functionsBySlug)}
                className="w-full lg:w-1/3"
              />
            </div>
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
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, envSlug, functionsBySlug),
              }}
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
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, envSlug, functionsBySlug),
              }}
              columns={[
                { valueName: 'cost', label: 'Total cost', format: formatCost },
                { valueName: 'tokens', label: 'Tokens used', format: formatCompactNumber },
              ]}
            />
          </Section>
          <Section
            title="Most expensive sessions"
            query={mostExpensiveSessions.data?.query}
            queryName="AI most expensive sessions"
            className="lg:col-span-2"
          >
            <RankedTable
              items={toListItems(mostExpensiveSessions.data)}
              isLoading={mostExpensiveSessions.fetching && !mostExpensiveSessions.data}
              identifierLabel="Session"
              renderIdentifier={(id, item) => renderSessionLink(id, item.sessionKey ?? '', envSlug)}
              sessionKeyColumn={{
                label: 'Session key',
                render: (key) => renderSessionKeyLink(key, envSlug),
              }}
              columns={[
                { valueName: 'runs', label: 'Runs', format: formatCompactNumber },
                { valueName: 'cost', label: 'Cost', format: formatCost },
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
              colors={CHART_COLORS}
              showYAxisLine={false}
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
              functionColumn={{
                label: 'Function',
                render: (id) => renderFunctionLink(id, envSlug, functionsBySlug),
              }}
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

function renderFunctionLink(
  identifier: string,
  envSlug: string,
  functionsBySlug: Map<string, { name: string; slug: string }>,
) {
  const fn = functionsBySlug.get(identifier);
  if (!fn) return identifier;
  return (
    <a
      className="text-link hover:underline"
      href={pathCreator.function({ envSlug, functionSlug: fn.slug })}
    >
      {fn.name}
    </a>
  );
}

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
  title?: string;
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
        {title && <h2 className="text-basis text-sm font-medium">{title}</h2>}
        {query && (
          <Button
            className="ml-auto"
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
