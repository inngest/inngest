'use client';

import { useMemo } from 'react';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { InformationCircleIcon } from '@heroicons/react/24/outline';
import { Tooltip } from '@inngest/components/Tooltip';
import {
  CartesianGrid,
  Tooltip as ChartTooltip,
  Line,
  LineChart,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from 'recharts';

import LoadingIcon from '@/icons/LoadingIcon';
import cn from '@/utils/cn';
import { minuteTime } from '@/utils/date';

type SimpleLineChartProps = {
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
    dataKey: string;
    name: string;
    color: string;
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
    <text
      x={props.x}
      y={props.y}
      dy={16}
      fill="#94A3B8"
      fontSize={10}
      className="font-medium"
      textAnchor="middle"
    >
      {minuteTime(props.payload.value)}
    </text>
  );
}

function CustomizedYAxisTick(props: AxisProps) {
  return (
    <text x={props.x} y={props.y} dy={16} fill="#94A3B8" fontSize={10} className="font-medium">
      {props.index > 0 ? props.payload.value : undefined}
    </text>
  );
}

export default function SimpleLineChart({
  className = '',
  height = 200,
  title,
  desc,
  data = [],
  legend = [],
  isLoading,
  error,
}: SimpleLineChartProps) {
  const flattenData = useMemo(() => {
    return data.map((d) => ({ ...d.values, name: d.name }));
  }, [data]);

  return (
    <div className={cn('border-b border-slate-200 bg-white px-6 py-4', className)}>
      <header className="flex items-center justify-between">
        <div className="flex gap-2">
          <h3 className="flex flex-row items-center gap-1.5 font-medium">{title}</h3>
          {desc && (
            <Tooltip content={desc}>
              <InformationCircleIcon className="h-4 w-4 text-slate-400" />
            </Tooltip>
          )}
        </div>
        <div className="flex justify-end gap-4">
          {legend.map((l) => (
            <span key={l.name} className="inline-flex items-center text-sm text-slate-800">
              <span
                className="mr-2 inline-flex h-3 w-3 rounded"
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
            <div className="flex h-full w-full flex-col items-center justify-center gap-5">
              <div className="inline-flex items-center gap-2 text-red-600">
                <ExclamationCircleIcon className="h-4 w-4" />
                <h2 className="text-sm">Failed to load chart</h2>
              </div>
            </div>
          ) : (
            <LineChart data={flattenData} margin={{ top: 16, bottom: 16 }} barCategoryGap={8}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="name"
                axisLine={false}
                tickLine={false}
                tickSize={2}
                tick={CustomizedXAxisTick}
              />
              <YAxis
                domain={[0, 'auto']}
                allowDataOverflow
                axisLine={false}
                tickLine={false}
                tick={CustomizedYAxisTick}
                width={10}
              />
              <ChartTooltip
                content={(props) => {
                  const { label, payload } = props;
                  return (
                    <div className="rounded-md border bg-white/90 px-3 pb-2 pt-1 text-sm shadow backdrop-blur-md">
                      <span className="text-xs text-slate-500">
                        {new Date(label).toLocaleString()}
                      </span>
                      {payload?.map((p, idx) => {
                        const l = legend.find((l) => l.dataKey == p.name);
                        return (
                          <div
                            key={idx}
                            className="flex items-center text-sm font-medium text-slate-800"
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
                <Line
                  dot={false}
                  key={l.name}
                  type="monotone"
                  dataKey={l.dataKey}
                  stroke={l.color}
                />
              ))}
            </LineChart>
          )}
        </ResponsiveContainer>
      </div>
    </div>
  );
}
