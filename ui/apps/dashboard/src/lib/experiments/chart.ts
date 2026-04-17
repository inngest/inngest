const MIN_CHART_HEIGHT = 120;
const ROW_HEIGHT = 36;
const MIN_Y_AXIS_WIDTH = 80;
const CHAR_WIDTH_PX = 6.5;

/**
 * Height and y-axis label width for horizontal bar charts whose rows are
 * labeled with variant names. Keeps every metric panel visually aligned.
 */
export function computeChartSizing(labels: readonly string[]): {
  chartHeight: number;
  yAxisWidth: number;
} {
  const longest = labels.reduce((max, label) => Math.max(max, label.length), 0);
  return {
    chartHeight: Math.max(MIN_CHART_HEIGHT, labels.length * ROW_HEIGHT),
    yAxisWidth: Math.max(MIN_Y_AXIS_WIDTH, longest * CHAR_WIDTH_PX),
  };
}
