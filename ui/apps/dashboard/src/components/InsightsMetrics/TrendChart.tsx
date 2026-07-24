import { useMemo } from 'react';
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { formatCompactNumber } from '@/components/InfraDashboard/utils';
import { dateFormat, lineColors, timeDiff } from '@/components/Metrics/utils';
import { ChartTooltip } from './ChartTooltip';
import { TrendAreaChartSkeleton, TrendChartSkeleton } from './ChartSkeleton';
import { BORDER_SUBTLE_COLOR, SURFACE_COLOR, toCssColor } from './colors';
import { valuesToMap, type InsightsMetricPoint } from './types';

export type TrendSeriesConfig = {
  // Which NamedValue.name to read per point.
  valueName: string;
  label: string;
  // Overrides the index-based default from `lineColors` (a [light, dark]
  // token/hex tuple, same shape as one `lineColors` entry) — for a series
  // whose identity has an established color independent of its position
  // (e.g. always green for "runs"), rather than whatever hue its index
  // would otherwise land on.
  color?: readonly [string, string];
  // Bar border color override (CategoricalChart only) — e.g. a subtle fill
  // paired with its own moderate/vivid hue as the border, rather than the
  // chart surface color CategoricalChart uses by default to separate
  // stacked segments.
  borderColor?: readonly [string, string];
  // Area fill override ('area' chartType only) — a design-system subtle
  // token, rather than the default computed mix of `color` toward the
  // chart surface.
  areaColor?: readonly [string, string];
};

type Props = {
  points: InsightsMetricPoint[] | undefined;
  // Fixed named series form — a caller-declared list of NamedValue keys
  // (e.g. input/output tokens), each with a full label and (optionally) a
  // dedicated color. Mutually exclusive with `valueName`.
  series?: TrendSeriesConfig[];
  // Derived series form, for points whose `dimensions` (not `values`) is
  // populated — e.g. one line per model. Which NamedValue.name to read per
  // dimension per bucket. Unlike `series`'s caller-supplied fixed config,
  // the category set here is open-ended and only known from the data: the
  // top `lineColors.length` identifiers by total value across the whole
  // range each get their own line in a fixed rank order (color follows the
  // entity, not a cycled hue), and every other identifier folds into one
  // "Other" line rather than adding an unbounded number of series.
  // Mutually exclusive with `series`.
  valueName?: string;
  // 'line' (default); 'bar' renders a time-bucketed histogram, which reads
  // better for coarse counts (e.g. run volume) than a line; 'area' adds a
  // light fill under each line (a visual highlight, not a stacked area —
  // each line still plots its own true value). 'area' is only meaningful
  // for the fixed `series` form.
  chartType?: 'line' | 'bar' | 'area';
  // Stack every series into one bar (bar) or one layered/mountain fill
  // (area) per bucket, instead of grouped bars or independently-overlapping
  // areas. No effect on plain lines.
  stacked?: boolean;
  // Formats the y-axis ticks and tooltip values — defaults to a compact
  // K/M number (formatCompactNumber). Callers with a unit-specific measure
  // (e.g. cost) pass their own formatter (e.g. formatCost) instead.
  format?: (value: number) => string;
  // Overrides `format` for the y-axis ticks specifically — for callers
  // whose tooltip wants a fixed decimal width but whose axis reads better
  // compact (e.g. formatSeconds' 3-decimal tooltip vs formatSecondsAxis'
  // trailing-zero-collapsed ticks). Defaults to `format`.
  axisFormat?: (value: number) => string;
  // Whether the y-axis can land on a decimal tick value at all — false
  // (the default) suits compact integer counts (runs, tokens); a
  // sub-1-unit-heavy measure (e.g. seconds) needs this true, or every tick
  // rounds to a whole number and there's nothing left for `axisFormat` to
  // format.
  allowDecimals?: boolean;
  // Legend icon shape — 'circle' (default) matches every other trend chart;
  // 'rect' for callers that want a square swatch instead.
  legendIcon?: 'circle' | 'rect';
  isLoading?: boolean;
  // Charts sharing the same group id sync their hover/tooltip position
  // (recharts' syncId) — hovering one chart highlights the same x position
  // on every other chart in the group.
  group?: string;
  className?: string;
};

const OTHER_LABEL = 'Other';

type ChartRow = { timestamp: string; [key: string]: string | number | null };

type EffectiveSeries = {
  key: string;
  label: string;
  color: string;
  areaColor: string;
  // Derived-mode "Other" bucket — a reserved neutral, never a categorical
  // hue, and (bar chartType) never gets the rounded outer-segment corner
  // treatment since it isn't the true top of the stack's ranked series.
  isOther: boolean;
};

