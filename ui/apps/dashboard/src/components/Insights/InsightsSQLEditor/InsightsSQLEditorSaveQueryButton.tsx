'use client';

import { Button } from '@inngest/components/Button/Button';
import { RiBookmarkLine } from '@remixicon/react';
import { ulid } from 'ulid';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { TabConfig } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: TabConfig;
};

export function InsightsSQLEditorSaveQueryButton({ tab }: InsightsSQLEditorSaveQueryButtonProps) {
  const { saveQuery } = useStoredQueries();
  const { query, queryName } = useInsightsStateMachineContext();

  return (
    <Button
      appearance="outlined"
      icon={<RiBookmarkLine className="h-4 w-4" />}
      kind="secondary"
      onClick={() => {
        saveQuery({
          id: tab.savedQueryId ?? ulid(),
          name: queryName,
          query,
          saved: true,
        });
      }}
      size="medium"
      title="Save query"
    />
  );
}
