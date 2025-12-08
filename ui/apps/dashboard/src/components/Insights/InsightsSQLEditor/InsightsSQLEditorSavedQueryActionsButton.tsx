import { useMemo } from 'react';
import { Button } from '@inngest/components/Button/Button';
import { RiMore2Fill } from '@remixicon/react';

import { QueryActionsMenu } from '../QueryActionsMenu';
import { useStoredQueries } from '../QueryHelperPanel/StoredQueriesContext';
import type { Tab } from '../types';

type InsightsSQLEditorSavedQueryActionsButtonProps = { tab: Tab };

export function InsightsSQLEditorSavedQueryActionsButton({
  tab,
}: InsightsSQLEditorSavedQueryActionsButtonProps) {
  const { deleteQuery, queries } = useStoredQueries();

  const savedQuery = useMemo(() => {
    if (tab.savedQueryId === undefined) return undefined;

    return queries.data?.find((q) => q.id === tab.savedQueryId);
  }, [queries.data, tab.savedQueryId]);

  return (
    <QueryActionsMenu
      onSelectDelete={(q) => deleteQuery(q.id)}
      query={savedQuery}
      trigger={
        <Button
          appearance="outlined"
          icon={<RiMore2Fill />}
          kind="secondary"
          size="medium"
        />
      }
    />
  );
}
