import { useEffect, useRef } from 'react';
import { getInstanceByDom, init, type EChartsOption, type SetOptionOpts } from 'echarts';

export interface ChartProps {
  option: EChartsOption;
  settings?: SetOptionOpts;
  theme?: 'light' | 'dark';
}

export const Chart = ({ option, settings, theme = 'light' }: ChartProps) => {
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

  return <div ref={chartRef} className="h-full w-full" />;
};
