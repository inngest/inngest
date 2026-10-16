import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Select, type Option } from '@inngest/components/Select/Select';
import Tabs from '@inngest/components/Tabs/Tabs';
import { useQuery } from 'urql';

import {
  HOUR_MS,
  createConcurrencyChartOptions,
} from '@/components/Billing/Usage/concurrencyChart';
import { graphql } from '@/gql';
import { Route as BillingRoute } from '@/routes/_authed/billing/route';
import { pathCreator } from '@/utils/urls';

const GetAccountConcurrencyTimeSeriesDocument = graphql(`
  query GetAccountConcurrencyTimeSeries($from: Time!, $to: Time!) {
    account {
      concurrencyTimeSeries(from: $from, to: $to) {
        baseLimit
        burstLimit
        buckets {
          bucket
          value
        }
        limitHits {
          bucket
          value
        }
      }
    }
  }
`);

const ref = 'app-billing-usage-concurrency';
const periodOptions = [
  { id: '24h', name: '24 hours', hours: 24 },
  { id: '7d', name: '7 days', hours: 7 * 24 },
  { id: '30d', name: '30 days', hours: 30 * 24 },
];

function burstMultiplier(burstLimit: number, baseLimit: number) {
  if (burstLimit % baseLimit === 0) {
    return `${baseLimit}*${burstLimit / baseLimit}`;
  }
  return undefined;
}

type TileProps = {
  title: React.ReactNode;
  children: React.ReactNode;
};

function Tile({ title, children }: TileProps) {
  return (
    <div className="bg-canvasBase border-subtle w-[245px] rounded-md border px-4 py-4">
      <p className="text-subtle text-sm">{title}</p>
      <p className="text-basis mt-1 text-2xl">{children}</p>
    </div>
  );
}

export function AccountConcurrency() {
  const { profile } = BillingRoute.useLoaderData();
  const [period, setPeriod] = useState(periodOptions[2]);
  const range = useMemo(() => {
    const to = Math.floor(Date.now() / HOUR_MS) * HOUR_MS;
    return {
      from: new Date(to - period.hours * HOUR_MS).toISOString(),
      to: new Date(to).toISOString(),
    };
  }, [period]);
  const [{ data, error, fetching, operation }] = useQuery({
    query: GetAccountConcurrencyTimeSeriesDocument,
    variables: range,
  });
  const series = data?.account.concurrencyTimeSeries;

  if (error || !series || !operation || series.burstLimit == null) {
    return null;
  }
  const { baseLimit, burstLimit } = series;
  const multiplier = burstMultiplier(burstLimit, baseLimit);
  const fetchedRange = operation.variables;

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
        <Tile title="Account concurrency">{baseLimit} steps</Tile>
        <Tile title="Concurrency during burst">
          {burstLimit} steps
          {multiplier && (
            <span className="text-muted ml-2 text-xs">({multiplier})</span>
          )}
        </Tile>
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
            onChange={(value: Option) =>
              setPeriod(periodOptions.find((o) => o.id === value.id) ?? period)
            }
            isLabelVisible
            label="Time"
            multiple={false}
            value={period}
          >
            <Select.Button isLabelVisible size="small">
              <div className="text-sm font-medium">{period.name}</div>
            </Select.Button>
            <Select.Options>
              {periodOptions.map((option) => (
                <Select.Option key={option.id} option={option}>
                  {option.name}
                </Select.Option>
              ))}
            </Select.Options>
          </Select>
        </div>
        <div className="mt-6">
          <Chart
            option={createConcurrencyChartOptions(series, fetchedRange)}
            className="h-[384px] w-full"
            loading={fetching}
          />
        </div>
      </div>
    </div>
  );
}
