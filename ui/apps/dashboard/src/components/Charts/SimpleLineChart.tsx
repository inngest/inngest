'use client';

import { useMemo } from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import cn from '@/utils/cn';
import { hourTime } from '@/utils/date';

type SimpleLineChartProps = {
  className?: string;
  height?: number;
  title: string | React.ReactNode;
  total?: number;
  totalDescription?: string;
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

export default function SimpleLineChart({
  className = '',
  height = 200,
  title,
  total,
  totalDescription,
  data = [],
  legend = [],
}: SimpleLineChartProps) {
  const flattenData = useMemo(() => {
    return data.map((d) => ({ ...d.values, name: d.name }));
  }, [data]);

  return (
    <div className={cn('border-b border-slate-200 bg-white px-6 py-4', className)}>
      <header className="flex items-center justify-between">
        <div className="flex gap-4">
          <h3 className="flex flex-row items-center gap-2 font-medium">{title}</h3>
        </div>
        <div>
          <div className="text-right text-lg font-medium">{total}</div>
          <div className="text-sm text-slate-600">{totalDescription}</div>
        </div>
      </header>
      <div style={{ minHeight: `${height}px` }}>
        {data.length ? (
          <ResponsiveContainer height={height} width="100%">
            <LineChart data={flattenData} margin={{ top: 16, bottom: 16 }} barCategoryGap={8}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="name"
                axisLine={false}
                tickLine={false}
                tickSize={2}
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
              <Legend />

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
          </ResponsiveContainer>
        ) : (
          'Loading...'
        )}
      </div>
    </div>
  );
}
