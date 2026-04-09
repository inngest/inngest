/**
 * Types for the Experiments UI components.
 */

export type ExperimentListItem = {
  experimentName: string;
  functionId: string;
  selectionStrategy: string;
  totalRuns: number;
  variantCount: number;
  firstSeen: Date;
  lastSeen: Date;
};

export type ExperimentVariantMetrics = {
  variantName: string;
  runCount: number;
  avgTokens: number;
  avgCost: number;
  avgAccuracy: number;
  avgSafety: number;
  avgDuration: number;
  failureRate: number;
};
