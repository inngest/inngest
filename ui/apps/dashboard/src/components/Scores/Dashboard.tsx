import { useMemo } from 'react';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Switch } from '@inngest/components/Switch';
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
import { parse } from 'graphql';
import { useQuery, type TypedDocumentNode } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { ScoreCard } from './ScoreCard';
import type { ScoreNamesResult, ScoreTimeSeriesResult } from './types';

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

// Parsed at runtime; codegen skips plain `parse(...)` calls so these don't get
// validated against the live schema until the cloud deploy lands. Swap to
// `graphql(...)` then for typed introspection.
const ScoreNamesDocument = parse(`
  query ScoreNames(
    $workspaceID: ID!
    $functionID: ID
    $filter: ScoreFilter!
  ) {
    scoreNames(
      workspaceID: $workspaceID
      functionID: $functionID
      filter: $filter
    ) {
      name
      kind
    }
  }
`) as TypedDocumentNode<
  ScoreNamesResult,
  {
    workspaceID: string;
    functionID?: string;
    filter: { timeRange: { from: string; to: string } };
  }
>;

const ScoreTimeSeriesDocument = parse(`
  query ScoreTimeSeries(
    $workspaceID: ID!
    $functionID: ID
    $filter: ScoreFilter!
    $scoreNames: [String!]
  ) {
    scoreTimeSeries(
      workspaceID: $workspaceID
      functionID: $functionID
      filter: $filter
      scoreNames: $scoreNames
    ) {
      scoreName
      kind
      bucketSeconds
      buckets {
        bucketStart
        p50
        p90
        p99
        trueCount
        falseCount
      }
    }
  }
`) as TypedDocumentNode<
  ScoreTimeSeriesResult,
  {
    workspaceID: string;
    functionID?: string;
    filter: { timeRange: { from: string; to: string } };
    scoreNames?: string[];
  }
>;

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

  // Stabilize range against the raw URL params so a fresh `now` doesn't
  // refire queries on every render.
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

  // TODO: scoring API only accepts one functionID; widen to [ID!] and pass all.
  const functionID = selectedFns?.[0];

  const [{ data: namesData, fetching: namesFetching, error: namesError }] =
    useQuery({
      query: ScoreNamesDocument,
      variables: { workspaceID, functionID, filter: { timeRange } },
    });

  const availableScores = useMemo(
    () => namesData?.scoreNames ?? [],
    [namesData],
  );

  const disabled = useMemo(() => new Set(disabledParam ?? []), [disabledParam]);

  const visibleScores = useMemo(
    () => availableScores.filter((s) => !disabled.has(s.name)),
    [availableScores, disabled],
  );

  const enabledNames = useMemo(
    () => visibleScores.map((s) => s.name),
    [visibleScores],
  );

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
        functionID,
        filter: { timeRange },
        scoreNames: enabledNames,
      },
      pause: enabledNames.length === 0,
    });

  const seriesByName = useMemo(() => {
    const m = new Map<
      string,
      ScoreTimeSeriesResult['scoreTimeSeries'][number]
    >();
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
  filterError &&
    console.error('Error fetching scores filter data', filterError);

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
                range={range}
                isLoading={seriesFetching}
                error={seriesError}
              />
            ))
          )}
        </div>
        <Legend
          scores={availableScores}
          disabled={disabled}
          onToggle={toggleScore}
          isLoading={namesFetching}
        />
      </div>
    </div>
  );
};

type LegendProps = {
  scores: { name: string }[];
  disabled: Set<string>;
  onToggle: (key: string) => void;
  isLoading: boolean;
};

const Legend = ({ scores, disabled, onToggle, isLoading }: LegendProps) => {
  return (
    <div className="border-subtle w-[220px] shrink-0 rounded-md border p-4">
      <div className="text-subtle mb-3 text-sm font-medium">Scores</div>
      {isLoading && scores.length === 0 ? (
        <Skeleton className="h-24 w-full" />
      ) : scores.length === 0 ? (
        <div className="text-muted text-xs">None in range.</div>
      ) : (
        <div className="flex flex-col gap-3">
          {scores.map((s) => (
            <div key={s.name} className="flex items-center justify-between">
              <span className="text-basis text-sm">{s.name}</span>
              <Switch
                checked={!disabled.has(s.name)}
                onCheckedChange={() => onToggle(s.name)}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
