'use client';

import type { SQLEditorMountCallback } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../../InsightsStateMachineContext/InsightsStateMachineContext';
import { useActiveTab, useTabManagerActions } from '../../InsightsTabManager/TabManagerContext';
import { HOME_TAB, TEMPLATES_TAB } from '../../InsightsTabManager/constants';
import type { Tab } from '../../types';
import { useSaveTabActions } from '../SaveTabContext';
import { bindEditorShortcuts } from '../actions/handleShortcuts';
import { markTemplateVars } from '../actions/markTemplateVars';
import { getCanRunQuery } from '../utils';
import { useLatest, useLatestCallback } from './useLatestCallback';

type UseInsightsSQLEditorOnMountCallbackReturn = {
  onMount: SQLEditorMountCallback;
};

export function useInsightsSQLEditorOnMountCallback(): UseInsightsSQLEditorOnMountCallbackReturn {
  const { query, runQuery, status } = useInsightsStateMachineContext();
  const { saveTab } = useSaveTabActions();
  const { tabManagerActions } = useTabManagerActions();
  const { activeTab } = useActiveTab();

  const latestQueryRef = useLatest(query);
  const isRunningRef = useLatest(status === 'loading');
  const activeTabRef = useLatest(activeTab);
  const saveTabRef = useLatest(saveTab);

  const onMount: SQLEditorMountCallback = useLatestCallback((editor, monaco) => {
    const shortcutsDisposable = bindEditorShortcuts(
      editor,
      {
        combo: { keyCode: monaco.KeyCode.Enter, metaOrCtrl: true },
        handler: () => {
          if (getCanRunQuery(latestQueryRef.current, isRunningRef.current)) runQuery();
        },
      },
      {
        combo: { alt: true, keyCode: monaco.KeyCode.KeyS, metaOrCtrl: true },
        handler: () => {
          const currentTab = activeTabRef.current;
          if (currentTab !== undefined) saveTabRef.current(currentTab);
        },
      },
      {
        combo: { alt: true, keyCode: monaco.KeyCode.KeyT, metaOrCtrl: true },
        handler: tabManagerActions.createNewTab,
      }
    );

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
