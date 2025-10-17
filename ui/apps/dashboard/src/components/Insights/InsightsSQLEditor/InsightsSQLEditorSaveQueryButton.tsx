'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkFill, RiBookmarkLine } from '@remixicon/react';

import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { getIsSavedQuery } from '../InsightsTabManager/InsightsTabManager';
import type { Tab } from '../types';
import { useDocumentShortcuts } from './actions/handleShortcuts';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: Tab;
};

export function InsightsSQLEditorSaveQueryButton({ tab }: InsightsSQLEditorSaveQueryButtonProps) {
  const [isSaving, setIsSaving] = useState(false);
  const { saveQuery } = useStoredQueries();

  const isSaved = getIsSavedQuery(tab);
  const disabled = tab.name === '' || tab.query === '' || isSaving;
  const Icon = isSaved ? RiBookmarkFill : RiBookmarkLine;

  useDocumentShortcuts({
    combo: { alt: true, code: 'KeyS', metaOrCtrl: true },
    handler: () => {
      if (disabled) return;
      setIsSaving(true);
      saveQuery(tab).finally(() => {
        setIsSaving(false);
      });
    },
  });

  return (
    <Button
      appearance="outlined"
      className="font-medium"
      disabled={disabled}
      icon={<Icon className={cn(disabled ? 'text-disabled' : 'text-muted')} size={16} />}
      iconSide="left"
      kind="secondary"
      label={`${isSaved ? 'Update' : 'Save'} query`}
      loading={isSaving}
      onClick={() => {
        setIsSaving(true);
        saveQuery(tab).finally(() => {
          setIsSaving(false);
        });
      }}
      size="medium"
    />
  );
}
