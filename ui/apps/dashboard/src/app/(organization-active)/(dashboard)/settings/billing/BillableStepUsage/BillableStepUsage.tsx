'use client';

import Link from 'next/link';
import { Alert } from '@inngest/components/Alert';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import { formatXAxis, formatYAxis, toLocaleUTCDateString } from './format';
import { transformData } from './transformData';

// import { Chart } from '@inngest/components/Chart/Chart';

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

const dataKeys = {
  additionalStepCount: {
    key: 'additionalStepCount',
    title: 'Additional steps',
  },
  includedStepCount: {
    key: 'includedStepCount',
    title: 'Plan-included steps',
  },
} as const;

type Props = {
  includedStepCountLimit?: number;
  selectedPeriod: 'current' | 'previous';
};

export function BillableStepUsage({ includedStepCountLimit, selectedPeriod }: Props) {
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

  if (fetching) {
    return (
      <div className="flex h-full min-h-[297px] w-full items-center justify-center overflow-hidden">
        <LoadingIcon />
      </div>
    );
  }
  if (!data) {
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

  const monthData = data.billableStepTimeSeries[0]?.data || [];
  const { series } = transformData(monthData, includedStepCountLimit);

  return (
    <div className="text-slate-800">
      {/* <Chart option={{series: series, yAxis: {}}}></Chart> */}
      <div>
        <ResponsiveContainer height={228}>
          <BarChart data={series}>
            <CartesianGrid strokeDasharray="10 4" vertical={false} stroke="rgb(226, 232, 240)" />

            <Tooltip
              wrapperStyle={{ outline: 'none' }}
              formatter={(value) => {
                return value.toLocaleString();
              }}
              labelFormatter={(value: unknown) => {
                // Should be impossible, but "value" isn't typed so it's good to
                // check.
                if (!(value instanceof Date)) {
                  return 'Unknown';
                }

                return toLocaleUTCDateString(value);
              }}
            />

            <Legend
              align="right"
              content={({ payload = [] }) => {
                return (
                  <div className="mt-4 flex items-center">
                    <div className="flex-grow" />

                    {payload.map((entry) => {
                      return (
                        <div className="ml-4 flex items-center" key={entry.value}>
                          <span
                            className="mr-2 h-4 w-4 rounded"
                            style={{ backgroundColor: entry.color }}
                          />
                          <span className="text-sm font-medium text-slate-600">{entry.value}</span>
                        </div>
                      );
                    })}
                  </div>
                );
              }}
            />

            <XAxis
              dataKey="time"
              tick={{ fontSize: 14 }}
              tickFormatter={formatXAxis}
              tickLine={false}
            />
            <YAxis
              axisLine={false}
              tick={{ fontSize: 14 }}
              tickFormatter={formatYAxis}
              tickLine={false}
            />

            <Bar
              dataKey={dataKeys.includedStepCount.key}
              fill={colors.slate['600']}
              name={dataKeys.includedStepCount.title}
              radius={3}
              stackId="slot"
            >
              {series.map((entry, i) => {
                let radius: number[] | undefined = [3, 3, 0, 0];

                // We don't want to round the bar if there's another bar on top
                // of it.
                if (entry[dataKeys.additionalStepCount.key] > 0) {
                  radius = undefined;
                }

                return (
                  <Cell
                    key={`cell-${i}`}
                    // @ts-expect-error Prop type says it can't accept number[]
                    // but it actually can.
                    radius={radius}
                  />
                );
              })}
            </Bar>

            <Bar
              dataKey={dataKeys.additionalStepCount.key}
              fill={colors.indigo['500']}
              name={dataKeys.additionalStepCount.title}
              radius={[3, 3, 0, 0]}
              stackId="slot"
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
