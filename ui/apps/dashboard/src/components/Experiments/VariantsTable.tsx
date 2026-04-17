import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import type {
  ExperimentScoringMetric,
  ExperimentVariantMetrics,
  VariantMetric,
} from '@inngest/components/Experiments';
import { Pill } from '@inngest/components/Pill';
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { Switch, SwitchLabel } from '@inngest/components/Switch';
import { Table } from '@inngest/components/Table';
import {
  RiAddLine,
  RiArrowRightUpLine,
  RiMoreFill,
  RiSettings3Line,
} from '@remixicon/react';
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table';

import { MetricAccordionItem } from '@/components/Experiments/ScoringFormulaSidebar';
import { scoreVariant } from '@/lib/experiments/score';

type Props = {
  variants: ExperimentVariantMetrics[];
  scoringConfig: ExperimentScoringMetric[];
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
  onEnableMetric: (key: string) => void;
  pointsLeft: number;
  onOpenInsights: () => void;
  showInactive: boolean;
  onShowInactiveChange: (v: boolean) => void;
};

type RowData = ExperimentVariantMetrics & {
  score: number;
};

const columnHelper = createColumnHelper<RowData>();

// ---------------------------------------------------------------------------
// Value formatting helpers
// ---------------------------------------------------------------------------

function formatMetricValue(val: number): string {
  if (Number.isNaN(val)) return '-';
  if (Math.abs(val) >= 1000)
    return val.toLocaleString(undefined, { maximumFractionDigits: 1 });
  if (Number.isInteger(val)) return String(val);
  // Small floats: trim trailing zeros but keep up to 3 decimal places
  return parseFloat(val.toFixed(3)).toString();
}

// ---------------------------------------------------------------------------
// Per-metric best / worst computation
// ---------------------------------------------------------------------------

type MetricStats = {
  bestAvg: number;
  worstAvg: number;
  bestVariant: string;
  worstVariant: string;
};

function computeMetricStats(
  rows: RowData[],
  metricKey: string,
  invert: boolean,
): MetricStats | null {
  const entries: { name: string; avg: number }[] = [];

  for (const row of rows) {
    const m = row.metrics.find((vm) => vm.key === metricKey);
    if (m) entries.push({ name: row.variantName, avg: m.avg });
  }

  if (entries.length === 0) return null;

  let best = entries[0]!;
  let worst = entries[0]!;

  for (const e of entries) {
    if (invert) {
      if (e.avg < best.avg) best = e;
      if (e.avg > worst.avg) worst = e;
    } else {
      if (e.avg > best.avg) best = e;
      if (e.avg < worst.avg) worst = e;
    }
  }

  return {
    bestAvg: best.avg,
    worstAvg: worst.avg,
    bestVariant: best.name,
    worstVariant: worst.name,
  };
}

// ---------------------------------------------------------------------------
// Metric cell sub-label
// ---------------------------------------------------------------------------

function MetricSubLabel({
  variantName,
  avg,
  stats,
  metric,
}: {
  variantName: string;
  avg: number;
  stats: MetricStats | null;
  metric: ExperimentScoringMetric;
}) {
  if (!stats) return null;

  if (variantName === stats.bestVariant) {
    return (
      <span className="text-success text-[10px]">
        {metric.labelBest || 'Best'}
      </span>
    );
  }

  if (variantName === stats.worstVariant) {
    return (
      <span className="text-error text-[10px]">
        {metric.labelWorst || 'Worst'}
      </span>
    );
  }

  // Middle variant: show delta vs best
  if (stats.bestAvg === 0) return null;

  const rawDelta = ((avg - stats.bestAvg) / stats.bestAvg) * 100;
  const delta = rawDelta * (metric.invert ? -1 : 1);
  const sign = delta >= 0 ? '+' : '';
  return (
    <span className="text-muted text-[10px] tabular-nums">
      {sign}
      {delta.toFixed(1)}% vs best
    </span>
  );
}

// ---------------------------------------------------------------------------
// Metric column header (stable component so Popover state survives re-renders)
// ---------------------------------------------------------------------------

