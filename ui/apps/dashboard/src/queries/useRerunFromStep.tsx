import { useMutation } from 'urql';

import { graphql } from '@/gql';

const rerun = graphql(`
  mutation Rerun($runID: ULID!, $fromStep: RerunFromStepInput) {
    rerun(runID: $runID, fromStep: $fromStep)
  }
`);

type RerunFromStep = {
  runID: string;
  fromStep: { stepID: string; input: string };
};

export function useRerunFromStep() {
  const [, rerunMutation] = useMutation(rerun);

  const rerunFromStep = async ({ runID, fromStep }: RerunFromStep) => {
    const result = await rerunMutation({
      runID,
      fromStep: {
        stepID: fromStep.stepID,
        input: fromStep.input,
      },
    });

    return result;
  };

  return rerunFromStep;
}
