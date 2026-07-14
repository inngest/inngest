import type { RowData } from './BoxPlot';
import { formatMetricValue } from './variantsTable/metricStats';

type TooltipEntry = {
  name?: string | number;
  payload?: RowData;
};

type Props = {
  active?: boolean;
  payload?: TooltipEntry[];
  label?: string | number;
};

export function BoxPlotTooltip({ active, payload, label }: Props) {
  if (!active || !payload?.length) return null;
  const first = payload[0];
  const data = first?.payload;
  if (!data) return null;
  const title = label ?? data.variantName ?? '';

  const stats = [
    { label: 'Average', value: formatMetricValue(data.avg) },
    { label: 'Median', value: formatMetricValue(data.med) },
    { label: 'Spread', value: `${formatMetricValue(data.q1)} – ${formatMetricValue(data.q3)}` },
    { label: 'Min/Max', value: `${formatMetricValue(data.min)} – ${formatMetricValue(data.max)}` },
  ];

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {title && (
        <div className="border-subtle mb-1.5 flex items-baseline justify-between gap-4 border-b pb-1.5">
          <span className="text-basis text-sm font-medium">{title}</span>
          <span className="text-muted tabular-nums">{data.runCount.toLocaleString()} runs</span>
        </div>
      )}
      <div className="flex flex-col gap-1">
        {stats.map(({ label, value }) => (
          <div key={label} className="flex items-baseline justify-between gap-4">
            <span className="text-muted">{label}</span>
            <span className="text-basis tabular-nums font-semibold">{value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
