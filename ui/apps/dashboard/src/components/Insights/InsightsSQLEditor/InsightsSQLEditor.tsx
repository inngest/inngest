'use client';

import { useState } from 'react';
import { SQLEditor } from '@inngest/components/SQLEditor/SQLEditor';

const DEFAULT_QUERY = `SELECT
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

export function InsightsSQLEditor() {
  const [content, setContent] = useState<string>(DEFAULT_QUERY);

  return <SQLEditor content={content} onChange={setContent} />;
}
