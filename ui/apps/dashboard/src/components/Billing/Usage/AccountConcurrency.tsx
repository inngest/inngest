import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Select } from '@inngest/components/Select/Select';
import Tabs from '@inngest/components/Tabs/Tabs';
import { cn } from '@inngest/components/utils/classNames';
import { useQuery } from 'urql';

import { GetBurstConcurrencyDocument } from '@/components/Billing/Usage/BurstConcurrency';
import { createConcurrencyChartOptions } from '@/components/Billing/Usage/concurrencyChart';
import { graphql } from '@/gql';
import { Route as BillingRoute } from '@/routes/_authed/billing/route';
import { pathCreator } from '@/utils/urls';

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

function formatBurstLimit(burstLimit: number, baseLimit?: number) {
  if (baseLimit && burstLimit % baseLimit === 0) {
    return `${burstLimit} (${baseLimit}×${burstLimit / baseLimit})`;
  }
  return `${burstLimit}`;
}

type Props = {
  baseLimit?: number;
};

export function AccountConcurrency({ baseLimit }: Props) {
  const { profile } = BillingRoute.useLoaderData();
  const [{ data, error }] = useQuery({ query: GetBurstConcurrencyDocument });
  const burstLimit = error ? undefined : data?.account.burstConcurrency?.limit;

  const range = useMemo(() => {
    const to = new Date();
    const from = new Date(to.getTime() - periodDays * 24 * 60 * 60 * 1000);
    return { from: from.toISOString(), to: to.toISOString() };
  }, []);
  const [{ data: seriesData, error: seriesError, fetching: seriesFetching }] =
    useQuery({
      query: GetAccountConcurrencyTimeSeriesDocument,
      variables: range,
    });
  const series = seriesError
    ? undefined
    : seriesData?.account.concurrencyTimeSeries;
  const chartOption = useMemo(
    () => (series ? createConcurrencyChartOptions(series, range) : undefined),
    [series, range],
  );

  return (
    <div className="mt-10">
      <h3 className="text-basis mb-3 text-xl">Account concurrency</h3>
      <div className="mb-6 flex items-start justify-between gap-6">
        <p className="text-muted text-sm">
          The maximum number of steps your account can execute at the same time.
          Steps above this limit wait in the queue until capacity frees up.
        </p>
        <div className="flex shrink-0 gap-2">
          <Button
            appearance="outlined"
            kind="secondary"
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
      <div className="bg-canvasBase border-subtle rounded-md border px-4 py-6">
        <div className="flex items-center justify-between">
          <Tabs defaultValue="account" className="w-auto">
            <Tabs.List className="border-b-0">
              <Tabs.Tab value="account" title="Account concurrency" />
            </Tabs.List>
          </Tabs>
          <Select
            disabled
            onChange={() => {}}
            isLabelVisible
            label="Period"
            multiple={false}
            value={periodOption}
          >
            <Select.Button isLabelVisible size="small">
              <div className="text-sm font-medium">{periodOption.name}</div>
            </Select.Button>
          </Select>
        </div>
        {chartOption && (
          <div className="mt-6">
            <Chart
              option={chartOption}
              settings={{ notMerge: true }}
              theme="light"
              className="h-[384px] w-full"
              loading={seriesFetching}
            />
          </div>
        )}
        <div className="mt-6 flex flex-wrap gap-10">
          {baseLimit !== undefined && (
            <div>
              <p className="text-basis text-sm">Concurrency limit</p>
              <p className="text-basis text-3xl">{baseLimit}</p>
            </div>
          )}
          {burstLimit !== undefined && (
            <div>
              <p className="text-basis text-sm">Burst limit</p>
              <p className="text-basis text-3xl">
                {formatBurstLimit(burstLimit, baseLimit)}
              </p>
            </div>
          )}
          {series && (
            <div>
              <p className="text-basis text-sm">
                Max concurrent steps{' '}
                <span className="text-muted">[{periodOption.id}]</span>
              </p>
              <p
                className={cn(
                  'text-3xl',
                  series.peak >= series.baseLimit ? 'text-error' : 'text-basis',
                )}
              >
                {series.peak}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
