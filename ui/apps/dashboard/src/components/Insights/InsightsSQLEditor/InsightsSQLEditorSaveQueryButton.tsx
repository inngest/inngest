'use client';

import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkFill, RiBookmarkLine } from '@remixicon/react';
import { toast } from 'sonner';
import { ulid } from 'ulid';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { TabConfig } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { hasSavedQueryWithUnsavedChanges } from '../InsightsTabManager/InsightsTabManager';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: TabConfig;
};

export function InsightsSQLEditorSaveQueryButton({ tab }: InsightsSQLEditorSaveQueryButtonProps) {
  const { queries, saveQuery } = useStoredQueries();
  const { query, queryName } = useInsightsStateMachineContext();

  const isSavedQuery = tab.savedQueryId !== undefined;
  const disabled =
    queryName === '' || (isSavedQuery && !hasSavedQueryWithUnsavedChanges(tab, queries));
  const Icon = tab.savedQueryId ? RiBookmarkFill : RiBookmarkLine;

  return (
    <Button
      appearance="outlined"
      disabled={disabled}
      icon={<Icon className={cn(disabled ? 'text-disabled' : 'text-muted')} size={16} />}
      iconSide="left"
      kind="secondary"
      label="Save"
      onClick={() => {
        const isUpdate = tab.savedQueryId !== undefined;

        saveQuery(
          {
            id: tab.savedQueryId ?? ulid(),
            name: queryName,
            query,
            saved: true,
          },
          tab.id
        );

        toast.success(`Successfully ${isUpdate ? 'updated' : 'saved'} query`);
      }}
      size="medium"
    />
  );
}
