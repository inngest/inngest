import { ScoresEmptyStateView } from '@inngest/components/FeatureEmptyState/ScoresEmptyStateView';
import { Header } from '@inngest/components/Header/Header';

import { useInfoQuery } from '@/store/devApi';
import { FakeDoorCard } from './FakeDoorCard';
import { useFakeDoorTracking } from './useFakeDoorTracking';

export function ScoresPage() {
  const track = useFakeDoorTracking('scores');
  const { data: info } = useInfoQuery();

  return (
    <div className="flex h-full flex-col overflow-y-scroll">
      <Header breadcrumb={[{ text: 'AI' }, { text: 'Scores' }]} />
      <ScoresEmptyStateView
        onView={() => track('viewed')}
        onPromptCopy={() => track('prompt_copied')}
        onExampleCopy={() => track('example_copied')}
        onDocsLinkClick={() => track('docs_clicked')}
        contentOverride={
          info?.hasSeenScores ? <FakeDoorCard feature="scores" /> : undefined
        }
      />
    </div>
  );
}
