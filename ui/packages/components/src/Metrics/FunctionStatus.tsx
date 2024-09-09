import { color } from 'framer-motion';

import { Chart, type ChartProps } from '../Chart/Chart';
import { FunctionInfo } from './FunctionInfo';

const data = [
  { value: 1048, name: 'Completed', itemStyle: { color: '#2c9b63' } },
  { value: 735, name: 'Running', itemStyle: { color: '#52b2fd' } },
  { value: 580, name: 'Queued', itemStyle: { color: '#8b74f9' } },
  { value: 484, name: 'Cancelled', itemStyle: { color: '#e2e2e2' } },
  { value: 300, name: 'Failed', itemStyle: { color: '#fa8d86' } },
];

const sum = data.reduce((acc, { value }) => acc + value, 0);

const percent = (part: number) => `${((part / sum) * 100).toFixed(0)}%`;
const holeLabel = {
  rich: {
    name: {
      fontSize: 12,
      lineHeight: 16,
    },
    value: {
      fontSize: 16,
      lineHeight: 20,
      fontWeight: 500,
    },
  },
};

export const FunctionStatus = () => {
  const option: ChartProps['option'] = {
    legend: {
      orient: 'vertical',
      right: '10%',
      top: 'center',
      icon: 'circle',
      formatter: (name: string) =>
        [name, percent(data.find((d) => d.name === name)?.value || 0)].join(' '),
    },

    series: [
      {
        name: 'Function Runs',
        type: 'pie',
        radius: ['35%', '60%'],
        center: ['25%', '50%'],
        itemStyle: {
          borderColor: '#fff',
          borderWidth: 2,
        },
        labelLayout: ({ dataIndex }) => {
          return { hideOverlap: true };
        },
        avoidLabelOverlap: true,
        label: {
          show: true,
          position: 'center',
          formatter: (): string => {
            return [`{name| Total runs}`, `{value| ${sum}}`].join('\n');
          },
          ...holeLabel,
        },
        emphasis: {
          label: {
            show: true,
            formatter: ({ data }: any): string => {
              return [`{name| ${data?.name}}`, `{value| ${data?.value}}`].join('\n');
            },
            backgroundColor: '#fff',
            width: 80,
            ...holeLabel,
          },
        },
        labelLine: {
          show: false,
        },
        data,
      },
    ],
  };
  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-[448px] shrink-0 flex-col rounded-lg p-5">
      <div className="text-subtle flex flex-row items-center gap-x-2 text-lg">
        Functions Status <FunctionInfo />
      </div>
      <Chart option={option} />
    </div>
  );
};
