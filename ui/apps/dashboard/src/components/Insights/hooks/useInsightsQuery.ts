import { useCallback, useState } from 'react';

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

export interface UseInsightsQueryReturn {
  content: string;
  isRunning: boolean;
  onChange: (value: string) => void;
  runQuery: () => void;
}

export function useInsightsQuery(): UseInsightsQueryReturn {
  const [content, setContent] = useState(DEFAULT_QUERY);
  const [isRunning, setIsRunning] = useState(false);

  // TODO: Implement actual query
  const runQuery = useCallback(() => {
    if (isRunning) return;

    setIsRunning(true);

    setTimeout(() => {
      setIsRunning(false);
    }, 2500);
  }, [isRunning, content]);

  return { content, isRunning, onChange: setContent, runQuery };
}
