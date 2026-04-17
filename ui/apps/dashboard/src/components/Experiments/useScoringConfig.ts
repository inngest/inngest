import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import useDebounce from '@inngest/components/hooks/useDebounce';

import {
  useExperimentScoringConfig,
  useUpdateExperimentScoringConfig,
} from './useExperiments';

const DEBOUNCE_MS = 600;

/**
 * Manages local scoring-config state with debounced persistence.
 *
 * Every mutator (`updateMetric`, `enableMetric`) applies optimistically and
 * schedules a save after DEBOUNCE_MS of inactivity. The save is skipped when
 * the local state matches the last-known server state.
 */
export function useScoringConfig(experimentName: string) {
  const scoring = useExperimentScoringConfig(experimentName);
  const updateScoring = useUpdateExperimentScoringConfig(experimentName);

  const [localMetrics, setLocalMetrics] = useState<
    ExperimentScoringMetric[] | null
  >(null);

  useEffect(() => {
    if (scoring.data) {
      setLocalMetrics(scoring.data.metrics);
    }
  }, [scoring.data]);

  const localMetricsRef = useRef(localMetrics);
  localMetricsRef.current = localMetrics;
  const serverMetricsRef = useRef<ExperimentScoringMetric[] | null>(null);
  const mutateRef = useRef(updateScoring.mutate);
  mutateRef.current = updateScoring.mutate;

  useEffect(() => {
    serverMetricsRef.current = scoring.data?.metrics ?? null;
  }, [scoring.data]);

  const debouncedSave = useDebounce(() => {
    const current = localMetricsRef.current;
    const serverMetrics = serverMetricsRef.current;
    if (!current || !serverMetrics) return;
    if (JSON.stringify(current) === JSON.stringify(serverMetrics)) return;
    serverMetricsRef.current = current;
    mutateRef.current(current);
  }, DEBOUNCE_MS);

  useEffect(() => {
    if (!localMetrics) return;
    debouncedSave();
  }, [localMetrics, debouncedSave]);

  // --- convenience mutators ---
  const updateMetric = useCallback(
    (key: string, patch: Partial<ExperimentScoringMetric>) => {
      setLocalMetrics((prev) =>
        prev ? prev.map((m) => (m.key === key ? { ...m, ...patch } : m)) : prev,
      );
    },
    [],
  );

  const enableMetric = useCallback((key: string) => {
    setLocalMetrics((prev) =>
      prev
        ? prev.map((m) => (m.key === key ? { ...m, enabled: true } : m))
        : prev,
    );
  }, []);

  const pointsLeft = useMemo(() => {
    if (!localMetrics) return 100;
    const allocated = localMetrics
      .filter((m) => m.enabled)
      .reduce((sum, m) => sum + m.points, 0);
    return 100 - allocated;
  }, [localMetrics]);

  return {
    metrics: localMetrics,
    updateMetric,
    enableMetric,
    pointsLeft,
    isSaving: updateScoring.isPending,
    isPending: scoring.isPending,
    error: scoring.error,
    refetch: scoring.refetch,
  };
}
