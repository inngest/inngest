type RerunFromStep = {
  runID: string;
  fromStep: { stepID: string; input: string };
};

export function useRerunFromStep({ runID, fromStep }: RerunFromStep) {
  const rerunFromStep = async ({
    runID,
    fromStep,
  }: {
    runID: string;
    fromStep: { stepID: string; input: string };
  }): Promise<void> => {
    console.log('not yet implemented in the dashboard');
  };

  return rerunFromStep;
}
