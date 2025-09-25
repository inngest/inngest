'use client';

import { Link } from '@inngest/components/Link/Link';
import { Resizeable } from '@inngest/components/Resizeable/Resizeable';

import type { Tab } from '@/components/Insights/types';
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
};

export function InsightsTabPanel({
  historyWindow,
  isHomeTab,
  isTemplatesTab,
  tab,
}: InsightsTabPanelProps) {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  // TODO: Adjust home tab to AI panel.
  if (isHomeTab) return <InsightsTabPanelTemplatesTab />;

  if (isTemplatesTab) return <InsightsTabPanelTemplatesTab />;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <Resizeable
        defaultSplitPercentage={37.5}
        minSplitPercentage={20}
        maxSplitPercentage={80}
        first={
          <Section
            actions={
              <>
                <InsightsSQLEditorSaveQueryButton tab={tab} />
                <InsightsSQLEditorQueryButton />
              </>
            }
            className="h-full"
            title={<InsightsSQLEditorQueryTitle tab={tab} />}
          >
            <div className="h-full min-h-0 overflow-hidden">
              <InsightsSQLEditor />
            </div>
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
            <div className="h-full min-h-0 overflow-auto">
              <InsightsDataTable />
            </div>
          </Section>
        }
        splitKey="insights-tab-panel-split-vertical"
      />
    </div>
  );
}
