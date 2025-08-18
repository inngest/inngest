'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import Tabs from '@inngest/components/Tabs/Tabs';
import {
  RiAddLine,
  RiBookReadLine,
  RiCircleFill,
  RiCodeSSlashLine,
  RiContractLeftLine,
  RiExpandRightLine,
} from '@remixicon/react';

import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import { hasSavedQueryWithUnsavedChanges, type TabConfig } from './InsightsTabManager';
import { useTabManagerActions } from './TabManagerContext';
import { TEMPLATES_TAB } from './constants';

interface InsightsTabsListProps {
  activeTabId: string;
  isQueryHelperPanelVisible: boolean;
  onToggleQueryHelperPanelVisibility: () => void;
  tabs: TabConfig[];
}

export function InsightsTabsList({
  activeTabId,
  isQueryHelperPanelVisible,
  onToggleQueryHelperPanelVisibility,
  tabs,
}: InsightsTabsListProps) {
  const { tabManagerActions } = useTabManagerActions();
  const { queries } = useStoredQueries();
  const [pendingCloseTabId, setPendingCloseTabId] = useState<string | null>(null);

  const ActionTabIcon = isQueryHelperPanelVisible ? RiContractLeftLine : RiExpandRightLine;
  const pendingCloseTab = pendingCloseTabId ? tabs.find((t) => t.id === pendingCloseTabId) : null;

  return (
    <>
      <Tabs
        onClose={(tabId: string) => {
          const tab = tabs.find((t) => t.id === tabId);
          if (tab === undefined) {
            tabManagerActions.closeTab(tabId);
            return;
          }

          if (hasSavedQueryWithUnsavedChanges(tab, queries)) {
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
            title={`${isQueryHelperPanelVisible ? 'Hide' : 'Show'} sidebar`}
          />
          {tabs.map((tab) => (
            <Tabs.Tab iconBefore={<IndicatorTabIcon tab={tab} />} key={tab.id} value={tab.id}>
              {tab.name}
            </Tabs.Tab>
          ))}
          <Tabs.IconTab
            icon={<RiAddLine size={16} />}
            onClick={tabManagerActions.createNewTab}
            title="Add new tab"
          />
        </Tabs.List>
      </Tabs>

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
            Are you sure you want to close <strong>{pendingCloseTab?.name}</strong> without saving
            your changes?
          </p>
          <Alert className="mt-4 text-sm" severity="warning">
            Your changes will be lost if you close this tab without saving it.
          </Alert>
        </div>
      </AlertModal>
    </>
  );
}

function IndicatorTabIcon({ tab }: { tab: TabConfig }) {
  const { queries } = useStoredQueries();

  if (tab.id === TEMPLATES_TAB.id) return <RiBookReadLine size={16} />;

  const savedQuery = tab.savedQueryId ? queries[tab.savedQueryId] : undefined;
  if (savedQuery === undefined || !hasSavedQueryWithUnsavedChanges(tab, queries)) {
    return <RiCodeSSlashLine size={16} />;
  }

  return <RiCircleFill className="fill-amber-500" size={16} />;
}
