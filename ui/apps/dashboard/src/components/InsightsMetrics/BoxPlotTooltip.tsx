import { formatPlainNumber } from './format';
import type { RowData } from './BoxPlot';

type TooltipEntry = {
  name?: string | number;
  payload?: RowData;
};

type Props = {
  active?: boolean;
  payload?: TooltipEntry[];
  label?: string | number;
  format?: (value: number) => string;
  // Wording for the row-count stat line (e.g. "runs", "points") — omitted
  // entirely when the row carries no `count`.
  countLabel?: string;
};

export function BoxPlotTooltip({
  active,
  payload,
  label,
  format = formatPlainNumber,
  countLabel = 'runs',
}: Props) {
  if (!active || !payload?.length) return null;
  const first = payload[0];
  const data = first?.payload;
  if (!data) return null;
  const title = label ?? data.label ?? '';

  const stats = [
    { label: 'Average', value: format(data.avg) },
    { label: 'Median', value: format(data.med) },
    { label: 'Spread', value: `${format(data.q1)} – ${format(data.q3)}` },
    { label: 'Min/Max', value: `${format(data.min)} – ${format(data.max)}` },
  ];

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {title && (
        <div className="border-subtle mb-1.5 flex items-baseline justify-between gap-4 border-b pb-1.5">
          <span className="text-basis text-sm font-medium">{title}</span>
          {typeof data.count === 'number' && (
            <span className="text-muted tabular-nums">
              {data.count.toLocaleString()} {countLabel}
            </span>
          )}
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
