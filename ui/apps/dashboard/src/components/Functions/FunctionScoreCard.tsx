import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { Error } from '@inngest/components/Error/Error';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowRightUpLine, RiMoreFill } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';
import type { CombinedError } from 'urql';

import { BooleanChart } from '@/components/InsightsMetrics/BooleanChart';
import { BoxPlot, type RowData } from '@/components/InsightsMetrics/BoxPlot';
import { CandlestickPlot, type CandleData } from '@/components/InsightsMetrics/CandlestickPlot';
import { CHART_COLORS, CHART_COLORS_SUBTLE } from '@/components/InsightsMetrics/colors';
import { formatPlainNumber } from '@/components/InsightsMetrics/format';
import { TrendChart } from '@/components/InsightsMetrics/TrendChart';
import type { InsightsMetricPoint } from '@/components/InsightsMetrics/types';
import { useEnvironment } from '@/components/Environments/environment-context';
import type { BooleanStats, ScoreOverviewStats } from '@/components/Scores/scoreAggregation';
import type { ScoreSeries } from '@/components/Scores/types';
import { pathCreator } from '@/utils/urls';

// Green/orange from CHART_COLORS — the same palette the box/candlestick
// charts above use — rather than the Metrics dashboard's own line palette,
// so every chart on this card draws from one consistent set of colors.
// TrendChart defaults a series' subtle fill to CHART_COLORS_SUBTLE at that
// series' *position* in the array, not the index of the color it was
// actually given — since "false" is CHART_COLORS[3] but sits at array
// position 1, that default would land on CHART_COLORS_SUBTLE[1] (blue)
// instead of the orange that matches its border. Passing fillColor
// explicitly at the same index as the border/color keeps them paired.
const TRUE_COLOR = CHART_COLORS[0];
const TRUE_SUBTLE_COLOR = CHART_COLORS_SUBTLE[0];
const FALSE_COLOR = CHART_COLORS[3];
const FALSE_SUBTLE_COLOR = CHART_COLORS_SUBTLE[3];

type Tab = 'overview' | 'timeseries';

type Props = {
  name: string;
  series: ScoreSeries | undefined;
  isLoading: boolean;
  // Resolved per-score color (and its paired subtle fill), both from
  // CHART_COLORS/CHART_COLORS_SUBTLE — the box/candlestick charts use these
  // so every score card draws from one consistent palette. Fall back to the
  // palette's blue when absent.
  color?: string;
  subtleColor?: string;
  // The `Error` import above is the banner component and shadows the global
  // Error type, so type this as what FunctionScores actually passes.
  error?: CombinedError;
  // True-quartile box-plot stats for this score across the whole selected
  // range (score_overall_aggregation) — undefined while loading or when the
  // score has no numeric observations in range.
  overview?: RowData;
  // The fuller stat set for the same range (adds p95/p99, which have no
  // place on a standard box plot but are still useful headline numbers) —
  // for the Overview tab's stat row.
  stats?: ScoreOverviewStats;
  // Same stats bucketed over time (score_aggregation_trend), one candle per
  // bucket.
  trend?: CandleData[];
  // Whether this score is boolean-kind — sourced from
  // score_overall_aggregation's own is_boolean column (every observation
  // collapsing to exactly 0 or 1), not ScoreNamesDocument's `kind`, since
  // it's computed straight from the same values the charts below plot.
  isBoolean: boolean;
  // Exact true/false split for a boolean-kind score, from the same
  // score_overall_aggregation/score_aggregation_trend calls' count_true
  // column — undefined when isBoolean is false or the data hasn't loaded.
  booleanOverview?: BooleanStats;
  booleanTrend?: (BooleanStats & { timestamp: string })[];
  // The exact Insights-dialect SQL behind the Overview/Timeseries tab's
  // data — one query shared by every score card, since
  // score_overall_aggregation/score_aggregation_trend each return every
  // score name in one call. Powers the "Open in Insights" menu action.
  overviewQuery?: string;
  trendQuery?: string;
};

