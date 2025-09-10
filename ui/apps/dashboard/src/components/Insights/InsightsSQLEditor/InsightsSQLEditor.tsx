'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { SQL_COMPLETION_CONFIG } from './constants';
import { useInsightsSQLEditorOnMountCallback } from './useInsightsSQLEditorOnMountCallback';

export function InsightsSQLEditor() {
  const { onChange, query } = useInsightsStateMachineContext();
  const { onMount } = useInsightsSQLEditorOnMountCallback();

  return (
    <SQLEditor
      completionConfig={SQL_COMPLETION_CONFIG}
      content={query}
      onChange={onChange}
      onMount={onMount}
    />
  );
}
