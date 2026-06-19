import { useMemo, useState } from 'react';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import {
  HelperPanelControl,
  HelperPanelFrame,
  type HelperItem,
} from '@inngest/components/HelperPanelControl';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { InsightsIcon } from '@inngest/components/icons/sections/Insights';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import {
  useBatchedSearchParams,
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParams';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
} from '@inngest/components/utils/date';
import { useRouterState } from '@tanstack/react-router';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { lineColors } from '@/components/Metrics/utils';
import { graphql } from '@/gql';
import { GetAccountEntitlementsDocument, ScoreKind } from '@/gql/graphql';
import { Legend } from './Legend';
import { ScoreCard } from './ScoreCard';
import type { ScoreSeries } from './types';

const DEFAULT_DURATION = { hours: 24 };
const SCORES_PANEL = 'Scores';

// Red (lineColors[3]) is reserved for boolean `false` bars and must never be
// used for a numeric line chart (nor any near-red hue). Numeric scores cycle
// through this red-free subset of the metrics palette.
const NUMERIC_LINE_COLORS = [
  lineColors[2], // blue
  lineColors[4], // purple
  lineColors[1], // green
  lineColors[0], // amber
];

const ScoresLookupDocument = graphql(`
  query ScoresLookup($envSlug: String!, $page: Int, $pageSize: Int) {
    envBySlug(slug: $envSlug) {
      workflows @paginated(perPage: $pageSize, page: $page) {
        data {
          name
          id
          slug
        }
      }
    }
  }
`);

const ScoreNamesDocument = graphql(`
  query ScoreNames(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $filter: ScoreFilter!
  ) {
    scoreNames(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      filter: $filter
    ) {
      name
      kind
    }
  }
`);

const ScoreTimeSeriesDocument = graphql(`
  query ScoreTimeSeries(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $filter: ScoreFilter!
    $scoreNames: [String!]
  ) {
    scoreTimeSeries(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      filter: $filter
      scoreNames: $scoreNames
    ) {
      scoreName
      kind
      buckets {
        bucketStart
        avg
        max
        p50
        p90
        p99
        trueCount
        falseCount
      }
    }
  }
`);

