'use client';

import { InsightsDataTable } from '@/components/Insights/InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryButton';
import { InsightsSQLEditorQueryTitle } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryTitle';
import { InsightsSQLEditorSaveQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorSaveQueryButton';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '@/components/Insights/Section';
import type { TabConfig } from './InsightsTabManager';
import { InsightsTabPanelTemplatesTab } from './InsightsTabPanelTemplatesTab/InsightsTabPanelTemplatesTab';

type InsightsTabPanelProps = {
  isTemplatesTab?: boolean;
  tab: TabConfig;
};

export function InsightsTabPanel({ isTemplatesTab, tab }: InsightsTabPanelProps) {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  if (isTemplatesTab) return <InsightsTabPanelTemplatesTab />;

  return (
    <>
      <Section
        actions={
          <>
            <InsightsSQLEditorSaveQueryButton tab={tab} />
            <InsightsSQLEditorQueryButton />
          </>
        }
        className="min-h-[255px]"
        title={<InsightsSQLEditorQueryTitle tab={tab} />}
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
