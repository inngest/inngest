import { analytics } from './segment';
import type { ScoreKind } from '@inngest/components/Experiments';

/**
 * Single source of truth for every Segment `track()` call in the dashboard.
 * Never call `analytics.track()` (or the internal `track()` below) directly
 * anywhere else — every event goes through a named `trackXxx()` function in
 * this file.
 *
 * THE PATTERN:
 * - `AnalyticsFeature` is the definitive, alphabetical list of feature
 *   slugs ('experiments', 'scores', etc). Nothing else defines a feature
 *   slug.
 * - `AnalyticsEventName` is the definitive, alphabetical list of event
 *   name strings. Feature names never appear inside an event name —
 *   `feature` is what tags an event with which feature fired it, which is
 *   what lets the same event name be shared and aggregated across
 *   features.
 * - The private `track()` function below is the only place that calls
 *   `analytics.track()`. It merges `feature` into the properties bag and
 *   drops undefined values.
 * - Every event gets exactly one exported `trackXxx()` function, named for
 *   the event (e.g. "Detail Viewed" -> `trackDetailViewed`). One function
 *   per event — never one per feature.
 * - Each `trackXxx()` takes a single object argument, typed by a small
 *   `XxxArgs` type declared directly above it. `XxxArgs` always includes
 *   `feature: AnalyticsFeature` plus whatever else that event needs. Every
 *   function gets its own `XxxArgs` — don't share, intersect, or extract a
 *   common "identity" type across functions, even if fields repeat.
 * - Properties passed to `track()` MUST be snake_case (`experiment_name`,
 *   never `experimentName`) — that's the required shape for Segment
 *   properties. The function's own argument keys can stay camelCase;
 *   convert to snake_case only inside the `track()` call.
 *
 * ADDING A NEW EVENT:
 * 1. Search this file for an event that already covers the action (e.g.
 *    the "Empty State" ones, reused across several features) before
 *    adding a new one. Reuse it with the right `feature` instead of
 *    creating a feature-specific duplicate.
 * 2. Keep events distinct when the trigger is genuinely different, even if
 *    the name looks similar. "Empty State Docs Link Opened" (the
 *    onboarding empty state) and "Docs Link Opened" (a general info
 *    panel) are different actions firing from different places — don't
 *    merge them just because they share a word.
 * 3. Name it `<Object> <Action>`, title case, with spaces (e.g. "Query
 *    Created", "Dashboard Chart Added"). The action MUST be past tense
 *    ("Viewed", "Created", "Updated" — never "View", "Create", "Update").
 *    Never bake a feature name into it.
 * 4. Add the literal string to `AnalyticsEventName`, in alphabetical
 *    order. Add a slug to `AnalyticsFeature` (also alphabetical) only if
 *    you're instrumenting a brand-new feature.
 * 5. Add an `XxxArgs` type and a `trackXxx()` function directly below it,
 *    following the pattern above.
 */

// Alphabetical — this list is the only place a valid feature name is defined.
export type AnalyticsFeature =
  | 'experiments'
  | 'sandboxes'
  | 'scores'
  | 'sessions';

// Alphabetical — this list is the only place a valid event name is defined.
// Feature names never appear here; `feature` carries that instead.
type AnalyticsEventName =
  | 'Detail Viewed'
  | 'Docs Link Opened'
  | 'Empty State Docs Link Opened'
  | 'Empty State Example Copied'
  | 'Empty State Prompt Copied'
  | 'Empty State Viewed'
  | 'List Viewed'
  | 'Opened In Insights'
  | 'Scoring Weight Updated'
  | 'Waitlist Form Submitted'
  | 'Waitlist Joined';

type AnalyticsEventProperties = Record<
  string,
  boolean | number | string | null | undefined
>;

function track(
  event: AnalyticsEventName,
  feature: AnalyticsFeature,
  properties: AnalyticsEventProperties = {},
) {
  const compactProperties = Object.fromEntries(
    Object.entries({ feature, ...properties }).filter(
      ([, value]) => value !== undefined,
    ),
  );
  analytics.track(event, compactProperties);
}

type EmptyStateViewedArgs = { feature: AnalyticsFeature };

export function trackEmptyStateViewed({ feature }: EmptyStateViewedArgs) {
  track('Empty State Viewed', feature);
}

type EmptyStatePromptCopiedArgs = { feature: AnalyticsFeature };

export function trackEmptyStatePromptCopied({
  feature,
}: EmptyStatePromptCopiedArgs) {
  track('Empty State Prompt Copied', feature);
}

type EmptyStateExampleCopiedArgs = { feature: AnalyticsFeature };

export function trackEmptyStateExampleCopied({
  feature,
}: EmptyStateExampleCopiedArgs) {
  track('Empty State Example Copied', feature);
}

type EmptyStateDocsLinkOpenedArgs = { feature: AnalyticsFeature };

export function trackEmptyStateDocsLinkOpened({
  feature,
}: EmptyStateDocsLinkOpenedArgs) {
  track('Empty State Docs Link Opened', feature);
}

type ListViewedArgs = {
  feature: AnalyticsFeature;
  experimentCount: number;
  functionCount: number;
};

export function trackListViewed({
  feature,
  experimentCount,
  functionCount,
}: ListViewedArgs) {
  track('List Viewed', feature, {
    experiment_count: experimentCount,
    function_count: functionCount,
    has_experiments: experimentCount > 0,
  });
}

type DetailViewedArgs = { feature: AnalyticsFeature };

export function trackDetailViewed({ feature }: DetailViewedArgs) {
  track('Detail Viewed', feature);
}

type OpenedInInsightsArgs = {
  feature: AnalyticsFeature;
  selectedVariantCount: number;
  variantCount: number;
};

export function trackOpenedInInsights({
  feature,
  selectedVariantCount,
  variantCount,
}: OpenedInInsightsArgs) {
  track('Opened In Insights', feature, {
    selected_variant_count: selectedVariantCount,
    variant_count: variantCount,
  });
}

type ScoringWeightUpdatedArgs = {
  feature: AnalyticsFeature;
  changedFields: string[];
  enabled: boolean;
  metricKey: string;
  metricKind: ScoreKind;
  points: number;
};

export function trackScoringWeightUpdated({
  feature,
  changedFields,
  enabled,
  metricKey,
  metricKind,
  points,
}: ScoringWeightUpdatedArgs) {
  track('Scoring Weight Updated', feature, {
    changed_fields: changedFields.join(','),
    enabled,
    metric_key: metricKey,
    metric_kind: metricKind,
    points,
  });
}

type DocsLinkOpenedArgs = { feature: AnalyticsFeature };

export function trackDocsLinkOpened({ feature }: DocsLinkOpenedArgs) {
  track('Docs Link Opened', feature);
}

type WaitlistJoinedArgs = { feature: AnalyticsFeature };

export function trackWaitlistJoined({ feature }: WaitlistJoinedArgs) {
  track('Waitlist Joined', feature);
}

type WaitlistFormSubmittedArgs = {
  feature: AnalyticsFeature;
  canContact: boolean;
  message: string;
};

export function trackWaitlistFormSubmitted({
  feature,
  canContact,
  message,
}: WaitlistFormSubmittedArgs) {
  track('Waitlist Form Submitted', feature, {
    can_contact: canContact,
    message: message,
  });
}
