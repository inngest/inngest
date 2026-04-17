import { useCallback, useMemo, useState } from 'react';
import { InlineCode } from '@inngest/components/Code';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import {
  ExperimentsBlankState,
  type TimeRangePreset,
} from '@inngest/components/Experiments';
import {
  HelperPanelControl,
  type HelperItem,
} from '@inngest/components/HelperPanelControl';
import { Header } from '@inngest/components/Header/Header';
import { Skeleton } from '@inngest/components/Skeleton';
import { RiCloseLine, RiFlaskLine, RiListOrdered2 } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { ExperimentDetailToolbar } from '@/components/Experiments/ExperimentDetailToolbar';
import { InfoSidebar } from '@/components/Experiments/InfoSidebar';
import { MetricPanel } from '@/components/Experiments/MetricPanel';
import { ScoreSummaryCard } from '@/components/Experiments/ScoreSummaryCard';
import { ScoringFormulaSidebar } from '@/components/Experiments/ScoringFormulaSidebar';
import { useExperimentDetail } from '@/components/Experiments/useExperiments';
import { useScoringConfig } from '@/components/Experiments/useScoringConfig';
import { VariantsTable } from '@/components/Experiments/VariantsTable';
import { findExtremum, scoreVariants } from '@/lib/experiments/score';
import { pathCreator } from '@/utils/urls';

type Props = {
  experimentName: string;
};

