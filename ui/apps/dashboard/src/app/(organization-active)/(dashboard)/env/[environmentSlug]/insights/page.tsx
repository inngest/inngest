'use client';

import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiPlayFill } from '@remixicon/react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { InsightsDataTable } from '@/components/Insights/InsightsDataTable';
import { InsightsSQLEditor } from '@/components/Insights/InsightsSQLEditor/InsightsSQLEditor';
import {
  InsightsStateMachineContextProvider,
  useInsightsStateMachineContext,
} from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Section } from '@/components/Insights/Section';
import { useInsightsQuery } from '@/components/Insights/hooks/useInsightsQuery';

function InsightsPageContent() {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');
  const { onChange, query, runQuery, status } = useInsightsStateMachineContext();
  const isRunning = status === 'loading';

  if (!isInsightsEnabled) return null;

  return (
    <>
      <Header breadcrumb={[{ text: 'Insights' }]} />
      <main className="grid h-full w-full flex-1 grid-rows-[3fr_5fr] gap-0">
        <Section
          actions={
            <Button
              className="w-[110px]"
              disabled={query.trim() === '' || isRunning}
              icon={<RiPlayFill className="h-4 w-4" />}
              iconSide="left"
              label={isRunning ? undefined : 'Run query'}
              loading={isRunning}
              onClick={runQuery}
              size="medium"
            />
          }
          className="min-h-[255px]"
          title="Query Editor"
        >
          <InsightsSQLEditor content={query} onChange={onChange} />
        </Section>
        <Section
          actions={
            <>
              {isRunning && <span className="text-muted mr-3 text-xs">Running query...</span>}
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <Button
                      appearance="ghost"
                      disabled
                      kind="secondary"
                      label="Download as .csv"
                      size="medium"
                    />
                  </span>
                </TooltipTrigger>
                <TooltipContent>Coming soon</TooltipContent>
              </Tooltip>
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
  const { content, isRunning, onChange, runQuery } = useInsightsQuery();

  if (!isInsightsEnabled) return null;

  return (
    <InsightsStateMachineContextProvider>
      <InsightsPageContent />
    </InsightsStateMachineContextProvider>
  );
}
