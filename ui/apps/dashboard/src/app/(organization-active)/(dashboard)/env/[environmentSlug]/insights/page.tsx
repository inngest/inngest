'use client';

import { Header } from '@inngest/components/Header/Header';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { InsightsDataTable } from '@/components/Insights/InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryButton';
import {
  InsightsStateMachineContextProvider,
  useInsightsStateMachineContext,
} from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '@/components/Insights/Section';

function InsightsContent() {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  return (
    <>
      <Header breadcrumb={[{ text: 'Insights' }]} />
      <main className="grid h-full w-full flex-1 grid-rows-[3fr_5fr] gap-0 overflow-hidden">
        <Section
          actions={<InsightsSQLEditorQueryButton />}
          className="min-h-[255px]"
          title="Query Editor"
        >
          <InsightsSQLEditor />
        </Section>
        <Section
          actions={
            <>
              {isRunning && <span className="text-muted mr-3 text-xs">Running query...</span>}
              <InsightsSQLEditorDownloadCSVButton />
            </>
          }
          title="Results"
        >
          <InsightsDataTable />
        </Section>
      </main>
    </>
  );
}

export default function InsightsPage() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');

  if (!isInsightsEnabled) return null;

  return (
    <InsightsStateMachineContextProvider>
      <InsightsContent />
    </InsightsStateMachineContextProvider>
  );
}
