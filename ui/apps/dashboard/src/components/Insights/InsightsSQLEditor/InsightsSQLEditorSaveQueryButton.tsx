'use client';

import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkFill, RiBookmarkLine } from '@remixicon/react';
import { toast } from 'sonner';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { hasDiffWithSavedQuery } from '../InsightsTabManager/InsightsTabManager';
import { useTabManagerActions } from '../InsightsTabManager/TabManagerContext';
import type { Query } from '../types';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: Query;
};

export function InsightsSQLEditorSaveQueryButton({ tab }: InsightsSQLEditorSaveQueryButtonProps) {
  const { tabManagerActions } = useTabManagerActions();
  const { queries, saveQuery } = useStoredQueries();
  const { query, queryName: name } = useInsightsStateMachineContext();

  const disabled = name === '' || (tab.saved && !hasDiffWithSavedQuery(queries, tab));
  const Icon = tab.saved ? RiBookmarkFill : RiBookmarkLine;

  return (
    <Button
      appearance="outlined"
      className="font-medium"
      disabled={disabled}
      icon={<Icon className={cn(disabled ? 'text-disabled' : 'text-muted')} size={16} />}
      iconSide="left"
      kind="secondary"
      label="Save query"
      onClick={() => {
        const newTab: Query = { id: tab.id, name, query, saved: true };
        saveQuery(newTab, () => {
          tabManagerActions.updateTab(tab.id, newTab);
          toast.success(`Successfully ${tab.saved ? 'updated' : 'saved'} query`);
        });
      }}
      size="medium"
    />
  );
}
