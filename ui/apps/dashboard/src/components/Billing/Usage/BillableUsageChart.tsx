'use client';

import Link from 'next/link';
import { Alert } from '@inngest/components/Alert';
import { Chart } from '@inngest/components/Chart/Chart';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { createChartOptions } from './transformData';

const GetBillableSteps = graphql(`
  query GetBillableSteps($month: Int!, $year: Int!) {
    billableStepTimeSeries(timeOptions: { month: $month, year: $year }) {
      data {
        time
        value
      }
    }
  }
`);

type Props = {
  includedStepCountLimit?: number;
  selectedPeriod: 'current' | 'previous';
  type: string;
};

export function BillableUsageChart({ includedStepCountLimit, selectedPeriod, type }: Props) {
  const currentMonthIndex = new Date().getUTCMonth();
  const options = {
    previous: {
      month: currentMonthIndex === 0 ? 12 : currentMonthIndex,
      year: currentMonthIndex === 0 ? new Date().getUTCFullYear() - 1 : new Date().getUTCFullYear(),
    },
    current: {
      month: currentMonthIndex + 1,
      year: new Date().getUTCFullYear(),
    },
  };

  const [{ data, fetching }] = useQuery({
    query: GetBillableSteps,
    variables: {
      month: options[selectedPeriod].month,
      year: options[selectedPeriod].year,
    },
  });

  if (!data && !fetching) {
    return (
      <div className="flex h-full min-h-[297px] w-full items-center justify-center overflow-hidden">
        <Alert severity="warning">
          Failed to load usage data. Please{' '}
          <Link href="/support" className="underline">
            contact support
          </Link>{' '}
          if this does not resolve.
        </Alert>
      </div>
    );
  }

  const monthData = data?.billableStepTimeSeries[0]?.data || [];
  const chartOption = createChartOptions(monthData, includedStepCountLimit, type);

  return (
    <div>
      <Chart
        option={chartOption}
        settings={{ notMerge: true }}
        theme="light"
        className="h-[297px] w-full"
        loading={fetching}
      />
    </div>
  );
}
