import { Chart, type ChartProps } from '../Chart/Chart';
import { FunctionInfo } from './FunctionInfo';

export const FailedFunctions = () => {
  const option: ChartProps['option'] = {
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      bottom: '10%',
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      data: [
        'Web analytics',
        'Deploy notifications',
        'New lead',
        'Stripe invoice',
        'Onboarding campaign',
      ],
      textStyle: { fontSize: '12px' },
    },
    grid: {
      top: '20%',
      left: '0%',
      right: '10%',
      bottom: '20%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'],
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: 'Web analytics',
        type: 'line',
        showSymbol: false,
        stack: 'Total',
        data: [120, 132, 101, 134, 90, 230, 210],
        itemStyle: { color: '#ec9923' },
      },
      {
        name: 'Deploy notifications',
        type: 'line',
        showSymbol: false,
        stack: 'Total',
        data: [220, 182, 191, 234, 290, 330, 310],
        itemStyle: { color: '#2c9b63' },
      },
      {
        name: 'New lead',
        type: 'line',
        showSymbol: false,
        stack: 'Total',
        data: [150, 232, 201, 154, 190, 330, 410],
        itemStyle: { color: '#2389f1' },
      },
      {
        name: 'Stripe invoice',
        type: 'line',
        showSymbol: false,
        stack: 'Total',
        data: [320, 332, 301, 334, 390, 330, 320],
        itemStyle: { color: '#f54a3f' },
      },
      {
        name: 'Onboarding campaign',
        type: 'line',
        showSymbol: false,
        stack: 'Total',
        data: [820, 932, 901, 934, 1290, 1330, 1320],
        itemStyle: { color: '#6222df' },
      },
    ],
  };
  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-full flex-col overflow-hidden rounded-lg p-5">
      <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
        Failed Functions <FunctionInfo />
      </div>
      <Chart option={option} />
    </div>
  );
};
