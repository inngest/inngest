import { useState } from "react";
import { Alert } from "@inngest/components/Alert/Alert";
import { AlertModal } from "@inngest/components/Modal/AlertModal";
import Tabs from "@inngest/components/Tabs/Tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@inngest/components/Tooltip";
import {
  RiAddLine,
  RiBookReadLine,
  RiCircleFill,
  RiCodeSSlashLine,
  RiContractLeftLine,
  RiExpandRightLine,
  RiHome4Line,
} from "@remixicon/react";

import { KeyboardShortcutTooltip } from "@/components/Insights/KeyboardShortcutTooltip";
import { useStoredQueries } from "@/components/Insights/QueryHelperPanel/StoredQueriesContext";
import type { Tab } from "@/components/Insights/types";
import { hasDiffWithSavedQuery } from "./InsightsTabManager";
import { useTabManagerActions } from "./TabManagerContext";
import { HOME_TAB, TEMPLATES_TAB } from "./constants";

interface InsightsTabsListProps {
  activeTabId: string;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: Tab[];
}

export function InsightsTabsList({
  activeTabId,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  tabs,
}: InsightsTabsListProps) {
  const { tabManagerActions } = useTabManagerActions();
  const { queries } = useStoredQueries();
  const [pendingCloseTabId, setPendingCloseTabId] = useState<string | null>(
    null,
  );

  const ActionTabIcon = isQueryHelperPanelVisible
    ? RiContractLeftLine
    : RiExpandRightLine;
  const pendingCloseTab = pendingCloseTabId
    ? tabs.find((t) => t.id === pendingCloseTabId)
    : null;

  return (
    <>
      <TooltipProvider>
        <Tabs
          onClose={(tabId: string) => {
            const tab = tabs.find((t) => t.id === tabId);
            if (tab === undefined) return;

            if (hasDiffWithSavedQuery(queries.data, tab)) {
              setPendingCloseTabId(tabId);
              return;
            }

            tabManagerActions.closeTab(tabId);
          }}
          onValueChange={tabManagerActions.focusTab}
          value={activeTabId}
        >
          <Tabs.List>
            <Tabs.IconTab
              icon={<ActionTabIcon size={16} />}
              onClick={onToggleQueryHelperPanelVisibility}
              title={`${isQueryHelperPanelVisible ? "Hide" : "Show"} sidebar`}
            />
            <Tabs.IconTab
              icon={<RiHome4Line size={16} />}
              onClick={() => tabManagerActions.focusTab(HOME_TAB.id)}
              value={HOME_TAB.id}
            />
            {tabs
              .filter((tab) => tab.id !== HOME_TAB.id)
              .map((tab) => (
                <Tabs.Tab
                  iconBefore={<IndicatorTabIcon tab={tab} />}
                  key={tab.id}
                  title={tab.name}
                  value={tab.id}
                />
              ))}
            <Tooltip>
              <TooltipTrigger asChild>
                <Tabs.IconTab
                  icon={<RiAddLine size={16} />}
                  onClick={tabManagerActions.createNewTab}
                />
              </TooltipTrigger>
              <TooltipContent>
                Add new tab (
                <KeyboardShortcutTooltip
                  combo={{ alt: true, key: "T", metaOrCtrl: true }}
                />
                )
              </TooltipContent>
            </Tooltip>
          </Tabs.List>
        </Tabs>
      </TooltipProvider>

      <AlertModal
        cancelButtonLabel="Cancel"
        className="w-[656px]"
        confirmButtonLabel="Confirm"
        isOpen={Boolean(pendingCloseTab)}
        onClose={() => {
          setPendingCloseTabId(null);
        }}
        onSubmit={() => {
          if (pendingCloseTabId) {
            tabManagerActions.closeTab(pendingCloseTabId);
            setPendingCloseTabId(null);
          }
        }}
        title="Unsaved changes"
      >
        <div className="p-6">
          <p className="text-subtle text-sm">
            Are you sure you want to close{" "}
            <strong>{pendingCloseTab?.name}</strong> without saving your
            changes?
          </p>
          <Alert className="mt-4 text-sm" severity="warning">
            Your changes will be lost if you close this tab without saving it.
          </Alert>
        </div>
      </AlertModal>
    </>
  );
}

function IndicatorTabIcon({ tab }: { tab: Tab }) {
  const { queries } = useStoredQueries();

  if (tab.id === TEMPLATES_TAB.id) {
    return <RiBookReadLine size={16} />;
  } else if (hasDiffWithSavedQuery(queries.data, tab)) {
    return <RiCircleFill className="fill-amber-500" size={16} />;
  } else {
    return <RiCodeSSlashLine size={16} />;
  }
}
