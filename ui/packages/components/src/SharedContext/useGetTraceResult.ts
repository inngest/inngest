import { useState } from 'react';

import { useShared } from './SharedContext';

export type GetTraceResultPayload = {
  traceID: string;
  preview?: boolean;
};

export type TraceResult = {
  input: string | null;
  data: string | null;
  error: {
    message: string;
    name: string | null;
    stack: string | null;
    cause: string | null;
  } | null;
};

export const useGetTraceResult = () => {
  const shared = useShared();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const getTraceResult = async (payload: GetTraceResultPayload) => {
    try {
      setLoading(true);
      setError(null);
      return await shared.getTraceResult(payload);
    } catch (err) {
      console.error('error gettting trace result output run', err);
      setError(err instanceof Error ? err : new Error('Error getting trace result output'));
    } finally {
      setLoading(false);
    }
  };

  return {
    loading,
    error,
    getTraceResult,
  };
};
