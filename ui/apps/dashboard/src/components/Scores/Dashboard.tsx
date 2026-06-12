import { useMemo } from 'react';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
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
import { graphql } from '@/gql';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { Legend } from './Legend';
import { ScoreCard } from './ScoreCard';
import type { ScoreSeries } from './types';

const DEFAULT_DURATION = { hours: 24 };

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
    <div className="flex h-full w-full flex-col">
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
      <div className="bg-canvasBase flex flex-row gap-4 px-4 pb-6">
        <div className="grid flex-1 grid-cols-1 gap-4 lg:grid-cols-2">
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
                isLoading={seriesFetching}
                error={seriesError}
              />
            ))
          )}
        </div>
        {!filterError && (
          <Legend
            scores={availableScores}
            disabled={disabled}
            onToggle={toggleScore}
            isLoading={namesFetching}
          />
        )}
      </div>
    </div>
  );
};
