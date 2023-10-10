import { useEffect, useState } from 'react';

import type { RunHistoryItem } from '@/store/generated';
import { HistoryParser } from './historyParser';
import type { HistoryNode } from './types';

export function useParsedHistory(rawHistory: RunHistoryItem[]): Record<string, HistoryNode> {
  const [history, setHistory] = useState<Record<string, HistoryNode>>({});

  useEffect(() => {
    if (rawHistory.length === 0) {
      if (Object.keys(history).length > 0) {
        setHistory({});
      }

      // Return early to prevent infinite rerendering when consumers default
      // the rawHistory param to an empty array.
      return;
    }

    setHistory(new HistoryParser(rawHistory).history);
  }, [rawHistory]);

  return history;
}
