import { useState, type ReactNode } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@inngest/components/DropdownMenu/DropdownMenu";

import { KeyboardShortcut } from "@/components/Insights/KeyboardShortcut";
import { useStoredQueries } from "@/components/Insights/QueryHelperPanel/StoredQueriesContext";
import type { Tab } from "@/components/Insights/types";
import { hasUnsavedChanges } from "./InsightsTabManager";
import { useTabManagerActions } from "./TabManagerContext";
import { HOME_TAB } from "./constants";

interface TabContextMenuProps {
  tabs: Tab[];
  onProcessPendingCloseTabs: (tabIds: string[]) => void;
  children: (props: {
    handleContextMenu: (e: React.MouseEvent, tabId: string) => void;
  }) => ReactNode;
}

export function TabContextMenu({
  tabs,
  onProcessPendingCloseTabs,
  children,
}: TabContextMenuProps) {
  const { tabManagerActions } = useTabManagerActions();
  const { queries } = useStoredQueries();
  const [contextMenu, setContextMenu] = useState<{
    tabId: string;
    x: number;
    y: number;
  } | null>(null);

  const handleContextMenu = (e: React.MouseEvent, tabId: string) => {
    e.preventDefault();
    setContextMenu({ tabId, x: e.clientX, y: e.clientY });
  };

  const handleCloseOtherTabs = (tabId: string) => {
    const tabsToClose = tabs.filter(
      (t) => t.id !== tabId && t.id !== HOME_TAB.id,
    );
    onProcessPendingCloseTabs(tabsToClose.map((t) => t.id));
    setContextMenu(null);
  };

  const handleCloseToTheRight = (tabId: string) => {
    const tabIndex = tabs.findIndex((t) => t.id === tabId);
    const tabsToClose = tabs.slice(tabIndex + 1);
    onProcessPendingCloseTabs(tabsToClose.map((t) => t.id));
    setContextMenu(null);
  };

  const handleCloseAll = () => {
    const tabsToClose = tabs.filter((t) => t.id !== HOME_TAB.id);
    onProcessPendingCloseTabs(tabsToClose.map((t) => t.id));
    setContextMenu(null);
  };

  return (
    <>
      {children({ handleContextMenu })}

      {contextMenu && (
        <DropdownMenu
          open={true}
          onOpenChange={(open) => !open && setContextMenu(null)}
        >
          <DropdownMenuContent
            align="start"
            className="fixed min-w-[240px]"
            style={{
              left: `${contextMenu.x}px`,
              top: `${contextMenu.y}px`,
            }}
          >
            <DropdownMenuItem
              className="text-basis flex items-center justify-between gap-8 px-4 outline-none"
              onSelect={tabManagerActions.createNewTab}
            >
              <span>New tab</span>
              <span className="ml-auto">
                <KeyboardShortcut
                  color="text-muted"
                  keys={["cmd", "ctrl", "alt", "t"]}
                />
              </span>
            </DropdownMenuItem>
            <div className="border-subtle my-1 border-t" />
            <DropdownMenuItem
              className="text-basis px-4 outline-none"
              onSelect={() => {
                const tab = tabs.find((t) => t.id === contextMenu.tabId);
                if (tab && hasUnsavedChanges(queries.data, tab)) {
                  onProcessPendingCloseTabs([contextMenu.tabId]);
                  setContextMenu(null);
                  return;
                }
                tabManagerActions.closeTab(contextMenu.tabId);
                setContextMenu(null);
              }}
            >
              <span>Close tab</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-basis px-4 outline-none"
              onSelect={() => handleCloseOtherTabs(contextMenu.tabId)}
            >
              <span>Close other tabs</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-basis px-4 outline-none"
              onSelect={() => handleCloseToTheRight(contextMenu.tabId)}
            >
              <span>Close to the right</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-basis px-4 outline-none"
              onSelect={handleCloseAll}
            >
              <span>Close all</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </>
  );
}
