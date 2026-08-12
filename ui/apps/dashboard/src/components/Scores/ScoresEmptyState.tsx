import { ScoresEmptyStateView } from '@inngest/components/FeatureEmptyState/ScoresEmptyStateView';

import {
  trackEmptyStateDocsLinkOpened,
  trackEmptyStateExampleCopied,
  trackEmptyStatePromptCopied,
  trackEmptyStateViewed,
} from '@/utils/analyticsEvents';

export function ScoresEmptyState() {
  return (
    <ScoresEmptyStateView
      onView={() => trackEmptyStateViewed({ feature: 'scores' })}
      onPromptCopy={() => trackEmptyStatePromptCopied({ feature: 'scores' })}
      onExampleCopy={() => trackEmptyStateExampleCopied({ feature: 'scores' })}
      onDocsLinkClick={() =>
        trackEmptyStateDocsLinkOpened({ feature: 'scores' })
      }
    />
  );
}
