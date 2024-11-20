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
  }): Promise<void> => {
    console.log('not yet implemented in the dashboard');
  };

  return rerunFromStep;
}
