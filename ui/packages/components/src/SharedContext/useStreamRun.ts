import { useEffect, useState } from 'react';

import type { Trace } from '../RunDetailsV3/types';
import { useShared } from './SharedContext';

export type StreamRunPayload = {
  runID: string;
};

export type StreamRunData = {
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
  status: string;
};

export type StreamRunCallbacks = {
  onData: (data: StreamRunData) => void;
  onError: (error: Error) => void;
  onComplete: () => void;
};

//
// StreamRun handler type - starts streaming and returns a cleanup function
export type StreamRunHandler = (
  payload: StreamRunPayload,
  callbacks: StreamRunCallbacks
) => () => void;

type UseStreamRunOptions = {
  runID?: string;
  enabled?: boolean;
};

export const useStreamRun = ({ runID, enabled = true }: UseStreamRunOptions) => {
  const shared = useShared();
  const [data, setData] = useState<StreamRunData | undefined>();
  const [error, setError] = useState<Error | undefined>();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!runID || !enabled || !shared.streamRun) {
      return;
    }

    setLoading(true);
    setError(undefined);

    const cleanup = shared.streamRun(
      { runID },
      {
        onData: setData,
        onError: (err) => {
          setError(err);
          setLoading(false);
        },
        onComplete: () => {
          setLoading(false);
        },
      }
    );

    return cleanup;
  }, [runID, enabled, shared.streamRun]);

  return {
    data,
    loading,
    error,
  };
};
