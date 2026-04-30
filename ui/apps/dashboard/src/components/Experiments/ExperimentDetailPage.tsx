import {
  useCallback,
  useDeferredValue,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import {
  ExperimentsBlankState,
  type ExperimentScoringMetric,
} from '@inngest/components/Experiments';
import {
  HelperPanelControl,
  HelperPanelFrame,
  type HelperItem,
} from '@inngest/components/HelperPanelControl';
import { Header } from '@inngest/components/Header/Header';
import { Skeleton } from '@inngest/components/Skeleton';
import { TableBlankState } from '@inngest/components/Table';
import { ExperimentsIcon } from '@inngest/components/icons/sections/Experiments';
import { subtractDuration } from '@inngest/components/utils/date';
import { RiFlaskLine, RiListOrdered2, RiRefreshLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { ExperimentDetailToolbar } from '@/components/Experiments/ExperimentDetailToolbar';
import { InfoSidebar } from '@/components/Experiments/InfoSidebar';
import { MetricPanel } from '@/components/Experiments/MetricPanel';
import { ScoreSummaryCard } from '@/components/Experiments/ScoreSummaryCard';
import { ScoringFormulaSidebar } from '@/components/Experiments/ScoringFormulaSidebar';
import {
  useExperimentDetail,
  useExperimentInsightsQuery,
  type ExperimentTimeRange as ExperimentQueryTimeRange,
} from '@/components/Experiments/useExperiments';
import { VariantsTable } from '@/components/Experiments/VariantsTable';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { insightsUrl } from '@/lib/experiments/insightsUrl';
import {
  getPointsLeft,
  getScoringMetricsForFormula,
  serializeScoringFormulaOverrides,
  updateScoringMetric,
} from '@/lib/experiments/scoringFormula';
import { findExtremum, scoreVariants } from '@/lib/experiments/score';
import {
  experimentTimeRangeToRangeChange,
  serializeExperimentScoringFormula,
  type ExperimentDetailPanel,
  type ExperimentScoringFormula,
  type ExperimentTimeRange as ExperimentUrlTimeRange,
} from '@/lib/experiments/urlState';
import { pathCreator } from '@/utils/urls';

type Props = {
  experimentName: string;
  // Disambiguates the experiment when two functions in the same workspace
  // declare experiment.weighted("X", ...). Required: detail/scoring/insights
  // queries scope by function ID server-side; passing nothing would silently
  // merge data across all matching functions.
  functionID: string;
  functionName: string;
  functionSlug: string;
  timeRange: ExperimentUrlTimeRange;
  hasTimeRangeSearch: boolean;
  onTimeRangeChange: (range: RangeChangeProps) => void;
  selectedVariants: string[];
  onSelectedVariantsChange: (variants: string[]) => void;
  showInactive: boolean;
  onShowInactiveChange: (showInactive: boolean) => void;
  activePanel: ExperimentDetailPanel;
  onActivePanelChange: (panel: ExperimentDetailPanel) => void;
  scoreFormula: ExperimentScoringFormula | null;
  scoreFormulaParam?: string;
  onScoringFormulaChange: (formulaParam: string | undefined) => void;
};

const PANEL_TITLES = {
  info: 'Info',
  scoring: 'Scoring formula',
} as const satisfies Record<Exclude<ExperimentDetailPanel, 'none'>, string>;
type PanelKey = keyof typeof PANEL_TITLES;
type RelativeDuration = Extract<
  RangeChangeProps,
  { type: 'relative' }
>['duration'];

const SCORE_FORMULA_URL_SYNC_DEBOUNCE_MS = 350;
const MAX_DEFAULT_DAYS = 30;

function rangeToTimeRange(range: RangeChangeProps): ExperimentQueryTimeRange {
  if (range.type === 'absolute') return { from: range.start, to: range.end };
  const to = new Date();
  return { from: subtractDuration(to, range.duration), to };
}

function getDurationScopeKey(duration: RelativeDuration): string {
  const { days, hours, minutes, months, seconds, weeks, years } = duration;
  return `${years ?? 0}:${months ?? 0}:${weeks ?? 0}:${days ?? 0}:${
    hours ?? 0
  }:${minutes ?? 0}:${seconds ?? 0}`;
}

function getScoringScopeKey({
  functionID,
  experimentName,
  range,
}: {
  functionID: string;
  experimentName: string;
  range: RangeChangeProps;
}) {
  const timeRangeKey =
    range.type === 'absolute'
      ? `absolute:${range.start.getTime()}:${range.end.getTime()}`
      : `relative:${getDurationScopeKey(range.duration)}`;
  return `${functionID}:${experimentName}:${timeRangeKey}`;
}

export function ExperimentDetailPage({
  experimentName,
  functionID,
  functionName,
  functionSlug,
  timeRange,
  hasTimeRangeSearch,
  onTimeRangeChange,
  selectedVariants,
  onSelectedVariantsChange,
  showInactive,
  onShowInactiveChange,
  activePanel,
  onActivePanelChange,
  scoreFormula,
  scoreFormulaParam,
  onScoringFormulaChange,
}: Props) {
  const environment = useEnvironment();

  const [{ data: accountData }] = useQuery({
    query: GetAccountEntitlementsDocument,
  });
  const daysAgoMax = accountData?.account.entitlements.history.limit ?? 7;
  const rangeReady = hasTimeRangeSearch || accountData !== undefined;
  const timeRangeType = timeRange.type;
  const liveDurationMs =
    timeRange.type === 'live' ? timeRange.durationMs : null;
  const fixedFromTs = timeRange.type === 'fixed' ? timeRange.fromTs : null;
  const fixedToTs = timeRange.type === 'fixed' ? timeRange.toTs : null;

  const range = useMemo<RangeChangeProps>(() => {
    if (hasTimeRangeSearch) {
      if (
        timeRangeType === 'fixed' &&
        fixedFromTs !== null &&
        fixedToTs !== null
      ) {
        return experimentTimeRangeToRangeChange({
          type: 'fixed',
          fromTs: fixedFromTs,
          toTs: fixedToTs,
          preset: null,
        });
      }

      if (liveDurationMs !== null) {
        return experimentTimeRangeToRangeChange({
          type: 'live',
          durationMs: liveDurationMs,
          preset: null,
        });
      }
    }

    return {
      type: 'relative',
      duration: { days: Math.min(daysAgoMax, MAX_DEFAULT_DAYS) },
    };
  }, [
    daysAgoMax,
    fixedFromTs,
    fixedToTs,
    hasTimeRangeSearch,
    liveDurationMs,
    timeRangeType,
  ]);

  const queryRange = useMemo(() => rangeToTimeRange(range), [range]);

  const detail = useExperimentDetail(
    functionID,
    experimentName,
    queryRange,
    null,
    { enabled: rangeReady },
  );
  const insightsQuery = useExperimentInsightsQuery(
    functionID,
    experimentName,
    queryRange,
    { enabled: rangeReady },
  );

  const activePanelKey = activePanel === 'none' ? null : activePanel;

  const urlScoringMetrics = useMemo(() => {
    if (!detail.data) return null;
    return getScoringMetricsForFormula({
      detail: detail.data,
      formula: scoreFormula,
    });
  }, [detail.data, scoreFormula]);

  const defaultScoringMetrics = useMemo(() => {
    if (!detail.data) return null;
    return getScoringMetricsForFormula({
      detail: detail.data,
      formula: null,
    });
  }, [detail.data]);

  const serializedUrlScoringMetrics = useMemo(() => {
    if (!urlScoringMetrics || urlScoringMetrics.length === 0) return undefined;
    return serializeExperimentScoringFormula(urlScoringMetrics);
  }, [urlScoringMetrics]);

  const [draftScoringMetrics, setDraftScoringMetrics] = useState<
    ExperimentScoringMetric[] | null
  >(null);
  const [pendingScoringFormula, setPendingScoringFormula] = useState<{
    param: string | undefined;
    scopeKey: string;
  } | null>(null);
  const scoringScopeKey = getScoringScopeKey({
    functionID,
    experimentName,
    range,
  });

  useEffect(() => {
    if (!pendingScoringFormula) return;
    if (pendingScoringFormula.scopeKey !== scoringScopeKey) return;

    const timeoutID = window.setTimeout(() => {
      onScoringFormulaChange(pendingScoringFormula.param);
    }, SCORE_FORMULA_URL_SYNC_DEBOUNCE_MS);

    return () => window.clearTimeout(timeoutID);
  }, [onScoringFormulaChange, pendingScoringFormula, scoringScopeKey]);

  useEffect(() => {
    if (!urlScoringMetrics) {
      setDraftScoringMetrics(null);
      setPendingScoringFormula(null);
      return;
    }

    if (pendingScoringFormula) {
      if (pendingScoringFormula.scopeKey !== scoringScopeKey) {
        setPendingScoringFormula(null);
      } else if (scoreFormulaParam !== pendingScoringFormula.param) {
        return;
      } else {
        setPendingScoringFormula(null);
      }
    }

    setDraftScoringMetrics((prev) => {
      if (
        prev &&
        serializedUrlScoringMetrics &&
        serializeExperimentScoringFormula(prev) === serializedUrlScoringMetrics
      ) {
        return prev;
      }

      return urlScoringMetrics;
    });
  }, [
    pendingScoringFormula,
    scoreFormulaParam,
    scoringScopeKey,
    serializedUrlScoringMetrics,
    urlScoringMetrics,
  ]);

  const scoringMetrics = draftScoringMetrics ?? urlScoringMetrics;
  const deferredScoringMetrics = useDeferredValue(scoringMetrics);
  const visualScoringMetrics = deferredScoringMetrics ?? scoringMetrics;

  const pointsLeft = useMemo(
    () => (scoringMetrics ? getPointsLeft(scoringMetrics) : 100),
    [scoringMetrics],
  );

  const togglePanel = useCallback(
    (key: PanelKey) => {
      onActivePanelChange(activePanel === key ? 'none' : key);
    },
    [activePanel, onActivePanelChange],
  );

  const availableVariants = useMemo(
    () => detail.data?.variants.map((v) => v.variantName) ?? [],
    [detail.data],
  );

  const selectedAvailableVariants = useMemo(() => {
    if (selectedVariants.length === 0) return [];
    const available = new Set(availableVariants);
    return selectedVariants.filter((variant) => available.has(variant));
  }, [availableVariants, selectedVariants]);

  const hasUnavailableVariantSelection =
    selectedVariants.length > 0 && selectedAvailableVariants.length === 0;

  const filteredDetail = useMemo(() => {
    if (!detail.data) return null;
    if (selectedVariants.length === 0) return detail.data;
    return {
      ...detail.data,
      variants: detail.data.variants.filter((v) =>
        selectedAvailableVariants.includes(v.variantName),
      ),
    };
  }, [detail.data, selectedAvailableVariants, selectedVariants.length]);

  // Score each variant once so downstream panels (VariantsTable, ScoreSummaryCard,
  // top-variant callout) share the same precomputed results.
  const scoredVariants = useMemo(() => {
    if (!filteredDetail || !visualScoringMetrics) return null;
    return scoreVariants(filteredDetail.variants, visualScoringMetrics);
  }, [filteredDetail, visualScoringMetrics]);

  const topVariantName = useMemo(() => {
    if (!scoredVariants) return null;
    const top = findExtremum(scoredVariants, (s) => s.result.total);
    return top?.variant.variantName ?? null;
  }, [scoredVariants]);

  // Per-metric min/max of observed avg values across all variants, so the
  // scoring inputs can offer a "fit to data" shortcut.
  const metricRanges = useMemo(() => {
    const map: Record<string, { min: number; max: number }> = {};
    if (!filteredDetail) return map;
    for (const variant of filteredDetail.variants) {
      for (const m of variant.metrics) {
        const range = map[m.key];
        if (!range) {
          map[m.key] = { min: m.avg, max: m.avg };
        } else {
          if (m.avg < range.min) range.min = m.avg;
          if (m.avg > range.max) range.max = m.avg;
        }
      }
    }
    return map;
  }, [filteredDetail]);

  const enabledMetrics = useMemo(
    () => (visualScoringMetrics ?? []).filter((m) => m.enabled),
    [visualScoringMetrics],
  );

  const handleUpdateMetric = useCallback(
    (key: string, patch: Partial<ExperimentScoringMetric>) => {
      if (!scoringMetrics || !defaultScoringMetrics) return;

      const next = updateScoringMetric(scoringMetrics, key, patch);
      const formulaParam = serializeScoringFormulaOverrides(
        next,
        defaultScoringMetrics,
      );
      setDraftScoringMetrics(next);
      setPendingScoringFormula({
        param: formulaParam,
        scopeKey: scoringScopeKey,
      });
    },
    [defaultScoringMetrics, scoringMetrics, scoringScopeKey],
  );

  const onOpenInsights = useCallback(() => {
    const sql = insightsQuery.data;
    if (!sql) return;
    window.open(insightsUrl(environment.slug, sql), '_blank');
  }, [insightsQuery.data, environment.slug]);

  const helperItems: HelperItem[] = [
    {
      title: PANEL_TITLES.info,
      icon: <RiFlaskLine className="h-4 w-4" />,
      action: () => togglePanel('info'),
    },
    {
      title: PANEL_TITLES.scoring,
      icon: <RiListOrdered2 className="h-4 w-4" />,
      action: () => togglePanel('scoring'),
    },
  ];

  return (
    <>
      <Header
        breadcrumb={[
          {
            text: 'All experiments',
            href: pathCreator.experiments({ envSlug: environment.slug }),
          },
          {
            text: functionName,
            href: pathCreator.function({
              envSlug: environment.slug,
              functionSlug,
            }),
          },
          { text: experimentName },
        ]}
      />

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <div className="flex min-w-0 flex-1 flex-col gap-4 overflow-y-auto px-6 pb-10 pt-4">
          <h1 className="text-basis text-lg font-semibold">{experimentName}</h1>

          {!rangeReady ? (
            <>
              <Skeleton className="h-8 w-72 rounded-lg" />
              <Skeleton className="h-96 w-full rounded-lg" />
            </>
          ) : (
            <>
              <ExperimentDetailToolbar
                range={range}
                onRangeChange={onTimeRangeChange}
                daysAgoMax={daysAgoMax}
                selectedVariants={selectedVariants}
                onSelectedVariantsChange={onSelectedVariantsChange}
                availableVariants={availableVariants}
              />

              {detail.isPending && (
                <Skeleton className="h-96 w-full rounded-lg" />
              )}

              {detail.error && (
                <ErrorCard
                  error={detail.error}
                  reset={() => detail.refetch()}
                />
              )}

              {!detail.isPending && !detail.error && detail.data === null && (
                <ExperimentsBlankState
                  title="No runs in this time range"
                  description="Try selecting a wider time range."
                  onRefresh={detail.refetch}
                />
              )}

              {filteredDetail &&
                scoringMetrics &&
                visualScoringMetrics &&
                scoredVariants &&
                (filteredDetail.variants.length === 0 ? (
                  hasUnavailableVariantSelection ? (
                    <TableBlankState
                      icon={<ExperimentsIcon />}
                      title="No selected variants in this time range"
                      description="Clear the variant filter or choose a wider time range."
                      actions={
                        <>
                          <Button
                            kind="primary"
                            appearance="solid"
                            label="Clear variant filter"
                            onClick={() => onSelectedVariantsChange([])}
                          />
                          <Button
                            kind="primary"
                            appearance="outlined"
                            label="Refresh"
                            icon={<RiRefreshLine />}
                            iconSide="left"
                            onClick={() => detail.refetch()}
                          />
                        </>
                      }
                    />
                  ) : (
                    <ExperimentsBlankState
                      title="No variant data yet"
                      description={
                        <>
                          Once your function emits runs for this experiment via{' '}
                          <InlineCode>group.experiment()</InlineCode>, variant
                          metrics will appear here.
                        </>
                      }
                      onRefresh={detail.refetch}
                    />
                  )
                ) : (
                  <div className="@container">
                    <div className="grid grid-cols-1 gap-3 @[800px]:grid-cols-2 @[1200px]:grid-cols-3">
                      <ScoreSummaryCard
                        className="col-span-full"
                        scoredVariants={scoredVariants}
                        metrics={visualScoringMetrics}
                      />

                      {enabledMetrics.map((metric) => (
                        <MetricPanel
                          key={metric.key}
                          metric={metric}
                          variants={filteredDetail.variants}
                        />
                      ))}

                      <VariantsTable
                        className="col-span-full"
                        scoredVariants={scoredVariants}
                        scoringConfig={scoringMetrics}
                        metricRanges={metricRanges}
                        onUpdateMetric={handleUpdateMetric}
                        pointsLeft={pointsLeft}
                        onOpenInsights={onOpenInsights}
                        showInactive={showInactive}
                        onShowInactiveChange={onShowInactiveChange}
                      />
                    </div>
                  </div>
                ))}
            </>
          )}
        </div>

        {activePanelKey && (
          <aside className="border-subtle flex w-[360px] shrink-0 flex-col overflow-hidden border-l">
            <HelperPanelFrame
              title={PANEL_TITLES[activePanelKey]}
              icon={
                helperItems.find(
                  (i) => i.title === PANEL_TITLES[activePanelKey],
                )?.icon
              }
              onClose={() => onActivePanelChange('none')}
            >
              {activePanelKey === 'info' && detail.data && (
                <InfoSidebar
                  detail={detail.data}
                  topVariantName={topVariantName}
                />
              )}
              {activePanelKey === 'scoring' && scoringMetrics && (
                <ScoringFormulaSidebar
                  metrics={scoringMetrics}
                  metricRanges={metricRanges}
                  onUpdateMetric={handleUpdateMetric}
                  pointsLeft={pointsLeft}
                />
              )}
            </HelperPanelFrame>
          </aside>
        )}

        <HelperPanelControl
          items={helperItems}
          activeTitle={activePanelKey ? PANEL_TITLES[activePanelKey] : null}
        />
      </div>
    </>
  );
}
