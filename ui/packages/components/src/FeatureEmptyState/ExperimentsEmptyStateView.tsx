import type { ReactNode } from 'react';

import { FeatureEmptyState } from './FeatureEmptyState';
import {
  DOCS_URL,
  INTRO_DESCRIPTION,
  PROMPT,
  USE_CASES,
  VARIANT_TABS,
} from './experimentsEmptyStateContent';

type ExperimentsEmptyStateViewProps = {
  onView?: () => void;
  onPromptCopy?: () => void;
  onExampleCopy?: () => void;
  onDocsLinkClick?: () => void;
  banner?: ReactNode;
};

// Shared onboarding empty state for Experiments, rendered identically by
// the cloud dashboard and the dev server. Tracking is injected per app.
export function ExperimentsEmptyStateView(props: ExperimentsEmptyStateViewProps) {
  return (
    <FeatureEmptyState
      {...props}
      title="Experiments"
      description={INTRO_DESCRIPTION}
      docsUrl={DOCS_URL}
      valueProps={USE_CASES.map(({ Icon, title, description }) => ({
        icon: Icon,
        title,
        description,
      }))}
      prompt={{
        description: 'Copy this prompt to learn about this feature and implement experiments',
        content: PROMPT,
      }}
      example={{
        tabs: VARIANT_TABS,
        height: 280,
      }}
    />
  );
}
