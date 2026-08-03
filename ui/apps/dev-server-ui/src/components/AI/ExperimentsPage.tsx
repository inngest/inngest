import { ExperimentsEmptyStateView } from '@inngest/components/FeatureEmptyState/ExperimentsEmptyStateView';
import { Header } from '@inngest/components/Header/Header';

import { useFakeDoorTracking } from './useFakeDoorTracking';

export function ExperimentsPage() {
  const track = useFakeDoorTracking('experiments');

  return (
    <div className="flex h-full flex-col overflow-y-scroll">
      <Header breadcrumb={[{ text: 'AI' }, { text: 'Experiments' }]} />
      <ExperimentsEmptyStateView
        onView={() => track('viewed')}
        onPromptCopy={() => track('prompt_copied')}
        onExampleCopy={() => track('example_copied')}
        onDocsLinkClick={() => track('docs_clicked')}
      />
    </div>
  );
}
