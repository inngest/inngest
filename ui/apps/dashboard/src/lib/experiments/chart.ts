const MIN_CHART_HEIGHT = 120;
const ROW_HEIGHT = 36;
const MIN_Y_AXIS_WIDTH = 80;
const MAX_Y_AXIS_WIDTH = 128;
const CHAR_WIDTH_PX = 8;

/** Maximum characters to render on a variant row label before truncating with a center ellipsis. */
export const MAX_LABEL_LENGTH = 18;

/**
 * Height and y-axis label width for horizontal bar charts whose rows are
 * labeled with variant names. Keeps every metric panel visually aligned.
 */
export function computeChartSizing(labels: readonly string[]): {
  chartHeight: number;
  yAxisWidth: number;
} {
  const longestRendered = labels.reduce(
    (max, label) => Math.max(max, Math.min(label.length, MAX_LABEL_LENGTH)),
    0,
  );
  return {
    chartHeight: Math.max(MIN_CHART_HEIGHT, labels.length * ROW_HEIGHT),
    yAxisWidth: Math.min(
      MAX_Y_AXIS_WIDTH,
      Math.max(MIN_Y_AXIS_WIDTH, longestRendered * CHAR_WIDTH_PX),
    ),
  };
}

/** Trims `s` with a center ellipsis so both the beginning and end remain visible. */
export function truncateCenter(s: string, max = MAX_LABEL_LENGTH): string {
  if (s.length <= max) return s;
  const headLen = Math.ceil((max - 1) / 2);
  const tailLen = Math.floor((max - 1) / 2);
  return `${s.slice(0, headLen)}…${s.slice(s.length - tailLen)}`;
}
