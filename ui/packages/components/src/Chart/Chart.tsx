'use client';

import { useEffect, useRef } from 'react';
import {
  getInstanceByDom,
  init,
  type EChartsOption,
  type LineSeriesOption,
  type SetOptionOpts,
} from 'echarts';

export type { LineSeriesOption };

export interface ChartProps {
  option: EChartsOption;
  settings?: SetOptionOpts;
  theme?: 'light' | 'dark';
  className?: string;
}

export const Chart = ({
  option,
  settings = { notMerge: true },
  theme = 'light',
  className,
}: ChartProps) => {
  const chartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (chartRef.current !== null) {
      const chart = init(chartRef.current, theme);

      const resizeChart = () => {
        chart?.resize();
      };
      window.addEventListener('resize', resizeChart);

      return () => {
        chart?.dispose();
        window.removeEventListener('resize', resizeChart);
      };
    }
  }, [theme]);

  useEffect(() => {
    if (chartRef.current !== null) {
      const chart = getInstanceByDom(chartRef.current);
      chart?.setOption(option, settings);
    }
  }, [option, settings]);

  return <div ref={chartRef} className={` ${className}`} />;
};
