import { useEffect, useReducer } from 'react';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import {
  DEFAULT_CHART_CONFIG,
  type ChartConfig,
  type ChartType,
} from './types';

export type ChartConfigAction =
  | { type: 'SET_CHART_TYPE'; chartType: ChartType }
  | { type: 'SET_X_AXIS'; column: string | null }
  | { type: 'SET_Y_AXIS'; column: string | null }
  | { type: 'SET_CONVERT_X_TO_FLOAT'; value: boolean }
  | { type: 'SET_CONVERT_Y_TO_FLOAT'; value: boolean }
  | { type: 'SET_SHOW_TOOLTIPS'; value: boolean }
  | { type: 'SET_SHOW_LABELS'; value: boolean }
  | { type: 'SET_DEFAULTS'; xAxis: string | null; yAxis: string | null };

function chartConfigReducer(
  state: ChartConfig,
  action: ChartConfigAction,
): ChartConfig {
  switch (action.type) {
    case 'SET_CHART_TYPE':
      return { ...state, chartType: action.chartType };
    case 'SET_X_AXIS':
      return { ...state, xAxisColumn: action.column };
    case 'SET_Y_AXIS':
      return { ...state, yAxisColumn: action.column };
    case 'SET_CONVERT_X_TO_FLOAT':
      return { ...state, convertXToFloat: action.value };
    case 'SET_CONVERT_Y_TO_FLOAT':
      return { ...state, convertYToFloat: action.value };
    case 'SET_SHOW_TOOLTIPS':
      return { ...state, showTooltips: action.value };
    case 'SET_SHOW_LABELS':
      return { ...state, showLabels: action.value };
    case 'SET_DEFAULTS':
      return { ...state, xAxisColumn: action.xAxis, yAxisColumn: action.yAxis };
    default:
      return state;
  }
}

function getDefaultAxes(columns: InsightsFetchResult['columns']) {
  const dateOrString =
    columns.find((c) => c.type === 'date') ??
    columns.find((c) => c.type === 'string');
  const numberCol = columns.find((c) => c.type === 'number');
  return {
    x: dateOrString?.name ?? columns[0]?.name ?? null,
    y: numberCol?.name ?? null,
  };
}

export function useChartConfig(data: InsightsFetchResult | undefined) {
  const [config, dispatch] = useReducer(
    chartConfigReducer,
    DEFAULT_CHART_CONFIG,
  );

  // Auto-select default axes when data columns change
  useEffect(() => {
    if (!data?.columns.length) return;
    if (config.xAxisColumn !== null || config.yAxisColumn !== null) return;

    const defaults = getDefaultAxes(data.columns);
    dispatch({ type: 'SET_DEFAULTS', xAxis: defaults.x, yAxis: defaults.y });
  }, [data?.columns, config.xAxisColumn, config.yAxisColumn]);

  return { config, dispatch };
}
