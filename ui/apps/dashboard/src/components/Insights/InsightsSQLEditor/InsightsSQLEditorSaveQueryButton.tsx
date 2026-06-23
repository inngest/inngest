import { Button } from '@inngest/components/Button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiSaveLine } from '@remixicon/react';
import { useCallback } from 'react';

import { useInsightsChatProviderOptional } from '../InsightsTabManager/InsightsHelperPanel/features/InsightsChat/InsightsChatProvider';
import { postQueryFeedback, sqlWasEdited } from '../queryFeedback';
import { KeyboardShortcutTooltip } from '../KeyboardShortcutTooltip';
import type { Tab } from '../types';
import { useSaveTab } from './SaveTabContext';
import { useDocumentShortcuts } from './actions/handleShortcuts';

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: Tab;
};

export function InsightsSQLEditorSaveQueryButton({
  tab,
}: InsightsSQLEditorSaveQueryButtonProps) {
  const { canSave, isSaving, saveTab } = useSaveTab(tab);
  const chat = useInsightsChatProviderOptional();

  // Saving an unmodified AI-generated query is a strong positive signal.
  const handleSave = useCallback(() => {
    saveTab();
    const threadId = chat?.currentThreadId;
    if (!chat || !threadId) return;
    const runId = chat.getLatestRunId(threadId);
    if (!runId) return;
    if (sqlWasEdited(chat.getLatestGeneratedSql(threadId), tab.query)) return;
    void postQueryFeedback({ runId, saved: true });
  }, [saveTab, chat, tab.query]);

  useDocumentShortcuts([
    {
      combo: { alt: true, code: 'KeyS', metaOrCtrl: true },
      handler: handleSave,
    },
  ]);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            appearance="outlined"
            className="font-medium"
            disabled={!canSave}
            icon={
              <RiSaveLine
                className={cn(!canSave ? 'text-disabled' : 'text-muted')}
                size={16}
              />
            }
            iconSide="left"
            kind="secondary"
            label="Save"
            loading={isSaving}
            onClick={() => {
              handleSave();
            }}
            size="medium"
          />
        </TooltipTrigger>
        <TooltipContent>
          <KeyboardShortcutTooltip
            combo={{ alt: true, key: 'S', metaOrCtrl: true }}
          />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
