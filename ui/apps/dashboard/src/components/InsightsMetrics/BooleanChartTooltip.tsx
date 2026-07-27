import { formatPlainNumber } from './format';
import type { RowData } from './BooleanChart';

type TooltipEntry = {
  name?: string | number;
  value?: number | string | Array<number | string>;
  color?: string;
  payload?: RowData;
};

type Props = {
  active?: boolean;
  payload?: TooltipEntry[];
  label?: string | number;
  format?: (value: number) => string;
  countLabel?: string;
};

export function BooleanChartTooltip({
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
  const value = typeof first?.value === 'number' ? first.value : null;

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
      {value !== null && (
        <div className="flex items-center justify-between gap-4">
          <span className="text-muted">{first?.name}</span>
          <span className="text-basis tabular-nums font-semibold">{format(value)}</span>
        </div>
      )}
    </div>
  );
}
