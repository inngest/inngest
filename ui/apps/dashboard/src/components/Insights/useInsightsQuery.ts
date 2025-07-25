import { useCallback, useState } from 'react';

import { generateInsightsMockData } from './mockData';

const simulateDelay = (millis: number) => new Promise((resolve) => setTimeout(resolve, millis));

type QueryResult = {
  data: Record<string, any>[];
  totalRows: number;
};

export function useInsightsQuery() {
  const [result, setResult] = useState<QueryResult>({ data: [], totalRows: 0 });
  const [isLoading, setIsLoading] = useState(false);

  const executeQuery = useCallback(async (_query: string) => {
    setIsLoading(true);

    try {
      await simulateDelay(1000);

      const mockData = generateInsightsMockData(100);
      setResult({ data: mockData, totalRows: mockData.length });
    } catch (e) {
      // TODO: Handle error.
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { executeQuery, isLoading, result };
}
