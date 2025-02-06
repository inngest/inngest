import { useState } from 'react';

import { useShared, type SharedDefinitions } from './SharedContext';

export interface InvokeRunPayload {
  envID?: string;
  functionSlug: string;
  data: Record<string, unknown>;
  user: Record<string, unknown> | null;
}
export type InvokeRunResult = {
  error?: unknown;
  loading?: boolean;
  data?: unknown;
};

export const useInvokeRun = () => {
  const shared = useShared();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const invoke = async (payload: SharedDefinitions['invokeRun']['payload']) => {
    try {
      setLoading(true);
      setError(null);
      return await shared.invokeRun(payload);
    } catch (err) {
      console.error('error invoking function', err);
      setError(err instanceof Error ? err : new Error('Error invoking function'));
    } finally {
      setLoading(false);
    }
  };

  return {
    invoke,
    loading,
    error,
  };
};
