import { useState } from 'react';

import { useSignals, type SignalDefinitions } from './SignalsContext';

export type RerunFromStepPayload = {
  runID: string;
  fromStep: { stepID: string; input: string };
};

export type RerunFromStepResult = {
  error?: Error;
  loading?: boolean;
  data?: {
    rerun: unknown;
  };
};

export const useRerunFromStep = () => {
  const signals = useSignals();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const rerun = async (payload: SignalDefinitions['rerunFromStep']['payload']) => {
    try {
      setLoading(true);
      setError(null);
      return await signals.rerunFromStep(payload);
    } catch (err) {
      console.error('error rerunning from step', err);
      setError(err instanceof Error ? err : new Error('Error rerunning from step'));
    } finally {
      setLoading(false);
    }
  };

  return {
    rerun,
    loading,
    error,
  };
};
