'use client';

import { RangePicker } from '@inngest/components/DatePicker';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { Error } from '@inngest/components/Error/Error';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import {
  useBatchedSearchParams,
  useBooleanSearchParam,
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParam';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
  type DurationType,
} from '@inngest/components/utils/date';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { GetAccountEntitlementsDocument, MetricsScope } from '@/gql/graphql';
import { MetricsOverview } from './Overview';
import { MetricsVolume } from './Volume';
import { convertLookup } from './utils';

export type EntityType = {
  id: string;
  name: string;
  slug?: string;
};

export type EntityLookup = { [id: string]: EntityType };

export type MetricsFilters = {
  from: Date;
  until?: Date;
  selectedApps?: string[];
  selectedFns?: string[];
  autoRefresh?: boolean;
  entities: EntityLookup;
  functions: EntityLookup;
  scope: MetricsScope;
};

export const DEFAULT_DURATION = { hours: 24 };

const getFrom = (start?: Date, duration?: DurationType | '') =>
  start || subtractDuration(new Date(), duration ? duration : DEFAULT_DURATION);

const getDefaultRange = (start?: Date, end?: Date, duration?: DurationType | '') =>
  start && end
    ? {
        type: 'absolute' as const,
        start: start,
        end: end,
      }
    : {
        type: 'relative' as const,
        duration: duration ? duration : DEFAULT_DURATION,
      };

const MetricsLookupDocument = graphql(`
  query MetricsLookups($envSlug: String!, $page: Int, $pageSize: Int) {
    envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
        isArchived
      }
      workflows @paginated(perPage: $pageSize, page: $page) {
        data {
          name
          id
          slug
        }
        page {
          page
          totalPages
          perPage
        }
      }
    }
  }
`);

const AccountConcurrencyLookupDocument = graphql(`
  query AccountConcurrencyLookup {
    account {
      entitlements {
        concurrency {
          limit
        }
      }
    }
  }
`);

export const Dashboard = ({ envSlug }: { envSlug: string }) => {
  const [selectedApps, setApps, removeApps] = useStringArraySearchParam('apps');
  const [selectedFns, setFns, removeFns] = useStringArraySearchParam('fns');
  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const [autoRefresh] = useBooleanSearchParam('autoRefresh');
  const batchUpdate = useBatchedSearchParams();

  const parsedDuration = duration && parseDuration(duration);
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  const page = 1;
  //
  // TODO: handle more
  const pageSize = 1000;
  const [{ data, fetching, error }] = useQuery({
    query: MetricsLookupDocument,
    variables: { envSlug, page, pageSize },
  });

  const [{ data: accountData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });

  const [{ data: accountConcurrencyLimitRes }] = useQuery({
    query: AccountConcurrencyLookupDocument,
  });

  const apps = data?.envBySlug?.apps
    .filter(({ isArchived }) => isArchived === false)
    .map((app: { id: string; externalID: string }) => ({
      id: app.id,
      name: app.externalID,
    }));

  const functions = data?.envBySlug?.workflows.data;

  const logRetention = accountData?.account.entitlements.history.limit || 7;
  const upgradeCutoff = subtractDuration(new Date(), { days: logRetention });
  const concurrencyLimit = accountConcurrencyLimitRes?.account.entitlements.concurrency.limit;

  const envLookup = apps?.length !== 1 && !selectedApps?.length && !selectedFns?.length;
  const mappedFunctions = convertLookup(functions);
  const mappedApps = convertLookup(apps);
  const mappedEntities = envLookup ? mappedApps : mappedFunctions;

  error && console.error('Error fetcthing metrics lookup data', error);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="bg-canvasBase flex h-16 w-full flex-row items-center justify-between px-3 py-5">
        <div className="flex flex-row items-center justify-start gap-x-2">
          {fetching ? (
            <Skeleton className="block h-8 w-60" />
          ) : (
            <>
              <EntityFilter
                type="app"
                onFilterChange={(apps) => (apps.length ? setApps(apps) : removeApps())}
                selectedEntities={selectedApps || []}
                entities={apps || []}
                className="h-8"
              />
              <EntityFilter
                type="function"
                onFilterChange={(fns) => (fns.length ? setFns(fns) : removeFns())}
                selectedEntities={selectedFns || []}
                entities={functions || []}
                className="h-8"
              />
            </>
          )}
        </div>
        <div className="flex flex-row items-center justify-end gap-x-2">
          <RangePicker
            className="w-full"
            upgradeCutoff={upgradeCutoff}
            defaultValue={getDefaultRange(parsedStart, parsedEnd, parsedDuration)}
            onChange={(range: RangeChangeProps) => {
              batchUpdate({
                duration: range.type === 'relative' ? durationToString(range.duration) : null,
                start: range.type === 'absolute' ? range.start.toISOString() : null,
                end: range.type === 'absolute' ? range.end.toISOString() : null,
              });
            }}
          />
        </div>
      </div>
      {error && <Error message="There was an error fetching metrics filter data." />}
      <div className="bg-canvasSubtle px-6">
        <MetricsOverview
          from={getFrom(parsedStart, parsedDuration)}
          until={parsedEnd}
          selectedApps={selectedApps}
          selectedFns={selectedFns}
          autoRefresh={autoRefresh}
          entities={mappedEntities}
          functions={mappedFunctions}
          scope={envLookup ? MetricsScope.App : MetricsScope.Fn}
        />
      </div>
      <div className="bg-canvasSubtle px-6 pb-6">
        <MetricsVolume
          from={getFrom(parsedStart, parsedDuration)}
          until={parsedEnd}
          selectedApps={selectedApps}
          selectedFns={selectedFns}
          autoRefresh={autoRefresh}
          entities={mappedEntities}
          scope={envLookup ? MetricsScope.App : MetricsScope.Fn}
          concurrencyLimit={concurrencyLimit}
        />
      </div>
    </div>
  );
};
