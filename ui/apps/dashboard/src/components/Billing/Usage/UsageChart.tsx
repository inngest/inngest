'use client';

import { Alert } from '@inngest/components/Alert';
import { Chart } from '@inngest/components/Chart/Chart';

import { createChartOptions } from './transformData';
import useGetUsageChartData from './useGetUsageChartData';

type Props = {
  includedCountLimit?: number;
  selectedPeriod: 'current' | 'previous';
  type: 'run' | 'step';
};

export default function BillableUsageChart({ includedCountLimit, selectedPeriod, type }: Props) {
  const { data, fetching } = useGetUsageChartData({
    selectedPeriod: selectedPeriod,
    type: type,
  });

  if (data.length === 0 && !fetching) {
    return (
      <div className="flex h-full min-h-[297px] w-full items-center justify-center overflow-hidden">
        <Alert severity="warning">
          Failed to load usage data. Please{' '}
          <Alert.Link severity="warning" href="/support">
            contact support
          </Alert.Link>{' '}
          if this does not resolve.
        </Alert>
      </div>
    );
  }

  const chartOption = createChartOptions(data, includedCountLimit, type);

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
