'use client';

import Link from 'next/link';
import { Alert } from '@inngest/components/Alert';
import { Chart } from '@inngest/components/Chart/Chart';

import { createChartOptions } from './transformData';
import useGetBillableSteps from './useGetBillableSteps';

type Props = {
  includedStepCountLimit?: number;
  selectedPeriod: 'current' | 'previous';
  type: string;
};

export default function BillableUsageChart({
  includedStepCountLimit,
  selectedPeriod,
  type,
}: Props) {
  const { data, fetching } = useGetBillableSteps({
    selectedPeriod: selectedPeriod,
  });

  if (data.length === 0 && !fetching) {
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

  const chartOption = createChartOptions(data, includedStepCountLimit, type);

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
