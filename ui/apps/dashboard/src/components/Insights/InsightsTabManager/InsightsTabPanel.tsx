'use client';

import { Link } from '@inngest/components/Link/Link';

import { InsightsDataTable } from '@/components/Insights/InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryButton';
import { InsightsSQLEditorQueryTitle } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryTitle';
import { InsightsSQLEditorResultsTitle } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorResultsTitle';
import { InsightsSQLEditorSaveQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorSaveQueryButton';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '@/components/Insights/Section';
import type { Tab } from '@/components/Insights/types';
import { MaximizeChatButton } from '../InsightsChat/header/MaximizeChatButton';
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

  // TODO: Adjust home tab to AI panel.
  if (isHomeTab) return <InsightsTabPanelTemplatesTab />;

  if (isTemplatesTab) return <InsightsTabPanelTemplatesTab />;

  return (
    <div className="grid h-full w-full grid-rows-[3fr_5fr] gap-0 overflow-hidden">
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
        className="min-h-[255px]"
        title={<InsightsSQLEditorQueryTitle tab={tab} />}
      >
        <InsightsSQLEditor />
      </Section>
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
        className="border-subtle border-t"
        title={<InsightsSQLEditorResultsTitle historyWindow={historyWindow} />}
      >
        <InsightsDataTable />
      </Section>
    </div>
  );
}

function VerticalDivider() {
  return <div className="border-subtle mx-1 h-[28px] border-l" />;
}
