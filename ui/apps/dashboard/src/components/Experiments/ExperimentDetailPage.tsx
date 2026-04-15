import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import type {
  ExperimentScoringMetric,
  TimeRangePreset,
} from '@inngest/components/Experiments';
import { Header } from '@inngest/components/Header/Header';
import { Skeleton } from '@inngest/components/Skeleton';

import { useEnvironment } from '@/components/Environments/environment-context';
import { ExperimentDetailToolbar } from '@/components/Experiments/ExperimentDetailToolbar';
import { InfoSidebar } from '@/components/Experiments/InfoSidebar';
import { MetricPanel } from '@/components/Experiments/MetricPanel';
import { ScoreSummaryCard } from '@/components/Experiments/ScoreSummaryCard';
import { ScoringFormulaSidebar } from '@/components/Experiments/ScoringFormulaSidebar';
import {
  SidebarRail,
  type SidebarTab,
} from '@/components/Experiments/SidebarRail';
import {
  useExperimentDetail,
  useExperimentScoringConfig,
  useUpdateExperimentScoringConfig,
} from '@/components/Experiments/useExperiments';
import { VariantsTable } from '@/components/Experiments/VariantsTable';
import { scoreVariant } from '@/lib/experiments/score';
import { pathCreator } from '@/utils/urls';

type Props = {
  experimentName: string;
};

export function ExperimentDetailPage({ experimentName }: Props) {
  const environment = useEnvironment();

  // --- state ---
  const [preset, setPreset] = useState<TimeRangePreset>('24h');
  const [variantFilter, setVariantFilter] = useState<string | null>(null);
  const [showInactive, setShowInactive] = useState(false);
  const [tab, setTab] = useState<SidebarTab>('scoring');
  const [localMetrics, setLocalMetrics] = useState<
    ExperimentScoringMetric[] | null
  >(null);

  // --- hooks ---
  const detail = useExperimentDetail(experimentName, preset, variantFilter);
  const scoring = useExperimentScoringConfig(experimentName);
  const updateScoring = useUpdateExperimentScoringConfig(experimentName);

  // Initialize localMetrics from server config
  useEffect(() => {
    if (scoring.data) {
      setLocalMetrics(scoring.data.metrics);
    }
  }, [scoring.data]);

  // --- debounced save ---
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const serverMetricsRef = useRef<ExperimentScoringMetric[] | null>(null);

  useEffect(() => {
    serverMetricsRef.current = scoring.data?.metrics ?? null;
  }, [scoring.data]);

  useEffect(() => {
    if (!localMetrics) return;
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      const serverMetrics = serverMetricsRef.current;
      if (
        serverMetrics &&
        JSON.stringify(localMetrics) !== JSON.stringify(serverMetrics)
      ) {
        updateScoring.mutate(localMetrics);
      }
    }, 600);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [localMetrics, updateScoring]);

  // Clean up timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  // --- compute top variant ---
  const topVariant = useMemo(() => {
    if (!detail.data || !localMetrics) return null;
    let bestName: string | null = null;
    let bestScore = -Infinity;
    for (const v of detail.data.variants) {
      const result = scoreVariant(v.metrics, localMetrics);
      if (result.total > bestScore) {
        bestScore = result.total;
        bestName = v.variantName;
      }
    }
    return bestName;
  }, [detail.data, localMetrics]);

  // --- available variants for toolbar filter ---
  const availableVariants = useMemo(
    () => detail.data?.variants.map((v) => v.variantName) ?? [],
    [detail.data],
  );

  // --- enabled metrics for MetricPanel grid ---
  const enabledMetrics = useMemo(
    () => (localMetrics ?? []).filter((m) => m.enabled),
    [localMetrics],
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

  // --- metric update helpers for VariantsTable ---
  const handleUpdateMetric = useCallback(
    (key: string, patch: Partial<ExperimentScoringMetric>) => {
      setLocalMetrics((prev) =>
        prev ? prev.map((m) => (m.key === key ? { ...m, ...patch } : m)) : prev,
      );
    },
    [],
  );

  const handleEnableMetric = useCallback((key: string) => {
    setLocalMetrics((prev) =>
      prev
        ? prev.map((m) => (m.key === key ? { ...m, enabled: true } : m))
        : prev,
    );
  }, []);

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

      <div className="flex flex-1 overflow-hidden">
        {/* Left content */}
        <div className="flex min-w-0 flex-1 flex-col gap-4 overflow-y-auto px-6 py-4">
          <h1 className="text-basis text-lg font-semibold">{experimentName}</h1>

          <ExperimentDetailToolbar
            preset={preset}
            onPresetChange={setPreset}
            variantFilter={variantFilter}
            onVariantFilterChange={setVariantFilter}
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

          {/* Data */}
          {detail.data && localMetrics && (
            <>
              <ScoreSummaryCard
                variants={detail.data.variants}
                metrics={localMetrics}
              />

              <div className="grid grid-cols-3 gap-3">
                {enabledMetrics.map((metric) => (
                  <MetricPanel
                    key={metric.key}
                    metric={metric}
                    variants={detail.data!.variants}
                  />
                ))}
              </div>

              <VariantsTable
                variants={detail.data.variants}
                scoringConfig={localMetrics}
                onUpdateMetric={handleUpdateMetric}
                onEnableMetric={handleEnableMetric}
                onOpenInsights={onOpenInsights}
                showInactive={showInactive}
                onShowInactiveChange={setShowInactive}
              />
            </>
          )}
        </div>

        {/* Sidebar rail */}
        <SidebarRail active={tab} onChange={setTab} />

        {/* Sidebar panel */}
        <aside className="border-subtle w-[360px] shrink-0 overflow-y-auto border-l">
          {tab === 'info' && detail.data && (
            <InfoSidebar detail={detail.data} topVariantName={topVariant} />
          )}
          {tab === 'scoring' && localMetrics && (
            <ScoringFormulaSidebar
              metrics={localMetrics}
              onChange={setLocalMetrics}
              isSaving={updateScoring.isPending}
            />
          )}
        </aside>
      </div>
    </>
  );
}
