"use client";

import { RiBookmarkLine, RiHistoryLine, RiTeamLine } from "@remixicon/react";

type QueryHelperPanelSectionContentNoDataProps = {
  primary: string;
  secondary: string;
  sectionType: "history" | "saved" | "shared";
};

export function QueryHelperPanelSectionContentNoData({
  primary,
  secondary,
  sectionType,
}: QueryHelperPanelSectionContentNoDataProps) {
  const Icon =
    sectionType === "history"
      ? RiHistoryLine
      : sectionType === "saved"
      ? RiBookmarkLine
      : RiTeamLine;

  return (
    <div className="border-subtle flex min-h-[120px] w-full flex-col items-center justify-center gap-4 rounded border border-dashed px-4 py-6">
      <Icon className="text-disabled h-4 w-4" />
      <div className="flex flex-col gap-1 text-center">
        <div className="text-subtle text-xs font-medium">{primary}</div>
        <div className="text-muted text-xs">{secondary}</div>
      </div>
    </div>
  );
}
