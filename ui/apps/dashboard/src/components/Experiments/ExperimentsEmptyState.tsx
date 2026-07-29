import { FeatureEmptyState } from '@/components/FeatureEmptyState/FeatureEmptyState';
import {
  DOCS_URL,
  INTRO_DESCRIPTION,
  PROMPT,
  USE_CASES,
  VARIANT_TABS,
} from './experimentsEmptyStateContent';

type ExperimentsEmptyStateProps = {
  onDocsLinkClick?: () => void;
};

export function ExperimentsEmptyState({
  onDocsLinkClick,
}: ExperimentsEmptyStateProps) {
  return (
    <FeatureEmptyState
      feature="experiments"
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
        description:
          'Copy this prompt to learn about this feature and implement experiments',
        content: PROMPT,
      }}
      example={{
        tabs: VARIANT_TABS,
        height: 280,
      }}
    />
  );
}
