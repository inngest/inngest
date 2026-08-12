import { ExperimentsEmptyStateView } from '@inngest/components/FeatureEmptyState/ExperimentsEmptyStateView';

import {
  trackEmptyStateExampleCopied,
  trackEmptyStatePromptCopied,
  trackEmptyStateViewed,
} from '@/utils/analyticsEvents';

type ExperimentsEmptyStateProps = {
  onDocsLinkClick?: () => void;
};

export function ExperimentsEmptyState({
  onDocsLinkClick,
}: ExperimentsEmptyStateProps) {
  return (
    <ExperimentsEmptyStateView
      onView={() => trackEmptyStateViewed({ feature: 'experiments' })}
      onPromptCopy={() =>
        trackEmptyStatePromptCopied({ feature: 'experiments' })
      }
      onExampleCopy={() =>
        trackEmptyStateExampleCopied({ feature: 'experiments' })
      }
      onDocsLinkClick={onDocsLinkClick}
    />
  );
}
