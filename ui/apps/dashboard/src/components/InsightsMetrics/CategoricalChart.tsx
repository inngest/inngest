import { useMemo } from 'react';
import {
  Bar,
  BarChart,
  Cell,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import { truncateCenter } from '@/lib/experiments/chart';
import { RankedChartSkeleton } from './ChartSkeleton';
import { BORDER_SUBTLE_COLOR, CHART_COLORS } from './colors';
import type { TrendSeriesConfig } from './TrendChart';
import { valuesToMap, type InsightsMetricItem, type TooltipExtra } from './types';

type Props = {
  items: InsightsMetricItem[] | undefined;
  // Single-measure form: one bar per category, sharing one hue (a magnitude
  // comparison, not an identity comparison — see the module doc below).
  // Mutually exclusive with `series`; exactly one must be given.
  valueName?: string;
  valueLabel?: string;
  // Overrides the default single shared magnitude hue (CHART_COLORS[1])
  // — single-measure form only; ignored when `colors` or `series` is given.
  color?: string;
  // Overrides the single shared magnitude hue with one distinct color per
  // bar/category, in category (sorted) order — for callers where each
  // category is itself worth distinguishing by color (e.g. a fixed top-N
  // ranking), not just by its position on the axis. Single-measure form
  // only; ignored when `series` is given (which already colors by series).
  colors?: readonly string[];
  // Single-measure form's border opt-in, mirroring `series[].borderColor` (see
  // TrendChart's TrendSeriesConfig): when set, each bar draws a solid border
  // in this color on top of `color`'s fill — pass a distinct fill color via
  // `color` alongside this for a fill/solid-border look, rather than relying
  // on any automatic derivation. Ignored when `series` is given.
  borderColor?: string;
  // Per-category counterpart to `borderColor`, mirroring `colors` — one
  // border color per bar/category, layered over `colors`' per-category fill.
  // Ignored when `series` is given.
  borderColors?: readonly string[];
  // Multi-measure form: `series.length` bars per category (e.g. input vs
  // output tokens by model), one categorical hue per series — this IS an
  // identity comparison, so every series gets a legend and a fixed-order hue.
  series?: TrendSeriesConfig[];
  // Stack every series into one bar per category instead of grouping them
  // side by side (multi-measure form only).
  stacked?: boolean;
  format?: (value: number) => string;
  // Maps an item's raw identifier to its y-axis display label — e.g.
  // resolving a function slug to its human-readable name. Defaults to the
  // identifier as-is (already human-readable for callers like model names).
  formatIdentifier?: (identifier: string) => string;
  // Hides the vertical y-axis line (ticks/labels stay) — for callers where
  // it's redundant against the chart's own card border.
  showYAxisLine?: boolean;
  // Shows each bar's formatted value(s) as a persistent label to its right,
  // instead of only on hover via the tooltip. Single-measure form: just that
  // series' value. Multi-measure form: only when `stacked` (a grouped,
  // non-stacked bar has no single row position to attach a combined label
  // to) — every series' value in declared order, joined by " | " (e.g.
  // "1.2K | 856" for input | output tokens).
  showValueLabels?: boolean;
  // Appends each series' name after its value in the hover tooltip — for
  // single-measure charts whose measure is already obvious from context
  // (e.g. "Cost by function", where every bar can only be cost), turning
  // this off keeps the tooltip to just the formatted value. Multi-measure
  // charts still need this to tell series apart, so it defaults to true.
  showTooltipValueName?: boolean;
  // Extra value(s) to show in the hover tooltip beyond what's plotted —
  // e.g. token counts alongside a "Cost by model" chart's cost bars. Reads
  // straight off each item's `values` (every named value is carried through
  // to chartData, not just the plotted series), so any column the query
  // returns works here without adding a visual bar/segment for it.
  tooltipExtras?: TooltipExtra[];
  isLoading?: boolean;
  className?: string;
};

const defaultFormat = (value: number) => value.toLocaleString();

// Every bar — stacked, grouped, or single — renders at this thickness, so
// switching between forms doesn't change a chart's row height. The custom
// stacked-segment shape (see below) draws its outline strokes at this same
// thickness so they line up with the fill bars exactly.
const BAR_THICKNESS = 20;
const STACKED_BAR_CATEGORY_GAP = '55%';

// showValueLabels' per-series value columns: each value's own fixed-width
// column, plus a "|" separator centered in the gap between columns.
const VALUE_COLUMN_WIDTH = 36;
const VALUE_COLUMN_GAP = 14;
const VALUE_COLUMN_STEP = VALUE_COLUMN_WIDTH + VALUE_COLUMN_GAP;

type ChartRow = { identifier: string; [valueName: string]: string | number };

type BarShapeProps = {
  x: number;
  y: number;
  width: number;
  height: number;
  fill: string;
};

// stackedSegmentShape draws one stacked bar segment: a plain fill rect plus
// a manually-drawn outline — a stacked bar's segments sit edge-to-edge, so a
// full 4-sided border on every segment would double the shared seam between
// adjacent segments. Every segment draws its top/left/bottom edges; only the
// last (rightmost) segment also draws its right edge, since that's the
// stack's true outer edge and isn't shared with a neighbor.
function stackedSegmentShape(props: BarShapeProps, isLastSegment: boolean, stroke: string) {
  const { x, y, width, height, fill } = props;
  const outline = `M ${x + width} ${y} L ${x} ${y} L ${x} ${y + height} L ${x + width} ${y + height}${
    isLastSegment ? ` L ${x + width} ${y}` : ''
  }`;
  return (
    <g>
      <rect x={x} y={y} width={width} height={height} fill={fill} shapeRendering="crispEdges" />
      <path d={outline} fill="none" stroke={stroke} strokeWidth={1} />
    </g>
  );
}

type TooltipPayloadEntry = {
  dataKey?: string;
  name?: string;
  value?: number;
  color?: string;
  // recharts attaches the full chartData row here regardless of which
  // dataKey the entry itself plots — this is how tooltipExtras reads a
  // column (e.g. cost) that has no Bar of its own.
  payload?: ChartRow;
};

function CategoricalTooltip({
  active,
  payload,
  label,
  formatIdentifier,
  format,
  colorByKey,
  colorByIdentifier,
  showValueLabel,
  tooltipExtras,
}: {
  active?: boolean;
  payload?: TooltipPayloadEntry[];
  label?: string;
  formatIdentifier: (identifier: string) => string;
  format: (value: number) => string;
  // Overrides recharts' default per-entry color (a Bar's fill) with the
  // bar's border color instead — keyed by dataKey, so the tooltip's swatch
  // matches the chart's own legend rather than the (possibly muted) fill.
  colorByKey?: Record<string, string>;
  // Overrides recharts' default color with each bar's own per-category
  // color — keyed by identifier (the hovered category), for single-measure
  // charts using the `colors` palette, since recharts' payload color
  // reflects the Bar's base fill, not its per-item Cell override.
  colorByIdentifier?: Record<string, string>;
  // Whether to append each entry's series name after its value — recharts
  // falls back to the raw dataKey (e.g. "cost") whenever the Bar's own
  // `name` prop is falsy, so an absent/empty `valueLabel` alone can't
  // suppress this; the caller has to opt out explicitly instead.
  showValueLabel: boolean;
  tooltipExtras?: TooltipExtra[];
}) {
  if (!active || !payload?.length) return null;
  const row = payload[0]?.payload;
  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 pb-2 pt-1 text-sm shadow-md">
      <div className="text-muted pb-2">{formatIdentifier(label ?? '')}</div>
      {payload.map((p, idx) => {
        const swatchColor =
          (p.dataKey && colorByKey?.[p.dataKey]) ?? (label && colorByIdentifier?.[label]) ?? p.color;
        return (
          <div key={idx} className="text-basis flex items-center font-medium">
            {swatchColor && (
              <span
                className="mr-2 inline-flex h-3 w-3 shrink-0"
                style={{ backgroundColor: swatchColor }}
              />
            )}
            <span className="truncate">
              {typeof p.value === 'number' ? format(p.value) : p.value}
              {showValueLabel && p.name ? ` ${p.name}` : ''}
            </span>
          </div>
        );
      })}
      {tooltipExtras?.map((extra) => {
        const raw = row?.[extra.valueName];
        const value = typeof raw === 'number' ? raw : extra.defaultValue;
        if (value === undefined) return null;
        return (
          <div key={extra.valueName} className="text-muted flex items-center justify-between gap-4 pt-1">
            <span>{extra.label}</span>
            <span className="tabular-nums">{(extra.format ?? defaultFormat)(value)}</span>
          </div>
        );
      })}
    </div>
  );
}

