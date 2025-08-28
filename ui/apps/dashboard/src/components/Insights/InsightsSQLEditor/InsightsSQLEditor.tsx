'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import { useInsightsSQLEditorOnMountCallback } from './useInsightsSQLEditorOnMountCallback';

export function InsightsSQLEditor() {
  const { onChange, query } = useInsightsStateMachineContext();
  const { onMount } = useInsightsSQLEditorOnMountCallback();

  return <SQLEditor content={query} onChange={onChange} onMount={onMount} />;
}
