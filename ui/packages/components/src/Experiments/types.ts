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

export type VariantMetric = {
  key: string;
  avg: number;
  min: number;
  max: number;
};

export type ExperimentVariantMetrics = {
  variantName: string;
  runCount: number;
  metrics: VariantMetric[];
};

export type VariantWeight = {
  variantName: string;
  weight: number;
};

export type ExperimentDetail = {
  name: string;
  variants: ExperimentVariantMetrics[];
  variantWeights: VariantWeight[];
  firstSeen: Date;
  lastSeen: Date;
  selectionStrategy: string;
};

export type ExperimentScoringMetric = {
  key: string;
  enabled: boolean;
  points: number;
  minValue: number;
  maxValue: number;
  invert: boolean;
  labelBest: string;
  labelWorst: string;
  displayName: string;
};

export type ExperimentScoringConfig = {
  experimentName: string;
  metrics: ExperimentScoringMetric[];
  updatedAt: Date;
};

export type TimeRangePreset = '24h' | '7d' | '30d';