// CategoricalChart renders an InsightsListMetricResult as a horizontal bar
// chart over nominal, unordered categories (e.g. calls or cost by model).
// With a single measure (`valueName`/`valueLabel`) this is a magnitude
// comparison, so every bar shares one hue by default — a color ramp on
// unordered categories would double-encode bar length as hue — unless the
// caller opts into per-category colors via `colors`. With several measures
// (`series`, e.g. input vs output tokens per model) each measure is its own
// identity, so each gets its own fixed-order categorical hue and a legend,
// grouped side by side or stacked per `stacked`. Generic over which
// value(s) it plots; the caller supplies one of the two forms.
export function CategoricalChart({
  items,
  valueName,
  valueLabel,
  color,
  colors,
  borderColor,
  borderColors,
  series,
  stacked = false,
  format = defaultFormat,
  formatIdentifier = (identifier) => identifier,
  showYAxisLine = true,
  showValueLabels = false,
  showTooltipValueName = true,
  tooltipExtras,
  isLoading = false,
  className,
}: Props) {
  const effectiveSeries = useMemo<TrendSeriesConfig[]>(
    () => series ?? [{ valueName: valueName ?? '', label: valueLabel ?? '' }],
    [series, valueName, valueLabel],
  );
  const isMultiSeries = effectiveSeries.length > 1;

  const sorted = useMemo(() => {
    if (!items || items.length === 0) return [];
    return [...items].sort((a, b) => {
      const aMap = valuesToMap(a.values);
      const bMap = valuesToMap(b.values);
      const av = effectiveSeries.reduce((sum, s) => sum + (aMap.get(s.valueName) ?? 0), 0);
      const bv = effectiveSeries.reduce((sum, s) => sum + (bMap.get(s.valueName) ?? 0), 0);
      return bv - av;
    });
  }, [items, effectiveSeries]);

  const chartData = useMemo<ChartRow[]>(
    () =>
      sorted.map((item) => {
        const map = valuesToMap(item.values);
        // Every named value the item carries, not just the plotted
        // series — lets tooltipExtras read a column (e.g. cost alongside
        // Tokens by model's input/output bars) without a Bar of its own.
        const row: ChartRow = { identifier: item.identifier, ...Object.fromEntries(map) };
        effectiveSeries.forEach((s) => {
          row[s.valueName] = map.get(s.valueName) ?? 0;
        });
        return row;
      }),
    [sorted, effectiveSeries],
  );

  // singleColor is the fill — pass a distinct tone via `color` directly when
  // pairing it with `borderColor` below; there's no automatic fill
  // derivation from the border color.
  const singleColor = useMemo(() => color ?? CHART_COLORS[1], [color]);

  // perCategoryColors mirrors chartData order — each bar gets its own fill
  // color from `colors`, cycling if there are more bars than colors.
  const perCategoryColors = useMemo(
    () => (colors ? chartData.map((_, idx) => colors[idx % colors.length]) : undefined),
    [chartData, colors],
  );
  // perCategoryBorderColors: `borderColors`' per-category counterpart to
  // `borderColor`, layered as a stroke over perCategoryColors' fill.
  const perCategoryBorderColors = useMemo(
    () => (borderColors ? chartData.map((_, idx) => borderColors[idx % borderColors.length]) : undefined),
    [chartData, borderColors],
  );

  // seriesColors/segmentBorderColors parallel effectiveSeries. A stacked
  // segment's border falls back to transparent (no seam at all) when no
  // explicit borderColor is set, rather than the chart's surface color —
  // opaque, non-subtle fills read better as one continuous stacked bar than
  // as visually separated segments.
  const seriesColors = useMemo(
    () => effectiveSeries.map((s, i) => s.color ?? CHART_COLORS[i % CHART_COLORS.length]),
    [effectiveSeries],
  );
  const segmentBorderColors = useMemo(
    () => effectiveSeries.map((s) => s.borderColor ?? 'transparent'),
    [effectiveSeries],
  );
  // legendSwatchColors mirrors each multi-series entry's *border* color
  // (falling back to its fill when no border override is set) — the legend
  // icon matches the bar's outline, not its (possibly muted) fill.
  const legendSwatchColors = useMemo(
    () => effectiveSeries.map((s, i) => (s.borderColor ? segmentBorderColors[i] : seriesColors[i])),
    [effectiveSeries, segmentBorderColors, seriesColors],
  );
  // colorByKey lets the tooltip look up each series' legend/border color by
  // dataKey, so its swatch matches the chart's border rather than recharts'
  // default (the Bar's fill).
  const colorByKey = useMemo(
    () =>
      isMultiSeries
        ? Object.fromEntries(effectiveSeries.map((s, i) => [s.valueName, legendSwatchColors[i]]))
        : undefined,
    [isMultiSeries, effectiveSeries, legendSwatchColors],
  );
  // colorByIdentifier lets the tooltip look up each bar's own per-category
  // color by identifier — recharts' payload color reflects the Bar's base
  // fill, not a per-item Cell override, so a `colors` palette needs this to
  // show the actual bar color rather than the shared fallback fill.
  const colorByIdentifier = useMemo(() => {
    if (isMultiSeries) return undefined;
    const perCategory = perCategoryBorderColors ?? perCategoryColors;
    if (perCategory) {
      return Object.fromEntries(chartData.map((row, idx) => [row.identifier, perCategory[idx]]));
    }
    if (borderColor) {
      return Object.fromEntries(chartData.map((row) => [row.identifier, borderColor]));
    }
    return undefined;
  }, [isMultiSeries, perCategoryBorderColors, perCategoryColors, borderColor, chartData]);

  // renderYAxisTick draws each category label as a single line of SVG text,
  // middle-truncated with an ellipsis (matching VariantAxisTick in
  // Experiments) — recharts' default category-axis tick wraps long labels
  // across multiple lines when constrained by `width`; a plain <text>
  // element never wraps, so labels stay on one line instead. The full label
  // is exposed via a native <title> tooltip.
  const renderYAxisTick = ({ x, y, payload }: { x: number; y: number; payload: { value: string } }) => {
    const full = formatIdentifier(payload.value);
    return (
      <text x={x} y={y} dy={4} textAnchor="end" fontSize={10} className="fill-basis">
        <title>{full}</title>
        {truncateCenter(full, 30)}
      </text>
    );
  };

  // rowByIdentifier backs the value-label column (showValueLabels): a
  // second category y-axis on the right, sharing the same row positions as
  // the identifier axis on the left, so every value lines up with its bar
  // regardless of that bar's own length.
  const rowByIdentifier = useMemo(
    () => new Map(chartData.map((row) => [row.identifier, row])),
    [chartData],
  );
  // Every series' value renders in its own fixed-width column (with a
  // vertical divider line at a fixed position between them) rather than one
  // joined string — so columns line up across rows regardless of how many
  // digits any one row's value happens to have. Degrades to a single column
  // with no divider for the single-measure form.
  const renderValueAxisTick = ({ x, y, payload }: { x: number; y: number; payload: { value: string } }) => {
    const row = rowByIdentifier.get(payload.value);
    return (
      <g>
        {effectiveSeries.map((s, i) => {
          const raw = row?.[s.valueName];
          const text = typeof raw === 'number' ? format(raw) : '';
          const columnX = x + i * VALUE_COLUMN_STEP;
          return (
            <g key={s.valueName}>
              {i > 0 && (
                <line
                  x1={columnX - VALUE_COLUMN_GAP / 2}
                  x2={columnX - VALUE_COLUMN_GAP / 2}
                  y1={y - 5}
                  y2={y + 5}
                  stroke={BORDER_SUBTLE_COLOR}
                  strokeWidth={1}
                />
              )}
              <text x={columnX} y={y} dy={4} textAnchor="start" fontSize={10} className="fill-basis">
                {text}
              </text>
            </g>
          );
        })}
      </g>
    );
  };

  if ((!items || items.length === 0) || isLoading) {
    return (
      <div className={className}>
        <RankedChartSkeleton animate={isLoading} className="h-full min-h-[200px]" />
      </div>
    );
  }

  return (
    <div className={className}>
      <div className="relative h-[200px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={chartData}
            layout="vertical"
            margin={{ top: 8, right: 12, bottom: -9.5, left: 12 }}
            barCategoryGap={stacked && isMultiSeries ? STACKED_BAR_CATEGORY_GAP : undefined}
          >
            <XAxis
              type="number"
              tickFormatter={formatCompactNumber}
              axisLine={false}
              tickLine={false}
              tick={{ fontSize: 12 }}
            />
            <YAxis
              type="category"
              dataKey="identifier"
              tick={renderYAxisTick}
              axisLine={showYAxisLine}
              tickLine={showYAxisLine}
              width={140}
              interval={0}
            />
            {(!isMultiSeries || stacked) && showValueLabels && (
              <YAxis
                yAxisId="values"
                orientation="right"
                type="category"
                dataKey="identifier"
                tick={renderValueAxisTick}
                axisLine={false}
                tickLine={false}
                width={effectiveSeries.length * VALUE_COLUMN_WIDTH + (effectiveSeries.length - 1) * VALUE_COLUMN_GAP}
                tickMargin={0}
                interval={0}
              />
            )}
            {chartData.map((row) => (
              // `stroke` as a direct prop, not a `stroke-[...]` className —
              // Tailwind's JIT scanner can't statically analyze a
              // template-literal-interpolated class name, so that class was
              // never actually generated and recharts' own default
              // `stroke="#ccc"` rendered instead.
              <ReferenceLine key={row.identifier} y={row.identifier} stroke={BORDER_SUBTLE_COLOR} />
            ))}
            <Tooltip
              cursor={false}
              content={
                <CategoricalTooltip
                  formatIdentifier={formatIdentifier}
                  format={format}
                  colorByKey={colorByKey}
                  colorByIdentifier={colorByIdentifier}
                  showValueLabel={showTooltipValueName}
                  tooltipExtras={tooltipExtras}
                />
              }
              // recharts' tooltip wrapper is `tabIndex={-1} role="dialog"`,
              // programmatically focused whenever the tooltip shows —
              // including on plain mouse hover, not just real keyboard
              // navigation (tabIndex -1 means it's never reachable via
              // sequential Tab in the first place). Without this, the
              // browser's default focus outline shows on every hover.
              wrapperStyle={{ outline: 'none' }}
            />
            {effectiveSeries.map((s, i) => {
              const isLastSegment = i === effectiveSeries.length - 1;
              return (
                <Bar
                  key={s.valueName}
                  dataKey={s.valueName}
                  name={s.label}
                  barSize={BAR_THICKNESS}
                  stackId={isMultiSeries && stacked ? 'stack' : undefined}
                  fill={isMultiSeries ? seriesColors[i] : singleColor}
                  stroke={isMultiSeries ? (!stacked ? segmentBorderColors[i] : undefined) : borderColor}
                  strokeWidth={isMultiSeries ? (!stacked ? 1 : undefined) : borderColor ? 1 : undefined}
                  isAnimationActive={false}
                  legendType="none"
                  shape={
                    isMultiSeries && stacked
                      ? (shapeProps: unknown) =>
                          stackedSegmentShape(
                            shapeProps as BarShapeProps,
                            isLastSegment,
                            segmentBorderColors[i],
                          )
                      : undefined
                  }
                >
                  {!isMultiSeries && perCategoryColors
                    ? chartData.map((row, idx) => (
                        <Cell
                          key={row.identifier}
                          fill={perCategoryColors[idx] ?? singleColor}
                          stroke={perCategoryBorderColors?.[idx]}
                          strokeWidth={perCategoryBorderColors ? 1 : undefined}
                        />
                      ))
                    : null}
                </Bar>
              );
            })}
          </BarChart>
        </ResponsiveContainer>
        {isMultiSeries && (
          // Absolutely positioned into the plot's own bottom-left corner —
          // a horizontal bar chart's categories rarely fill the fixed-height
          // container down to its last pixel, so the legend can sit in that
          // otherwise-empty margin instead of pushing the whole card taller.
          <ul className="absolute bottom-0 flex flex-wrap gap-4">
            {effectiveSeries.map((s, i) => (
              <li key={s.valueName} className="flex items-center gap-1.5 text-xs">
                <span
                  className="h-2.5 w-2.5 shrink-0"
                  style={{ backgroundColor: legendSwatchColors[i] }}
                />
                <span className="text-basis">{s.label}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
