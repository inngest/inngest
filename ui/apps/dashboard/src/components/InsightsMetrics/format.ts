// Default value formatter shared by BoxPlot/CandlestickPlot's tooltips and
// axis ticks — rounds to 2 decimals and trims trailing zeros via
// toLocaleString, the same approach Experiments' formatMetricValue uses.
// Callers with a unit-specific measure (seconds, cost, ...) pass their own
// `format` instead.
export function formatPlainNumber(value: number): string {
  if (!Number.isFinite(value)) return '-';
  return (+value.toFixed(2)).toLocaleString(undefined, {
    maximumFractionDigits: 2,
  });
}
