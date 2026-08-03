import {
  FeatureEmptyState as SharedFeatureEmptyState,
  type FeatureEmptyStateProps as SharedFeatureEmptyStateProps,
  type FeatureEmptyStateValueProp,
} from '@inngest/components/FeatureEmptyState/FeatureEmptyState';

import {
  trackEmptyStateExampleCopied,
  trackEmptyStatePromptCopied,
  trackEmptyStateViewed,
  type AnalyticsFeature,
} from '@/utils/analyticsEvents';

export type { FeatureEmptyStateValueProp };

export type FeatureEmptyStateProps = Omit<
  SharedFeatureEmptyStateProps,
  'onView' | 'onPromptCopy' | 'onExampleCopy'
> & {
  feature: AnalyticsFeature;
};

// Dashboard wrapper around the shared component: maps the feature slug to
// the Segment tracking calls so consumers keep the pre-extraction API and
// event surface.
export function FeatureEmptyState({
  feature,
  ...props
}: FeatureEmptyStateProps) {
  return (
    <SharedFeatureEmptyState
      {...props}
      onView={() => trackEmptyStateViewed({ feature })}
      onPromptCopy={() => trackEmptyStatePromptCopied({ feature })}
      onExampleCopy={() => trackEmptyStateExampleCopied({ feature })}
    />
  );
}
