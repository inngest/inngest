import { useState } from 'react';

import type { Trace } from '../RunDetailsV3/types';
import { useShared } from './SharedContext';

export type GetRunPayload = {
  runID: string;
};

export type GetRunResult = {
  error?: Error;
  loading: boolean;
  data?: {
    app: {
      externalID: string;
      name: string;
    };
    fn: {
      id: string;
      name: string;
      slug: string;
    };
    id: string;
    trace: Trace;
    hasAI: boolean;
  };
};

export const useGetRun = () => {
  const shared = useShared();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const getRun = async (payload: GetRunPayload) => {
    try {
      setLoading(true);
      setError(null);
      return await shared.getRun(payload);
    } catch (err) {
      console.error('error gettting function run', err);
      setError(err instanceof Error ? err : new Error('Error getting function run'));
    } finally {
      setLoading(false);
    }
  };

  return {
    loading,
    error,
    getRun,
  };
};
