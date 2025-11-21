'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { useActiveTab } from '../InsightsTabManager/TabManagerContext';
import { SQLEditorContextMenu } from './SQLEditorContextMenu';
import { useSQLEditorInstance } from './SQLEditorInstanceContext';
import { useSaveTabActions } from './SaveTabContext';
import { useInsightsSQLEditorOnMountCallback } from './hooks/useInsightsSQLEditorOnMountCallback';
import { useSQLCompletionConfig } from './hooks/useSQLCompletionConfig';

export function InsightsSQLEditor() {
  const { onChange, query, runQuery } = useInsightsStateMachineContext();
  const { onMount } = useInsightsSQLEditorOnMountCallback();
  const completionConfig = useSQLCompletionConfig();
  const { editorRef } = useSQLEditorInstance();
  const { activeTab } = useActiveTab();
  const { saveTab } = useSaveTabActions();

  const hasSelection = () => {
    const editor = editorRef.current;
    if (!editor) return false;
    const selection = editor.getSelection();
    return selection ? !selection.isEmpty() : false;
  };

  const handleCopy = async () => {
    const editor = editorRef.current;
    if (!editor) return;

    const selection = editor.getSelection();
    if (selection && !selection.isEmpty()) {
      const text = editor.getModel()?.getValueInRange(selection);
      if (text) {
        try {
          await navigator.clipboard.writeText(text);
        } catch (err) {
          console.error('Failed to copy:', err);
        }
      }
    }
  };

  const handleCut = async () => {
    const editor = editorRef.current;
    if (!editor) return;

    const selection = editor.getSelection();
    if (selection && !selection.isEmpty()) {
      const text = editor.getModel()?.getValueInRange(selection);
      if (text) {
        try {
          await navigator.clipboard.writeText(text);
          editor.executeEdits('cut', [
            {
              range: selection,
              text: '',
            },
          ]);
        } catch (err) {
          console.error('Failed to cut:', err);
        }
      }
    }
  };

  const handlePaste = async () => {
    const editor = editorRef.current;
    if (!editor) return;

    try {
      const text = await navigator.clipboard.readText();
      const selection = editor.getSelection();
      if (selection) {
        editor.executeEdits('paste', [
          {
            range: selection,
            text,
          },
        ]);
      }
    } catch (err) {
      console.error('Failed to paste:', err);
    }
  };

  const handlePrettifySQL = () => {
    const editor = editorRef.current;
    if (!editor) return;
    editor.getAction('editor.action.formatDocument')?.run();
  };

  const handleRunQuery = () => {
    runQuery();
  };

  const handleSaveQuery = () => {
    if (activeTab) {
      saveTab(activeTab);
    }
  };

  return (
    <div className="h-full min-h-0 overflow-hidden">
      <SQLEditor
        completionConfig={completionConfig}
        content={query}
        onChange={onChange}
        onMount={onMount}
      />
      <SQLEditorContextMenu
        onCopy={handleCopy}
        onCut={handleCut}
        onPaste={handlePaste}
        onPrettifySQL={handlePrettifySQL}
        onRunQuery={handleRunQuery}
        onSaveQuery={handleSaveQuery}
        hasSelection={hasSelection}
      />
    </div>
  );
}
