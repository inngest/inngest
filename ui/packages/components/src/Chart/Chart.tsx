'use client';

import { useEffect, useRef } from 'react';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import {
  connect,
  getInstanceByDom,
  init,
  type EChartsOption,
  type LegendComponentOption,
  type LineSeriesOption,
  type SetOptionOpts,
} from 'echarts';
import resolveConfig from 'tailwindcss/resolveConfig';

import tailwindConfig from '../../tailwind.config';

const {
  theme: { textColor, colors },
} = resolveConfig(tailwindConfig);

export type { LegendComponentOption, LineSeriesOption, EChartsOption };

export interface ChartProps {
  option: EChartsOption;
  settings?: SetOptionOpts;
  theme?: 'light' | 'dark';
  className?: string;
  group?: string;
  loading?: boolean;
}

export const Chart = ({
  option,
  settings = { notMerge: true },
  theme = 'light',
  className,
  group,
  loading = false,
}: ChartProps) => {
  const dark = isDark();
  const chartRef = useRef<HTMLDivElement>(null);

  const toggleTooltips = (show: boolean) => {
    if (chartRef.current !== null) {
      try {
        const chart = getInstanceByDom(chartRef.current);
        chart?.setOption({ tooltip: { show }, xAxis: { axisPointer: { label: { show } } } });
      } catch (e) {
        //
        // fast successive toggling occasionally throws errors,
        // catch them so we don't pollute sentry
        console.info('there was a problem toggling tooltip', e);
      }
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

      if (chart) {
        if (loading) {
          chart.showLoading('default', {
            text: 'Loading...',
            color: resolveColor(colors.primary.moderate, dark), // Spinner color
            textColor: resolveColor(textColor.basis, dark),
            maskColor: dark ? 'rgba(2, 2, 2, 0.8)' : 'rgba(254, 254, 254, 0.8)', // bg-canvasBase
            spinnerRadius: 10,
            lineWidth: 2,
            fontSize: 14,
          });
        } else {
          chart.hideLoading();
          if (option) {
            chart.setOption(option, settings); // Update chart with new data
          }
        }

        if (group) {
          chart.group = group;
          connect(group);
        }
      }
    }
  }, [option, settings, loading]);

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
