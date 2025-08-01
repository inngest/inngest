'use client';

import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

interface InsightsSQLEditorProps {
  content: string;
  onChange: (value: string) => void;
}

export function InsightsSQLEditor({ content, onChange }: InsightsSQLEditorProps) {
  return <SQLEditor content={content} onChange={onChange} />;
}
