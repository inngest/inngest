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

export type { LineSeriesOption, LegendComponentOption };

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

  const toggleTooltip = (show: boolean) => {
    if (chartRef.current !== null) {
      const chart = getInstanceByDom(chartRef.current);
      chart?.setOption({ tooltip: { show } });
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
      {...(group && { onMouseEnter: () => toggleTooltip(true) })}
      {...(group && { onMouseLeave: () => toggleTooltip(false) })}
    />
  );
};
