'use client';

import { Pill } from '@inngest/components/Pill';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

import { InsightsDataTable } from '@/components/Insights/InsightsDataTable/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';
import { InsightsSQLEditorDownloadCSVButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorDownloadCSVButton';
import { InsightsSQLEditorQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryButton';
import { InsightsSQLEditorQueryTitle } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorQueryTitle';
import { InsightsSQLEditorSaveQueryButton } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditorSaveQueryButton';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '@/components/Insights/Section';
import type { Query } from '@/components/Insights/types';
import { InsightsTabPanelTemplatesTab } from './InsightsTabPanelTemplatesTab/InsightsTabPanelTemplatesTab';

type InsightsTabPanelProps = {
  isHomeTab?: boolean;
  isTemplatesTab?: boolean;
  tab: Query;
};

export function InsightsTabPanel({ isHomeTab, isTemplatesTab, tab }: InsightsTabPanelProps) {
  const { status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  // TODO: Adjust home tab to AI panel.
  if (isHomeTab) return <InsightsTabPanelTemplatesTab />;

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
        title={
          <div className="mr-1 flex items-center gap-2">
            <span>Results</span>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Pill appearance="outlined" kind="info">
                    LIMIT 1000
                  </Pill>
                </span>
              </TooltipTrigger>
              <TooltipContent className="p-2 text-xs" side="right" sideOffset={3}>
                Results are currently limited to at most 1000 rows.
              </TooltipContent>
            </Tooltip>
          </div>
        }
      >
        <InsightsDataTable />
      </Section>
    </>
  );
}
