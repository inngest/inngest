import { useMemo } from 'react';
import { Button } from '@inngest/components/Button';
import type {
  ExperimentScoringMetric,
  ExperimentVariantMetrics,
  VariantMetric,
} from '@inngest/components/Experiments';
import { Pill } from '@inngest/components/Pill';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { Switch, SwitchLabel } from '@inngest/components/Switch';
import { Table } from '@inngest/components/Table';
import { cn } from '@inngest/components/utils/classNames';
import { RiAddLine, RiArrowRightUpLine, RiMoreFill } from '@remixicon/react';
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table';

import { findBestAndWorst, type ScoredVariant } from '@/lib/experiments/score';
import {
  computeMetricStats,
  formatMetricValue,
  type MetricStats,
} from './variantsTable/metricStats';
import {
  AddMetricPopover,
  MetricColumnHeader,
  MetricSubLabel,
} from './variantsTable/parts';

type Props = {
  scoredVariants: ScoredVariant[];
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
  className?: string;
};

type RowData = ExperimentVariantMetrics & {
  score: number;
  /** O(1) lookup from metric key → VariantMetric, to avoid `.find()` in cell renders. */
  metricsByKey: Map<string, VariantMetric>;
};

const columnHelper = createColumnHelper<RowData>();

export function VariantsTable({
  scoredVariants,
  scoringConfig,
  onUpdateMetric,
  onEnableMetric,
  pointsLeft,
  onOpenInsights,
  showInactive,
  onShowInactiveChange,
  className,
}: Props) {
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
    const allRows = scoredVariants.map(({ variant, result }) => ({
      ...variant,
      score: result.total,
      metricsByKey: new Map(variant.metrics.map((m) => [m.key, m])),
    }));
    const filtered = showInactive
      ? allRows
      : allRows.filter((r) => r.runCount > 0);
    return filtered.sort((a, b) => b.score - a.score);
  }, [scoredVariants, showInactive]);

  const { bestScore, worstScore } = useMemo(() => {
    const active = rows.filter((r) => r.runCount > 0);
    const pair = findBestAndWorst(active, (r) => r.score);
    if (!pair) return { bestScore: null, worstScore: null };
    return { bestScore: pair.best.score, worstScore: pair.worst.score };
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
    // tanstack-react-table's ColumnDef is invariant in its value type, so a
    // mixed list of accessor columns (score: number, variantName: string) plus
    // display columns only unifies at `any`. Scoped to this local array.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const cols: ColumnDef<RowData, any>[] = [];

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

    for (const metric of enabledMetrics) {
      cols.push(
        columnHelper.display({
          id: `metric_${metric.key}`,
          header: () => (
            <MetricColumnHeader
              metric={metric}
              pointsLeft={pointsLeft}
              onUpdateMetric={onUpdateMetric}
            />
          ),
          cell: (info) => {
            const row = info.row.original;
            const vm = row.metricsByKey.get(metric.key);
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
    statsMap,
    pointsLeft,
    bestScore,
    worstScore,
    onUpdateMetric,
    onEnableMetric,
  ]);

  return (
    <div className={cn('flex flex-col', className)}>
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
