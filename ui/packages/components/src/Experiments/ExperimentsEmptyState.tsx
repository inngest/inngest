import { FeatureEmptyState } from '../FeatureEmptyState';
import {
  DOCS_URL,
  INTRO_DESCRIPTION,
  PROMPT,
  USE_CASES,
  VARIANT_TABS,
} from './experimentsEmptyStateContent';

type ExperimentsEmptyStateProps = {
  onViewed?: () => void;
  onDocsLinkClick?: () => void;
  onPromptCopy?: () => void;
  onExampleCopy?: () => void;
};

export function ExperimentsEmptyState({
  onViewed,
  onDocsLinkClick,
  onPromptCopy,
  onExampleCopy,
}: ExperimentsEmptyStateProps) {
  return (
    <FeatureEmptyState
      title="Experiments"
      description={INTRO_DESCRIPTION}
      docsUrl={DOCS_URL}
      onDocsLinkClick={onDocsLinkClick}
      valueProps={USE_CASES.map(({ Icon, title, description }) => ({
        icon: Icon,
        title,
        description,
      }))}
      prompt={{
        description: 'Copy this prompt to learn about this feature and implement experiments',
        content: PROMPT,
        onCopy: onPromptCopy,
      }}
      example={{
        tabs: VARIANT_TABS,
        height: 280,
        onCopy: onExampleCopy,
      }}
      onViewed={onViewed}
    />
  );
}
