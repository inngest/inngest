"use client";

import { Button } from "@inngest/components/Button/Button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@inngest/components/Tooltip";
import { cn } from "@inngest/components/utils/classNames";
import { RiSaveLine } from "@remixicon/react";

import { KeyboardShortcutTooltip } from "../KeyboardShortcutTooltip";
import type { Tab } from "../types";
import { useSaveTab } from "./SaveTabContext";
import { useDocumentShortcuts } from "./actions/handleShortcuts";

type InsightsSQLEditorSaveQueryButtonProps = {
  tab: Tab;
};

export function InsightsSQLEditorSaveQueryButton({
  tab,
}: InsightsSQLEditorSaveQueryButtonProps) {
  const { canSave, isSaving, saveTab } = useSaveTab(tab);

  useDocumentShortcuts([
    {
      combo: { alt: true, code: "KeyS", metaOrCtrl: true },
      handler: saveTab,
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
                className={cn(!canSave ? "text-disabled" : "text-muted")}
                size={16}
              />
            }
            iconSide="left"
            kind="secondary"
            label="Save"
            loading={isSaving}
            onClick={() => {
              saveTab();
            }}
            size="medium"
          />
        </TooltipTrigger>
        <TooltipContent>
          <KeyboardShortcutTooltip
            combo={{ alt: true, key: "S", metaOrCtrl: true }}
          />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
