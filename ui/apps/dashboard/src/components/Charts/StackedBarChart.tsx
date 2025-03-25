'use client';

import { useMemo } from 'react';
import { Error } from '@inngest/components/Error/Error';
import {
  Tooltip as CustomTooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { minuteTime } from '@inngest/components/utils/date';
import { RiInformationLine } from '@remixicon/react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import LoadingIcon from '@/icons/LoadingIcon';

type BarChartProps = {
  className?: string;
  height?: number;
  title: string | React.ReactNode;
  desc?: string;
  data?: {
    name: string;
    values: {
      [key: string]: number;
    };
  }[];
  legend: {
    /** dataKey should match the data's value map */
    dataKey: string;
    name: string;
    /** A hex color code */
    color: string;
    /** The default series to show the min bar height when no data is present */
    default?: boolean;
  }[];
  isLoading: boolean;
  error?: Error;
};

type AxisProps = {
  y: number;
  x: number;
  index: number;
  payload: {
    value: string;
  };
};

function CustomizedXAxisTick(props: AxisProps) {
  return (
    <text x={props.x} y={props.y} dy={16} fontSize={12} className="fill-muted" textAnchor="middle">
      {minuteTime(new Date(props.payload.value))}
    </text>
  );
}

function CustomizedYAxisTick(props: AxisProps) {
  return (
    <text x={props.x} y={props.y} dy={2} fontSize={12} className="fill-muted">
      {props.index > 0 ? props.payload.value : undefined}
    </text>
  );
}

export default function StackedBarChart({
  className = '',
  height = 200,
  title,
  desc,
  data = [],
  legend = [],
  error,
  isLoading,
}: BarChartProps) {
  const flattenedData = useMemo(() => data.map((d) => ({ ...d.values, name: d.name })), [data]);

  const defaultLegend = legend.find((element) => element.default);
  const defaultDataKey = defaultLegend?.dataKey;

  return (
    <div className={cn('border-subtle bg-canvasBase border-b px-6 py-4', className)}>
      <header className="mb-2 flex items-center justify-between">
        <div className="flex gap-4">
          <h3 className="flex flex-row items-center gap-2 text-base">{title}</h3>
          {desc && (
            <CustomTooltip>
              <TooltipTrigger>
                <RiInformationLine className="text-subtle h-4 w-4" />
              </TooltipTrigger>
              <TooltipContent>{desc}</TooltipContent>
            </CustomTooltip>
          )}
        </div>
        <div className="flex justify-end gap-4">
          {legend.map((l) => (
            <span key={l.name} className="inline-flex items-center text-xs">
              <span
                className="mr-2 inline-flex h-3 w-3 rounded-full"
                style={{ backgroundColor: l.color }}
              ></span>
              {l.name}
            </span>
          ))}
        </div>
      </header>
      <div style={{ minHeight: `${height}px` }}>
        <ResponsiveContainer height={height} width="100%">
          {isLoading ? (
            <div className="flex h-full w-full items-center justify-center">
              <LoadingIcon />
            </div>
          ) : error ? (
            <div className="h-full w-full">
              <Error message="Failed to load chart" />
            </div>
          ) : (
            <BarChart
              data={flattenedData}
              margin={{
                top: 16,
                bottom: 16,
              }}
            >
              <CartesianGrid strokeDasharray="0" vertical={false} className="stroke-disabled" />
              <XAxis
                allowDecimals={false}
                dataKey="name"
                axisLine={false}
                tickLine={false}
                tickSize={2}
                tick={CustomizedXAxisTick}
              />
              <YAxis
                allowDecimals={false}
                domain={[0, 'auto']}
                allowDataOverflow
                axisLine={false}
                tickLine={false}
                tick={CustomizedYAxisTick}
                width={20}
                tickMargin={8}
              />

              <Tooltip
                content={(props) => {
                  const { label, payload } = props;
                  return (
                    <div className="bg-canvasBase shadow-tooltip rounded-md px-3 pb-2 pt-1 text-sm shadow-md">
                      <div className="text-muted pb-2">{new Date(label).toLocaleString()}</div>
                      {payload?.map((p, idx) => {
                        const l = legend.find((l) => l.dataKey == p.name);
                        return (
                          <div
                            key={idx}
                            className="text-muted flex items-center text-sm font-medium"
                          >
                            <span
                              className="mr-2 inline-flex h-3 w-3 rounded"
                              style={{ backgroundColor: l?.color || p.color }}
                            ></span>
                            {p.value?.toLocaleString(undefined, {
                              notation: 'compact',
                              compactDisplay: 'short',
                            })}{' '}
                            {l?.name || p.name}
                          </div>
                        );
                      }) || ''}
                    </div>
                  );
                }}
                wrapperStyle={{ outline: 'none' }}
                cursor={false}
              />

              {legend.map((l) => (
                /* @ts-ignore */
                <Bar key={l.name} dataKey={l.dataKey} stackId="default" fill={l.color}>
                  {data.map((entry, i) => {
                    const isRadius =
                      l.default || (defaultDataKey && entry.values[defaultDataKey] === 0);

                    return (
                      // @ts-ignore
                      <Cell key={`cell-${i}`} radius={isRadius ? [3, 3, 0, 0] : undefined} />
                    );
                  })}
                </Bar>
              ))}
            </BarChart>
          )}
        </ResponsiveContainer>
      </div>
    </div>
  );
}
