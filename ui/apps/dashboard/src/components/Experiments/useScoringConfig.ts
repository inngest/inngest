import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';

import {
  useExperimentScoringConfig,
  useUpdateExperimentScoringConfig,
} from './useExperiments';

const DEBOUNCE_MS = 600;

/**
 * Manages local scoring-config state with debounced persistence.
 *
 * Every call to `setMetrics`, `updateMetric`, or `enableMetric` applies
 * optimistically and schedules a save after DEBOUNCE_MS of inactivity.
 * The save is skipped when the local state matches the last-known server state.
 */
export function useScoringConfig(experimentName: string) {
  const scoring = useExperimentScoringConfig(experimentName);
  const updateScoring = useUpdateExperimentScoringConfig(experimentName);

  const [localMetrics, setLocalMetrics] = useState<
    ExperimentScoringMetric[] | null
  >(null);

  // --- server → local sync (initial load & refetch) ---
  useEffect(() => {
    if (scoring.data) {
      setLocalMetrics(scoring.data.metrics);
    }
  }, [scoring.data]);

  // --- debounced save ---
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const serverMetricsRef = useRef<ExperimentScoringMetric[] | null>(null);
  const mutateRef = useRef(updateScoring.mutate);
  mutateRef.current = updateScoring.mutate;

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
        serverMetricsRef.current = localMetrics;
        mutateRef.current(localMetrics);
      }
    }, DEBOUNCE_MS);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [localMetrics]);

  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

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
    setMetrics: setLocalMetrics,
    updateMetric,
    enableMetric,
    pointsLeft,
    isSaving: updateScoring.isPending,
    isPending: scoring.isPending,
    error: scoring.error,
    refetch: scoring.refetch,
  };
}
