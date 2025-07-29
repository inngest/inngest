'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

import { useInsightsQueryContext } from '../context';

export function InsightsSQLEditor() {
  const { content, onChange } = useInsightsQueryContext();

  return <SQLEditor content={content} onChange={onChange} />;
}