// TrendChart renders an InsightsTimeSeriesMetricResult as a multi-line (or
// multi-bar/area) chart. Generic over which values it plots — the caller
// supplies either a fixed `series` config (one line per declared
// NamedValue, e.g. input vs output tokens) or a single `valueName` to
// derive series from the data itself (one line per dimension identifier,
// e.g. one per model); exactly one of the two must be given. This
// component has no AI-specific knowledge either way.
export function TrendChart({
  points,
  series,
  valueName,
  chartType = 'line',
  stacked = false,
  format = formatCompactNumber,
  axisFormat = format,
  allowDecimals = false,
  legendIcon = 'circle',
  isLoading = false,
  group,
  className,
}: Props) {
  // Derived-mode ranking: the top identifiers by total value across every
  // point's dimensions, plus whether there's a remainder to fold into
  // "Other". Only computed when `valueName` (not `series`) is given.
  const { topIdentifiers, topSet, hasOther } = useMemo(() => {
    if (!valueName) return { topIdentifiers: [] as string[], topSet: new Set<string>(), hasOther: false };
    const totals = new Map<string, number>();
    for (const p of points ?? []) {
      for (const dim of p.dimensions ?? []) {
        const v = valuesToMap(dim.values).get(valueName) ?? 0;
        totals.set(dim.identifier, (totals.get(dim.identifier) ?? 0) + v);
      }
    }
    const ranked = [...totals.entries()].sort((a, b) => b[1] - a[1]);
    const topIdentifiers = ranked.slice(0, lineColors.length).map(([id]) => id);
    return { topIdentifiers, topSet: new Set(topIdentifiers), hasOther: ranked.length > lineColors.length };
  }, [points, valueName]);

  const effectiveSeries = useMemo<EffectiveSeries[]>(() => {
    if (valueName) {
      const names = hasOther ? [...topIdentifiers, OTHER_LABEL] : topIdentifiers;
      return names.map((name, i) => {
        const isOther = name === OTHER_LABEL;
        const color = isOther ? BORDER_SUBTLE_COLOR : toCssColor(lineColors[i][0]);
        return { key: name, label: name, color, areaColor: color, isOther };
      });
    }
    return (series ?? []).map((s, i) => {
      const color = toCssColor((s.color ?? lineColors[i % lineColors.length])[0]);
      const areaColor = s.areaColor
        ? toCssColor(s.areaColor[0])
        : `color-mix(in srgb, ${color} 40%, ${SURFACE_COLOR} 60%)`;
      return { key: s.valueName, label: s.label, color, areaColor, isOther: false };
    });
  }, [valueName, hasOther, topIdentifiers, series]);

  const chartData = useMemo<ChartRow[]>(
    () =>
      (points ?? []).map((p) => {
        const row: ChartRow = { timestamp: p.timestamp };
        if (valueName) {
          effectiveSeries.forEach((s) => {
            if (s.isOther) {
              const rest = (p.dimensions ?? []).filter((d) => !topSet.has(d.identifier));
              row[s.key] =
                rest.length === 0
                  ? null
                  : rest.reduce((sum, d) => sum + (valuesToMap(d.values).get(valueName) ?? 0), 0);
              return;
            }
            const match = (p.dimensions ?? []).find((d) => d.identifier === s.key);
            row[s.key] = match ? valuesToMap(match.values).get(valueName) ?? null : null;
          });
        } else {
          const map = valuesToMap(p.values ?? []);
          effectiveSeries.forEach((s) => {
            row[s.key] = map.get(s.key) ?? null;
          });
        }
        return row;
      }),
    [points, effectiveSeries, valueName, topSet],
  );

  const colorByKey = useMemo(
    () => Object.fromEntries(effectiveSeries.map((s) => [s.key, s.color])),
    [effectiveSeries],
  );

  const diff = timeDiff(points?.[0]?.timestamp, points?.[points.length - 1]?.timestamp);
  // Denser trends would render an unreadable pile of overlapping labels —
  // skip enough ticks to keep them legible (matches getXAxis's interval
  // formula in the ECharts-era Metrics charts).
  const tickInterval = (points?.length ?? 0) <= 40 ? 2 : 12;

  // Derived mode always shows a legend (there's no other way to tell which
  // line is which entity); fixed mode only needs one when there's more
  // than one declared series to distinguish.
  const showLegend = valueName !== undefined || effectiveSeries.length > 1;
  const TrendSkeleton = chartType === 'area' ? TrendAreaChartSkeleton : TrendChartSkeleton;

  if ((!points || points.length === 0) || isLoading) {
    return (
      <div className={className}>
        <TrendSkeleton animate={isLoading} className="h-full min-h-[240px]" />
      </div>
    );
  }

  return (
    <div className={className}>
      <div className="relative h-[240px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart
              data={chartData}
              syncId={group}
              margin={{ top: 8, right: 8, bottom: showLegend ? 24 : 8, left: 8 }}
            >
              {/* `stroke` as a direct prop, not a `stroke-[...]` className —
                  Tailwind's JIT scanner can't statically analyze a
                  template-literal-interpolated class name, so that class was
                  never actually generated and recharts' own default
                  `stroke="#ccc"` rendered instead. */}
              <CartesianGrid horizontal vertical={false} stroke={BORDER_SUBTLE_COLOR} />
              <XAxis
                dataKey="timestamp"
                tickFormatter={(value: string) => dateFormat(value, diff)}
                interval={tickInterval}
                tick={{ fontSize: 12 }}
                className="fill-basis"
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                allowDecimals={allowDecimals}
                tickFormatter={axisFormat}
                tick={{ fontSize: 12 }}
                className="fill-basis"
                axisLine={false}
                tickLine={false}
                width={50}
              />
              <Tooltip
                cursor={{ stroke: BORDER_SUBTLE_COLOR }}
                content={<ChartTooltip colorByKey={colorByKey} format={format} />}
                // recharts' tooltip wrapper is `tabIndex={-1} role="dialog"`,
                // programmatically focused whenever the tooltip shows —
                // including on plain mouse hover, not just real keyboard
                // navigation (tabIndex -1 means it's never reachable via
                // sequential Tab in the first place). Without this, the
                // browser's default focus outline shows on every hover.
                wrapperStyle={{ outline: 'none' }}
              />
              {showLegend && (
                <Legend
                  verticalAlign="bottom"
                  align="left"
                  content={() => (
                    <ul className="mt-2 flex flex-wrap gap-4">
                      {effectiveSeries.map((s) => (
                        <li key={s.key} className="flex items-center gap-1.5 text-xs">
                          <span
                            className={`h-2.5 w-2.5 shrink-0 ${legendIcon === 'circle' ? 'rounded-full' : ''}`}
                            style={{ backgroundColor: s.color }}
                          />
                          <span className="text-basis">{s.label}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                />
              )}
              {effectiveSeries.map((s, i) => {
                const isLastInStack = i === effectiveSeries.length - 1;
                if (chartType === 'bar') {
                  return (
                    <Bar
                      key={s.key}
                      dataKey={s.key}
                      name={s.label}
                      fill={s.color}
                      barSize={24}
                      stackId={stacked ? 'trend' : undefined}
                      // Rounded data-end anchored away from the baseline —
                      // only the outermost segment of a stack, since inner
                      // segments meet another fill, not the axis. Only
                      // meaningful in derived mode, where several series
                      // can share one stacked bar.
                      radius={valueName && (!stacked || isLastInStack) ? [4, 4, 0, 0] : undefined}
                      isAnimationActive={false}
                    />
                  );
                }
                if (chartType === 'area') {
                  return (
                    <Area
                      key={s.key}
                      type="monotone"
                      dataKey={s.key}
                      name={s.label}
                      stroke={s.color}
                      strokeWidth={1}
                      fill={s.areaColor}
                      // The active dot for stacked areas is rendered
                      // separately below, after every area fill — each
                      // area paints in declared order, so a lower (earlier)
                      // layer's own inline activeDot would sit right at
                      // its shared boundary with the next layer up and get
                      // covered by that later-painted fill.
                      activeDot={false}
                      connectNulls
                      // recharts stacks Areas in declared order (first =
                      // bottom layer) — already the order we want, unlike
                      // ECharts, which needed the series array reversed to
                      // get the same visual result.
                      stackId={stacked ? 'trend' : undefined}
                      isAnimationActive={false}
                    />
                  );
                }
                return (
                  <Line
                    key={s.key}
                    type="monotone"
                    dataKey={s.key}
                    name={s.label}
                    stroke={s.color}
                    strokeWidth={1}
                    dot={false}
                    // Hover-point marker. Fill reuses `areaColor`, already
                    // the "subtle" tier mix of this series' color (the same
                    // tint an area fill uses); the border matches the
                    // line's own stroke color, so the dot reads as "this
                    // line, lit up" rather than an unrelated hue.
                    activeDot={{ r: 4, fill: s.areaColor, stroke: s.color, strokeWidth: 2 }}
                    connectNulls
                    isAnimationActive={false}
                  />
                );
              })}
              {chartType === 'area' &&
                effectiveSeries.map((s, i) => (
                  // Dot-only overlay, one per area series, rendered after
                  // every area fill above — an invisible line whose only
                  // visible output is its activeDot, so hovering any
                  // stacked layer's dot always paints on top regardless of
                  // that layer's position in the stack. When stacked, the
                  // real Area's own boundary sits at the *cumulative* sum
                  // through this layer (recharts stacks Areas internally),
                  // not this series' own raw value — so the overlay's
                  // dataKey has to recompute that same cumulative value
                  // itself rather than reading the raw field.
                  <Line
                    key={`dot-${s.key}`}
                    dataKey={(row: ChartRow) =>
                      stacked
                        ? effectiveSeries
                            .slice(0, i + 1)
                            .reduce((sum, s2) => sum + (typeof row[s2.key] === 'number' ? (row[s2.key] as number) : 0), 0)
                        : row[s.key]
                    }
                    stroke="none"
                    dot={false}
                    activeDot={{ r: 4, fill: s.areaColor, stroke: s.color, strokeWidth: 2 }}
                    legendType="none"
                    connectNulls
                    isAnimationActive={false}
                  />
                ))}
            </ComposedChart>
          </ResponsiveContainer>
      </div>
    </div>
  );
}
