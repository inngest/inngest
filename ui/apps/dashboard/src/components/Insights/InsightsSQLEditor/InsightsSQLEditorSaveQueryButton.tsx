'use client';

import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkFill, RiBookmarkLine } from '@remixicon/react';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { hasDiffWithSavedQuery } from '../InsightsTabManager/InsightsTabManager';
import { isSavedQuery } from '../queries';
import type { Query } from '../types';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: Query;
};

export function InsightsSQLEditorSaveQueryButton({ tab }: InsightsSQLEditorSaveQueryButtonProps) {
  const { queries, saveQuery } = useStoredQueries();
  const { query, queryName: name } = useInsightsStateMachineContext();

  const isSaved = isSavedQuery(tab);
  const disabled = name === '' || (isSaved && !hasDiffWithSavedQuery(queries, tab));
  const Icon = isSaved ? RiBookmarkFill : RiBookmarkLine;

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
        saveQuery({ ...tab, name, query });
      }}
      size="medium"
    />
  );
}
