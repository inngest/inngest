'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ChartBarIcon } from '@heroicons/react/20/solid';
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

import { Alert } from '@/components/Alert';
import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import { StepCounter } from './StepCounter';
import { formatXAxis, formatYAxis, toLocaleUTCDateString } from './format';
import { transformData } from './transformData';

const GetBillableSteps = graphql(`
  query GetBillableSteps($month: Int!) {
    billableStepTimeSeries(timeOptions: { month: $month }) {
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
  // Step count included in the plan.
  includedStepCountLimit?: number;
};

export function BillableStepUsage({ includedStepCountLimit }: Props) {
  const currentMonthIndex = new Date().getUTCMonth();
  const options = {
    prevMonth: currentMonthIndex === 0 ? 11 : currentMonthIndex,
    thisMonth: currentMonthIndex + 1,
  };

  const [selectedMonth, setSelectedMonth] = useState<'prevMonth' | 'thisMonth'>('thisMonth');
  const [{ data, fetching }] = useQuery({
    query: GetBillableSteps,
    variables: {
      month: options[selectedMonth],
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

  const monthData = data?.billableStepTimeSeries?.[0]?.data || [];
  const { additionalStepCount, series, totalStepCount } = transformData(
    monthData,
    includedStepCountLimit
  );

  return (
    <div className="text-slate-800">
      <div className="mb-4 flex items-center justify-end gap-x-8">
        <div className="flex text-lg text-slate-600">
          <ChartBarIcon className="mr-2 w-5" />
          <span className="font-medium">Function Usage</span>
        </div>

        <div className="flex-grow" />

        <StepCounter count={includedStepCountLimit} title="Plan-included steps" />
        <StepCounter
          count={additionalStepCount}
          numberClassName="text-indigo-500"
          title="Additional steps"
        />
        <StepCounter count={totalStepCount} title="Total steps" />
      </div>

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
                    <select
                      className="font-regular shadow-outline-secondary-light inline-flex flex-shrink-0 items-center justify-center gap-1 overflow-hidden rounded-[6px] bg-white text-sm font-medium text-slate-700 transition-all"
                      onChange={(event) => {
                        const { value } = event.target;
                        if (value !== 'thisMonth' && value !== 'prevMonth') {
                          throw new Error(`invalid value: ${value}`);
                        }

                        setSelectedMonth(value);
                      }}
                      value={selectedMonth}
                    >
                      <option value="thisMonth">This Month</option>

                      <option value="prevMonth">Previous Month</option>
                    </select>

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
