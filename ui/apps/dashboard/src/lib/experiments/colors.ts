import { CHART_COLORS, CHART_COLORS_SUBTLE } from "@/components/InsightsMetrics/colors";

export function colorForMetric(index: number): string {
  return CHART_COLORS[index % CHART_COLORS.length];
}

export function colorForVariant(index: number): string {
  return CHART_COLORS[index % CHART_COLORS.length];
}

export function subtleColorForVariant(index: number): string {
  return CHART_COLORS_SUBTLE[index % CHART_COLORS_SUBTLE.length];
}

type MetricLike = { key: string };

/**
 * Maps each metric's key to its chart color. Colors are assigned by position in
 * the full metrics list so a metric keeps the same color regardless of whether
 * it (or any other metric) is enabled. The Score Summary chart builds its
 * segment colors from the same map, so the two views stay in sync.
 */
export function buildMetricColorMap(
  metrics: MetricLike[],
): Record<string, string> {
  const map: Record<string, string> = {};
  metrics.forEach((metric, index) => {
    map[metric.key] = colorForMetric(index);
  });
  return map;
}
