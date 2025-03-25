'use client';

import { useMemo } from 'react';
import { Pill } from '@inngest/components/Pill/Pill';
import { cn } from '@inngest/components/utils/classNames';
import { minuteTime } from '@inngest/components/utils/date';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

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
    <div className={cn('border-subtle bg-canvasBase border-b px-6 py-4', className)}>
      <header className="mb-2 flex items-center justify-between">
        <div className="flex">
          <h3 className="mr-4 flex flex-row items-center gap-2 text-base">{title}</h3>
          <Pill>{period}</Pill>
        </div>
        <div>
          <div className="text-right text-lg font-medium">{total}</div>
          <div className="text-subtle text-sm">{totalDescription}</div>
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
              <CartesianGrid strokeDasharray="0" vertical={false} className="stroke-disabled" />
              <XAxis
                allowDecimals={false}
                dataKey="name"
                axisLine={false}
                tickLine={false}
                tickSize={2}
                interval={1}
                /* @ts-ignore */
                tick={<CustomizedXAxisTick />}
              />
              <YAxis
                allowDecimals={false}
                domain={[0, 'auto']}
                allowDataOverflow
                axisLine={false}
                tickLine={false}
                /* @ts-ignore */
                tick={<CustomizedYAxisTick />}
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
                          <div key={idx} className="flex items-center font-medium">
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
                  radius={[0, 0, 0, 0]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        ) : (
          loading && <p className="text-basis text-center text-sm leading-[200px]">Loading...</p>
        )}
      </div>
    </div>
  );
}
