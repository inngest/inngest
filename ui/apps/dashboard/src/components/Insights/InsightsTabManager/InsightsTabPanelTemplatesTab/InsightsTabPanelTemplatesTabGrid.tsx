"use client";

import { cn } from "@inngest/components/utils/classNames";
import { RiAddLine } from "@remixicon/react";

import { useTabManagerActions } from "../TabManagerContext";
import {
  BUTTON_CARD_STYLES,
  InsightsTabPanelTemplatesTabCard,
} from "./InsightsTabPanelTemplatesTabCard";
import { TEMPLATES } from "./templates";

export function InsightsTabPanelTemplatesTabGrid() {
  const { tabManagerActions } = useTabManagerActions();

  return (
    <div className="flex flex-wrap gap-6 pb-12">
      <button
        className={cn(
          BUTTON_CARD_STYLES,
          "bg-canvasSubtle text-muted flex flex-col justify-center gap-3 border-none text-sm shadow-none",
        )}
        onClick={() => {
          tabManagerActions.createNewTab();
        }}
      >
        <div className="border-muted text-basis bg-surfaceSubtle dark:bg-surfaceMuted self-center rounded-[4px] border p-1">
          <RiAddLine className="h-5 w-5" />
        </div>
        <p className="text-basis self-center text-sm">Start from scratch</p>
      </button>
      {TEMPLATES.map((template) => (
        <InsightsTabPanelTemplatesTabCard
          key={template.id}
          template={template}
        />
      ))}
    </div>
  );
}
