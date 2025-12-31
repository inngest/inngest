import { Resizable } from '@inngest/components/Resizable/Resizable';

import type { Tab } from '@/components/Insights/types';
import { InsightsDataTable } from '../InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '../InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '../InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '../InsightsSQLEditor/InsightsSQLEditorQueryButton';
import { InsightsSQLEditorQueryEditHistoryButton } from '../InsightsSQLEditor/InsightsSQLEditorQueryEditHistoryButton';
import { InsightsSQLEditorQueryTitle } from '../InsightsSQLEditor/InsightsSQLEditorQueryTitle';
import { InsightsSQLEditorResultsTitle } from '../InsightsSQLEditor/InsightsSQLEditorResultsTitle';
import { InsightsSQLEditorSaveQueryButton } from '../InsightsSQLEditor/InsightsSQLEditorSaveQueryButton';
import { InsightsSQLEditorSavedQueryActionsButton } from '../InsightsSQLEditor/InsightsSQLEditorSavedQueryActionsButton';
import { SQLEditorInstanceProvider } from '../InsightsSQLEditor/SQLEditorInstanceContext';
import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '../Section';
import { InsightsTabPanelTemplatesTab } from './InsightsTabPanelTemplatesTab/InsightsTabPanelTemplatesTab';

type InsightsTabPanelProps = {
  historyWindow?: number;
  isHomeTab?: boolean;
  isTemplatesTab?: boolean;
  tab: Tab;
};

export function InsightsTabPanel({
  historyWindow,
  isHomeTab,
  isTemplatesTab,
  tab,
}: InsightsTabPanelProps) {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  // TODO: Adjust home tab to AI panel
  if (isHomeTab) return <InsightsTabPanelTemplatesTab />;

  if (isTemplatesTab) return <InsightsTabPanelTemplatesTab />;

  return (
    <SQLEditorInstanceProvider>
      <div className="flex h-full min-h-0 flex-col">
        <Resizable
          defaultSplitPercentage={37.5}
          minSplitPercentage={20}
          maxSplitPercentage={80}
          first={
            <Section
              actions={
                <div className="relative flex gap-2">
                  <div className="absolute right-full">
                    <InsightsSQLEditorQueryEditHistoryButton tab={tab} />
                  </div>
                  <InsightsSQLEditorSavedQueryActionsButton tab={tab} />
                  <InsightsSQLEditorSaveQueryButton tab={tab} />
                  <InsightsSQLEditorQueryButton />
                </div>
              }
              className="h-full"
              title={<InsightsSQLEditorQueryTitle tab={tab} />}
            >
              <InsightsSQLEditor />
            </Section>
          }
          orientation="vertical"
          second={
            <Section
              actions={
                <>
                  <InsightsSQLEditorDownloadCSVButton />
                  {isRunning && (
                    <span className="text-muted mr-3 text-xs">
                      Running query...
                    </span>
                  )}
                </>
              }
              className="border-subtle h-full border-t"
              title={
                <InsightsSQLEditorResultsTitle historyWindow={historyWindow} />
              }
            >
              <InsightsDataTable />
            </Section>
          }
          splitKey="insights-tab-panel-split-vertical"
        />
      </div>
    </SQLEditorInstanceProvider>
  );
}
