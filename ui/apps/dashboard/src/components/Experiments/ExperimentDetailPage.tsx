import { useCallback, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { ExperimentsBlankState } from '@inngest/components/Experiments';
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
  type ExperimentTimeRange,
} from '@/components/Experiments/useExperiments';
import { useScoringConfig } from '@/components/Experiments/useScoringConfig';
import { VariantsTable } from '@/components/Experiments/VariantsTable';
import { GetAccountEntitlementsDocument } from '@/gql/graphql';
import { insightsUrl } from '@/lib/experiments/insightsUrl';
import { findExtremum, scoreVariants } from '@/lib/experiments/score';
import {
  experimentTimeRangeToRangeChange,
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
};

const INFO_PANEL = 'Info';
const SCORING_PANEL = 'Scoring formula';
type PanelKey = typeof INFO_PANEL | typeof SCORING_PANEL;

// TimeFilter's longest preset is "Last 30 days". Don't default the range
// past that even if the account's history retention is longer.
const MAX_DEFAULT_DAYS = 30;

function rangeToTimeRange(range: RangeChangeProps): ExperimentTimeRange {
  if (range.type === 'absolute') return { from: range.start, to: range.end };
  const to = new Date();
  return { from: subtractDuration(to, range.duration), to };
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

  const [showInactive, setShowInactive] = useState(false);
  const [activePanel, setActivePanel] = useState<PanelKey | null>(INFO_PANEL);

  const queryRange = useMemo<ExperimentTimeRange>(() => {
    return rangeToTimeRange(range);
  }, [range]);

  const detail = useExperimentDetail(
    functionID,
    experimentName,
    queryRange,
    null,
    { enabled: rangeReady },
  );
  const scoring = useScoringConfig(functionID, experimentName);
  const insightsQuery = useExperimentInsightsQuery(
    functionID,
    experimentName,
    queryRange,
    { enabled: rangeReady },
  );

  const togglePanel = useCallback((key: PanelKey) => {
    setActivePanel((panel) => (panel === key ? null : key));
  }, []);

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

  // Score each variant once so downstream panels (VariantsTable,
  // ScoreSummaryCard, top-variant callout) share the same precomputed results.
  const scoredVariants = useMemo(() => {
    if (!filteredDetail || !scoring.metrics) return null;
    return scoreVariants(filteredDetail.variants, scoring.metrics);
  }, [filteredDetail, scoring.metrics]);

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
      for (const metric of variant.metrics) {
        const range = map[metric.key];
        if (!range) {
          map[metric.key] = { min: metric.avg, max: metric.avg };
        } else {
          if (metric.avg < range.min) range.min = metric.avg;
          if (metric.avg > range.max) range.max = metric.avg;
        }
      }
    }
    return map;
  }, [filteredDetail]);

  const enabledMetrics = useMemo(
    () => (scoring.metrics ?? []).filter((metric) => metric.enabled),
    [scoring.metrics],
  );

  const onOpenInsights = useCallback(() => {
    const sql = insightsQuery.data;
    if (!sql) return;
    window.open(insightsUrl(environment.slug, sql), '_blank');
  }, [insightsQuery.data, environment.slug]);

  const helperItems: HelperItem[] = [
    {
      title: INFO_PANEL,
      icon: <RiFlaskLine className="h-4 w-4" />,
      action: () => togglePanel(INFO_PANEL),
    },
    {
      title: SCORING_PANEL,
      icon: <RiListOrdered2 className="h-4 w-4" />,
      action: () => togglePanel(SCORING_PANEL),
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

              {(detail.isPending || scoring.isPending) && (
                <Skeleton className="h-96 w-full rounded-lg" />
              )}

              {detail.error && (
                <ErrorCard
                  error={detail.error}
                  reset={() => detail.refetch()}
                />
              )}

              {scoring.error && (
                <ErrorCard
                  error={scoring.error}
                  reset={() => scoring.refetch()}
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
                scoring.metrics &&
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
                        metrics={scoring.metrics}
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
                        scoringConfig={scoring.metrics}
                        metricRanges={metricRanges}
                        onUpdateMetric={scoring.updateMetric}
                        pointsLeft={scoring.pointsLeft}
                        onOpenInsights={onOpenInsights}
                        showInactive={showInactive}
                        onShowInactiveChange={setShowInactive}
                      />
                    </div>
                  </div>
                ))}
            </>
          )}
        </div>

        {activePanel && (
          <aside className="border-subtle flex w-[360px] shrink-0 flex-col overflow-hidden border-l">
            <HelperPanelFrame
              title={activePanel}
              icon={
                helperItems.find((item) => item.title === activePanel)?.icon
              }
              onClose={() => setActivePanel(null)}
            >
              {activePanel === INFO_PANEL && detail.data && (
                <InfoSidebar
                  detail={detail.data}
                  topVariantName={topVariantName}
                />
              )}
              {activePanel === SCORING_PANEL && scoring.metrics && (
                <ScoringFormulaSidebar
                  metrics={scoring.metrics}
                  metricRanges={metricRanges}
                  onUpdateMetric={scoring.updateMetric}
                  pointsLeft={scoring.pointsLeft}
                />
              )}
            </HelperPanelFrame>
          </aside>
        )}

        <HelperPanelControl items={helperItems} activeTitle={activePanel} />
      </div>
    </>
  );
}
