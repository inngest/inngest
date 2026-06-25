import type { ReactElement } from 'react';
import type { BoxPlotData } from './types';
import { formatMetricValue } from './variantsTable/metricStats';

type TooltipEntry = {
  name?: string | number;
  dataKey?: string | number;
  value?: number | string | Array<number | string>;
  color?: string;
  payload: BoxPlotData;
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
export function BoxPlotTooltip({ active, payload, label }: Props) {
  if (!active || !payload?.length) return null;

  const first = payload[0];
  const title = label ?? first?.payload?.variantName ?? '';

  const statGroups: [ReactElement | string, (_: BoxPlotData) => string][][] = [
    [
      ['Avg', (p) => formatMetricValue(p.avg)],
      ['StdDev', (p) => formatMetricValue(p.stddev)],
      [
        <>
          Z<sub>-1</sub>
        </>,
        (p) => formatMetricValue(p.avg - p.stddev),
      ],
      [
        <>
          Z<sub>1</sub>
        </>,
        (p) => formatMetricValue(p.avg + p.stddev),
      ],
    ],
    [
      ['Min', (p) => formatMetricValue(p.min)],
      ['Q1', (p) => formatMetricValue(p.q1)],
      ['Median', (p) => formatMetricValue(p.med)],
      ['Q3', (p) => formatMetricValue(p.q3)],
      ['Max', (p) => formatMetricValue(p.max)],
    ],
  ];

  return (
    <div className="bg-canvasBase border-subtle shadow-tooltip rounded-md border px-3 py-2 text-xs shadow-md">
      {title && (
        <div className="text-basis mb-1.5 text-sm font-medium">{title}</div>
      )}
      <div className="flex flex-col gap-1">
        {payload.map((p, i) => (
          <div key={i} className="flex flex-col gap-2">
            {statGroups.map((group, j) => (
              <div key={j} className="flex gap-2">
                {group.map(([label, valueFn], k) => (
                  <div key={k} className="flex gap-2 items-center">
                    <span className="text-muted">{label}</span>
                    <span className="text-basis tabular-nums">
                      {valueFn(p.payload)}
                    </span>
                  </div>
                ))}
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
