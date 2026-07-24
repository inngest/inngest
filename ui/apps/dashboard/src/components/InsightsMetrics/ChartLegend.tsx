import { lineColors } from '@/components/Metrics/utils';
import { LegendSkeleton } from './ChartSkeleton';
import { toCssColor } from './colors';
import { valuesToMap, type InsightsMetricItem } from './types';

type Props = {
  items: InsightsMetricItem[] | undefined;
  valueName: string;
  // Overrides the single shared swatch color with one distinct color per
  // row, in the same sorted order as CategoricalChart's `colors` prop —
  // pass the same palette to both so swatches match their bars.
  colors?: readonly (readonly [string, string])[];
  format?: (value: number) => string;
  // Renders the identifier — callers use this to resolve a function slug to
  // a display name/link, matching RankedTable's convention.
  renderIdentifier?: (identifier: string) => React.ReactNode;
  isLoading?: boolean;
  className?: string;
};

const defaultFormat = (value: number) => value.toLocaleString();

// ChartLegend renders a CategoricalChart's single-measure ranking as a
// legend list — a swatch (matching the chart's bar hue: one shared color by
// default, since this is a magnitude comparison, or one per row when the
// caller passes `colors`), the identifier, and its value — meant to sit
// beside the chart it describes. Not a sortable/bordered data grid: no
// headers, no borders, no pagination.
export function ChartLegend({
  items,
  valueName,
  colors,
  format = defaultFormat,
  renderIdentifier,
  isLoading = false,
  className,
}: Props) {
  const sharedSwatchColor = toCssColor(lineColors[2][0]);

  const sorted = [...(items ?? [])].sort((a, b) => {
    const av = valuesToMap(a.values).get(valueName) ?? 0;
    const bv = valuesToMap(b.values).get(valueName) ?? 0;
    return bv - av;
  });

  if (sorted.length === 0) {
    return (
      <div className={className}>
        <LegendSkeleton animate={isLoading} />
      </div>
    );
  }

  return (
    <ul className={`flex flex-col gap-3 ${className ?? ''}`}>
      {sorted.map((item, idx) => {
        const value = valuesToMap(item.values).get(valueName);
        const swatchColor = colors ? toCssColor(colors[idx % colors.length][0]) : sharedSwatchColor;
        return (
          <li key={item.identifier} className="flex items-center gap-2 text-sm">
            <span
              className="h-2.5 w-2.5 shrink-0"
              style={{ backgroundColor: swatchColor }}
            />
            <span className="text-basis min-w-0 flex-1 truncate">
              {renderIdentifier ? renderIdentifier(item.identifier) : item.identifier}
            </span>
            <span className="text-muted shrink-0 tabular-nums">
              {value === undefined ? '—' : format(value)}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
