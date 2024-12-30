import type { RerunResult } from '@inngest/components/Rerun/RerunModal';
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

export function useRerunFromStep({ runID, fromStep }: RerunFromStep) {
  const [, rerunMutation] = useMutation(rerun);

  const rerunFromStep = async ({
    runID,
    fromStep,
  }: {
    runID: string;
    fromStep: { stepID: string; input: string };
  }): Promise<RerunResult> => {
    const result = await rerunMutation({
      runID,
      fromStep: {
        stepID: fromStep.stepID,
        input: fromStep.input,
      },
    });
    return { data: { rerun: { id: result.data?.rerun || '' } } };
  };

  return rerunFromStep;
}
