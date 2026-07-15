import { analytics } from '@/utils/segment';
import type { ScoreKind } from '@inngest/components/Experiments';

/**
 * Events tracked via Segment should always follow these patterns:
 * - Name: <Object> <Action, past tense>, using title case, with spaces, ex. "Query Created" "Dashboard Chart Added"
 * - Properties: Always snake_case
 */

type ExperimentEventName =
  | 'Experiment List Viewed'
  | 'Experiment Detail Viewed'
  | 'Experiment Opened In Insights'
  | 'Experiment Scoring Weight Updated'
  | 'Experiment Variant Filter Changed'
  | 'Experiment Time Range Changed'
  | 'Experiment Docs Link Opened'
  | 'Experiment Empty State Viewed'
  | 'Experiment Empty State Prompt Copied'
  | 'Experiment Empty State Example Copied'
  | 'Experiment Empty State Docs Link Opened';

type ExperimentEventProperties = Record<
  string,
  boolean | number | string | null | undefined
>;

type ExperimentIdentityArgs = {
  experimentName: string;
  functionSlug: string;
};

function trackExperimentsEvent(
  event: ExperimentEventName,
  properties: ExperimentEventProperties = {},
) {
  const compactProperties = Object.fromEntries(
    Object.entries({
      feature: 'experiments',
      ...properties,
    }).filter(([, value]) => value !== undefined),
  );

  analytics.track(event, compactProperties);
}

function getExperimentMetadata({
  experimentName,
  functionSlug,
}: ExperimentIdentityArgs): ExperimentEventProperties {
  return {
    experiment_name: experimentName,
    function_slug: functionSlug,
  };
}

export function trackExperimentsListViewed({
  experimentCount,
  functionCount,
}: {
  experimentCount: number;
  functionCount: number;
}) {
  trackExperimentsEvent('Experiment List Viewed', {
    experiment_count: experimentCount,
    function_count: functionCount,
    has_experiments: experimentCount > 0,
  });
}

type ExperimentDetailResult =
  | 'success'
  | 'no_runs'
  | 'no_variant_data'
  | 'error';

export function trackExperimentDetailViewed({
  durationMs,
  errorType,
  result,
  runCount,
  selectionStrategy,
  variantCount,
  ...identity
}: ExperimentIdentityArgs & {
  durationMs: number;
  errorType?: 'network' | 'graphql';
  result: ExperimentDetailResult;
  runCount?: number;
  selectionStrategy?: string;
  variantCount?: number;
}) {
  trackExperimentsEvent('Experiment Detail Viewed', {
    ...getExperimentMetadata(identity),
    duration_ms: durationMs,
    error_type: errorType,
    result,
    run_count: runCount,
    selection_strategy: selectionStrategy,
    variant_count: variantCount,
  });
}

export function trackExperimentOpenedInInsights({
  selectedVariantCount,
  variantCount,
  ...identity
}: ExperimentIdentityArgs & {
  selectedVariantCount: number;
  variantCount: number;
}) {
  trackExperimentsEvent('Experiment Opened In Insights', {
    ...getExperimentMetadata(identity),
    selected_variant_count: selectedVariantCount,
    variant_count: variantCount,
  });
}

export const SCORING_METRIC_CHANGED_FIELDS = [
  ['enabled', 'enabled'],
  ['points', 'points'],
  ['invert', 'invert'],
  ['minValue', 'min_value'],
  ['maxValue', 'max_value'],
  ['labelWorst', 'label_worst'],
  ['labelBest', 'label_best'],
  ['displayName', 'display_name'],
] as const;

export function trackExperimentScoringWeightUpdated({
  changedFields,
  enabled,
  metricKey,
  metricKind,
  points,
  ...identity
}: ExperimentIdentityArgs & {
  changedFields: string[];
  enabled: boolean;
  metricKey: string;
  metricKind: ScoreKind;
  points: number;
}) {
  trackExperimentsEvent('Experiment Scoring Weight Updated', {
    ...getExperimentMetadata(identity),
    changed_fields: changedFields.join(','),
    enabled,
    metric_key: metricKey,
    metric_kind: metricKind,
    points,
  });
}

type ExperimentVariantFilterType = 'variant_selection' | 'show_inactive';

export function trackExperimentVariantFilterChanged({
  availableVariantCount,
  filterType,
  resultedInEmpty,
  selectedVariantCount,
  ...identity
}: ExperimentIdentityArgs & {
  availableVariantCount: number;
  filterType: ExperimentVariantFilterType;
  resultedInEmpty: boolean;
  selectedVariantCount: number;
}) {
  trackExperimentsEvent('Experiment Variant Filter Changed', {
    ...getExperimentMetadata(identity),
    available_variant_count: availableVariantCount,
    filter_type: filterType,
    resulted_in_empty: resultedInEmpty,
    selected_variant_count: selectedVariantCount,
  });
}

export function trackExperimentTimeRangeChanged({
  hitMaxRange,
  rangeDays,
  ...identity
}: ExperimentIdentityArgs & {
  hitMaxRange: boolean;
  rangeDays: number;
}) {
  trackExperimentsEvent('Experiment Time Range Changed', {
    ...getExperimentMetadata(identity),
    hit_max_range: hitMaxRange,
    range_days: rangeDays,
  });
}

export function trackExperimentDocsLinkOpened() {
  trackExperimentsEvent('Experiment Docs Link Opened');
}

export function trackExperimentEmptyStateViewed() {
  trackExperimentsEvent('Experiment Empty State Viewed');
}

export function trackExperimentEmptyStatePromptCopied() {
  trackExperimentsEvent('Experiment Empty State Prompt Copied');
}

export function trackExperimentEmptyStateExampleCopied() {
  trackExperimentsEvent('Experiment Empty State Example Copied');
}

export function trackExperimentEmptyStateDocsLinkOpened() {
  trackExperimentsEvent('Experiment Empty State Docs Link Opened');
}
