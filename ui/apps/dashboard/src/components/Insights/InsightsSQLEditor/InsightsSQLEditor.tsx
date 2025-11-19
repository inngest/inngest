'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { useInsightsSQLEditorOnMountCallback } from './hooks/useInsightsSQLEditorOnMountCallback';
import { useSQLCompletionConfig } from './hooks/useSQLCompletionConfig';

export function InsightsSQLEditor() {
  const { onChange, query } = useInsightsStateMachineContext();
  const { onMount } = useInsightsSQLEditorOnMountCallback();
  const completionConfig = useSQLCompletionConfig();

  return (
    <div className="h-full min-h-0 overflow-hidden">
      <SQLEditor
        completionConfig={completionConfig}
        content={query}
        onChange={onChange}
        onMount={onMount}
      />
    </div>
  );
}
