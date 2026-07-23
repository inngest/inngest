import { formatNumber } from '@/components/Metrics/utils';

export type ChartTooltipPayloadEntry = {
  dataKey?: string;
  name?: string;
  value?: number | null;
  color?: string;
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
}: {
  active?: boolean;
  payload?: ChartTooltipPayloadEntry[];
  label?: string;
  colorByKey: Record<string, string>;
  format?: (value: number) => string;
}) {
  if (!active || !payload?.length) return null;
  const sorted = [...payload]
    .filter((p): p is ChartTooltipPayloadEntry & { value: number } => typeof p.value === 'number')
    .sort((a, b) => b.value - a.value);
  if (sorted.length === 0) return null;
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
    </div>
  );
}
