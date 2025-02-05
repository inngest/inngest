import { useState } from 'react';

import { useSignals, type SignalDefinitions } from './SignalsContext';

export const useInvokeRun = () => {
  const signals = useSignals();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const invoke = async (payload: SignalDefinitions['invokeRun']['payload']) => {
    try {
      setLoading(true);
      setError(null);
      return await signals.invokeRun(payload);
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
