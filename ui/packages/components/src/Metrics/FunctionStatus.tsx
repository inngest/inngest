import { Chart, type ChartProps } from '../Chart/Chart';
import { FunctionInfo } from './FunctionInfo';

const data = [
  { value: 1048, name: 'Completed' },
  { value: 735, name: 'Running' },
  { value: 580, name: 'Queued' },
  { value: 484, name: 'Cancelled' },
  { value: 300, name: 'Failed' },
];

const sum = data.reduce((acc, { value }) => acc + value, 0);

const percent = (part: number) => `${((part / sum) * 100).toFixed(0)}%`;

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
      radius: ['45%', '80%'],
      center: ['25%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: {
        borderColor: '#fff',
        borderWidth: 2,
      },
      label: {
        show: false,
        position: 'center',
      },
      emphasis: {
        label: {
          show: true,
          fontSize: 10,
          formatter: ({ data }: { data: { name: string; value: string } }) => {
            return [`{name| ${data.name}}`, `{value| ${data.value}}`].join('\n');
          },
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
        },
      },
      labelLine: {
        show: false,
      },
      data,
    },
  ],
};

export const FunctionStatus = () => (
  <div className="bg-canvasBase border-subtle flex h-[300px] w-[448px] flex-col rounded-lg p-5">
    <div className="text-subtle flex flex-row items-center gap-x-2 text-lg">
      Function Status <FunctionInfo />
    </div>
    <Chart option={option} />
  </div>
);