export const FunctionScoreCard = ({
  name,
  series,
  isLoading,
  color,
  subtleColor,
  error,
  overview,
  stats,
  trend,
  isBoolean,
  booleanOverview,
  booleanTrend,
  overviewQuery,
  trendQuery,
}: Props) => {
  const navigate = useNavigate();
  const environment = useEnvironment();
  const [tab, setTab] = useState<Tab>('timeseries');
  // Whichever tab is showing decides which query "Open in Insights" opens —
  // Overview's box stats and Timeseries' candles come from two separate
  // insightsMetric calls (score_overall_aggregation/score_aggregation_trend).
  const activeQuery = tab === 'overview' ? overviewQuery : trendQuery;

  const resolvedColor = color ?? CHART_COLORS[1];
  const resolvedSubtleColor = subtleColor ?? CHART_COLORS_SUBTLE[1];

  const overviewRow = useMemo<RowData | undefined>(
    () =>
      overview && { ...overview, color: resolvedColor, subtleColor: resolvedSubtleColor },
    [overview, resolvedColor, resolvedSubtleColor],
  );

  const boxDomain = useMemo<[number, number]>(() => {
    if (!overviewRow) return [0, 1];
    const pad = (overviewRow.max - overviewRow.min) * 0.1 || 1;
    return [overviewRow.min - pad, overviewRow.max + pad];
  }, [overviewRow]);

  // Boolean scores get their true/false counts from count_true (an exact
  // countIf(score_value = 1), not derived from avg*count) rather than the
  // old ScoreTimeSeriesDocument buckets. Timeseries renders a plain
  // true/false stacked-bar histogram via the shared Recharts TrendChart
  // (not ECharts — ECharts' canvas renderer can't parse CHART_COLORS'
  // `rgb(var(--...))` strings the way Recharts' SVG output can); Overview
  // collapses it into one fraction-true row on the same lollipop chart
  // Experiments' boolean metrics use.
  const booleanTrendPoints = useMemo<InsightsMetricPoint[]>(() => {
    if (!isBoolean || !booleanTrend) return [];
    return booleanTrend.map((b) => ({
      timestamp: b.timestamp,
      values: [
        { name: 'true', value: b.countTrue },
        { name: 'false', value: b.count - b.countTrue },
      ],
    }));
  }, [isBoolean, booleanTrend]);

  const booleanOverviewRow = useMemo(() => {
    if (!isBoolean || !booleanOverview) return undefined;
    const { count, countTrue } = booleanOverview;
    return {
      label: name,
      avg: count > 0 ? countTrue / count : 0,
      count,
      color: resolvedColor,
      subtleColor: resolvedSubtleColor,
      opacity: 1,
    };
  }, [isBoolean, booleanOverview, name, resolvedColor, resolvedSubtleColor]);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-hidden rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-muted flex w-full flex-row items-center gap-x-2 font-mono text-base">
          {name}
        </div>
        {activeQuery && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                size="small"
                kind="secondary"
                appearance="ghost"
                icon={<RiMoreFill />}
              />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onSelect={() =>
                  navigate({
                    to: pathCreator.insights({ envSlug: environment.slug }),
                    search: { sql: activeQuery, name },
                  })
                }
              >
                <RiArrowRightUpLine className="h-4 w-4" />
                Open in Insights
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      {series ? (
        <ScoreTabs selected={tab} onSelect={setTab} />
      ) : (
        // Reserve the tab row's height while loading so the chart container
        // doesn't shrink after the chart has measured it.
        <TabRow>
          <span className="invisible -mb-px border-b-2 border-transparent pb-1.5 text-sm">
            Overview
          </span>
        </TabRow>
      )}
      {error ? (
        <Error message="Failed to load chart" />
      ) : !series ? (
        isLoading ? (
          <div className="flex min-h-0 flex-1 flex-row items-center">
            <Chart option={{}} className="relative h-full w-full" group="functionScores" loading />
          </div>
        ) : (
          <div className="text-muted flex min-h-0 flex-1 items-center justify-center text-sm">
            No data in selected range
          </div>
        )
      ) : !isBoolean ? (
        <div className="min-h-0 flex-1">
          {tab === 'overview' ? (
            <div className="flex h-full flex-col">
              <StatRow
                stats={[
                  { label: 'Runs', value: stats?.count.toLocaleString() ?? '—' },
                  { label: 'Avg', value: stats ? formatPlainNumber(stats.avg) : '—' },
                  { label: 'Median', value: stats ? formatPlainNumber(stats.med) : '—' },
                  {
                    label: 'Q1/Q3',
                    value: stats ? `${formatPlainNumber(stats.q1)} / ${formatPlainNumber(stats.q3)}` : '—',
                  },
                  { label: 'p95', value: stats?.p95 !== undefined ? formatPlainNumber(stats.p95) : '—' },
                  { label: 'p99', value: stats?.p99 !== undefined ? formatPlainNumber(stats.p99) : '—' },
                  {
                    label: 'Min/Max',
                    value: stats ? `${formatPlainNumber(stats.min)} / ${formatPlainNumber(stats.max)}` : '—',
                  },
                ]}
              />
              <div className="flex flex-1 flex-col items-center justify-center">
                <BoxPlot
                  rows={overviewRow ? [overviewRow] : []}
                  domain={boxDomain}
                  metricDisplayName={name}
                />
              </div>
            </div>
          ) : (
            <CandlestickPlot
              className="h-full"
              candles={trend ?? []}
              color={resolvedColor}
              subtleColor={resolvedSubtleColor}
              isLoading={isLoading && !trend}
              group="functionScores"
            />
          )}
        </div>
      ) : (
        <div className="min-h-0 flex-1">
          {tab === 'overview' ? (
            <div className="flex h-full flex-col">
              <StatRow
                stats={[
                  { label: 'Runs', value: booleanOverviewRow?.count?.toLocaleString() ?? '—' },
                  {
                    label: 'True rate',
                    value: booleanOverviewRow ? `${(booleanOverviewRow.avg * 100).toFixed(1)}%` : '—',
                  },
                ]}
              />
              <div className="flex flex-1 flex-col items-center justify-center">
                <BooleanChart
                  rows={booleanOverviewRow ? [booleanOverviewRow] : []}
                  domain={[0, 1]}
                  metricDisplayName={name}
                />
              </div>
            </div>
          ) : (
            <TrendChart
              className="h-full"
              points={booleanTrendPoints}
              series={[
                {
                  valueName: 'true',
                  label: 'True',
                  color: TRUE_COLOR,
                  borderColor: TRUE_COLOR,
                  fillColor: TRUE_SUBTLE_COLOR,
                },
                {
                  valueName: 'false',
                  label: 'False',
                  color: FALSE_COLOR,
                  borderColor: FALSE_COLOR,
                  fillColor: FALSE_SUBTLE_COLOR,
                },
              ]}
              chartType="bar"
              stacked
              allowDecimals={false}
              isLoading={isLoading && !booleanTrend}
              group="functionScores"
            />
          )}
        </div>
      )}
    </div>
  );
};

