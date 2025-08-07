'use client';

import { InsightsDataTable } from '@/components/Insights/InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryButton';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '@/components/Insights/Section';

export function InsightsTabPanel() {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  return (
    <>
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
    </>
  );
}
