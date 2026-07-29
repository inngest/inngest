import { formatMetricValue } from './variantsTable/metricStats';

type TooltipEntry = {
  name?: string | number;
  dataKey?: string | number;
  value?: number | string | Array<number | string>;
  color?: string;
  payload?: { variantName?: string; total?: number; runCount?: number };
};

type Props = {
  active?: boolean;
  payload?: TooltipEntry[];
  /** Falls back to `payload[0].payload.variantName` when not provided. */
  label?: string | number;
};

export function ChartTooltip({ active, payload, label }: Props) {
  if (!active || !payload?.length) return null;

  const first = payload[0];
  const title = label ?? first?.payload?.variantName ?? '';
  const total = first?.payload?.total;
  const runCount = first?.payload?.runCount;

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {title && (
        <div className="border-subtle mb-1.5 flex items-baseline justify-between gap-4 border-b pb-1.5">
          <span className="text-basis text-sm font-medium">{title}</span>
          {runCount != null && (
            <span className="text-muted tabular-nums">{runCount.toLocaleString()} runs</span>
          )}
        </div>
      )}
      <div className="flex flex-col gap-1">
        {payload.map((p, i) => (
          <div key={i} className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <span
                className="h-2.5 w-2.5 shrink-0 rounded"
                style={{ backgroundColor: p.color }}
              />
              <span className="text-muted">{p.name ?? p.dataKey}</span>
            </div>
            <span className="text-basis tabular-nums">
              {typeof p.value === 'number'
                ? formatMetricValue(p.value)
                : p.value}
            </span>
          </div>
        ))}
      </div>
      {typeof total === 'number' && (
        <div className="mt-1.5 flex items-baseline justify-between gap-3">
          <span className="text-muted">Total</span>
          <span className="text-basis text-sm font-semibold tabular-nums">
            {formatMetricValue(total)}
          </span>
        </div>
      )}
    </div>
  );
}
