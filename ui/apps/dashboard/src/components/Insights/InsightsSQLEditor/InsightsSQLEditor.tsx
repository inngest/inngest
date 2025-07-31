'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';

export function InsightsSQLEditor() {
  const { onChange, query } = useInsightsStateMachineContext();

  return <SQLEditor content={query} onChange={onChange} />;
}
