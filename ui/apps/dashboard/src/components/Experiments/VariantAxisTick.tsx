import { truncateCenter } from '@/lib/experiments/chart';

type Props = {
  x?: number;
  y?: number;
  payload?: { value?: string };
  textAnchor?: string;
};

/**
 * Recharts YAxis tick that renders variant names with a center ellipsis for
 * long values. The full name is exposed via a native `<title>` tooltip so it
 * remains accessible.
 */
export function VariantAxisTick({
  x = 0,
  y = 0,
  payload,
  textAnchor = 'end',
}: Props) {
  const full = payload?.value ?? '';
  const display = truncateCenter(full);
  return (
    <g transform={`translate(${x},${y})`}>
      <text dy={4} textAnchor={textAnchor} fontSize={12} className="fill-muted">
        <title>{full}</title>
        {display}
      </text>
    </g>
  );
}
