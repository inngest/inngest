import { ScoresEmptyStateView } from '@inngest/components/FeatureEmptyState/ScoresEmptyStateView';
import { Header } from '@inngest/components/Header/Header';

import { useFakeDoorTracking } from './useFakeDoorTracking';

export function ScoresPage() {
  const track = useFakeDoorTracking('scores');

  return (
    <div className="flex h-full flex-col overflow-y-scroll">
      <Header breadcrumb={[{ text: 'AI' }, { text: 'Scores' }]} />
      <ScoresEmptyStateView
        onView={() => track('viewed')}
        onPromptCopy={() => track('prompt_copied')}
        onExampleCopy={() => track('example_copied')}
        onDocsLinkClick={() => track('docs_clicked')}
      />
    </div>
  );
}
