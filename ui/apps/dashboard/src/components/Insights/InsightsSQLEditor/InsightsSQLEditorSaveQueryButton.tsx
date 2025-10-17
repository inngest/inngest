'use client';

import { Button } from '@inngest/components/Button/Button';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkFill, RiBookmarkLine } from '@remixicon/react';

import type { Tab } from '../types';
import { useSaveTab } from './SaveTabContext';
import { useDocumentShortcuts } from './actions/handleShortcuts';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: Tab;
};

export function InsightsSQLEditorSaveQueryButton({ tab }: InsightsSQLEditorSaveQueryButtonProps) {
  const { canSave, isSaved, isSaving, saveTab } = useSaveTab(tab);

  const Icon = isSaved ? RiBookmarkFill : RiBookmarkLine;

  useDocumentShortcuts([
    {
      combo: { alt: true, code: 'KeyS', metaOrCtrl: true },
      handler: saveTab,
    },
  ]);

  return (
    <Button
      appearance="outlined"
      className="font-medium"
      disabled={!canSave}
      icon={<Icon className={cn(!canSave ? 'text-disabled' : 'text-muted')} size={16} />}
      iconSide="left"
      kind="secondary"
      label={`${isSaved ? 'Update' : 'Save'} query`}
      loading={isSaving}
      onClick={() => {
        saveTab();
      }}
      size="medium"
    />
  );
}
