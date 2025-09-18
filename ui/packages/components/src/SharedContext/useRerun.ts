import { useShared } from './SharedContext';

export type RerunPayload = {
  runID: string;
  //
  // this is only required for the dashboard
  // and is validated there
  fnID?: string;
  debugRunID?: string;
  debugSessionID?: string;
};

export type RerunResult = {
  error?: Error;
  data?: {
    newRunID?: string;
  };
  redirect?: string;
};

export const useRerun = () => {
  const shared = useShared();
  const rerun = async (payload: RerunPayload): Promise<RerunResult> => shared.rerun(payload);

  return {
    rerun,
  };
};
