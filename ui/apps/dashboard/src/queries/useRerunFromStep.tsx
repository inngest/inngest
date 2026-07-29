import type { RerunFromStepPayload } from '@inngest/components/SharedContext/useRerunFromStep';
import { useMutation } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

const rerun = graphql(`
  mutation Rerun($envID: UUID!, $runID: ULID!, $fromStep: RerunFromStepInput) {
    rerun(envID: $envID, runID: $runID, fromStep: $fromStep)
  }
`);

export const useRerunFromStep = () => {
  const env = useEnvironment();
  const [, rerunMutation] = useMutation(rerun);

  const rerunFromStep = async ({ runID, fromStep }: RerunFromStepPayload) => {
    try {
      const result = await rerunMutation({
        envID: env.id,
        runID,
        fromStep: {
          stepID: fromStep.stepID,
          input: fromStep.input,
        },
      });

      return {
        ...result,
        redirect: result.data?.rerun
          ? pathCreator.runPopout({
              envSlug: env.slug,
              runID: result.data.rerun,
            })
          : undefined,
      };
    } catch (error) {
      console.error('error rerunning from step', error);
      return {
        error:
          error instanceof Error
            ? error
            : new Error('Error rerunning from step'),
        data: undefined,
      };
    }
  };

  return rerunFromStep;
};
