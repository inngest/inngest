import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { dateFormat, timeDiff } from '@/components/Metrics/utils';
import { CandlestickTooltip } from './CandlestickTooltip';
import { TrendChartSkeleton } from './ChartSkeleton';
import { BORDER_SUBTLE_COLOR } from './colors';

const BAR_SIZE = 16;
const Y_AXIS_WIDTH = 50;

// One time bucket's box-plot stats — a "candle" is a vertical box plot
// (min/q1/med/q3/max), the same shape BoxPlot draws horizontally per
// category, just one per bucket along a time axis instead of one per row.
// The stat fields are optional so a bucket with no observations can still
// occupy its rightful slot on the time axis (see fillTrendBuckets-style
// gap-filling upstream) — CandleShape just draws nothing for it, rather
// than the axis compressing to only the buckets that have data.
export type CandleData = {
  timestamp: string;
  count?: number;
  avg?: number;
  min?: number;
  q1?: number;
  med?: number;
  q3?: number;
  max?: number;
};

type CompleteCandle = Required<CandleData>;

function hasStats(candle: CandleData): candle is CompleteCandle {
  return (
    typeof candle.min === 'number' &&
    typeof candle.max === 'number' &&
    typeof candle.q1 === 'number' &&
    typeof candle.q3 === 'number' &&
    typeof candle.med === 'number'
  );
}

type BarShapeProps = {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  payload?: CandleData;
};

// makeCandleShape draws one vertical box plot per bucket: a filled box from
// q1 to q3, whiskers out to min/max, and a median tick — the same visual
// vocabulary as BoxPlot's BoxShape, transposed so value increases upward
// (pixel y decreases) instead of left-to-right.
function makeCandleShape(color: string, subtleColor: string) {
  return function CandleShape({ x = 0, y = 0, width = 0, height = 0, payload }: BarShapeProps) {
    if (!payload || !hasStats(payload)) return <g />;
    const { min, max, q1, q3, med } = payload;
    const range = max - min;
    const cx = x + width / 2;

    if (range === 0) {
      const r = width / 2;
      return <circle cx={cx} cy={y + height / 2} r={r} fill={subtleColor} stroke={color} strokeWidth={1} />;
    }

    const py = (v: number) => y + ((max - v) / range) * height;

    return (
      <g>
        <line x1={cx} x2={cx} y1={py(max)} y2={py(q3)} stroke={color} strokeWidth={1} />
        <line x1={cx} x2={cx} y1={py(q1)} y2={py(min)} stroke={color} strokeWidth={1} />
        <rect
          x={x}
          y={py(q3)}
          width={width}
          height={Math.max(py(q1) - py(q3), 0)}
          fill={subtleColor}
          stroke={color}
          strokeWidth={1}
        />
        <line x1={x} x2={x + width} y1={py(med)} y2={py(med)} stroke={color} strokeWidth={1.5} />
      </g>
    );
  };
}

type Props = {
  candles: CandleData[];
  color: string;
  subtleColor: string;
  // Formats the tooltip's stat values — defaults to a plain 2-decimal number.
  format?: (value: number) => string;
  countLabel?: string;
  isLoading?: boolean;
  group?: string;
  className?: string;
};

export function CandlestickPlot({
  candles,
  color,
  subtleColor,
  format,
  countLabel,
  isLoading = false,
  group,
  className,
}: Props) {
  const withStats = candles.filter(hasStats);

  if (candles.length === 0 || withStats.length === 0 || isLoading) {
    return (
      <div className={className}>
        <TrendChartSkeleton animate={isLoading} className="h-full min-h-[240px]" />
      </div>
    );
  }

  const domain = withStats.reduce<[number, number]>(
    ([lo, hi], c) => [Math.min(lo, c.min), Math.max(hi, c.max)],
    [withStats[0].min, withStats[0].max],
  );
  const pad = (domain[1] - domain[0]) * 0.1 || 1;
  const yDomain: [number, number] = [domain[0] - pad, domain[1] + pad];

  const diff = timeDiff(candles[0]?.timestamp, candles[candles.length - 1]?.timestamp);
  const tickInterval = candles.length <= 40 ? 2 : 12;
  const candleShape = makeCandleShape(color, subtleColor);

  return (
    <div className={className}>
      <div className="relative h-full min-h-[240px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={candles}
            syncId={group}
            barSize={BAR_SIZE}
            margin={{ top: 8, right: 8, bottom: 0, left: 8 }}
          >
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
              domain={yDomain}
              tickFormatter={(v: number) => +v.toFixed(2) + ''}
              tick={{ fontSize: 12 }}
              className="fill-basis"
              axisLine={false}
              tickLine={false}
              width={Y_AXIS_WIDTH}
            />
            <Tooltip
              cursor={{ stroke: BORDER_SUBTLE_COLOR }}
              content={
                <CandlestickTooltip
                  format={format}
                  countLabel={countLabel}
                  formatLabel={(value) => dateFormat(value, diff)}
                />
              }
              wrapperStyle={{ outline: 'none' }}
            />
            <Bar
              dataKey={(entry: CandleData): [number, number] =>
                hasStats(entry) ? [entry.min, entry.max] : [0, 0]
              }
              isAnimationActive={false}
              shape={candleShape}
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
