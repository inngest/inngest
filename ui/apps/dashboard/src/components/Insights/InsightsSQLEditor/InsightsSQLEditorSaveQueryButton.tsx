'use client';

import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkFill, RiBookmarkLine } from '@remixicon/react';
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

  const disabled = query === '' || queryName === '';
  const Icon = tab.savedQueryId ? RiBookmarkFill : RiBookmarkLine;

  return (
    <Button
      appearance="outlined"
      disabled={disabled}
      icon={<Icon className={cn('h-4 w-4', disabled ? 'text-disabled' : 'text-muted')} size={16} />}
      kind="secondary"
      onClick={() => {
        const queryId = tab.savedQueryId ?? ulid();
        saveQuery(
          {
            id: queryId,
            name: queryName,
            query,
            saved: true,
          },
          tab.id
        );
      }}
      size="medium"
      title="Save query"
    />
  );
}
