'use client';

import { useMemo } from 'react';
import { Error } from '@inngest/components/Error/Error';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { minuteTime } from '@inngest/components/utils/date';
import { RiInformationLine } from '@remixicon/react';
import {
  CartesianGrid,
  Tooltip as ChartTooltip,
  Line,
  LineChart,
  ReferenceArea,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from 'recharts';

import LoadingIcon from '@/icons/LoadingIcon';

type SimpleLineChartProps = {
  className?: string;
  height?: number;
  title: string | React.ReactNode;
  desc?: string;
  data?: {
    name: string;
    values: {
      [key: string]: number | boolean;
    };
  }[];
  legend: {
    dataKey: string;
    name: string;
    color: string;
    default?: boolean;
    referenceArea?: boolean;
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

function omit(obj: Record<string, any>, props: string[]) {
  obj = { ...obj };
  props.forEach((prop) => delete obj[prop]);
  return obj;
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
  const referenceAreas = useMemo(() => legend.filter((k) => k.referenceArea), [legend]);
  const referenceAreaKeys = referenceAreas.map((k) => k.dataKey);
  const flattenData = useMemo(() => {
    return data.map((d) => {
      const values = omit(d.values, referenceAreaKeys);
      return { ...values, name: d.name };
    });
  }, [data, referenceAreaKeys]);

  return (
    <div className={cn('border-subtle bg-canvasBase border-b px-6 py-4', className)}>
      <header className="mb-2 flex items-center justify-between">
        <div className="flex gap-2">
          <h3 className="flex flex-row items-center gap-1.5 text-base">{title}</h3>
          {desc && (
            <Tooltip>
              <TooltipTrigger>
                <RiInformationLine className="text-subtle h-4 w-4" />
              </TooltipTrigger>
              <TooltipContent>{desc}</TooltipContent>
            </Tooltip>
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
            <LineChart data={flattenData} margin={{ top: 16, bottom: 16 }} barCategoryGap={8}>
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
              <ReferenceArea
                y1={0}
                y2={1}
                x1="2023-10-17T12:30:00Z"
                x2="2023-10-17T12:30:00Z"
                stroke="red"
                fill="red"
              />
              {referenceAreas.map((k) =>
                data.map(({ name, values }, index) => {
                  if (!values[k.dataKey]) return;
                  return (
                    <ReferenceArea
                      key={`${name}+${k.dataKey}`}
                      x1={name}
                      x2={index < data.length - 1 ? data[index + 1]!.name : name}
                      fill={k.color}
                      fillOpacity={0.15}
                    />
                  );
                })
              )}
              <ChartTooltip
                content={(props) => {
                  const { label, payload } = props;
                  return (
                    <div className="shadow-tooltip bg-canvasBase rounded-md px-3 pb-2 pt-1 text-sm shadow-md">
                      <div className="text-muted pb-2">{new Date(label).toLocaleString()}</div>
                      {payload?.map((p, idx) => {
                        // @ts-ignore
                        if (p.value === false) return;
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
                            {typeof p.value === 'number'
                              ? p.value.toLocaleString(undefined, {
                                  notation: 'compact',
                                  compactDisplay: 'short',
                                })
                              : ''}{' '}
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
