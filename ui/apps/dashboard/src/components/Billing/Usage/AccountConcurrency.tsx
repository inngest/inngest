import { useMemo } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Select } from '@inngest/components/Select/Select';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import Tabs from '@inngest/components/Tabs/Tabs';
import { cn } from '@inngest/components/utils/classNames';
import { useQuery } from 'urql';

import { createConcurrencyChartOptions } from '@/components/Billing/Usage/concurrencyChart';
import { graphql } from '@/gql';
import { Route as BillingRoute } from '@/routes/_authed/billing/route';
import { pathCreator } from '@/utils/urls';

const GetBurstConcurrencyLimitDocument = graphql(`
  query GetBurstConcurrencyLimit {
    account {
      burstConcurrency {
        limit
      }
    }
  }
`);

const GetAccountConcurrencyTimeSeriesDocument = graphql(`
  query GetAccountConcurrencyTimeSeries($from: Time!, $to: Time!) {
    account {
      concurrencyTimeSeries(from: $from, to: $to) {
        baseLimit
        burstLimit
        peak
        buckets {
          bucket
          value
        }
        limitHits {
          bucket
          value
        }
        bursts {
          startedAt
          endedAt
          peak
          limitHits
        }
      }
    }
  }
`);

const ref = 'app-billing-usage-concurrency';
const periodOption = { id: '30d', name: '30 days' };
const periodDays = 30;

function burstMultiplier(burstLimit: number, baseLimit?: number) {
  if (baseLimit && burstLimit % baseLimit === 0) {
    return `${baseLimit}*${burstLimit / baseLimit}`;
  }
  return undefined;
}

type TileProps = {
  title: React.ReactNode;
  loading?: boolean;
  children: React.ReactNode;
};

function Tile({ title, loading, children }: TileProps) {
  return (
    <div className="bg-canvasBase border-subtle w-[245px] rounded-md border px-4 py-4">
      <p className="text-subtle text-sm">{title}</p>
      <p className="text-basis mt-1 text-2xl">
        {loading ? <Skeleton className="h-8 w-24" /> : children}
      </p>
    </div>
  );
}

type Props = {
  baseLimit?: number;
};

export function AccountConcurrency({ baseLimit }: Props) {
  const { profile } = BillingRoute.useLoaderData();
  const [{ data: burstData, error: burstError }] = useQuery({
    query: GetBurstConcurrencyLimitDocument,
  });
  const burstLimit = burstError
    ? undefined
    : burstData?.account.burstConcurrency?.limit;
  const multiplier =
    burstLimit !== undefined
      ? burstMultiplier(burstLimit, baseLimit)
      : undefined;

  const range = useMemo(() => {
    const to = new Date();
    const from = new Date(to.getTime() - periodDays * 24 * 60 * 60 * 1000);
    return { from: from.toISOString(), to: to.toISOString() };
  }, []);
  const [{ data: seriesData, error: seriesError, fetching: seriesFetching }] =
    useQuery({
      query: GetAccountConcurrencyTimeSeriesDocument,
      variables: range,
      pause: burstLimit === undefined,
    });
  const series = seriesError
    ? undefined
    : seriesData?.account.concurrencyTimeSeries;
  const chartOption = useMemo(
    () => (series ? createConcurrencyChartOptions(series, range) : undefined),
    [series, range],
  );

  if (burstLimit === undefined) {
    return null;
  }

  return (
    <div className="mt-10">
      <div className="mb-6 flex items-center justify-between gap-6">
        <div>
          <h3 className="text-basis mb-1 text-xl">Account concurrency</h3>
          <p className="text-subtle text-sm italic">
            Maximum number of concurrently executing steps{' '}
            <span className="font-bold">in your entire account</span>
          </p>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button
            appearance="outlined"
            label="Talk to sales"
            href={pathCreator.support({ ref })}
            target="_blank"
          />
          {!profile.isMarketplace && (
            <Button
              label="Increase concurrency"
              href={pathCreator.billing({ highlight: 'concurrency', ref })}
            />
          )}
        </div>
      </div>
      <div className="mb-4 flex flex-wrap gap-4">
        <Tile title="Account concurrency" loading={baseLimit === undefined}>
          {baseLimit} steps
        </Tile>
        <Tile title="Concurrency during burst">
          {burstLimit} steps
          {multiplier && (
            <span className="text-muted ml-2 text-xs">({multiplier})</span>
          )}
        </Tile>
        {!seriesError && (
          <Tile
            title={
              <>
                Peak concurrency{' '}
                <span className="text-muted">[{periodOption.id}]</span>
              </>
            }
            loading={!series}
          >
            <span
              className={cn(
                series?.limitHits.some((point) => (point.value ?? 0) > 0)
                  ? 'text-error'
                  : 'text-basis',
              )}
            >
              {series?.peak} steps
            </span>
          </Tile>
        )}
      </div>
      <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
        <div className="flex items-center justify-between">
          <Tabs defaultValue="account" className="mr-4 flex-1">
            <Tabs.List>
              <Tabs.Tab
                value="account"
                title="Account concurrency"
                className="flex-none max-w-none border-r-0"
              />
            </Tabs.List>
          </Tabs>
          <Select
            disabled
            onChange={() => {}}
            isLabelVisible
            label="Time"
            multiple={false}
            value={periodOption}
          >
            <Select.Button isLabelVisible size="small">
              <div className="text-sm font-medium">{periodOption.name}</div>
            </Select.Button>
          </Select>
        </div>
        <div className="mt-6">
          {seriesError ? (
            <div className="flex h-full min-h-[384px] w-full items-center justify-center overflow-hidden">
              <Alert severity="warning">
                Failed to load concurrency data. Please{' '}
                <Alert.Link
                  severity="warning"
                  href="https://support.inngest.com"
                >
                  contact support
                </Alert.Link>{' '}
                if this does not resolve.
              </Alert>
            </div>
          ) : (
            <Chart
              option={chartOption ?? {}}
              settings={{ notMerge: true }}
              theme="light"
              className="h-[384px] w-full"
              loading={seriesFetching}
            />
          )}
        </div>
      </div>
    </div>
  );
}
