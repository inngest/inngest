import type { ReactNode } from 'react';
import { InlineCode } from '@inngest/components/Code';

import { FeatureEmptyState } from './FeatureEmptyState';
import { DESCRIPTION, DOCS_URL, EXAMPLE, PROMPT, VALUE_PROPS } from './scoresEmptyStateContent';

type ScoresEmptyStateViewProps = {
  onView?: () => void;
  onPromptCopy?: () => void;
  onExampleCopy?: () => void;
  onDocsLinkClick?: () => void;
  banner?: ReactNode;
};

// Shared onboarding empty state for Scores, rendered identically by the
// cloud dashboard and the dev server. Tracking is injected per app.
export function ScoresEmptyStateView(props: ScoresEmptyStateViewProps) {
  return (
    <FeatureEmptyState
      {...props}
      title="Scores"
      description={DESCRIPTION}
      docsUrl={DOCS_URL}
      valueProps={VALUE_PROPS}
      prompt={{
        description: 'Copy this prompt to learn about this feature and implement scores',
        content: PROMPT,
      }}
      example={{
        description: (
          <>
            add <InlineCode>inngest.score()</InlineCode> to any function
          </>
        ),
        tabs: [
          {
            title: 'Code',
            content: EXAMPLE,
            readOnly: true,
            language: 'typescript',
          },
        ],
      }}
    />
  );
}
