import { useMemo } from 'react';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import type { ChartConfig } from './types';

export type RechartsDataPoint = {
  name: string;
  [yAxisKey: string]: string | number;
};

export function useChartData(
  data: InsightsFetchResult | undefined,
  config: ChartConfig,
): RechartsDataPoint[] | null {
  return useMemo(() => {
    if (!data?.rows.length || !config.xAxisColumn || !config.yAxisColumn)
      return null;

    const xCol = data.columns.find((c) => c.name === config.xAxisColumn);
    const yCol = data.columns.find((c) => c.name === config.yAxisColumn);
    if (!xCol || !yCol) return null;

    const points: RechartsDataPoint[] = [];

    for (const row of data.rows) {
      let xValue = row.values[config.xAxisColumn];
      let yValue = row.values[config.yAxisColumn];

      // Format X-axis value as string
      let xLabel: string;
      if (xValue instanceof Date) {
        xLabel = xValue.toLocaleDateString(undefined, {
          month: 'short',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
        });
      } else if (config.convertXToFloat && typeof xValue === 'string') {
        const parsed = parseFloat(xValue);
        xLabel = isNaN(parsed) ? xValue : String(parsed);
      } else {
        xLabel = String(xValue ?? '');
      }

      // Parse Y-axis value as number
      let yNumeric: number;
      if (typeof yValue === 'number') {
        yNumeric = yValue;
      } else if (yValue instanceof Date) {
        yNumeric = yValue.getTime();
      } else if (config.convertYToFloat && typeof yValue === 'string') {
        const parsed = parseFloat(yValue);
        yNumeric = isNaN(parsed) ? 0 : parsed;
      } else if (typeof yValue === 'string') {
        const parsed = parseFloat(yValue);
        yNumeric = isNaN(parsed) ? 0 : parsed;
      } else {
        yNumeric = 0;
      }

      points.push({
        name: xLabel,
        [config.yAxisColumn]: yNumeric,
      });
    }

    return points;
  }, [
    data,
    config.xAxisColumn,
    config.yAxisColumn,
    config.convertXToFloat,
    config.convertYToFloat,
  ]);
}