export function ExperimentDetailPage({ experimentName }: Props) {
  const environment = useEnvironment();

  // --- state ---
  const [preset, setPreset] = useState<TimeRangePreset>('24h');
  const [selectedVariants, setSelectedVariants] = useState<string[]>([]);
  const [showInactive, setShowInactive] = useState(false);
  const [activePanel, setActivePanel] = useState<string | null>('Info');
  // --- hooks ---
  const detail = useExperimentDetail(experimentName, preset, null);
  const scoring = useScoringConfig(experimentName);

  // --- available variants for toolbar filter ---
  const availableVariants = useMemo(
    () => detail.data?.variants.map((v) => v.variantName) ?? [],
    [detail.data],
  );

  // --- filtered variants based on multi-select ---
  const filteredDetail = useMemo(() => {
    if (!detail.data) return null;
    if (selectedVariants.length === 0) return detail.data;
    return {
      ...detail.data,
      variants: detail.data.variants.filter((v) =>
        selectedVariants.includes(v.variantName),
      ),
    };
  }, [detail.data, selectedVariants]);

  // Score each variant once so downstream panels (VariantsTable, ScoreSummaryCard,
  // top-variant callout) share the same precomputed results.
  const scoredVariants = useMemo(() => {
    if (!filteredDetail || !scoring.metrics) return null;
    return scoreVariants(filteredDetail.variants, scoring.metrics);
  }, [filteredDetail, scoring.metrics]);

  const topVariantName = useMemo(() => {
    if (!scoredVariants) return null;
    const top = findExtremum(scoredVariants, (s) => s.result.total);
    return top?.variant.variantName ?? null;
  }, [scoredVariants]);

  // --- enabled metrics for MetricPanel grid ---
  const enabledMetrics = useMemo(
    () => (scoring.metrics ?? []).filter((m) => m.enabled),
    [scoring.metrics],
  );

  // --- onOpenInsights callback ---
  const onOpenInsights = useCallback(() => {
    const sql = `SELECT * FROM isteps WHERE \`inngest.experiment.values.experiment_name\` = '${experimentName.replace(
      /'/g,
      "''",
    )}' ORDER BY started_at DESC`;
    window.open(
      `/env/${environment.slug}/insights?sql=${encodeURIComponent(sql)}`,
      '_blank',
    );
  }, [experimentName, environment.slug]);

  const helperItems: HelperItem[] = [
    {
      title: 'Info',
      icon: <RiFlaskLine className="h-4 w-4" />,
      action: () => setActivePanel((p) => (p === 'Info' ? null : 'Info')),
    },
    {
      title: 'Scoring formula',
      icon: <RiListOrdered2 className="h-4 w-4" />,
      action: () =>
        setActivePanel((p) =>
          p === 'Scoring formula' ? null : 'Scoring formula',
        ),
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
          { text: experimentName },
        ]}
      />

      <div className="flex min-h-0 flex-1 overflow-y-auto">
        {/* Main content */}
        <div className="flex min-w-0 flex-1 flex-col gap-4 px-6 py-4">
          <h1 className="text-basis text-lg font-semibold">{experimentName}</h1>

          <ExperimentDetailToolbar
            preset={preset}
            onPresetChange={setPreset}
            selectedVariants={selectedVariants}
            onSelectedVariantsChange={setSelectedVariants}
            availableVariants={availableVariants}
          />

          {/* Loading state */}
          {(detail.isPending || scoring.isPending) && (
            <Skeleton className="h-96 w-full rounded-lg" />
          )}

          {/* Error states */}
          {detail.error && (
            <ErrorCard error={detail.error} reset={() => detail.refetch()} />
          )}
          {scoring.error && (
            <ErrorCard error={scoring.error} reset={() => scoring.refetch()} />
          )}

          {/* Results */}
          {filteredDetail &&
            scoring.metrics &&
            scoredVariants &&
            (filteredDetail.variants.length === 0 ? (
              <ExperimentsBlankState
                title="No variant data yet"
                description={
                  <>
                    Once your function emits runs for this experiment via{' '}
                    <InlineCode>group.experiment()</InlineCode>, variant metrics
                    will appear here.
                  </>
                }
                onRefresh={detail.refetch}
              />
            ) : (
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
                <div className="col-span-1 md:col-span-2 xl:col-span-3">
                  <ScoreSummaryCard
                    scoredVariants={scoredVariants}
                    metrics={scoring.metrics}
                  />
                </div>

                {enabledMetrics.map((metric, i) => (
                  <MetricPanel
                    key={metric.key}
                    metric={metric}
                    variants={filteredDetail.variants}
                    colorIndex={i}
                  />
                ))}

                <div className="col-span-1 md:col-span-2 xl:col-span-3">
                  <VariantsTable
                    scoredVariants={scoredVariants}
                    scoringConfig={scoring.metrics}
                    onUpdateMetric={scoring.updateMetric}
                    onEnableMetric={scoring.enableMetric}
                    pointsLeft={scoring.pointsLeft}
                    onOpenInsights={onOpenInsights}
                    showInactive={showInactive}
                    onShowInactiveChange={setShowInactive}
                  />
                </div>
              </div>
            ))}
        </div>

        {/* Sidebar panel */}
        {activePanel && (
          <aside className="border-subtle flex w-[360px] shrink-0 flex-col overflow-hidden border-l">
            {/* Panel header — matches InsightsHelperPanel */}
            <div className="border-subtle flex h-[49px] shrink-0 flex-row items-center justify-between border-b px-3">
              <div className="flex flex-row items-center gap-2">
                {helperItems.find((i) => i.title === activePanel)?.icon ?? null}
                <div className="text-sm font-normal">{activePanel}</div>
              </div>
              <button
                aria-label="Close panel"
                className="hover:bg-canvasSubtle hover:text-basis text-subtle -mr-1 flex h-8 w-8 items-center justify-center rounded-md transition-colors"
                onClick={() => setActivePanel(null)}
                type="button"
              >
                <RiCloseLine className="h-4 w-4" />
              </button>
            </div>
            {/* Panel content */}
            <div className="flex-1 overflow-y-auto">
              {activePanel === 'Info' && detail.data && (
                <InfoSidebar
                  detail={detail.data}
                  topVariantName={topVariantName}
                />
              )}
              {activePanel === 'Scoring formula' && scoring.metrics && (
                <ScoringFormulaSidebar
                  metrics={scoring.metrics}
                  onUpdateMetric={scoring.updateMetric}
                  pointsLeft={scoring.pointsLeft}
                  isSaving={scoring.isSaving}
                />
              )}
            </div>
          </aside>
        )}

        {/* Icon rail — far right */}
        <HelperPanelControl items={helperItems} activeTitle={activePanel} />
      </div>
    </>
  );
}
