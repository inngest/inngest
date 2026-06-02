import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { InsightsChartEmptyState } from './InsightsChartEmptyState';
import type { ChartConfig } from './types';
import { useChartData } from './useChartData';

const CHART_COLOR = 'rgb(44, 155, 99)'; // primary-moderate (matcha-500)

type CustomTooltipProps = {
  active?: boolean;
  payload?: Array<{ name: string; value: number }>;
  label?: string;
};

function ChartTooltip({ active, payload, label }: CustomTooltipProps) {
  if (!active || !payload?.length) return null;

  return (
    <div className="bg-canvasBase shadow-tooltip rounded-md px-3 pb-2 pt-1 text-sm shadow-md">
      <div className="text-muted pb-2">{label}</div>
      {payload.map((p, idx) => (
        <div key={idx} className="text-basis flex items-center font-medium">
          <span
            className="mr-2 inline-flex h-3 w-3 rounded"
            style={{ backgroundColor: CHART_COLOR }}
          />
          {typeof p.value === 'number'
            ? p.value.toLocaleString(undefined, {
                notation: 'compact',
                compactDisplay: 'short',
              })
            : p.value}{' '}
          {p.name}
        </div>
      ))}
    </div>
  );
}

type InsightsChartRendererProps = {
  data: InsightsFetchResult;
  config: ChartConfig;
};

export function InsightsChartRenderer({
  data,
  config,
}: InsightsChartRendererProps) {
  const chartData = useChartData(data, config);

  if (!chartData || chartData.length === 0) {
    return <InsightsChartEmptyState reason="no-plottable-data" />;
  }

  const yAxisKey = config.yAxisColumn!;

  const sharedAxisProps = {
    axisLine: false,
    tickLine: false,
    tick: { fontSize: 12, className: 'fill-muted' },
  };

  return (
    <div className="flex h-full items-center justify-center p-4">
      <ResponsiveContainer width="100%" height="100%">
        {config.chartType === 'line' ? (
          <LineChart
            data={chartData}
            margin={{ top: 16, right: 16, bottom: 16, left: 0 }}
          >
            <CartesianGrid
              strokeDasharray="0"
              vertical={false}
              className="stroke-disabled"
            />
            <XAxis
              dataKey="name"
              {...sharedAxisProps}
              tickSize={2}
              interval="preserveStartEnd"
            />
            <YAxis
              allowDecimals={false}
              domain={[0, 'auto']}
              allowDataOverflow
              {...sharedAxisProps}
              width={50}
              tickMargin={8}
            />
            {config.showTooltips && (
              <Tooltip
                content={<ChartTooltip />}
                wrapperStyle={{ outline: 'none' }}
                cursor={false}
              />
            )}
            <Line
              type="monotone"
              dataKey={yAxisKey}
              stroke={CHART_COLOR}
              strokeWidth={2}
              dot={false}
              label={
                config.showLabels
                  ? {
                      position: 'top' as const,
                      fontSize: 11,
                      className: 'fill-muted',
                    }
                  : false
              }
            />
          </LineChart>
        ) : (
          <BarChart
            data={chartData}
            margin={{ top: 16, right: 16, bottom: 16, left: 0 }}
          >
            <CartesianGrid
              strokeDasharray="0"
              vertical={false}
              className="stroke-disabled"
            />
            <XAxis
              dataKey="name"
              {...sharedAxisProps}
              tickSize={2}
              interval="preserveStartEnd"
            />
            <YAxis
              allowDecimals={false}
              domain={[0, 'auto']}
              allowDataOverflow
              {...sharedAxisProps}
              width={50}
              tickMargin={8}
            />
            {config.showTooltips && (
              <Tooltip
                content={<ChartTooltip />}
                wrapperStyle={{ outline: 'none' }}
                cursor={false}
              />
            )}
            <Bar
              dataKey={yAxisKey}
              fill={CHART_COLOR}
              radius={[4, 4, 0, 0]}
              label={
                config.showLabels
                  ? {
                      position: 'top' as const,
                      fontSize: 11,
                      className: 'fill-muted',
                    }
                  : false
              }
            />
          </BarChart>
        )}
      </ResponsiveContainer>
    </div>
  );
}
