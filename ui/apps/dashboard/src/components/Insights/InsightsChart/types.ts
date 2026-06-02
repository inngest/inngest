export type ChartViewMode = 'table' | 'chart';

export type ChartType = 'line' | 'bar';

export interface ChartConfig {
  chartType: ChartType;
  xAxisColumn: string | null;
  yAxisColumn: string | null;
  convertXToFloat: boolean;
  convertYToFloat: boolean;
  showTooltips: boolean;
  showLabels: boolean;
}

export const DEFAULT_CHART_CONFIG: ChartConfig = {
  chartType: 'bar',
  xAxisColumn: null,
  yAxisColumn: null,
  convertXToFloat: false,
  convertYToFloat: false,
  showTooltips: true,
  showLabels: true,
};
