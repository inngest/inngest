import type { ExperimentVariantMetrics } from '@inngest/components/Experiments';

import {
  colorForVariant,
} from '@/lib/experiments/colors';
import type { RowData } from '@/components/InsightsMetrics/BoxPlot';

export function rowsForMetric(
  variants: ExperimentVariantMetrics[],
  metricKey: string,
  colorIndexForVariant?: Map<string, number>,
): RowData[] {
  return variants
    .map((v, variantIndex): RowData | null => {
      const m = v.metrics.find((vm) => vm.key === metricKey);
      const colorIndex = colorIndexForVariant?.get(v.variantName) ?? variantIndex;
      return m
        ? {
            label: v.variantName,
            count: v.runCount,
            avg: m.avg,
            min: m.min,
            q1: m.q1,
            med: m.med,
            q3: m.q3,
            max: m.max,
            color: colorForVariant(colorIndex),
            subtleColor: colorForVariant(colorIndex),
            opacity: 1,
          }
        : null;
    })
    .filter((r): r is RowData => r !== null);
}
