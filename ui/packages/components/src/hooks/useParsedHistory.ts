import { useEffect, useState } from 'react';
import { HistoryParser, type RawHistoryItem } from '@inngest/components/utils/historyParser';

export function useParsedHistory(rawHistory: RawHistoryItem[]): HistoryParser {
  const [history, setHistory] = useState<HistoryParser>(new HistoryParser());

  useEffect(() => {
    if (rawHistory.length === 0) {
      if (Object.keys(history.getGroups()).length > 0) {
        setHistory(new HistoryParser());
      }

      // Return early to prevent infinite rerendering when consumers default
      // the rawHistory param to an empty array.
      return;
    }

    setHistory(new HistoryParser(rawHistory));
  }, [rawHistory]);

  return history;
}
