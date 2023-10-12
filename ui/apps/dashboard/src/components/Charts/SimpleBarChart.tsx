'use client';

import { useMemo } from 'react';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import cn from '@/utils/cn';
import { calendarTime, hourTime } from '@/utils/date';

type NestedKeyOf<T> = {
  [Values in keyof T]: T[Values];
};

type BarChartProps = {
  className?: string;
  height?: number;
  title: string | React.ReactNode;
  period: string | '24 Hours';
  total?: number;
  totalDescription?: string;
  loading?: boolean;
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
      {hourTime(props.payload.value)}
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

export default function SimpleBarChart({
  className = '',
  height = 200,
  title,
  period,
  total,
  totalDescription,
  loading,
  data = [],
  legend = [],
}: BarChartProps) {
  const flattenedData = useMemo(() => data.map((d) => ({ ...d.values, name: d.name })), [data]);

  return (
    <div className={cn('border-b border-slate-200 bg-white px-6 py-4', className)}>
      <header className="flex items-center justify-between">
        <div className="flex">
          <h3 className="mr-4 flex flex-row items-center gap-2 font-medium">{title}</h3>
          <div className="flex items-center rounded-full bg-slate-800 px-3 py-1 text-xs capitalize leading-none text-white">
            {period}
          </div>
        </div>
        <div>
          <div className="text-right text-lg font-medium">{total}</div>
          <div className="text-sm text-slate-600">{totalDescription}</div>
        </div>
      </header>
      <div style={{ minHeight: `${height}px` }}>
        {data.length ? (
          <ResponsiveContainer height={height} width="100%">
            <BarChart
              data={flattenedData}
              margin={{
                top: 16,
                bottom: 16,
                left: 0,
                right: 0,
              }}
              barCategoryGap={8}
            >
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="name"
                axisLine={false}
                tickLine={false}
                tickSize={2}
                interval={1}
                className="text-white"
                /* @ts-ignore */
                tick={<CustomizedXAxisTick />}
              />
              <YAxis
                domain={[0, 'auto']}
                allowDataOverflow
                axisLine={false}
                tickLine={false}
                /* @ts-ignore */
                tick={<CustomizedYAxisTick />}
                width={10}
              />

              <Tooltip
                content={(props) => {
                  const { label, payload } = props;
                  return (
                    <div className="rounded-md border bg-white/90 px-3 pb-2 pt-1 text-sm shadow backdrop-blur-md">
                      <span className="text-xs text-slate-500">{calendarTime(label)}</span>
                      {payload?.map((p, idx) => {
                        const l = legend.find((l) => l.dataKey == p.name);
                        return (
                          <div key={idx} className="flex items-center font-medium text-slate-800">
                            <span
                              className="mr-2 inline-flex h-3 w-3 rounded"
                              style={{ backgroundColor: l?.color || p.color }}
                            ></span>
                            {p.value} {l?.name || p.name}
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
                <Bar
                  key={l.name}
                  dataKey={l.dataKey}
                  stackId="default"
                  fill={l.color}
                  minPointSize={1}
                  radius={[3, 3, 0, 0]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        ) : (
          loading && (
            <p className="text-smn text-center leading-[200px] text-slate-700">Loading...</p>
          )
        )}
      </div>
    </div>
  );
}