function MetricColumnHeader({
  metric,
  pointsLeft,
  onUpdateMetric,
  isOpen,
  onOpenChange,
}: {
  metric: ExperimentScoringMetric;
  pointsLeft: number;
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <div className="flex w-full items-center gap-1">
      <span className="text-muted min-w-0 flex-1 truncate text-xs font-medium">
        {metric.displayName}
      </span>
      <Popover open={isOpen} onOpenChange={onOpenChange}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="text-muted hover:text-basis ml-auto flex shrink-0 items-center"
          >
            <RiSettings3Line className="h-3.5 w-3.5" />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start">
          <MetricAccordionItem
            metric={metric}
            pointsLeft={pointsLeft}
            collapsible={false}
            onUpdate={(patch) => onUpdateMetric(metric.key, patch)}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Add-metric popover
// ---------------------------------------------------------------------------

function AddMetricPopover({
  disabledMetrics,
  onEnable,
}: {
  disabledMetrics: ExperimentScoringMetric[];
  onEnable: (key: string) => void;
}) {
  return (
    <div className="flex w-52 flex-col gap-1 p-2">
      <p className="text-muted px-2 py-1 text-xs font-medium">
        Enable a metric
      </p>
      {disabledMetrics.map((m) => (
        <PopoverClose key={m.key} asChild>
          <button
            type="button"
            className="text-basis hover:bg-canvasSubtle rounded px-2 py-1.5 text-left text-sm"
            onClick={() => onEnable(m.key)}
          >
            {m.displayName}
          </button>
        </PopoverClose>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function VariantsTable({
  variants,
  scoringConfig,
  onUpdateMetric,
  onEnableMetric,
  pointsLeft,
  onOpenInsights,
  showInactive,
  onShowInactiveChange,
}: Props) {
  const [openMetricPopover, setOpenMetricPopover] = useState<string | null>(
    null,
  );

  const enabledMetrics = useMemo(
    () =>
      [...scoringConfig]
        .filter((m) => m.enabled)
        .sort((a, b) => b.points - a.points),
    [scoringConfig],
  );

  const disabledMetrics = useMemo(
    () => scoringConfig.filter((m) => !m.enabled),
    [scoringConfig],
  );

  const rows: RowData[] = useMemo(() => {
    const allRows = variants.map((v) => {
      const result = scoreVariant(v.metrics, scoringConfig);
      return { ...v, score: result.total };
    });

    const filtered = showInactive
      ? allRows
      : allRows.filter((r) => r.runCount > 0);
    return filtered.sort((a, b) => b.score - a.score);
  }, [variants, scoringConfig, showInactive]);

  const { bestScore, worstScore } = useMemo(() => {
    const active = rows.filter((r) => r.runCount > 0);
    if (active.length < 2) return { bestScore: null, worstScore: null };
    let best = -Infinity;
    let worst = Infinity;
    for (const r of active) {
      if (r.score > best) best = r.score;
      if (r.score < worst) worst = r.score;
    }
    return { bestScore: best, worstScore: worst };
  }, [rows]);

  // Pre-compute stats for each enabled metric
  const statsMap = useMemo(() => {
    const map = new Map<string, MetricStats | null>();
    for (const m of enabledMetrics) {
      map.set(m.key, computeMetricStats(rows, m.key, m.invert));
    }
    return map;
  }, [rows, enabledMetrics]);

  const columns = useMemo(() => {
    const cols: ColumnDef<RowData, any>[] = [];

    // 1. Score
    cols.push(
      columnHelper.accessor('score', {
        header: 'Score',
        cell: (info) => {
          const val = info.getValue();
          const hasRuns = info.row.original.runCount > 0;
          let kind: 'primary' | 'error' | 'default' = 'default';
          if (hasRuns && bestScore !== null && worstScore !== null) {
            if (val === bestScore) {
              kind = 'primary';
            } else if (val === worstScore) {
              kind = 'error';
            }
          }
          return (
            <Pill kind={kind} appearance="solid">
              {Math.round(val)}
            </Pill>
          );
        },
        enableSorting: false,
      }),
    );

    // 2. Variant name
    cols.push(
      columnHelper.accessor('variantName', {
        header: 'Variant',
        cell: (info) => (
          <span className="text-basis text-sm font-medium">
            {info.getValue()}
          </span>
        ),
        enableSorting: false,
      }),
    );

    // 3. One column per enabled metric
    for (const metric of enabledMetrics) {
      cols.push(
        columnHelper.display({
          id: `metric_${metric.key}`,
          header: () => (
            <MetricColumnHeader
              metric={metric}
              pointsLeft={pointsLeft}
              onUpdateMetric={onUpdateMetric}
              isOpen={openMetricPopover === metric.key}
              onOpenChange={(open) =>
                setOpenMetricPopover(open ? metric.key : null)
              }
            />
          ),
          cell: (info) => {
            const row = info.row.original;
            const vm: VariantMetric | undefined = row.metrics.find(
              (m) => m.key === metric.key,
            );
            if (!vm) {
              return <span className="text-muted text-sm">-</span>;
            }

            const stats = statsMap.get(metric.key) ?? null;

            return (
              <div className="flex flex-col">
                <span className="text-basis text-sm tabular-nums">
                  {formatMetricValue(vm.avg)}
                </span>
                <MetricSubLabel
                  variantName={row.variantName}
                  avg={vm.avg}
                  stats={stats}
                  metric={metric}
                />
              </div>
            );
          },
          enableSorting: false,
        }),
      );
    }

    // 4. Add metric column
    cols.push(
      columnHelper.display({
        id: '__add_metric',
        header: () => (
          <Popover>
            <PopoverTrigger asChild>
              <Button
                kind="secondary"
                appearance="ghost"
                size="small"
                icon={<RiAddLine className="h-3.5 w-3.5" />}
                disabled={disabledMetrics.length === 0}
              />
            </PopoverTrigger>
            {disabledMetrics.length > 0 && (
              <PopoverContent align="end">
                <AddMetricPopover
                  disabledMetrics={disabledMetrics}
                  onEnable={onEnableMetric}
                />
              </PopoverContent>
            )}
          </Popover>
        ),
        cell: () => null,
        enableSorting: false,
      }),
    );

    return cols;
  }, [
    enabledMetrics,
    disabledMetrics,
    rows,
    statsMap,
    pointsLeft,
    bestScore,
    worstScore,
    openMetricPopover,
    onUpdateMetric,
    onEnableMetric,
  ]);

  return (
    <div className="flex flex-col">
      {/* Header toolbar */}
      <div className="flex items-center justify-between py-2">
        <div className="flex flex-col gap-px">
          <span className="text-basis text-sm font-medium">Variants</span>
          <span className="text-subtle text-xs">
            Adjust the scoring weight via the column title or sidebar.
          </span>
        </div>

        <div className="flex items-center gap-2">
          <Button
            kind="primary"
            appearance="ghost"
            size="small"
            label="Open with insights"
            icon={<RiArrowRightUpLine className="h-3.5 w-3.5" />}
            iconSide="left"
            onClick={onOpenInsights}
          />

          <Button
            kind="primary"
            appearance="solid"
            size="small"
            label="Compare"
            disabled
          />

          <Popover>
            <PopoverTrigger asChild>
              <Button
                kind="secondary"
                appearance="outlined"
                size="small"
                icon={<RiMoreFill className="h-4 w-4" />}
              />
            </PopoverTrigger>
            <PopoverContent align="end">
              <div className="flex items-center gap-2 px-3 py-2">
                <div className="flex min-w-0 flex-1 flex-col">
                  <SwitchLabel
                    htmlFor="show-inactive"
                    className="text-basis text-sm"
                  >
                    Show inactive variants
                  </SwitchLabel>
                  <span className="text-subtle text-xs">
                    Includes variants with no recent runs.
                  </span>
                </div>
                <Switch
                  id="show-inactive"
                  checked={showInactive}
                  onCheckedChange={onShowInactiveChange}
                />
              </div>
            </PopoverContent>
          </Popover>
        </div>
      </div>

      <Table
        columns={columns}
        data={rows}
        blankState={
          <span className="text-muted text-sm">No variant data available</span>
        }
        cellClassName="py-2"
      />
    </div>
  );
}
