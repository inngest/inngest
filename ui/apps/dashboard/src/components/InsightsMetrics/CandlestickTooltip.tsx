import { formatPlainNumber } from './format';
import type { CandleData } from './CandlestickPlot';

type TooltipEntry = {
  payload?: CandleData;
};

type Props = {
  active?: boolean;
  payload?: TooltipEntry[];
  label?: string | number;
  format?: (value: number) => string;
  countLabel?: string;
  // Formats the bucket's raw timestamp into the tooltip title — defaults to
  // the raw value, since callers already control the x-axis tickFormatter
  // and typically want the same date formatting here.
  formatLabel?: (timestamp: string) => string;
};

export function CandlestickTooltip({
  active,
  payload,
  label,
  format = formatPlainNumber,
  countLabel = 'runs',
  formatLabel,
}: Props) {
  if (!active || !payload?.length) return null;
  const data = payload[0]?.payload;
  // A gap-filled empty bucket still occupies an x-axis slot (see
  // fillTimeBuckets), but has no stats to show — nothing to hover into.
  if (!data || data.min === undefined || data.max === undefined) return null;
  const title = formatLabel && typeof label === 'string' ? formatLabel(label) : label;

  const stats = [
    ...(typeof data.avg === 'number' ? [{ label: 'Average', value: format(data.avg) }] : []),
    ...(typeof data.med === 'number' ? [{ label: 'Median', value: format(data.med) }] : []),
    ...(typeof data.q1 === 'number' && typeof data.q3 === 'number'
      ? [{ label: 'Spread', value: `${format(data.q1)} – ${format(data.q3)}` }]
      : []),
    { label: 'Min/Max', value: `${format(data.min)} – ${format(data.max)}` },
  ];

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {title !== undefined && (
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
