import { formatNumber } from '@/components/Metrics/utils';
import type { TooltipExtra } from './types';

export type ChartTooltipPayloadEntry = {
  // string for a real series (TrendChart sets `dataKey={s.key}` directly);
  // a stacked area's invisible dot-marker overlay instead sets a *function*
  // dataKey (it recomputes each layer's cumulative value for its hover dot)
  // — recharts still reports that overlay as its own tooltip payload entry,
  // so filtering to string dataKeys is what excludes it below.
  dataKey?: string | ((row: unknown) => unknown);
  name?: string;
  value?: number | null;
  color?: string;
  // recharts attaches the full chartData row here regardless of which
  // dataKey the entry itself plots — this is how tooltipExtras reads a
  // column (e.g. cost) that has no Line/Bar/Area of its own.
  payload?: Record<string, string | number | null>;
};

// ChartTooltip renders TrendChart's shared axis tooltip: every series
// present at the hovered bucket, sorted descending by value, each with a
// colored swatch. `colorByKey` overrides recharts' default per-entry color
// (a Line/Bar/Area's own stroke/fill), since that varies by series type
// (Area entries report fill, Line entries report stroke) — keying by
// dataKey keeps the swatch consistent regardless.
export function ChartTooltip({
  active,
  payload,
  label,
  colorByKey,
  format = formatNumber,
  tooltipExtras,
}: {
  active?: boolean;
  payload?: ChartTooltipPayloadEntry[];
  label?: string;
  colorByKey: Record<string, string>;
  format?: (value: number) => string;
  tooltipExtras?: TooltipExtra[];
}) {
  if (!active || !payload?.length) return null;
  const sorted = payload
    .filter((p): p is ChartTooltipPayloadEntry & { dataKey: string; value: number } =>
      typeof p.dataKey === 'string' && p.dataKey in colorByKey && typeof p.value === 'number',
    )
    .sort((a, b) => b.value - a.value);
  if (sorted.length === 0) return null;
  const row = payload[0]?.payload;
  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 pb-2 pt-1 text-sm shadow-md">
      <div className="text-muted pb-2">{label}</div>
      {sorted.map((p, idx) => (
        <div key={idx} className="text-basis flex items-center justify-between gap-4 font-medium">
          <span className="flex min-w-0 items-center truncate">
            <span
              className="mr-2 inline-flex h-2.5 w-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: (p.dataKey && colorByKey[p.dataKey]) ?? p.color }}
            />
            {p.name}
          </span>
          <span className="shrink-0 tabular-nums">{format(p.value)}</span>
        </div>
      ))}
      {tooltipExtras?.map((extra) => {
        const raw = row?.[extra.valueName];
        const value = typeof raw === 'number' ? raw : extra.defaultValue;
        if (value === undefined) return null;
        return (
          <div key={extra.valueName} className="text-muted flex items-center justify-between gap-4 pt-1">
            <span>{extra.label}</span>
            <span className="tabular-nums">{(extra.format ?? formatNumber)(value)}</span>
          </div>
        );
      })}
    </div>
  );
}
