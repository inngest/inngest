import { useMemo } from 'react';
import { Alert } from '@inngest/components/Alert';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import {
  useBatchedSearchParams,
  useSearchParam,
} from '@inngest/components/hooks/useSearchParams';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
} from '@inngest/components/utils/date';
import { useQuery } from 'urql';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { CHART_COLORS, CHART_COLORS_SUBTLE } from '@/components/InsightsMetrics/colors';
import { TREND_BUCKET_LIMIT } from '@/components/InsightsMetrics/queries';
import { useEnvironment } from '@/components/Environments/environment-context';
import {
  ScoreNamesDocument,
  ScoreTimeSeriesDocument,
} from '@/components/Scores/queries';
import {
  useScoreAggregationTrend,
  useScoreOverallAggregation,
} from '@/components/Scores/scoreAggregation';
import type { ScoreSeries } from '@/components/Scores/types';
import { ScoreKind } from '@/gql/graphql';
import { useFunction } from '@/queries/functions';
import { useAccountFeatures } from '@/utils/useAccountFeatures';
import { FunctionScoreCard } from './FunctionScoreCard';

const DEFAULT_DURATION = { hours: 24 };

type Props = {
  functionSlug: string;
};

export const FunctionScores = ({ functionSlug }: Props) => {
  const environment = useEnvironment();
  const workspaceID = environment.id;

  const [{ data, fetching: isFetchingFunction }] = useFunction({
    functionSlug,
  });
  const function_ = data?.workspace.workflow;
  const functionID = function_?.id;

  const features = useAccountFeatures();
  const daysAgoMax = features.data?.history ?? 7;

  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const batchUpdate = useBatchedSearchParams();

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

  const [{ data: namesData, fetching: namesFetching, error: namesError }] =
    useQuery({
      query: ScoreNamesDocument,
      variables: {
        workspaceID,
        functionIDs: functionID ? [functionID] : [],
        filter: { timeRange },
      },
      pause: !functionID,
    });

  const availableScores = namesData?.scoreNames ?? [];

  const kindByName = useMemo(() => {
    const m = new Map<string, ScoreKind>();
    for (const s of availableScores) m.set(s.name, s.kind);
    return m;
  }, [availableScores]);

  const enabledNames = useMemo(
    () => availableScores.map((s) => s.name),
    [availableScores],
  );

  const [{ data: seriesData, fetching: seriesFetching, error: seriesError }] =
    useQuery({
      query: ScoreTimeSeriesDocument,
      variables: {
        workspaceID,
        functionIDs: functionID ? [functionID] : [],
        filter: { timeRange },
        scoreNames: enabledNames,
      },
      pause: !functionID || enabledNames.length === 0,
    });

  const seriesByName = useMemo(() => {
    const m = new Map<string, ScoreSeries>();
    for (const s of seriesData?.scoreTimeSeries ?? []) {
      m.set(s.scoreName, s);
    }
    return m;
  }, [seriesData]);

  const {
    byName: overviewByName,
    statsByName,
    isBooleanByName,
    booleanByName: booleanOverviewByName,
    order,
    query: overviewQuery,
    fetching: overviewFetching,
  } = useScoreOverallAggregation({
    workspaceID,
    functionIDs: functionID ? [functionID] : [],
    range: timeRange,
    pause: !functionID,
  });
  const {
    byName: trendByName,
    booleanByName: booleanTrendByName,
    query: trendQuery,
    fetching: trendFetching,
  } = useScoreAggregationTrend({
    workspaceID,
    functionIDs: functionID ? [functionID] : [],
    range: timeRange,
    limit: TREND_BUCKET_LIMIT,
    pause: !functionID,
  });

  // The identifiers score_overall_aggregation returns — falls back to
  // ScoreNamesDocument's list only while that query hasn't resolved yet, so
  // the grid isn't empty on first paint — sorted in natural (alphanumeric,
  // e.g. "score2" before "score10") rather than the backend's own row
  // order, which defaults to ORDER BY count DESC.
  const orderedNames = useMemo(() => {
    const names = order.length > 0 ? order : availableScores.map((s) => s.name);
    return [...names].sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));
  }, [order, availableScores]);

  // CHART_COLORS/CHART_COLORS_SUBTLE, index-paired the same way BoxPlot's
  // Overview/Timeseries charts use them elsewhere in InsightsMetrics — each
  // score keeps its color regardless of which others are toggled off.
  const scoreColors = useMemo(() => {
    const color = new Map<string, string>();
    const subtleColor = new Map<string, string>();
    orderedNames.forEach((name, i) => {
      color.set(name, CHART_COLORS[i % CHART_COLORS.length]);
      subtleColor.set(name, CHART_COLORS_SUBTLE[i % CHART_COLORS_SUBTLE.length]);
    });
    return { color, subtleColor };
  }, [orderedNames.join(',')]);

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
        <Alert severity="warning">
          Function not yet deployed to this environment
        </Alert>
      </div>
    );
  }

  const isLoading = namesFetching || seriesFetching;
  const filterError = namesError;

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      {filterError && (
        <Error message="There was an error fetching scores for this function." />
      )}
      <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto p-5 [&::-webkit-scrollbar]:hidden">
        <div className="bg-canvasBase mb-4 flex flex-row items-center gap-1.5">
          <TimeFilter
            daysAgoMax={daysAgoMax}
            defaultValue={defaultRange}
            onDaysChange={(r: RangeChangeProps) => {
              batchUpdate({
                duration:
                  r.type === 'relative' ? durationToString(r.duration) : null,
                start: r.type === 'absolute' ? r.start.toISOString() : null,
                end: r.type === 'absolute' ? r.end.toISOString() : null,
              });
            }}
          />
        </div>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {orderedNames.length === 0 && !isLoading && !namesError ? (
            <div className="text-muted col-span-full py-10 text-center text-sm">
              No scores recorded in this range.
            </div>
          ) : (
            orderedNames.map((name) => (
              <FunctionScoreCard
                key={name}
                name={name}
                series={seriesByName.get(name)}
                color={scoreColors.color.get(name)}
                subtleColor={scoreColors.subtleColor.get(name)}
                isLoading={seriesFetching || overviewFetching || trendFetching}
                error={seriesError}
                overview={overviewByName.get(name)}
                stats={statsByName.get(name)}
                trend={trendByName.get(name)}
                // isBooleanByName comes straight from score_overall_aggregation's
                // own is_boolean column; falls back to the ScoreNames kind while
                // that query hasn't resolved yet.
                isBoolean={isBooleanByName.get(name) ?? kindByName.get(name) === ScoreKind.Boolean}
                booleanOverview={booleanOverviewByName.get(name)}
                booleanTrend={booleanTrendByName.get(name)}
                overviewQuery={overviewQuery}
                trendQuery={trendQuery}
              />
            ))
          )}
        </div>
      </div>
    </div>
  );
};
