'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiPlayFill } from '@remixicon/react';

import CodeEditor from '@/components/Textarea/CodeEditor';

const PLACEHOLDER_QUERY = `SELECT 
  HOUR(ts) as hour, 
  COUNT(*) as count 
WHERE 
  name = 'cli/dev_ui.loaded' 
  AND data.os != 'linux'
  AND ts > 1752845983000 
GROUP BY
  hour 
ORDER BY 
  hour desc`;

type SQLEditorProps = {
  isLoading?: boolean;
  onRunQuery: (query: string) => void;
};

export function SQLEditor({ isLoading = false, onRunQuery }: SQLEditorProps) {
  const [query, setQuery] = useState(PLACEHOLDER_QUERY);

  return (
    <div className="overflow-hidden">
      <div className="border-subtle flex h-12 items-center justify-between border-b px-4">
        <h3 className="text-basis text-sm font-medium">SQL Query</h3>
        <Button
          disabled={!query.trim() || isLoading}
          icon={<RiPlayFill />}
          iconSide="left"
          kind="primary"
          label="Run query"
          loading={isLoading}
          onClick={() => onRunQuery(query)}
          size="medium"
        />
      </div>
      <div className="bg-codeEditor py-2 pl-4">
        <CodeEditor
          className="h-[250px]"
          initialCode={PLACEHOLDER_QUERY}
          label="SQL Query"
          language="sql"
          name="sql-query"
          onCodeChange={setQuery}
        />
      </div>
    </div>
  );
}
