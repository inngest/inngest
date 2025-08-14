import { useShared } from './SharedContext';

export type RerunFromStepPayload = {
  runID: string;
  fromStep: { stepID: string; input?: string };
  debugSessionID?: string;
  debugRunID?: string;
};

export type RerunFromStepResult = {
  error?: Error;
  data?: {
    rerun: unknown;
  };
  redirect?: string;
};

export const useRerunFromStep = () => {
  const shared = useShared();
  const rerun = async (payload: RerunFromStepPayload) => shared.rerunFromStep(payload);

  return {
    rerun,
  };
};