// StatRow is a compact, always-visible summary line above the Overview
// chart — the box/lollipop chart's own tooltip already shows these numbers
// on hover, but a card this small benefits from surfacing the headline
// figures without requiring a hover.
const StatRow = ({ stats }: { stats: { label: string; value: string }[] }) => (
  <div className="mb-3 flex flex-wrap gap-x-4 gap-y-1">
    {stats.map((s) => (
      <div key={s.label} className="flex items-baseline gap-1.5 text-xs">
        <span className="text-muted">{s.label}</span>
        <span className="text-basis font-semibold tabular-nums">{s.value}</span>
      </div>
    ))}
  </div>
);

const ScoreTabs = ({
  selected,
  onSelect,
}: {
  selected: Tab;
  onSelect: (tab: Tab) => void;
}) => (
  <TabRow>
    <TabButton
      label="Over Time"
      isActive={selected === 'timeseries'}
      onClick={() => onSelect('timeseries')}
    />
    <TabButton label="Summary" isActive={selected === 'overview'} onClick={() => onSelect('overview')} />
  </TabRow>
);

const TabRow = ({ children }: React.PropsWithChildren) => (
  <div className="border-subtle mb-2 flex flex-row gap-4 border-b">
    {children}
  </div>
);

const TabButton = ({
  label,
  isActive,
  onClick,
}: {
  label: string;
  isActive: boolean;
  onClick?: () => void;
}) => (
  <button
    type="button"
    onClick={onClick}
    className={cn(
      '-mb-px pb-1.5 text-sm',
      isActive
        ? 'text-basis border-contrast border-b-2 font-medium'
        : 'text-muted hover:text-basis',
    )}
  >
    {label}
  </button>
);
