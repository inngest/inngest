'use client';

import { useEffect, useRef } from 'react';
import {
  connect,
  getInstanceByDom,
  init,
  type EChartsOption,
  type LegendComponentOption,
  type LineSeriesOption,
  type SetOptionOpts,
} from 'echarts';

export type { LegendComponentOption, LineSeriesOption };

export interface ChartProps {
  option: EChartsOption;
  settings?: SetOptionOpts;
  theme?: 'light' | 'dark';
  className?: string;
  group?: string;
}

export const Chart = ({
  option,
  settings = { notMerge: true },
  theme = 'light',
  className,
  group,
}: ChartProps) => {
  const chartRef = useRef<HTMLDivElement>(null);

  const toggleTooltips = (show: boolean) => {
    if (chartRef.current !== null) {
      const chart = getInstanceByDom(chartRef.current);
      chart?.setOption({ tooltip: { show }, xAxis: { axisPointer: { label: { show } } } });
    }
  };

  useEffect(() => {
    if (chartRef.current !== null) {
      const chart = init(chartRef.current, theme);

      const resizeChart = () => {
        chart?.resize();
      };
      window.addEventListener('resize', resizeChart);
      window.addEventListener('navToggle', resizeChart);

      return () => {
        chart?.dispose();
        window.removeEventListener('resize', resizeChart);
        window.removeEventListener('navToggle', resizeChart);
      };
    }
  }, [theme]);

  useEffect(() => {
    if (chartRef.current !== null) {
      const chart = getInstanceByDom(chartRef.current);
      chart?.setOption(option, settings);

      if (chart && group) {
        chart.group = group;
        connect(group);
      }
    }
  }, [option, settings]);

  return (
    <div
      ref={chartRef}
      className={className}
      //
      // for grouped charts, we only want tooltips to show for chart in focus
      {...(group && {
        onMouseLeave: () => toggleTooltips(false),
        onMouseEnter: () => toggleTooltips(true),
      })}
    />
  );
};
