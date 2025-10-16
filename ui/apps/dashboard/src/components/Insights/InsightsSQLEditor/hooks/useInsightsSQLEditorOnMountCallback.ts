'use client';

import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import { useActiveTab, useTabManagerActions } from '../../InsightsTabManager/TabManagerContext';
import { HOME_TAB, TEMPLATES_TAB } from '../../InsightsTabManager/constants';
import { useStoredQueries } from '../../QueryHelperPanel/StoredQueriesContext';
import type { Tab } from '../../types';
import { handleShortcuts } from '../actions/handleShortcuts';
import { markTemplateVars } from '../actions/markTemplateVars';
import { getCanRunQuery } from '../utils';
import { useLatest, useLatestCallback } from './useLatestCallback';

type UseInsightsSQLEditorOnMountCallbackReturn = {
  onMount: SQLEditorMountCallback;
};

export type SQLShortcutActions = {
  onRun: () => void;
  onSave: () => void;
  onNewTab: () => void;
};

export function useInsightsSQLEditorOnMountCallback(): UseInsightsSQLEditorOnMountCallbackReturn {
  const { query, runQuery, status } = useInsightsStateMachineContext();
  const { saveQuery } = useStoredQueries();
  const { tabManagerActions } = useTabManagerActions();
  const { activeTab } = useActiveTab();

  const latestQueryRef = useLatest(query);
  const isRunningRef = useLatest(status === 'loading');
  const activeTabRef = useLatest(activeTab);

  const onMount: SQLEditorMountCallback = useLatestCallback((editor, monaco) => {
    const shortcutsDisposable = handleShortcuts(editor, monaco, {
      onRun: () => {
        if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) runQuery();
      },
      onSave: () => {
        const currentTab = activeTabRef.current;
        if (currentTab !== undefined && isQueryTab(currentTab)) {
          saveQuery(currentTab);
        }
      },
      onNewTab: tabManagerActions.createNewTab,
    });

    const markersDisposable = markTemplateVars(editor, monaco);

    // TODO: This code is not currently running. It turns out that actually doing so would
    // require messy exterior code. This is here to demonstrate roughly the pattern that would
    // be needed if any of these "actions" truly needed to run disposable functions. As for now,
    // neither of them do anything truly global, so all necessary cleanup should happen just as
    // a result of the monaco editor unmounting.
    return () => {
      shortcutsDisposable.dispose();
      markersDisposable.dispose();
    };
  });

  return { onMount };
}

function isQueryTab(tab: Tab): boolean {
  return tab.id !== HOME_TAB.id && tab.id !== TEMPLATES_TAB.id;
}
