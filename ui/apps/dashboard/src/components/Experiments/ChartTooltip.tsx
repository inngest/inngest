import { formatMetricValue } from './variantsTable/metricStats';

type TooltipEntry = {
  name?: string | number;
  dataKey?: string | number;
  value?: number | string | Array<number | string>;
  color?: string;
  payload?: { variantName?: string };
};

type Props = {
  active?: boolean;
  payload?: TooltipEntry[];
  /** Falls back to `payload[0].payload.variantName` when not provided. */
  label?: string | number;
};

/**
 * Themed recharts tooltip that uses semantic Tailwind tokens so it stays
 * readable in both light and dark mode. Recharts' default renders white-on-white
 * text in dark mode.
 */
export function ChartTooltip({ active, payload, label }: Props) {
  if (!active || !payload?.length) return null;

  const first = payload[0];
  const title = label ?? first?.payload?.variantName ?? '';

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {title && (
        <div className="text-basis mb-1.5 text-sm font-medium">{title}</div>
      )}
      <div className="flex flex-col gap-1">
        {payload.map((p, i) => (
          <div key={i} className="flex items-center gap-2">
            <span
              className="h-2.5 w-2.5 shrink-0 rounded"
              style={{ backgroundColor: p.color }}
            />
            <span className="text-muted">{p.name ?? p.dataKey}</span>
            <span className="text-basis tabular-nums">
              {typeof p.value === 'number'
                ? formatMetricValue(p.value)
                : p.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
