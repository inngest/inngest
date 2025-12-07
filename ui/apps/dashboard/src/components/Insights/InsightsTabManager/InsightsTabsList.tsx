import { useState } from "react";
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

import { useSaveTabActions } from "@/components/Insights/InsightsSQLEditor/SaveTabContext";
import { KeyboardShortcutTooltip } from "@/components/Insights/KeyboardShortcutTooltip";
import { useStoredQueries } from "@/components/Insights/QueryHelperPanel/StoredQueriesContext";
import type { Tab } from "@/components/Insights/types";
import { hasUnsavedChanges } from "./InsightsTabManager";
import { TabContextMenu } from "./TabContextMenu";
import { useTabManagerActions } from "./TabManagerContext";
import { UnsavedChangesModal } from "./UnsavedChangesModal";
import { HOME_TAB, TEMPLATES_TAB } from "./constants";

/**
 * Filters a list of tab IDs to return only tabs with unsaved changes
 */
function getTabsWithUnsavedChanges(
  tabIds: string[],
  allTabs: Tab[],
  queriesData: ReturnType<typeof useStoredQueries>["queries"]["data"],
): Tab[] {
  return tabIds
    .map((id) => allTabs.find((t) => t.id === id))
    .filter(
      (tab): tab is Tab =>
        tab !== undefined && hasUnsavedChanges(queriesData, tab),
    );
}

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
  const { saveTab } = useSaveTabActions();
  const [pendingCloseTabIds, setPendingCloseTabIds] = useState<string[]>([]);

  const ActionTabIcon = isQueryHelperPanelVisible
    ? RiContractLeftLine
    : RiExpandRightLine;
  // Get all tabs with unsaved changes that are pending close
  const unsavedTabsPendingClose = getTabsWithUnsavedChanges(
    pendingCloseTabIds,
    tabs,
    queries.data,
  );

  const processPendingCloseTabs = (tabIdsToClose: string[]) => {
    // Find all tabs with unsaved changes
    const tabsWithUnsavedChanges = getTabsWithUnsavedChanges(
      tabIdsToClose,
      tabs,
      queries.data,
    );

    if (tabsWithUnsavedChanges.length > 0) {
      // Show bulk confirmation modal for all unsaved tabs
      setPendingCloseTabIds(tabIdsToClose);
      return;
    }

    // No unsaved changes, close all tabs
    tabIdsToClose.forEach((id) => {
      tabManagerActions.closeTab(id);
    });
    setPendingCloseTabIds([]);
  };

  const handleDiscardAll = () => {
    // Close all pending tabs without saving
    pendingCloseTabIds.forEach((id) => {
      tabManagerActions.closeTab(id);
    });
    setPendingCloseTabIds([]);
  };

  const handleSaveAll = async () => {
    // Save tabs sequentially and track which ones succeeded
    const savedTabIds: string[] = [];
    let encounteredError = false;

    for (const tab of unsavedTabsPendingClose) {
      try {
        await saveTab(tab);
        // If saveTab completes without error, mark as saved
        savedTabIds.push(tab.id);
      } catch (error) {
        // Stop on first error
        encounteredError = true;
        break;
      }
    }

    // Close all tabs that were successfully saved
    savedTabIds.forEach((id) => {
      tabManagerActions.closeTab(id);
    });

    if (
      !encounteredError &&
      savedTabIds.length === unsavedTabsPendingClose.length
    ) {
      // All tabs saved and closed successfully, close any remaining tabs without unsaved changes
      pendingCloseTabIds
        .filter((id) => !savedTabIds.includes(id))
        .forEach((id) => {
          tabManagerActions.closeTab(id);
        });
      setPendingCloseTabIds([]);
    } else {
      // Update pending list to only include tabs that weren't saved
      setPendingCloseTabIds((prev) =>
        prev.filter((id) => !savedTabIds.includes(id)),
      );
    }
  };

  return (
    <>
      <TabContextMenu
        tabs={tabs}
        onProcessPendingCloseTabs={processPendingCloseTabs}
      >
        {({ handleContextMenu }) => (
          <TooltipProvider>
            <Tabs
              onClose={(tabId: string) => {
                const tab = tabs.find((t) => t.id === tabId);
                if (tab === undefined) return;

                if (hasUnsavedChanges(queries.data, tab)) {
                  processPendingCloseTabs([tabId]);
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
                  title={`${
                    isQueryHelperPanelVisible ? "Hide" : "Show"
                  } sidebar`}
                />
                <Tabs.IconTab
                  icon={<RiHome4Line size={16} />}
                  onClick={() => tabManagerActions.focusTab(HOME_TAB.id)}
                  value={HOME_TAB.id}
                />
                <div className="-mr-px flex overflow-x-auto">
                  {tabs
                    .filter((tab) => tab.id !== HOME_TAB.id)
                    .map((tab) => (
                      <Tabs.Tab
                        iconBefore={<IndicatorTabIcon tab={tab} />}
                        key={tab.id}
                        onContextMenu={(e) => handleContextMenu(e, tab.id)}
                        title={tab.name}
                        value={tab.id}
                      />
                    ))}
                </div>
                <div className="border-subtle border-l">
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
                </div>
              </Tabs.List>
            </Tabs>
          </TooltipProvider>
        )}
      </TabContextMenu>

      <UnsavedChangesModal
        isOpen={unsavedTabsPendingClose.length > 0}
        unsavedTabs={unsavedTabsPendingClose}
        onCancel={() => setPendingCloseTabIds([])}
        onDiscardAll={handleDiscardAll}
        onSaveAll={handleSaveAll}
      />
    </>
  );
}

function IndicatorTabIcon({ tab }: { tab: Tab }) {
  const { queries } = useStoredQueries();

  if (tab.id === TEMPLATES_TAB.id) {
    return <RiBookReadLine size={16} />;
  } else if (hasUnsavedChanges(queries.data, tab)) {
    return <RiCircleFill className="fill-amber-500" size={16} />;
  } else {
    return <RiCodeSSlashLine size={16} />;
  }
}
