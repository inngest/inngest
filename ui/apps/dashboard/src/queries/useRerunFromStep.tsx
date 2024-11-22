import type { RerunResult } from '@inngest/components/Rerun/RerunModal';

type RerunFromStep = {
  runID: string;
  fromStep: { stepID: string; input: string };
};

export function useRerunFromStep(
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  { runID, fromStep }: RerunFromStep
) {
  const rerunFromStep = async ({
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    runID,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    fromStep,
  }: {
    runID: string;
    fromStep: { stepID: string; input: string };
  }): Promise<RerunResult> => {
    console.log('not yet implemented in the dashboard');
    return { data: { rerun: {} } };
  };

  return rerunFromStep;
}
