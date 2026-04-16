import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import type {
  ExperimentScoringMetric,
  ExperimentVariantMetrics,
  VariantMetric,
} from '@inngest/components/Experiments';
import { Input } from '@inngest/components/Forms/Input';
import { Pill } from '@inngest/components/Pill';
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch';
import { Table } from '@inngest/components/Table';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import {
  RiAddLine,
  RiArrowRightUpLine,
  RiEqualizerLine,
  RiMore2Line,
} from '@remixicon/react';
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table';

import { scoreVariant } from '@/lib/experiments/score';

type Props = {
  variants: ExperimentVariantMetrics[];
  scoringConfig: ExperimentScoringMetric[];
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
  onEnableMetric: (key: string) => void;
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
// Metric editor popover (with draft state)
// ---------------------------------------------------------------------------

function MetricEditorPopover({
  metric,
  onApply,
  maxPoints,
}: {
  metric: ExperimentScoringMetric;
  onApply: (patch: Partial<ExperimentScoringMetric>) => void;
  maxPoints: number;
}) {
  const [draft, setDraft] = useState<Partial<ExperimentScoringMetric>>({});

  const current = { ...metric, ...draft };

  function reset() {
    setDraft({});
  }

  function apply() {
    onApply(draft);
    setDraft({});
  }

  return (
    <div className="flex w-64 flex-col gap-3 p-3">
      <p className="text-basis text-xs font-medium">Edit metric</p>

      <Input
        label="Display name"
        inngestSize="small"
        value={current.displayName}
        onChange={(e) => setDraft({ ...draft, displayName: e.target.value })}
      />

      <div className="grid grid-cols-2 gap-2">
        <Input
          label="Min"
          inngestSize="small"
          type="number"
          value={current.minValue}
          onChange={(e) =>
            setDraft({ ...draft, minValue: parseFloat(e.target.value) || 0 })
          }
        />
        <Input
          label="Max"
          inngestSize="small"
          type="number"
          value={current.maxValue}
          onChange={(e) =>
            setDraft({ ...draft, maxValue: parseFloat(e.target.value) || 0 })
          }
        />
      </div>

      <Input
        label="Points"
        inngestSize="small"
        type="number"
        min={0}
        max={maxPoints}
        value={current.points}
        onChange={(e) => {
          const parsed = parseInt(e.target.value, 10) || 0;
          setDraft({
            ...draft,
            points: Math.max(0, Math.min(maxPoints, parsed)),
          });
        }}
      />

      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={current.invert}
          onChange={(e) => setDraft({ ...draft, invert: e.target.checked })}
          className="accent-primary-moderate h-3.5 w-3.5 rounded"
        />
        <span className="text-basis text-xs">Invert (lower is better)</span>
      </label>

      <div className="flex items-center justify-end gap-2">
        <PopoverClose asChild>
          <Button
            kind="secondary"
            appearance="ghost"
            size="small"
            label="Cancel"
            onClick={reset}
          />
        </PopoverClose>
        <PopoverClose asChild>
          <Button
            kind="primary"
            appearance="solid"
            size="small"
            label="Apply"
            onClick={apply}
          />
        </PopoverClose>
      </div>
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
  onOpenInsights,
  showInactive,
  onShowInactiveChange,
}: Props) {
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());

  const enabledMetrics = useMemo(
    () =>
      [...scoringConfig]
        .filter((m) => m.enabled)
        .sort((a, b) => b.points - a.points),
    [scoringConfig],
  );

  const totalAllocated = useMemo(
    () =>
      scoringConfig
        .filter((m) => m.enabled)
        .reduce((sum, m) => sum + m.points, 0),
    [scoringConfig],
  );
  const pointsLeft = 100 - totalAllocated;

  const disabledMetrics = useMemo(
    () => scoringConfig.filter((m) => !m.enabled),
    [scoringConfig],
  );

  const rows: RowData[] = useMemo(() => {
    const allRows = variants.map((v) => {
      const result = scoreVariant(v.metrics, scoringConfig);
      return { ...v, score: result.total };
    });

    if (showInactive) return allRows;
    return allRows.filter((r) => r.runCount > 0);
  }, [variants, scoringConfig, showInactive]);

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

    // 1. Select checkbox
    cols.push(
      columnHelper.display({
        id: '__select',
        header: () => (
          <input
            type="checkbox"
            className="accent-primary-moderate h-3.5 w-3.5 rounded"
            checked={selectedRows.size > 0 && selectedRows.size === rows.length}
            onChange={(e) => {
              if (e.target.checked) {
                setSelectedRows(new Set(rows.map((r) => r.variantName)));
              } else {
                setSelectedRows(new Set());
              }
            }}
          />
        ),
        cell: (info) => {
          const name = info.row.original.variantName;
          return (
            <input
              type="checkbox"
              className="accent-primary-moderate h-3.5 w-3.5 rounded"
              checked={selectedRows.has(name)}
              onChange={(e) => {
                const next = new Set(selectedRows);
                if (e.target.checked) {
                  next.add(name);
                } else {
                  next.delete(name);
                }
                setSelectedRows(next);
              }}
            />
          );
        },
        enableSorting: false,
      }),
    );

    // 2. Score
    cols.push(
      columnHelper.accessor('score', {
        header: 'Score',
        cell: (info) => {
          const val = info.getValue();
          const hasRuns = info.row.original.runCount > 0;
          return (
            <Pill kind={hasRuns ? 'primary' : 'default'} appearance="outlined">
              {Math.round(val)}
            </Pill>
          );
        },
        enableSorting: false,
      }),
    );

    // 3. Variant name
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

    // 4. One column per enabled metric
    for (const metric of enabledMetrics) {
      cols.push(
        columnHelper.display({
          id: `metric_${metric.key}`,
          header: () => (
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="text-muted hover:text-basis flex items-center gap-1 text-xs font-medium"
                >
                  {metric.displayName}
                  <RiEqualizerLine className="h-3 w-3" />
                </button>
              </PopoverTrigger>
              <PopoverContent align="start">
                <MetricEditorPopover
                  metric={metric}
                  maxPoints={metric.points + pointsLeft}
                  onApply={(patch) => onUpdateMetric(metric.key, patch)}
                />
              </PopoverContent>
            </Popover>
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

    // 5. Add metric column
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
    selectedRows,
    rows,
    statsMap,
    pointsLeft,
    onUpdateMetric,
    onEnableMetric,
  ]);

  return (
    <div className="flex flex-col">
      <Table
        columns={columns}
        data={rows}
        blankState={
          <span className="text-muted text-sm">No variant data available</span>
        }
        cellClassName="py-2"
      />

      {/* Footer toolbar */}
      <div className="border-subtle flex items-center gap-2 border-t px-4 py-2">
        <Button
          kind="secondary"
          appearance="ghost"
          size="small"
          label="Open with insights"
          icon={<RiArrowRightUpLine className="h-3.5 w-3.5" />}
          iconSide="left"
          onClick={onOpenInsights}
        />

        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button
                kind="secondary"
                appearance="ghost"
                size="small"
                label="Compare"
                disabled
              />
            </span>
          </TooltipTrigger>
          <TooltipContent>Coming soon</TooltipContent>
        </Tooltip>

        <div className="flex-1" />

        <Popover>
          <PopoverTrigger asChild>
            <Button
              kind="secondary"
              appearance="ghost"
              size="small"
              icon={<RiMore2Line className="h-4 w-4" />}
            />
          </PopoverTrigger>
          <PopoverContent align="end">
            <div className="p-3">
              <SwitchWrapper>
                <Switch
                  id="show-inactive"
                  checked={showInactive}
                  onCheckedChange={onShowInactiveChange}
                />
                <SwitchLabel htmlFor="show-inactive">
                  Show inactive variants
                </SwitchLabel>
              </SwitchWrapper>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}
