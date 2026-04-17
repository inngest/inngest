import { useCallback, useMemo, useState } from 'react';
import { InlineCode } from '@inngest/components/Code';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import {
  ExperimentsBlankState,
  type TimeRangePreset,
} from '@inngest/components/Experiments';
import {
  HelperPanelControl,
  HelperPanelFrame,
  type HelperItem,
} from '@inngest/components/HelperPanelControl';
import { Header } from '@inngest/components/Header/Header';
import { Skeleton } from '@inngest/components/Skeleton';
import { RiFlaskLine, RiListOrdered2 } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { ExperimentDetailToolbar } from '@/components/Experiments/ExperimentDetailToolbar';
import { InfoSidebar } from '@/components/Experiments/InfoSidebar';
import { MetricPanel } from '@/components/Experiments/MetricPanel';
import { ScoreSummaryCard } from '@/components/Experiments/ScoreSummaryCard';
import { ScoringFormulaSidebar } from '@/components/Experiments/ScoringFormulaSidebar';
import { useExperimentDetail } from '@/components/Experiments/useExperiments';
import { useScoringConfig } from '@/components/Experiments/useScoringConfig';
import { VariantsTable } from '@/components/Experiments/VariantsTable';
import { experimentInsightsUrl } from '@/lib/experiments/insightsUrl';
import { findExtremum, scoreVariants } from '@/lib/experiments/score';
import { pathCreator } from '@/utils/urls';

type Props = {
  experimentName: string;
};

const INFO_PANEL = 'Info';
const SCORING_PANEL = 'Scoring formula';
type PanelKey = typeof INFO_PANEL | typeof SCORING_PANEL;

export function ExperimentDetailPage({ experimentName }: Props) {
  const environment = useEnvironment();

  const [preset, setPreset] = useState<TimeRangePreset>('24h');
  const [selectedVariants, setSelectedVariants] = useState<string[]>([]);
  const [showInactive, setShowInactive] = useState(false);
  const [activePanel, setActivePanel] = useState<PanelKey | null>(INFO_PANEL);

  const detail = useExperimentDetail(experimentName, preset, null);
  const scoring = useScoringConfig(experimentName);

  const togglePanel = useCallback((key: PanelKey) => {
    setActivePanel((p) => (p === key ? null : key));
  }, []);

  const availableVariants = useMemo(
    () => detail.data?.variants.map((v) => v.variantName) ?? [],
    [detail.data],
  );

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

  const enabledMetrics = useMemo(
    () => (scoring.metrics ?? []).filter((m) => m.enabled),
    [scoring.metrics],
  );

  const onOpenInsights = useCallback(() => {
    window.open(
      experimentInsightsUrl(environment.slug, experimentName),
      '_blank',
    );
  }, [experimentName, environment.slug]);

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
          { text: experimentName },
        ]}
      />

      <div className="flex min-h-0 flex-1 overflow-y-auto">
        <div className="flex min-w-0 flex-1 flex-col gap-4 px-6 py-4">
          <h1 className="text-basis text-lg font-semibold">{experimentName}</h1>

          <ExperimentDetailToolbar
            preset={preset}
            onPresetChange={setPreset}
            selectedVariants={selectedVariants}
            onSelectedVariantsChange={setSelectedVariants}
            availableVariants={availableVariants}
          />

          {(detail.isPending || scoring.isPending) && (
            <Skeleton className="h-96 w-full rounded-lg" />
          )}

          {detail.error && (
            <ErrorCard error={detail.error} reset={() => detail.refetch()} />
          )}
          {scoring.error && (
            <ErrorCard error={scoring.error} reset={() => scoring.refetch()} />
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
                <ScoreSummaryCard
                  className="col-span-1 md:col-span-2 xl:col-span-3"
                  scoredVariants={scoredVariants}
                  metrics={scoring.metrics}
                />

                {enabledMetrics.map((metric, i) => (
                  <MetricPanel
                    key={metric.key}
                    metric={metric}
                    variants={filteredDetail.variants}
                    colorIndex={i}
                  />
                ))}

                <VariantsTable
                  className="col-span-1 md:col-span-2 xl:col-span-3"
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
            ))}
        </div>

        {activePanel && (
          <aside className="border-subtle flex w-[360px] shrink-0 flex-col overflow-hidden border-l">
            <HelperPanelFrame
              title={activePanel}
              icon={helperItems.find((i) => i.title === activePanel)?.icon}
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
