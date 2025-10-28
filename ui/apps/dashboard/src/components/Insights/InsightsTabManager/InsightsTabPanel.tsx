'use client';

import { Link } from '@inngest/components/Link/Link';
import { Resizable } from '@inngest/components/Resizable/Resizable';

import type { Tab } from '@/components/Insights/types';
import { MaximizeChatButton } from '../InsightsChat/header/MaximizeChatButton';
import { InsightsDataTable } from '../InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '../InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '../InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '../InsightsSQLEditor/InsightsSQLEditorQueryButton';
import { InsightsSQLEditorQueryTitle } from '../InsightsSQLEditor/InsightsSQLEditorQueryTitle';
import { InsightsSQLEditorResultsTitle } from '../InsightsSQLEditor/InsightsSQLEditorResultsTitle';
import { InsightsSQLEditorSaveQueryButton } from '../InsightsSQLEditor/InsightsSQLEditorSaveQueryButton';
import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '../Section';
import { InsightsTabPanelTemplatesTab } from './InsightsTabPanelTemplatesTab/InsightsTabPanelTemplatesTab';
import { EXTERNAL_FEEDBACK_LINK } from './constants';

type InsightsTabPanelProps = {
  historyWindow?: number;
  isHomeTab?: boolean;
  isTemplatesTab?: boolean;
  tab: Tab;
  isChatPanelVisible: boolean;
  isInsightsAgentEnabled: boolean;
  onToggleChatPanelVisibility: () => void;
};

export function InsightsTabPanel({
  historyWindow,
  isHomeTab,
  isTemplatesTab,
  tab,
  isChatPanelVisible,
  isInsightsAgentEnabled,
  onToggleChatPanelVisibility,
}: InsightsTabPanelProps) {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  // TODO: Adjust home tab to AI panel
  if (isHomeTab) return <InsightsTabPanelTemplatesTab />;

  if (isTemplatesTab) return <InsightsTabPanelTemplatesTab />;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <Resizable
        defaultSplitPercentage={37.5}
        minSplitPercentage={20}
        maxSplitPercentage={80}
        first={
          <Section
            actions={
              <>
                <InsightsSQLEditorSaveQueryButton tab={tab} />
                <InsightsSQLEditorQueryButton />
                {isInsightsAgentEnabled && !isChatPanelVisible && (
                  <>
                    <VerticalDivider />
                    <MaximizeChatButton onClick={onToggleChatPanelVisibility} />
                  </>
                )}
              </>
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
                <InsightsSQLEditorDownloadCSVButton temporarilyHide />
                {isRunning && <span className="text-muted mr-3 text-xs">Running query...</span>}
                <Link href={EXTERNAL_FEEDBACK_LINK} rel="noopener noreferrer" target="_blank">
                  Send us feedback
                </Link>
              </>
            }
            className="border-subtle h-full border-t"
            title={<InsightsSQLEditorResultsTitle historyWindow={historyWindow} />}
          >
            <InsightsDataTable />
          </Section>
        }
        splitKey="insights-tab-panel-split-vertical"
      />
    </div>
  );
}

function VerticalDivider() {
  return <div className="border-subtle mx-1 h-[28px] border-l" />;
}