export const ScoresDashboard = ({ envSlug }: { envSlug: string }) => {
  const environment = useEnvironment();
  const workspaceID = environment.id;

  const [selectedFns, setFns, removeFns] = useStringArraySearchParam('fns');
  const [disabledParam, setDisabled, removeDisabled] =
    useStringArraySearchParam('disabled');
  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const batchUpdate = useBatchedSearchParams();

  const parsedDuration = duration ? parseDuration(duration) : '';
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  // `loadedAt` bumps on router.invalidate(), so RefreshButton refires queries.
  const loadedAt = useRouterState({ select: (s) => s.loadedAt });

  // Stabilize range against the raw URL params so a fresh `now` doesn't
  // refire queries on every render.
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

  const [{ data: lookupData, fetching: lookupFetching, error: lookupError }] =
    useQuery({
      query: ScoresLookupDocument,
      variables: { envSlug, page: 1, pageSize: 1000 },
    });

  const [{ data: accountData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const daysAgoMax = accountData?.account.entitlements.history.limit ?? 7;

  const functions = lookupData?.envBySlug?.workflows.data ?? [];

  const [{ data: namesData, fetching: namesFetching, error: namesError }] =
    useQuery({
      query: ScoreNamesDocument,
      variables: {
        workspaceID,
        functionIDs: selectedFns,
        filter: { timeRange },
      },
    });

  const { availableScores, disabled, visibleScores, enabledNames } =
    useMemo(() => {
      const availableScores = namesData?.scoreNames ?? [];
      const disabled = new Set(disabledParam ?? []);
      const visibleScores = availableScores.filter(
        (s) => !disabled.has(s.name),
      );
      return {
        availableScores,
        disabled,
        visibleScores,
        enabledNames: visibleScores.map((s) => s.name),
      };
    }, [namesData, disabledParam]);

  const toggleScore = (key: string) => {
    const current = disabledParam ?? [];
    const next = disabled.has(key)
      ? current.filter((k) => k !== key)
      : [...current, key];
    next.length ? setDisabled(next) : removeDisabled();
  };

  // One stable color per score, assigned by position in the full available
  // list so a score keeps its color regardless of which others are toggled
  // off. Reuses the metrics palette; cycles when there are more than 5 scores.
  const scoreColors = useMemo(() => {
    const dark = isDark();
    const m = new Map<string, string>();
    availableScores.forEach((s, i) => {
      const [token, hex] = NUMERIC_LINE_COLORS[i % NUMERIC_LINE_COLORS.length];
      m.set(s.name, resolveColor(token, dark, hex));
    });
    return m;
  }, [availableScores]);

  const [panelOpen, setPanelOpen] = useState(true);
  const helperItems: HelperItem[] = [
    {
      title: SCORES_PANEL,
      icon: <InsightsIcon className="h-5 w-5" />,
      action: () => setPanelOpen((open) => !open),
    },
  ];

  const [{ data: seriesData, fetching: seriesFetching, error: seriesError }] =
    useQuery({
      query: ScoreTimeSeriesDocument,
      variables: {
        workspaceID,
        functionIDs: selectedFns,
        filter: { timeRange },
        scoreNames: enabledNames,
      },
      pause: enabledNames.length === 0,
    });

  const seriesByName = useMemo(() => {
    const m = new Map<string, ScoreSeries>();
    for (const s of seriesData?.scoreTimeSeries ?? []) {
      m.set(s.scoreName, s);
    }
    return m;
  }, [seriesData]);

  // Toggle tint: numeric scores match their per-score line color; boolean
  // scores use the same green as their `true` bars (lineColors[1]) so every
  // boolean follows the shared green/red pattern. `kind` comes from the score
  // names query so the correct color is known on first paint (no flip once the
  // time series loads).
  const toggleColors = useMemo(() => {
    const dark = isDark();
    const booleanGreen = resolveColor(lineColors[1][0], dark, lineColors[1][1]);
    const m = new Map<string, string>();
    for (const s of availableScores) {
      const isBoolean = s.kind === ScoreKind.Boolean;
      m.set(
        s.name,
        isBoolean ? booleanGreen : scoreColors.get(s.name) ?? booleanGreen,
      );
    }
    return m;
  }, [availableScores, scoreColors]);

  const defaultRange =
    parsedStart && parsedEnd
      ? {
          type: 'absolute' as const,
          start: parsedStart,
          end: parsedEnd,
        }
      : {
          type: 'relative' as const,
          duration: parsedDuration || DEFAULT_DURATION,
        };

  const isLoading = namesFetching || seriesFetching;

  const filterError = lookupError ?? namesError;

  return (
    <div className="flex min-h-0 w-full flex-1 flex-row overflow-hidden">
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="bg-canvasBase flex flex-row items-center gap-1.5 px-3 py-[9px]">
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
          {lookupFetching ? (
            <Skeleton className="block h-6 w-60" />
          ) : (
            <EntityFilter
              type="function"
              onFilterChange={(fns) => (fns.length ? setFns(fns) : removeFns())}
              selectedEntities={selectedFns || []}
              entities={functions}
            />
          )}
        </div>
        {filterError && (
          <Error message="There was an error fetching scores filter data." />
        )}
        <div className="no-scrollbar min-h-0 flex-1 overflow-y-auto px-3 pb-6 [&::-webkit-scrollbar]:hidden">
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {visibleScores.length === 0 && !isLoading && !namesError ? (
              <div className="text-muted col-span-full py-10 text-center text-sm">
                {availableScores.length === 0
                  ? 'No scores recorded in this range.'
                  : 'No scores selected.'}
              </div>
            ) : (
              visibleScores.map((s) => (
                <ScoreCard
                  key={s.name}
                  name={s.name}
                  series={seriesByName.get(s.name)}
                  color={scoreColors.get(s.name)}
                  isLoading={seriesFetching}
                  error={seriesError}
                />
              ))
            )}
          </div>
        </div>
      </div>
      {!filterError && panelOpen && (
        <aside className="border-subtle flex w-[300px] shrink-0 flex-col overflow-hidden border-l">
          <HelperPanelFrame
            title={SCORES_PANEL}
            icon={<InsightsIcon className="h-5 w-5" />}
            onClose={() => setPanelOpen(false)}
          >
            <Legend
              scores={availableScores}
              disabled={disabled}
              colors={toggleColors}
              onToggle={toggleScore}
              isLoading={namesFetching}
            />
          </HelperPanelFrame>
        </aside>
      )}
      {!filterError && (
        <HelperPanelControl
          items={helperItems}
          activeTitle={panelOpen ? SCORES_PANEL : null}
        />
      )}
    </div>
  );
};
