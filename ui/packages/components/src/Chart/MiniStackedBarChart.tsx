import { cn } from '@inngest/components/utils/classNames';
import { Bar, BarChart, Cell, ResponsiveContainer } from 'recharts';

type MiniStackedBarChartProps = {
  data: {
    startCount: number;
    failureCount?: number;
    concurrencyLimitReached?: boolean;
  }[];
  className?: string;
};

export default function MiniStackedBarChart({ data, className = '' }: MiniStackedBarChartProps) {
  // Recharts doesn't support stacked bar charts with negative values, so we need to map the data
  // to a new format that works with the chart.
  const mappedData = data.map((d) => ({
    nonFailureCount: d.startCount - (d.failureCount ?? 0),
    failureCount: d.failureCount ?? 0,
    concurrencyLimitReached: d.concurrencyLimitReached ?? false,
  }));

  const cells = mappedData.map((slot, index) => (
    <Cell
      className={cn(
        'fill-primary-xSubtle',
        slot.concurrencyLimitReached && 'fill-accent-xSubtle dark:fill-accent-xIntense'
      )}
      key={index}
    />
  ));

  return (
    <div className={cn('h-8 w-40', className)}>
      <ResponsiveContainer>
        <BarChart
          data={mappedData}
          barCategoryGap={2}
          margin={{ top: 4, right: 4, bottom: 4, left: 4 }}
        >
          <Bar
            dataKey="failureCount"
            stackId="slot"
            fill={`rgb(var(--color-tertiary-subtle))`}
            radius={1}
            barSize={4}
          />
          <Bar
            dataKey="nonFailureCount"
            stackId="slot"
            fill={`rgb(var(--color-primary-xSubtle))`}
            minPointSize={1}
            barSize={4}
            radius={[1, 1, 0, 0]}
          >
            {cells}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
