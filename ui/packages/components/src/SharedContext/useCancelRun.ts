import { useShared } from './SharedContext';

export type CancelRunPayload = {
  runID: string;
};

export type CancelRunResult = {
  error?: Error;
  data?: {
    cancelRun?: { id?: string };
  };
};

export const useCancelRun = () => {
  const shared = useShared();
  const cancelRun = async (payload: CancelRunPayload): Promise<CancelRunResult> =>
    shared.cancelRun(payload);

  return {
    cancelRun,
  };
};
