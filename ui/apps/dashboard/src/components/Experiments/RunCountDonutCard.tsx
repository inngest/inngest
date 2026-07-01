import { useCallback, useMemo, useState } from 'react';
import { Card } from '@inngest/components/Card';
import type { ExperimentVariantMetrics } from '@inngest/components/Experiments';
import { cn } from '@inngest/components/utils/classNames';
import { Cell, Pie, PieChart, ResponsiveContainer, Sector } from 'recharts';

import { computeChartSizing, truncateCenter } from '@/lib/experiments/chart';
import { colorForVariant, subtleColorForVariant } from '@/lib/experiments/colors';

type Props = {
  variants: ExperimentVariantMetrics[];
  className?: string;
  variantColorIndex?: Map<string, number>;
  onVariantHover?: (name: string | null) => void;
  highlightedVariantName?: string | null;
};

export function RunCountDonutCard({ variants, className, variantColorIndex, onVariantHover, highlightedVariantName }: Props) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  const totalRunCount = useMemo(
    () => variants.reduce((sum, v) => sum + v.runCount, 0),
    [variants],
  );

  const hoveredVariant = hoveredIndex !== null ? (variants[hoveredIndex] ?? null) : null;

  const externalActiveIndex = useMemo(() => {
    if (hoveredIndex !== null || !highlightedVariantName) return null;
    const idx = variants.findIndex((v) => v.variantName === highlightedVariantName);
    return idx >= 0 ? idx : null;
  }, [hoveredIndex, highlightedVariantName, variants]);

  const externalVariant = useMemo(
    () =>
      highlightedVariantName && !hoveredVariant
        ? (variants.find((v) => v.variantName === highlightedVariantName) ?? null)
        : null,
    [highlightedVariantName, hoveredVariant, variants],
  );

  const displayVariant = hoveredVariant ?? externalVariant;
  // Only dim segments when the highlight is external (metric panel hover).
  // When the donut itself is hovered, the active segment grows via activeShape — no dimming.
  const effectiveHighlight = hoveredVariant ? null : (highlightedVariantName ?? null);

  const activeShape = useCallback(
    (props: { outerRadius?: number; [key: string]: unknown }) => (
      <Sector {...(props as Record<string, unknown>)} outerRadius={(props.outerRadius ?? 0) + 6} />
    ),
    [],
  );

  const data = useMemo(
    () => variants.map((v) => ({ name: v.variantName, value: v.runCount })),
    [variants],
  );

  // Match ScoreSummaryCard's chart height (base + 28px legend row) so the
  // donut never grows taller than the score chart sitting beside it.
  const maxDonutSize = useMemo(() => {
    const { chartHeight } = computeChartSizing(variants.map((v) => v.variantName));
    return Math.max(200, Math.round((chartHeight + 28) * 0.95));
  }, [variants]);

  return (
    <Card className={cn(className)}>
      <Card.Header className="rounded-t-md border-b-0 py-2 pl-3 pr-2">
        <span className="text-basis text-sm">Total Run Count</span>
      </Card.Header>
      <Card.Content className="flex items-center gap-4 px-3 py-2">
        <div className="relative flex-none" style={{ width: maxDonutSize, height: maxDonutSize }}>
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                cx="50%"
                cy="50%"
                innerRadius="55%"
                outerRadius="85%"
                dataKey="value"
                strokeWidth={1}
                isAnimationActive={false}
                activeIndex={hoveredIndex ?? externalActiveIndex ?? undefined}
                activeShape={activeShape}
                onMouseEnter={(_, index) => {
                  setHoveredIndex(index);
                  onVariantHover?.(variants[index]?.variantName ?? null);
                }}
                onMouseLeave={() => {
                  setHoveredIndex(null);
                  onVariantHover?.(null);
                }}
              >
                {data.map((entry, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={subtleColorForVariant(variantColorIndex?.get(entry.name) ?? index)}
                    stroke={colorForVariant(variantColorIndex?.get(entry.name) ?? index)}
                    opacity={effectiveHighlight && entry.name !== effectiveHighlight ? 0.25 : 1}
                  />
                ))}
              </Pie>
            </PieChart>
          </ResponsiveContainer>
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
            <span className="text-basis text-sm font-semibold leading-tight">
              {(displayVariant?.runCount ?? totalRunCount).toLocaleString()}
            </span>
            <span className="text-muted text-[10px] leading-tight uppercase tracking-wide">
              {displayVariant ? truncateCenter(displayVariant.variantName) : 'Total Runs'}
            </span>
          </div>
        </div>

        <div className="flex min-w-0 flex-1 flex-col items-center justify-center gap-1.5 pl-0 pr-5">
          {variants.map((v, index) => (
            <div key={v.variantName} className="flex w-full items-center gap-1">
              <span
                className="h-2.5 w-2.5 shrink-0 rounded-full border"
                style={{
                  backgroundColor: subtleColorForVariant(variantColorIndex?.get(v.variantName) ?? index),
                  borderColor: colorForVariant(variantColorIndex?.get(v.variantName) ?? index),
                }}
              />
              <span
                className="text-subtle min-w-0 flex-1 pr-3 text-left text-sm"
                title={v.variantName}
              >
                {truncateCenter(v.variantName)}
              </span>
              <span className="text-basis shrink-0 pl-3 text-right text-sm tabular-nums">
                {v.runCount.toLocaleString()}{' '}
                <span className="text-muted">runs</span>
              </span>
            </div>
          ))}
        </div>
      </Card.Content>
    </Card>
  );
}
